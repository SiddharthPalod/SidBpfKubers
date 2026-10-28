package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
	"sidkubers.io/operator/internal/resources"
	"sidkubers.io/operator/internal/security"
)

const databaseFinalizer = "database.sidkubers.io/finalizer"

// DatabaseClusterReconciler orchestrates lifecycle reconciliation for DatabaseCluster
type DatabaseClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile handles observed state versus desired state
func (r *DatabaseClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Printf("--> [Reconcile] Triggered for: %s\n", req.NamespacedName)

	// 1. Fetch DatabaseCluster
	var db databasev1alpha1.DatabaseCluster
	if err := r.Get(ctx, req.NamespacedName, &db); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Finalizer lifecycle management
	if result, err := r.handleFinalizers(ctx, &db); err != nil || result.Requeue || !db.DeletionTimestamp.IsZero() {
		return result, err
	}

	// 3. Secret hash for automated rotation
	secretHash := r.fetchSecretHash(ctx, &db)

	// 4. Ensure Init Script ConfigMap (Sync/Async WAL Streaming Replication)
	if err := r.ensureConfigMap(ctx, &db, resources.DesiredInitScriptConfigMap(&db)); err != nil {
		return ctrl.Result{}, err
	}

	// 5. Ensure CNPG Three-Tier Services (-rw, -ro, -r) + Headless DNS
	if err := r.ensureThreeTierServices(ctx, &db); err != nil {
		return ctrl.Result{}, err
	}

	// 6. Reconcile StatefulSet
	sts, stsFound, err := r.reconcileStatefulSet(ctx, &db, secretHash)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 7. Failover / Switchover & Primary Routing
	// Primary is stored durably in db.Status.Primary
	activePrimary := db.Status.Primary
	if activePrimary == "" {
		activePrimary = fmt.Sprintf("%s-0", db.Name)
	}

	// Failover check: if activePrimary is unhealthy, promote other pod
	activePrimary = r.detectFailover(ctx, &db, sts, activePrimary)

	// Planned Switchover check: if annotation set, intentionally flip primary
	if db.Annotations != nil && db.Annotations["database.sidkubers.io/switchover"] == "true" {
		activePrimary = r.executeSwitchover(ctx, &db, activePrimary)
	}

	// Sync pod role labels (role: primary / role: replica)
	r.syncPodRoleLabels(ctx, &db, activePrimary)

	// 8. On-demand Backup / Restore Triggers
	r.handleBackupRestoreTriggers(ctx, &db)

	// 9. Update observed Status & Conditions
	if stsFound {
		if err := r.updateClusterStatus(ctx, req, &db, sts, activePrimary); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseClusterReconciler) ensureThreeTierServices(ctx context.Context, db *databasev1alpha1.DatabaseCluster) error {
	svcs := []*corev1.Service{
		resources.DesiredHeadlessService(db),
		resources.DesiredRWService(db),
		resources.DesiredROService(db),
		resources.DesiredRService(db),
	}
	for _, svc := range svcs {
		if err := r.ensureService(ctx, db, svc); err != nil {
			return err
		}
	}
	return nil
}

func (r *DatabaseClusterReconciler) syncPodRoleLabels(ctx context.Context, db *databasev1alpha1.DatabaseCluster, primaryPod string) {
	for i := int32(0); i < db.Spec.Replicas; i++ {
		podName := fmt.Sprintf("%s-%d", db.Name, i)
		var pod corev1.Pod
		if err := r.Get(ctx, client.ObjectKey{Namespace: db.Namespace, Name: podName}, &pod); err == nil {
			expectedRole := "replica"
			if podName == primaryPod {
				expectedRole = "primary"
			}
			if pod.Labels == nil {
				pod.Labels = make(map[string]string)
			}
			if pod.Labels["role"] != expectedRole {
				pod.Labels["role"] = expectedRole
				fmt.Printf("    Setting pod label %s -> role: %s\n", podName, expectedRole)
				_ = r.Update(ctx, &pod)
			}
		}
	}
}

func (r *DatabaseClusterReconciler) executeSwitchover(ctx context.Context, db *databasev1alpha1.DatabaseCluster, currentPrimary string) string {
	candidate := fmt.Sprintf("%s-1", db.Name)
	if currentPrimary == fmt.Sprintf("%s-1", db.Name) {
		candidate = fmt.Sprintf("%s-0", db.Name)
	}

	var candidatePod corev1.Pod
	if err := r.Get(ctx, client.ObjectKey{Namespace: db.Namespace, Name: candidate}, &candidatePod); err == nil && r.isPodReady(&candidatePod) {
		fmt.Printf("    [SWITCHOVER] Executing planned promotion of %s to Primary\n", candidate)
		delete(db.Annotations, "database.sidkubers.io/switchover")
		_ = r.Update(ctx, db)
		return candidate
	}

	fmt.Printf("    [SWITCHOVER] Candidate %s is not ready, aborting switchover\n", candidate)
	delete(db.Annotations, "database.sidkubers.io/switchover")
	_ = r.Update(ctx, db)
	return currentPrimary
}

func (r *DatabaseClusterReconciler) detectFailover(ctx context.Context, db *databasev1alpha1.DatabaseCluster, sts *appsv1.StatefulSet, currentPrimary string) string {
	if db.Spec.Replicas <= 1 {
		return currentPrimary
	}

	var primaryPod corev1.Pod
	if err := r.Get(ctx, client.ObjectKey{Namespace: db.Namespace, Name: currentPrimary}, &primaryPod); err == nil {
		if !r.isPodReady(&primaryPod) {
			// Current primary is unhealthy, look for standby candidate
			candidate := fmt.Sprintf("%s-1", db.Name)
			if currentPrimary == fmt.Sprintf("%s-1", db.Name) {
				candidate = fmt.Sprintf("%s-0", db.Name)
			}

			var candidatePod corev1.Pod
			if err := r.Get(ctx, client.ObjectKey{Namespace: db.Namespace, Name: candidate}, &candidatePod); err == nil && r.isPodReady(&candidatePod) {
				fmt.Printf("    [FAILOVER] Primary %s UNHEALTHY! Auto-promoting %s\n", currentPrimary, candidate)
				return candidate
			}
		}
	}
	return currentPrimary
}

func (r *DatabaseClusterReconciler) handleFinalizers(ctx context.Context, db *databasev1alpha1.DatabaseCluster) (ctrl.Result, error) {
	if db.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(db, databaseFinalizer) {
			controllerutil.AddFinalizer(db, databaseFinalizer)
			fmt.Printf("    Added finalizer: %s\n", databaseFinalizer)
			return ctrl.Result{Requeue: true}, r.Update(ctx, db)
		}
	} else {
		if controllerutil.ContainsFinalizer(db, databaseFinalizer) {
			fmt.Printf("    [Finalizer] Running pre-delete cleanup for %s...\n", db.Name)
			controllerutil.RemoveFinalizer(db, databaseFinalizer)
			return ctrl.Result{}, r.Update(ctx, db)
		}
	}
	return ctrl.Result{}, nil
}

func (r *DatabaseClusterReconciler) fetchSecretHash(ctx context.Context, db *databasev1alpha1.DatabaseCluster) string {
	var secret corev1.Secret
	secretKey := client.ObjectKey{Namespace: db.Namespace, Name: db.Spec.CredentialsSecretRef.Name}
	if err := r.Get(ctx, secretKey, &secret); err == nil {
		return security.HashSecret(&secret)
	}
	return ""
}

func (r *DatabaseClusterReconciler) ensureConfigMap(ctx context.Context, db *databasev1alpha1.DatabaseCluster, desired *corev1.ConfigMap) error {
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(existing), existing); err != nil {
		if apierrors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(db, desired, r.Scheme); err != nil {
				return err
			}
			fmt.Printf("    Creating ConfigMap: %s\n", desired.Name)
			return r.Create(ctx, desired)
		}
		return err
	}

	if existing.Data["entrypoint.sh"] != desired.Data["entrypoint.sh"] {
		existing.Data = desired.Data
		fmt.Printf("    Updating ConfigMap: %s with latest entrypoint script\n", existing.Name)
		return r.Update(ctx, existing)
	}
	return nil
}

func (r *DatabaseClusterReconciler) reconcileStatefulSet(ctx context.Context, db *databasev1alpha1.DatabaseCluster, secretHash string) (*appsv1.StatefulSet, bool, error) {
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name,
			Namespace: db.Namespace,
		},
	}
	desired := resources.DesiredStatefulSet(db, secretHash)

	stsFound := true
	if err := r.Get(ctx, client.ObjectKeyFromObject(sts), sts); err != nil {
		if apierrors.IsNotFound(err) {
			stsFound = false
			sts = desired
			if err := ctrl.SetControllerReference(db, sts, r.Scheme); err != nil {
				return nil, false, err
			}
			fmt.Printf("    Creating StatefulSet: %s (replicas: %d)\n", sts.Name, *sts.Spec.Replicas)
			if err := r.Create(ctx, sts); err != nil {
				return nil, false, err
			}
		} else {
			return nil, false, err
		}
	}

	if stsFound {
		needsUpdate := false

		if *sts.Spec.Replicas != db.Spec.Replicas {
			fmt.Printf("    Scaling StatefulSet: %d -> %d\n", *sts.Spec.Replicas, db.Spec.Replicas)
			sts.Spec.Replicas = &db.Spec.Replicas
			needsUpdate = true
		}

		if len(sts.Spec.Template.Spec.Containers) > 0 && len(desired.Spec.Template.Spec.Containers) > 0 {
			target := &sts.Spec.Template.Spec.Containers[0]
			desiredContainer := desired.Spec.Template.Spec.Containers[0]

			if target.Image != desiredContainer.Image {
				target.Image = desiredContainer.Image
				needsUpdate = true
			}

			if target.LivenessProbe == nil || target.ReadinessProbe == nil {
				target.LivenessProbe = desiredContainer.LivenessProbe
				target.ReadinessProbe = desiredContainer.ReadinessProbe
				needsUpdate = true
			}

			if target.Resources.Requests == nil || target.Resources.Limits == nil {
				target.Resources = desiredContainer.Resources
				needsUpdate = true
			}

			if len(target.Command) == 0 {
				target.Command = desiredContainer.Command
				needsUpdate = true
			}
		}

		if len(sts.Spec.Template.Spec.TopologySpreadConstraints) == 0 {
			sts.Spec.Template.Spec.TopologySpreadConstraints = desired.Spec.Template.Spec.TopologySpreadConstraints
			needsUpdate = true
		}

		if len(sts.Spec.Template.Spec.Volumes) < len(desired.Spec.Template.Spec.Volumes) {
			sts.Spec.Template.Spec.Volumes = desired.Spec.Template.Spec.Volumes
			needsUpdate = true
		}

		if secretHash != "" {
			if sts.Spec.Template.Annotations == nil {
				sts.Spec.Template.Annotations = make(map[string]string)
			}
			if sts.Spec.Template.Annotations["database.sidkubers.io/secret-hash"] != secretHash {
				sts.Spec.Template.Annotations["database.sidkubers.io/secret-hash"] = secretHash
				needsUpdate = true
			}
		}

		if needsUpdate {
			if err := r.Update(ctx, sts); err != nil {
				return nil, true, err
			}
		}
	}

	return sts, stsFound, nil
}

func (r *DatabaseClusterReconciler) ensureService(ctx context.Context, db *databasev1alpha1.DatabaseCluster, desired *corev1.Service) error {
	existing := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(existing), existing); err != nil {
		if apierrors.IsNotFound(err) {
			if err := ctrl.SetControllerReference(db, desired, r.Scheme); err != nil {
				return err
			}
			fmt.Printf("    Creating Service: %s\n", desired.Name)
			return r.Create(ctx, desired)
		}
		return err
	}
	return nil
}

func (r *DatabaseClusterReconciler) isPodReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *DatabaseClusterReconciler) handleBackupRestoreTriggers(ctx context.Context, db *databasev1alpha1.DatabaseCluster) {
	if db.Annotations == nil {
		return
	}

	if db.Annotations["database.sidkubers.io/trigger-backup"] == "true" {
		job := resources.NewBackupJob(db)
		if err := ctrl.SetControllerReference(db, job, r.Scheme); err == nil {
			if err := r.Create(ctx, job); err == nil {
				fmt.Printf("    [BACKUP] Triggered Job: %s\n", job.Name)
				delete(db.Annotations, "database.sidkubers.io/trigger-backup")
				_ = r.Update(ctx, db)
			}
		}
	}

	if db.Annotations["database.sidkubers.io/trigger-restore"] == "true" {
		job := resources.NewRestoreJob(db)
		if err := ctrl.SetControllerReference(db, job, r.Scheme); err == nil {
			if err := r.Create(ctx, job); err == nil {
				fmt.Printf("    [RESTORE] Triggered Job: %s\n", job.Name)
				delete(db.Annotations, "database.sidkubers.io/trigger-restore")
				_ = r.Update(ctx, db)
			}
		}
	}
}

func (r *DatabaseClusterReconciler) updateClusterStatus(ctx context.Context, req ctrl.Request, db *databasev1alpha1.DatabaseCluster, sts *appsv1.StatefulSet, primary string) error {
	newStatus := db.Status
	newStatus.ReadyReplicas = sts.Status.ReadyReplicas
	newStatus.Primary = primary

	if sts.Status.ReadyReplicas == db.Spec.Replicas && db.Spec.Replicas > 0 {
		newStatus.Phase = "Running"
		meta.SetStatusCondition(&newStatus.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "MinimumReplicasAvailable",
			Message:            fmt.Sprintf("%d/%d replicas ready and serving", sts.Status.ReadyReplicas, db.Spec.Replicas),
			LastTransitionTime: metav1.Now(),
		})
	} else {
		newStatus.Phase = "Creating"
		meta.SetStatusCondition(&newStatus.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "ReplicasNotReady",
			Message:            fmt.Sprintf("%d/%d replicas ready", sts.Status.ReadyReplicas, db.Spec.Replicas),
			LastTransitionTime: metav1.Now(),
		})
	}

	if db.Status.Phase != newStatus.Phase || db.Status.ReadyReplicas != newStatus.ReadyReplicas || db.Status.Primary != newStatus.Primary || len(db.Status.Conditions) != len(newStatus.Conditions) {
		fmt.Printf("    Updating CR Status: Phase=%s, ReadyReplicas=%d, Primary=%s\n", newStatus.Phase, newStatus.ReadyReplicas, newStatus.Primary)
		for attempt := 0; attempt < 3; attempt++ {
			var latestDb databasev1alpha1.DatabaseCluster
			if err := r.Get(ctx, req.NamespacedName, &latestDb); err != nil {
				return err
			}
			latestDb.Status = newStatus
			err := r.Status().Update(ctx, &latestDb)
			if err == nil {
				break
			}
			if !apierrors.IsConflict(err) {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.DatabaseCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

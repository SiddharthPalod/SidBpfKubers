package resources

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// ResolveVersion defaults version to 16 if unspecified
func ResolveVersion(v string) string {
	if v == "" {
		return "16"
	}
	return v
}

// DesiredImage formats the PostgreSQL image tag
func DesiredImage(version string) string {
	return fmt.Sprintf("postgres:%s-alpine", ResolveVersion(version))
}

// DesiredStatefulSet constructs hardened PostgreSQL StatefulSet with probes, storage, limits, and WAL replication entrypoint
func DesiredStatefulSet(db *databasev1alpha1.DatabaseCluster, secretHash string) *appsv1.StatefulSet {
	storageSize := db.Spec.Storage
	if storageSize == "" {
		storageSize = "10Gi"
	}

	replicas := db.Spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	labels := map[string]string{
		"app": db.Name,
	}

	annotations := make(map[string]string)
	if secretHash != "" {
		annotations["database.sidkubers.io/secret-hash"] = secretHash
	}

	scriptMode := int32(0755)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name,
			Namespace: db.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: db.Name + "-headless",
			Replicas:    &replicas,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       "kubernetes.io/hostname",
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: labels,
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "postgres",
							Image:   DesiredImage(db.Spec.Version),
							Command: []string{"/bin/sh", "/scripts/entrypoint.sh"},
							Ports: []corev1.ContainerPort{
								{
									Name:          "postgres",
									ContainerPort: 5432,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"pg_isready", "-U", "postgres", "-h", "127.0.0.1"},
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       10,
								TimeoutSeconds:      3,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"pg_isready", "-U", "postgres", "-h", "127.0.0.1"},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
								TimeoutSeconds:      3,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PGDATA",
									Value: "/var/lib/postgresql/data/pgdata",
								},
								{
									Name: "POSTGRES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: db.Spec.CredentialsSecretRef.Name,
											},
											Key: "username",
										},
									},
								},
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: db.Spec.CredentialsSecretRef.Name,
											},
											Key: "password",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/lib/postgresql/data",
								},
								{
									Name:      "scripts",
									MountPath: "/scripts",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: db.Name + "-init-script",
									},
									DefaultMode: &scriptMode,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(storageSize),
							},
						},
					},
				},
			},
		},
	}

	if db.Spec.StorageClassName != nil && *db.Spec.StorageClassName != "" {
		sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = db.Spec.StorageClassName
	}

	return sts
}

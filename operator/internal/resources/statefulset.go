package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
	"sidkubers.io/operator/internal/engines"
)

// DesiredStatefulSet constructs hardened StatefulSet using the configured Engine strategy
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

	eng := engines.GetEngine(db.GetEngine())
	container := eng.BuildContainer(db)
	volumes := eng.Volumes(db)

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
					Containers: []corev1.Container{container},
					Volumes:    volumes,
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

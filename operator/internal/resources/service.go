package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// DesiredHeadlessService generates headless service for stable StatefulSet DNS
func DesiredHeadlessService(db *databasev1alpha1.DatabaseCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-headless",
			Namespace: db.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: corev1.ClusterIPNone,
			Selector: map[string]string{
				"app": db.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name: "postgres",
					Port: 5432,
				},
			},
		},
	}
}

// DesiredRWService routes traffic exclusively to the writable Primary (CNPG -rw pattern)
func DesiredRWService(db *databasev1alpha1.DatabaseCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-rw",
			Namespace: db.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app":  db.Name,
				"role": "primary",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "postgres",
					Port: 5432,
				},
			},
		},
	}
}

// DesiredROService routes read-only queries exclusively to Standby Replicas (CNPG -ro pattern)
func DesiredROService(db *databasev1alpha1.DatabaseCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-ro",
			Namespace: db.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app":  db.Name,
				"role": "replica",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "postgres",
					Port: 5432,
				},
			},
		},
	}
}

// DesiredRService load-balances read queries across ALL instances: Primary + Replicas (CNPG -r pattern)
func DesiredRService(db *databasev1alpha1.DatabaseCluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-r",
			Namespace: db.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": db.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name: "postgres",
					Port: 5432,
				},
			},
		},
	}
}

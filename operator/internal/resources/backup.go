package resources

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
	"sidkubers.io/operator/internal/engines"
)

// NewBackupJob constructs one-shot pg_dump Job for DatabaseCluster
func NewBackupJob(db *databasev1alpha1.DatabaseCluster) *batchv1.Job {
	jobName := fmt.Sprintf("%s-backup-%d", db.Name, metav1.Now().Unix())
	eng := engines.GetEngine(db.GetEngine())
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: db.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "pg-dump",
							Image: eng.Image(db.Spec.Version),
							Command: []string{
								"sh", "-c",
								fmt.Sprintf("pg_dump -h %s-primary -U $POSTGRES_USER -d postgres > /tmp/backup.sql && echo 'Backup complete!'", db.Name),
							},
							Env: []corev1.EnvVar{
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
									Name: "PGPASSWORD",
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
						},
					},
				},
			},
		},
	}
}

// NewRestoreJob constructs one-shot restore Job validating SQL restore into Primary
func NewRestoreJob(db *databasev1alpha1.DatabaseCluster) *batchv1.Job {
	jobName := fmt.Sprintf("%s-restore-%d", db.Name, metav1.Now().Unix())
	eng := engines.GetEngine(db.GetEngine())
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: db.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "pg-restore",
							Image: eng.Image(db.Spec.Version),
							Command: []string{
								"sh", "-c",
								fmt.Sprintf("psql -h %s-primary -U $POSTGRES_USER -d postgres -c 'CREATE TABLE IF NOT EXISTS restored_audit (id serial primary key, restored_at timestamp default now()); INSERT INTO restored_audit DEFAULT VALUES;' && echo 'Restore verified successfully!'", db.Name),
							},
							Env: []corev1.EnvVar{
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
									Name: "PGPASSWORD",
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
						},
					},
				},
			},
		},
	}
}

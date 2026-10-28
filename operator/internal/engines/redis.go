package engines

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// RedisEngine implements Engine strategy for Redis (Master/Replica replication)
type RedisEngine struct{}

func (r *RedisEngine) Name() string {
	return "redis"
}

func (r *RedisEngine) Port() int32 {
	return 6379
}

func (r *RedisEngine) PortName() string {
	return "redis"
}

func (r *RedisEngine) VolumeMountPath() string {
	return "/data"
}

func (r *RedisEngine) Image(version string) string {
	if version == "" {
		version = "7"
	}
	return fmt.Sprintf("redis:%s-alpine", version)
}

func (r *RedisEngine) BuildInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
	masterHost := fmt.Sprintf("%s-0.%s-headless", db.Name, db.Name)

	script := fmt.Sprintf(`#!/bin/sh
set -e

HOSTNAME=$(hostname)
MASTER_HOST="%s"

echo "==> Starting Redis node: $HOSTNAME"
mkdir -p /data

AUTH_ARGS=""
if [ -n "$REDIS_PASSWORD" ]; then
    AUTH_ARGS="--requirepass $REDIS_PASSWORD --masterauth $REDIS_PASSWORD"
fi

if [ "$HOSTNAME" = "%s-0" ]; then
    echo "==> Node is MASTER (RW). Launching redis-server..."
    exec redis-server --protected-mode no --dir /data $AUTH_ARGS
else
    echo "==> Node is REPLICA (RO). Waiting for Master $MASTER_HOST:6379 to be ready..."
    until nc -z -w 2 "$MASTER_HOST" 6379; do
        echo "    Master not reachable yet, sleeping 2s..."
        sleep 2
    done

    echo "==> Master reachable. Starting replicaof $MASTER_HOST 6379..."
    exec redis-server --protected-mode no --dir /data --replicaof "$MASTER_HOST" 6379 $AUTH_ARGS
fi
`, masterHost, db.Name)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-init-script",
			Namespace: db.Namespace,
		},
		Data: map[string]string{
			"entrypoint.sh": script,
		},
	}
}

func (r *RedisEngine) BuildContainer(db *databasev1alpha1.DatabaseCluster) corev1.Container {
	var envVars []corev1.EnvVar

	if db.Spec.CredentialsSecretRef.Name != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "REDIS_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: db.Spec.CredentialsSecretRef.Name,
					},
					Key: "password",
				},
			},
		})
	}

	probeCmd := []string{"redis-cli", "ping"}

	return corev1.Container{
		Name:    "redis",
		Image:   r.Image(db.Spec.Version),
		Command: []string{"/bin/sh", "/scripts/entrypoint.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          r.PortName(),
				ContainerPort: r.Port(),
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("300m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       10,
			TimeoutSeconds:      3,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			InitialDelaySeconds: 3,
			PeriodSeconds:       5,
			TimeoutSeconds:      3,
		},
		Env: envVars,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: r.VolumeMountPath(),
			},
			{
				Name:      "scripts",
				MountPath: "/scripts",
			},
		},
	}
}

func (r *RedisEngine) Volumes(db *databasev1alpha1.DatabaseCluster) []corev1.Volume {
	scriptMode := int32(0755)
	return []corev1.Volume{
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
	}
}

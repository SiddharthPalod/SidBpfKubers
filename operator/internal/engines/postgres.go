package engines

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// PostgresEngine implements Engine strategy for PostgreSQL
type PostgresEngine struct{}

func (p *PostgresEngine) Name() string {
	return "postgres"
}

func (p *PostgresEngine) Port() int32 {
	return 5432
}

func (p *PostgresEngine) PortName() string {
	return "postgres"
}

func (p *PostgresEngine) VolumeMountPath() string {
	return "/var/lib/postgresql/data"
}

func (p *PostgresEngine) Image(version string) string {
	if version == "" {
		version = "16"
	}
	return fmt.Sprintf("postgres:%s-alpine", version)
}

func (p *PostgresEngine) BuildInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
	primaryHost := fmt.Sprintf("%s-0.%s-headless", db.Name, db.Name)

	syncFlags := ""
	if db.Spec.Replication != nil && db.Spec.Replication.Mode == "sync" {
		syncFlags = "-c synchronous_commit=on -c synchronous_standby_names='FIRST 1 (*)'"
	}

	script := fmt.Sprintf(`#!/bin/sh
set -e

HOSTNAME=$(hostname)
PRIMARY_HOST="%s"
export PGDATA="/var/lib/postgresql/data/pgdata"

echo "==> Starting PostgreSQL node: $HOSTNAME (PGDATA=$PGDATA)"
mkdir -p "$PGDATA"
chmod 700 "$PGDATA"

if [ "$HOSTNAME" = "%s-0" ]; then
    echo "==> Node is PRIMARY. Ensuring pg_hba replication permissions..."
    mkdir -p /docker-entrypoint-initdb.d
    cat <<'EOF' > /docker-entrypoint-initdb.d/01-replication.sh
#!/bin/sh
echo "host replication all 0.0.0.0/0 trust" >> "$PGDATA/pg_hba.conf"
EOF
    chmod +x /docker-entrypoint-initdb.d/01-replication.sh

    if [ -f "$PGDATA/pg_hba.conf" ]; then
        grep -q "host replication" "$PGDATA/pg_hba.conf" || echo "host replication all 0.0.0.0/0 trust" >> "$PGDATA/pg_hba.conf"
    fi

    exec docker-entrypoint.sh postgres \
        -c wal_level=replica \
        -c max_wal_senders=10 \
        -c max_replication_slots=10 \
        -c hot_standby=on \
        %s
else
    echo "==> Node is REPLICA. Waiting for Primary $PRIMARY_HOST to be ready..."
    until pg_isready -h "$PRIMARY_HOST" -U "$POSTGRES_USER" -t 2; do
        echo "    Primary not ready yet, sleeping 2s..."
        sleep 2
    done

    # Clean partial attempts to prevent 'directory exists but is not empty' error
    if [ ! -s "$PGDATA/PG_VERSION" ]; then
        echo "==> Cleaning stale files and cloning primary via pg_basebackup with -R (standby mode)..."
        rm -rf "${PGDATA:?}"/*
        PGPASSWORD="$POSTGRES_PASSWORD" pg_basebackup -h "$PRIMARY_HOST" -D "$PGDATA" -U "$POSTGRES_USER" -v -P -R -X stream
    fi

    chmod 700 "$PGDATA"
    echo "==> Standby clone ready. Launching PostgreSQL streaming replica..."
    exec docker-entrypoint.sh postgres -c hot_standby=on
fi
`, primaryHost, db.Name, syncFlags)

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

func (p *PostgresEngine) BuildContainer(db *databasev1alpha1.DatabaseCluster) corev1.Container {
	envVars := []corev1.EnvVar{
		{
			Name:  "PGDATA",
			Value: "/var/lib/postgresql/data/pgdata",
		},
	}

	if db.Spec.CredentialsSecretRef.Name != "" {
		envVars = append(envVars,
			corev1.EnvVar{
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
			corev1.EnvVar{
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
		)
	}

	return corev1.Container{
		Name:    "postgres",
		Image:   p.Image(db.Spec.Version),
		Command: []string{"/bin/sh", "/scripts/entrypoint.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          p.PortName(),
				ContainerPort: p.Port(),
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
		Env: envVars,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: p.VolumeMountPath(),
			},
			{
				Name:      "scripts",
				MountPath: "/scripts",
			},
		},
	}
}

func (p *PostgresEngine) Volumes(db *databasev1alpha1.DatabaseCluster) []corev1.Volume {
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

package resources

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// DesiredInitScriptConfigMap generates entrypoint script managing WAL replication and sync/async mode
func DesiredInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
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
        rm -rf "$PGDATA"/*
        PGPASSWORD="$POSTGRES_PASSWORD" pg_basebackup -h "$PRIMARY_HOST" -U "$POSTGRES_USER" -D "$PGDATA" -Fp -Xs -R
        chmod 700 "$PGDATA"
    fi

    echo "==> Booting standby PostgreSQL..."
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

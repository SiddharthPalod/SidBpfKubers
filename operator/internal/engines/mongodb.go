package engines

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// MongoDBEngine implements Engine strategy for MongoDB (Replica Set replication)
type MongoDBEngine struct{}

func (m *MongoDBEngine) Name() string {
	return "mongodb"
}

func (m *MongoDBEngine) Port() int32 {
	return 27017
}

func (m *MongoDBEngine) PortName() string {
	return "mongodb"
}

func (m *MongoDBEngine) VolumeMountPath() string {
	return "/data/db"
}

func (m *MongoDBEngine) Image(version string) string {
	if version == "" {
		version = "7.0"
	}
	return fmt.Sprintf("mongo:%s", version)
}

func (m *MongoDBEngine) BuildInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
	replSetName := "rs0"
	headlessDomain := fmt.Sprintf("%s-headless", db.Name)

	peerJoinScript := ""
	replicas := int(db.Spec.Replicas)
	if replicas <= 0 {
		replicas = 3
	}

	if replicas > 1 {
		var peerIndices []string
		for i := 1; i < replicas; i++ {
			peerIndices = append(peerIndices, fmt.Sprintf("%d", i))
		}
		peerJoinScript = fmt.Sprintf(`
        for i in %s; do
            PEER="%s-$i.%s:27017"
            echo "==> Waiting to join peer: $PEER"
            while true; do
                RES=$(mongosh "mongodb://127.0.0.1:27017/?directConnection=true" --quiet --eval "try { const m = (rs.status().members||[]).map(x=>x.name); if(m.includes('$PEER')) { print('ALREADY_MEMBER'); } else { const r = rs.add('$PEER'); if(r.ok===1) print('ADDED_SUCCESS'); } } catch(e){}" 2>/dev/null || true)
                if echo "$RES" | grep -Eq "ALREADY_MEMBER|ADDED_SUCCESS"; then
                    echo "==> Peer $PEER replica set status: $RES"
                    break
                fi
                sleep 5
            done
        done`, strings.Join(peerIndices, " "), db.Name, headlessDomain)
	}

	script := fmt.Sprintf(`#!/bin/sh
HOSTNAME=$(hostname)
echo "==> Starting MongoDB Replica Set Node: $HOSTNAME"
mkdir -p /data/db

# Background replica set auto-initializer on node-0
if [ "$HOSTNAME" = "%s-0" ]; then
    (
        echo "==> Node-0: Waiting for local mongod to accept connections..."
        until mongosh "mongodb://127.0.0.1:27017/?directConnection=true" --quiet --eval "db.adminCommand('ping')" > /dev/null 2>&1; do
            sleep 2
        done
        echo "==> Node-0: Local mongod ready. Checking replica set status..."
        while true; do
            RS_OK=$(mongosh "mongodb://127.0.0.1:27017/?directConnection=true" --quiet --eval "try { rs.status().ok } catch(e) { 0 }" 2>/dev/null || echo "0")
            if [ "$RS_OK" = "1" ]; then
                echo "==> Replica set already active."
                break
            fi
            echo "==> Initiating single-node replica set '%s'..."
            if mongosh "mongodb://127.0.0.1:27017/?directConnection=true" --quiet --eval "rs.initiate({ _id: '%s', members: [{ _id: 0, host: '%s-0.%s:27017' }] })" 2>/dev/null; then
                echo "==> Replica set '%s' initiated successfully."
                break
            fi
            sleep 2
        done
%s
    ) &
fi

echo "==> Launching mongod --replSet %s --bind_ip_all --wiredTigerCacheSizeGB 0.25"
exec mongod --replSet %s --bind_ip_all --dbpath /data/db --wiredTigerCacheSizeGB 0.25
`, db.Name,
		replSetName, replSetName, db.Name, headlessDomain,
		replSetName,
		peerJoinScript,
		replSetName, replSetName)

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

func (m *MongoDBEngine) BuildContainer(db *databasev1alpha1.DatabaseCluster) corev1.Container {
	var envVars []corev1.EnvVar

	envVars = append(envVars, corev1.EnvVar{
		Name:  "NODE_OPTIONS",
		Value: "--max-old-space-size=128",
	})

	if db.Spec.CredentialsSecretRef.Name != "" {
		envVars = append(envVars,
			corev1.EnvVar{
				Name: "MONGO_INITDB_ROOT_USERNAME",
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
				Name: "MONGO_INITDB_ROOT_PASSWORD",
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

	probeCmd := []string{"mongosh", "mongodb://127.0.0.1:27017/?directConnection=true", "--quiet", "--eval", "db.adminCommand('ping')"}

	return corev1.Container{
		Name:    "mongodb",
		Image:   m.Image(db.Spec.Version),
		Command: []string{"/bin/sh", "/scripts/entrypoint.sh"},
		Ports: []corev1.ContainerPort{
			{
				Name:          m.PortName(),
				ContainerPort: m.Port(),
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1536Mi"),
			},
		},
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       5,
			TimeoutSeconds:      5,
			FailureThreshold:    30,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			PeriodSeconds:  15,
			TimeoutSeconds: 8,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			PeriodSeconds:  10,
			TimeoutSeconds: 8,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: m.VolumeMountPath(),
			},
			{
				Name:      "scripts",
				MountPath: "/scripts",
			},
		},
		Env: envVars,
	}
}

func (m *MongoDBEngine) Volumes(db *databasev1alpha1.DatabaseCluster) []corev1.Volume {
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

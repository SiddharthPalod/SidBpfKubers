package engines

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// ETCDEngine implements Engine strategy for etcd (Raft Consensus Quorum)
type ETCDEngine struct{}

func (e *ETCDEngine) Name() string {
	return "etcd"
}

func (e *ETCDEngine) Port() int32 {
	return 2379
}

func (e *ETCDEngine) PortName() string {
	return "client"
}

func (e *ETCDEngine) VolumeMountPath() string {
	return "/var/run/etcd"
}

func (e *ETCDEngine) Image(version string) string {
	if version == "" {
		version = "v3.5.12"
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return fmt.Sprintf("quay.io/coreos/etcd:%s", version)
}

func (e *ETCDEngine) BuildInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
	headlessDomain := fmt.Sprintf("%s-headless", db.Name)
	replicas := int(db.Spec.Replicas)
	if replicas <= 0 {
		replicas = 3
	}

	var initialClusterMembers []string
	for i := 0; i < replicas; i++ {
		member := fmt.Sprintf("%s-%d=http://%s-%d.%s:2380", db.Name, i, db.Name, i, headlessDomain)
		initialClusterMembers = append(initialClusterMembers, member)
	}
	initialClusterStr := strings.Join(initialClusterMembers, ",")

	script := fmt.Sprintf(`#!/bin/sh
set -e

HOSTNAME=$(hostname)
HEADLESS="%s"
echo "==> Starting etcd Raft Quorum Node: ${HOSTNAME}"
mkdir -p /var/run/etcd

PEER_URL="http://${HOSTNAME}.${HEADLESS}:2380"
CLIENT_URL="http://${HOSTNAME}.${HEADLESS}:2379"

echo "==> Raft Peer URL:   ${PEER_URL}"
echo "==> Client API URL:  ${CLIENT_URL}"
echo "==> Initial Cluster: %s"

exec etcd \
  --name "${HOSTNAME}" \
  --data-dir /var/run/etcd \
  --listen-client-urls "http://0.0.0.0:2379" \
  --advertise-client-urls "${CLIENT_URL},http://127.0.0.1:2379" \
  --listen-peer-urls "http://0.0.0.0:2380" \
  --initial-advertise-peer-urls "${PEER_URL}" \
  --initial-cluster "%s" \
  --initial-cluster-token "etcd-token-%s" \
  --initial-cluster-state new
`, headlessDomain, initialClusterStr, initialClusterStr, db.Name)

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

func (e *ETCDEngine) BuildContainer(db *databasev1alpha1.DatabaseCluster) corev1.Container {
	headlessDomain := fmt.Sprintf("%s-headless", db.Name)
	replicas := int(db.Spec.Replicas)
	if replicas <= 0 {
		replicas = 3
	}

	var initialClusterMembers []string
	for i := 0; i < replicas; i++ {
		member := fmt.Sprintf("%s-%d=http://%s-%d.%s:2380", db.Name, i, db.Name, i, headlessDomain)
		initialClusterMembers = append(initialClusterMembers, member)
	}
	initialClusterStr := strings.Join(initialClusterMembers, ",")

	probeCmd := []string{"etcdctl", "endpoint", "health"}

	return corev1.Container{
		Name:    "etcd",
		Image:   e.Image(db.Spec.Version),
		Command: []string{"etcd"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "client",
				ContainerPort: 2379,
			},
			{
				Name:          "peer",
				ContainerPort: 2380,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			InitialDelaySeconds: 2,
			PeriodSeconds:       2,
			TimeoutSeconds:      2,
			FailureThreshold:    20,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			PeriodSeconds:  10,
			TimeoutSeconds: 3,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: probeCmd,
				},
			},
			PeriodSeconds:  5,
			TimeoutSeconds: 3,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: e.VolumeMountPath(),
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
			{
				Name:  "ETCD_NAME",
				Value: "$(POD_NAME)",
			},
			{
				Name:  "ETCD_DATA_DIR",
				Value: "/var/run/etcd",
			},
			{
				Name:  "ETCD_LISTEN_CLIENT_URLS",
				Value: "http://0.0.0.0:2379",
			},
			{
				Name:  "ETCD_ADVERTISE_CLIENT_URLS",
				Value: fmt.Sprintf("http://$(POD_NAME).%s:2379,http://127.0.0.1:2379", headlessDomain),
			},
			{
				Name:  "ETCD_LISTEN_PEER_URLS",
				Value: "http://0.0.0.0:2380",
			},
			{
				Name:  "ETCD_INITIAL_ADVERTISE_PEER_URLS",
				Value: fmt.Sprintf("http://$(POD_NAME).%s:2380", headlessDomain),
			},
			{
				Name:  "ETCD_INITIAL_CLUSTER",
				Value: initialClusterStr,
			},
			{
				Name:  "ETCD_INITIAL_CLUSTER_TOKEN",
				Value: fmt.Sprintf("etcd-token-%s", db.Name),
			},
			{
				Name:  "ETCD_INITIAL_CLUSTER_STATE",
				Value: "new",
			},
			{
				Name:  "ETCDCTL_API",
				Value: "3",
			},
		},
	}
}

func (e *ETCDEngine) Volumes(db *databasev1alpha1.DatabaseCluster) []corev1.Volume {
	return nil
}

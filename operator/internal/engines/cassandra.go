package engines

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// CassandraEngine implements Engine strategy for Apache Cassandra (Masterless Peer-to-Peer DHT Hash Ring)
type CassandraEngine struct{}

func (e *CassandraEngine) Name() string {
	return "cassandra"
}

func (e *CassandraEngine) Port() int32 {
	return 9042
}

func (e *CassandraEngine) PortName() string {
	return "cql"
}

func (e *CassandraEngine) VolumeMountPath() string {
	return "/var/lib/cassandra"
}

func (e *CassandraEngine) Image(version string) string {
	if version == "" {
		version = "4.1"
	}
	return fmt.Sprintf("cassandra:%s", version)
}

func (e *CassandraEngine) BuildInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-init-script",
			Namespace: db.Namespace,
		},
		Data: map[string]string{
			"entrypoint.sh": "#!/bin/sh\n# Cassandra initializes natively via environment variables\n",
		},
	}
}

func (e *CassandraEngine) BuildContainer(db *databasev1alpha1.DatabaseCluster) corev1.Container {
	headlessDomain := fmt.Sprintf("%s-headless", db.Name)
	seedNode := fmt.Sprintf("%s-0.%s", db.Name, headlessDomain)

	return corev1.Container{
		Name:  "cassandra",
		Image: e.Image(db.Spec.Version),
		Ports: []corev1.ContainerPort{
			{
				Name:          "cql",
				ContainerPort: 9042,
			},
			{
				Name:          "intra-node",
				ContainerPort: 7000,
			},
			{
				Name:          "jmx",
				ContainerPort: 7199,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1536Mi"),
			},
		},
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(9042),
				},
			},
			InitialDelaySeconds: 20,
			PeriodSeconds:       5,
			TimeoutSeconds:      2,
			FailureThreshold:    70,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(9042),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
			TimeoutSeconds:      2,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(9042),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       5,
			TimeoutSeconds:      2,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: e.VolumeMountPath(),
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
			{
				Name:  "CASSANDRA_CLUSTER_NAME",
				Value: fmt.Sprintf("%s-ring", db.Name),
			},
			{
				Name:  "CASSANDRA_NUM_TOKENS",
				Value: "128",
			},
			{
				Name:  "CASSANDRA_SEEDS",
				Value: seedNode,
			},
			{
				Name:  "CASSANDRA_DC",
				Value: "datacenter1",
			},
			{
				Name:  "CASSANDRA_RACK",
				Value: "rack1",
			},
			{
				Name:  "CASSANDRA_ENDPOINT_SNITCH",
				Value: "GossipingPropertyFileSnitch",
			},
			{
				Name:  "MAX_HEAP_SIZE",
				Value: "512M",
			},
			{
				Name:  "HEAP_NEWSIZE",
				Value: "128M",
			},
		},
	}
}

func (e *CassandraEngine) Volumes(db *databasev1alpha1.DatabaseCluster) []corev1.Volume {
	return nil
}

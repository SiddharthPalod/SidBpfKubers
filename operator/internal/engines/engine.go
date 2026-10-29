package engines

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
)

// Engine defines the strategy interface for database storage engines (PostgreSQL, Redis, etc.)
type Engine interface {
	// Name returns the canonical engine name ("postgres" or "redis")
	Name() string

	// Port returns the primary service port (e.g. 5432, 6379)
	Port() int32

	// PortName returns the descriptive port identifier
	PortName() string

	// Image returns the container image tag for the specified version
	Image(version string) string

	// BuildInitScriptConfigMap generates the engine-specific init/entrypoint ConfigMap
	BuildInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap

	// BuildContainer constructs the primary workload container specification
	BuildContainer(db *databasev1alpha1.DatabaseCluster) corev1.Container

	// Volumes returns engine-specific pod volumes
	Volumes(db *databasev1alpha1.DatabaseCluster) []corev1.Volume

	// VolumeMountPath returns the mount point for persistent storage
	VolumeMountPath() string
}

// GetEngine returns the engine strategy based on the specification
func GetEngine(engineType string) Engine {
	switch strings.ToLower(strings.TrimSpace(engineType)) {
	case "redis":
		return &RedisEngine{}
	case "mongodb", "mongo":
		return &MongoDBEngine{}
	case "etcd":
		return &ETCDEngine{}
	case "postgres", "postgresql", "":
		return &PostgresEngine{}
	default:
		return &PostgresEngine{}
	}
}

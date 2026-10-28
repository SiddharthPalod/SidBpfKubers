package resources

import (
	corev1 "k8s.io/api/core/v1"

	databasev1alpha1 "sidkubers.io/operator/api/v1alpha1"
	"sidkubers.io/operator/internal/engines"
)

// DesiredInitScriptConfigMap delegates ConfigMap generation to the configured Engine strategy
func DesiredInitScriptConfigMap(db *databasev1alpha1.DatabaseCluster) *corev1.ConfigMap {
	eng := engines.GetEngine(db.GetEngine())
	return eng.BuildInitScriptConfigMap(db)
}

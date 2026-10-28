package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SecretReference points to a Secret in the same namespace
type SecretReference struct {
	Name string `json:"name"`
}

// ReplicationSpec defines replication behavior (CloudNativePG pattern)
type ReplicationSpec struct {
	// Mode is "async" (default) or "sync" (synchronous_commit with quorum)
	// +optional
	Mode string `json:"mode,omitempty"`
}

// DatabaseClusterSpec defines the desired state of DatabaseCluster
type DatabaseClusterSpec struct {
	// Replicas is the number of database pods (1-9)
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Version is PostgreSQL version ("15", "16", "17")
	// +optional
	Version string `json:"version,omitempty"`

	// Storage size (e.g. "10Gi")
	// +optional
	Storage string `json:"storage,omitempty"`

	// StorageClassName points to specific StorageClass
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Replication configures sync vs async replication
	// +optional
	Replication *ReplicationSpec `json:"replication,omitempty"`

	// CredentialsSecretRef references Secret containing POSTGRES_USER and POSTGRES_PASSWORD
	CredentialsSecretRef SecretReference `json:"credentialsSecretRef"`
}

// DatabaseClusterStatus defines the observed state of DatabaseCluster
type DatabaseClusterStatus struct {
	// Phase is current status: Creating, Running, Degraded, Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// ReadyReplicas is number of healthy running pods
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Primary is pod name of active writable master
	// +optional
	Primary string `json:"primary,omitempty"`

	// Conditions represent latest available observations of current state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// DatabaseCluster is the Schema for the databaseclusters API
type DatabaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseClusterSpec   `json:"spec,omitempty"`
	Status DatabaseClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// DatabaseClusterList contains a list of DatabaseCluster
type DatabaseClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseCluster `json:"items"`
}

// DeepCopyInto copies all properties of receiver into out.
func (in *DatabaseCluster) DeepCopyInto(out *DatabaseCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	if in.Spec.StorageClassName != nil {
		in, out := &in.Spec.StorageClassName, &out.Spec.StorageClassName
		*out = new(string)
		**out = **in
	}
	if in.Spec.Replication != nil {
		in, out := &in.Spec.Replication, &out.Spec.Replication
		*out = new(ReplicationSpec)
		**out = **in
	}
	out.Status = in.Status
	if in.Status.Conditions != nil {
		in, out := &in.Status.Conditions, &out.Status.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates deepcopy of DatabaseCluster.
func (in *DatabaseCluster) DeepCopy() *DatabaseCluster {
	if in == nil {
		return nil
	}
	out := new(DatabaseCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns generic runtime.Object.
func (in *DatabaseCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies all properties of receiver into out.
func (in *DatabaseClusterList) DeepCopyInto(out *DatabaseClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DatabaseCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates deepcopy of DatabaseClusterList.
func (in *DatabaseClusterList) DeepCopy() *DatabaseClusterList {
	if in == nil {
		return nil
	}
	out := new(DatabaseClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns generic runtime.Object.
func (in *DatabaseClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

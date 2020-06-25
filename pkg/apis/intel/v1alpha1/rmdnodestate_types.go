package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkloadMap stores string values of workload data for RmdNodeStatus
type WorkloadMap map[string]string

// RmdNodeStateSpec defines the desired state of RmdNodeState
type RmdNodeStateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Node    string `json:"node"`
	NodeUID string `json:"nodeUid"`
}

// RmdNodeStateStatus defines the observed state of RmdNodeState
type RmdNodeStateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Workloads map[string]WorkloadMap `json:"workloads"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RmdNodeState is the Schema for the rmdnodestates API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rmdnodestates,scope=Namespaced
type RmdNodeState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RmdNodeStateSpec   `json:"spec,omitempty"`
	Status RmdNodeStateStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RmdNodeStateList contains a list of RmdNodeState
type RmdNodeStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RmdNodeState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RmdNodeState{}, &RmdNodeStateList{})
}

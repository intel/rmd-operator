package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Cache defines cache parameters for workload
type Cache struct {
	Max int `json:"max"`
	Min int `json:"min"`
}

// Pstate defines pstate parametes for workload
type Pstate struct {
	Ratio      string `json:"ratio,omniempty"`
	Monitoring string `json:"monitoring,omniempty"`
}

// WorkloadState defines state of a workload for a single node
type WorkloadState struct {
	Response string `json:"response"`
	ID       string `json:"id"`
	CosName  string `json:"cosName"`
	Status   string `json:"status"`
}

// RmdWorkloadSpec defines the desired state of RmdWorkload
type RmdWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	CoreIds []string `json:"coreIds"`
	Policy  string   `json:"policy,omniempty"`
	Cache   Cache    `json:"cache"`
	Pstate  Pstate   `json:"pstate,omniempty"`
	Nodes   []string `json:"nodes"`
}

// RmdWorkloadStatus defines the observed state of RmdWorkload
type RmdWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	WorkloadStates map[string]WorkloadState `json:"workloadStates"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RmdWorkload is the Schema for the rmdworkloads API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rmdworkloads,scope=Namespaced
type RmdWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RmdWorkloadSpec   `json:"spec,omitempty"`
	Status RmdWorkloadStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RmdWorkloadList contains a list of RmdWorkload
type RmdWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RmdWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RmdWorkload{}, &RmdWorkloadList{})
}

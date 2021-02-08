package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Rdt related settings (Cache, MBA)
type Rdt struct {
	Cache Cache `json:"cache,omitempty"`
	Mba   Mba   `json:"mba,omitempty"`
}

// Cache defines cache parameters for workload
type Cache struct {
	Max int `json:"max,omitempty"`
	Min int `json:"min,omitempty"`
}

// Mba defines mba parameters for workload
type Mba struct {
	Percentage int `json:"percentage,omitempty"`
	Mbps       int `json:"mbps,omitempty"`
}

// Plugins contains individual RMD plugin types
type Plugins struct {
	Pstate Pstate `json:"pstate,omitempty"`
}

// Pstate defines pstate parametes for workload
type Pstate struct {
	Ratio      string `json:"ratio,omitempty"`
	Monitoring string `json:"monitoring,omitempty"`
}

// WorkloadState defines state of a workload for a single node
type WorkloadState struct {
	Response string   `json:"response,omitempty"`
	ID       string   `json:"id,omitempty"`
	CosName  string   `json:"cosName,omitempty"`
	Status   string   `json:"status,omitempty"`
	CoreIds  []string `json:"coreIds,omitempty"`
	Policy   string   `json:"policy,omitempty"`
	Rdt      Rdt      `json:"rdt,omitempty"`
	Plugins  Plugins  `json:"plugins,omitempty"`
}

// RmdWorkloadSpec defines the desired state of RmdWorkload
type RmdWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	AllCores        bool              `json:"allCores,omitempty"`
	CoreIds         []string          `json:"coreIds,omitempty"`
	ReservedCoreIds []string          `json:"reservedCoreIds,omitempty"`
	Policy          string            `json:"policy,omitempty,omitempty"`
	Rdt             Rdt               `json:"rdt,omitempty"`
	Plugins         Plugins           `json:"plugins,omitempty"`
	NodeSelector    map[string]string `json:"nodeSelector,omitempty"`
	Nodes           []string          `json:"nodes,omitempty"`
}

// RmdWorkloadStatus defines the observed state of RmdWorkload
type RmdWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	WorkloadStates map[string]WorkloadState `json:"workloadStates,omitempty"`
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

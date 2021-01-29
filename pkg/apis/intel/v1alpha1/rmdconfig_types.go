package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RmdConfigSpec defines the desired state of RmdConfig
type RmdConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	RmdImage        string            `json:"rmdImage,omitempty"`
	DeployNodeAgent bool              `json:"deployNodeAgent,omitempty"`
	RmdNodeSelector map[string]string `json:"rmdNodeSelector,omitempty"`
}

// RmdConfigStatus defines the observed state of RmdConfig
type RmdConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Nodes []string `json:"nodes,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RmdConfig is the Schema for the rmdconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rmdconfigs,scope=Namespaced
type RmdConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RmdConfigSpec   `json:"spec,omitempty"`
	Status RmdConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RmdConfigList contains a list of RmdConfig
type RmdConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RmdConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RmdConfig{}, &RmdConfigList{})
}

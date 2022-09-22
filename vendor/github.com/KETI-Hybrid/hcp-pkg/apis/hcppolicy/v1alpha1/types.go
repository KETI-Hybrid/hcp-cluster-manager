package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// HCPPolicy is the Schema for the HCPpolicies API
type HCPPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HCPPolicySpec   `json:"spec,omitempty"`
	Status HCPPolicyStatus `json:"status,omitempty"`
}

type HCPPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	//Template - 생성
	Template HCPPolicyTemplate `json:"template"`
	/*Template struct {
		Spec struct {
			TargetController struct {
				Kind string `json:"kind"`
			} `json:"targetController"`
			Policies []struct {
				Type string `json:"type"`
				Value string `json:"value"`
			} `json:"policies"`
		} `json:"spec"`
	} `json:"template"`*/
	RangeOfApplication string `json:"rangeOfApplication"`
	PolicyStatus       string `json:"policyStatus"`
	//Placement

}

// HCPPolicyStatus defines the observed state of HCPPolicy
type HCPPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Replicas    int32            `json:"replicas"`
	ClusterMaps map[string]int32 `json:"clusters"`
}

type HCPPolicyTemplate struct {
	Spec HCPPolicyTemplateSpec `json:"spec"`
}

type HCPPolicyTemplateSpec struct {
	TargetController HCPPolicyTartgetController `json:"targetController"`
	Policies         []HCPPolicies              `json:"policies"`
}

type HCPPolicyTartgetController struct {
	Kind string `json:"kind"`
}

type HCPPolicies struct {
	Type  string   `json:"type"`
	Value []string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// HCPPolicyList contains a list of HCPPolicy
type HCPPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HCPPolicy `json:"items"`
}

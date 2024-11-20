/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VNISpec defines the desired state of VNI.
type VNISpec struct {
	ASN           uint32   `json:"asn,omitempty"`
	LocalNeighbor Neighbor `json:"asn,omitempty"`
	VNI           uint32   `json:"asn,omitempty"`
	LocalCIDR     string   `json:"localip,omitempty"`
}

// VNIStatus defines the observed state of VNI.
type VNIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VNI is the Schema for the vnis API.
type VNI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VNISpec   `json:"spec,omitempty"`
	Status VNIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VNIList contains a list of VNI.
type VNIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VNI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VNI{}, &VNIList{})
}

/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +k8s:openapi-gen=true
type ForSpec struct {
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// RbacNegotiationSpec defines the desired state of RbacNegotiation
// +k8s:openapi-gen=true
type RbacNegotiationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	For ForSpec `json:"for"`
}

// RbacNegotiationStatus defines the observed state of RbacNegotiation
type RbacNegotiationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RbacNegotiation is the Schema for the rbacnegotiations API
// +kubebuilder:printcolumn:name="kind",type=string,JSONPath=`.spec.for.kind`
// +kubebuilder:printcolumn:name="name",type=string,JSONPath=`.spec.for.name`
// +kubebuilder:printcolumn:name="status",type=string,JSONPath=`.spec.status.status`
// +kubebuilder:resource:shortName={rn,rbacn}
type RbacNegotiation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RbacNegotiationSpec   `json:"spec,omitempty"`
	Status RbacNegotiationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RbacNegotiationList contains a list of RbacNegotiation
type RbacNegotiationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RbacNegotiation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RbacNegotiation{}, &RbacNegotiationList{})
}

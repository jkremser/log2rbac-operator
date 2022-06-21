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

// ForSpec identifies the application of which the logs will be used for RBAC negotiation
// +k8s:openapi-gen=true
type ForSpec struct {
	//+kubebuilder:validation:Enum={Deployment,deployment,deploy,ReplicaSet,replicaset,rs,DaemonSet,daemonset,ds,StatefulSet,statefulset,ss,Service,service,svc}
	Kind string `json:"kind,omitempty"`
	// +optional
	// this can override the real pod selector that's associated for the deployment,rs,ds,ss or svc
	PodSelector map[string]string `json:"podSelector,omitempty"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
}

// RoleSpec identifies the role that would be updated by the operator
// +k8s:openapi-gen=true
type RoleSpec struct {
	Name             string `json:"name"`
	IsClusterRole    bool   `json:"isClusterRole,omitempty"`
	CreateIfNotExist bool   `json:"createIfNotExist,omitempty"`
}

// RbacNegotiationSpec defines the desired state of RbacNegotiation
// +k8s:openapi-gen=true
type RbacNegotiationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	For  ForSpec  `json:"for"`
	Role RoleSpec `json:"role,omitempty"`
	// +optional This needs to be provided if .spec.for.kind == service
	// this can override the real service account that's specified in the deployment,rs,ds or ss
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// RbacNegotiationStatus defines the observed state of RbacNegotiation
type RbacNegotiationStatus struct {
	//+kubebuilder:validation:Enum={Requested,InProgress,Error,NoChange,Synced}
	//+kubebuilder:default=Requested
	Status    string      `json:"status,omitempty"`
	LastCheck metav1.Time `json:"lastCheck,omitempty" protobuf:"bytes,8,opt,name=lastCheck"`
	// +kubebuilder:validation:Minimum=0
	EntriesAdded int32 `json:"entriesAdded,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RbacNegotiation is the Schema for the rbacnegotiations API
// +kubebuilder:printcolumn:name="for kind",type=string,JSONPath=`.spec.for.kind`,description="For which kind the rbac negotiation was requested"
// +kubebuilder:printcolumn:name="for name",type=string,JSONPath=`.spec.for.name`,description="Name of the {kind}"
// +kubebuilder:printcolumn:name="status",type=string,JSONPath=`.status.status`,description="State of the negotiation"
// +kubebuilder:printcolumn:name="role",type=string,JSONPath=`.spec.role.name`,priority=10,description="Name of the associated role"
// +kubebuilder:printcolumn:name="entries",type=string,JSONPath=`.status.entriesAdded`,priority=12,description="How many RBAC entries have been added to the role"
// +kubebuilder:printcolumn:name="checked",type=date,JSONPath=`.status.lastCheck`,priority=13,description="When the last reconciliation was done"
// +kubebuilder:resource:shortName={rn,rbacn}
// +kubebuilder:pruning:PreserveUnknownFields
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

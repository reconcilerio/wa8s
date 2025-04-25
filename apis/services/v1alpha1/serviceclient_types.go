/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
)

// +die
// +die:field:name=Ref,die=ServiceInstanceReferenceDie

// ServiceClientSpec defines the desired state of ServiceClient
type ServiceClientSpec struct {
	Ref                   ServiceInstanceReference `json:"ref"`
	Scopes                []string                 `json:"scopes,omitempty"`
	Duration              metav1.Duration          `json:"duration,omitempty"`
	RenewBefore           metav1.Duration          `json:"renewBefore,omitempty"`
	RenewBeforePercentage int32                    `json:"renewBeforePercentage,omitempty"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie,package=reconciler.io/wa8s/apis/components/v1alpha1
// +die:field:name=Binding,die=LocalObjectReferenceDie,package=_/core/v1

// ServiceClientStatus defines the observed state of ServiceClient
type ServiceClientStatus struct {
	apis.Status                               `json:",inline"`
	componentsv1alpha1.GenericComponentStatus `json:",inline"`

	ServiceBindingId string                      `json:"serviceBindingId,omitempty"`
	Binding          corev1.LocalObjectReference `json:"binding,omitempty"`
	RenewsAfter      metav1.Time                 `json:"renewsAfter,omitempty"`
	ExpiresAfter     metav1.Time                 `json:"expiresAfter,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-services
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Binding ID",type=string,JSONPath=`.status.serviceBindingId`
// +kubebuilder:printcolumn:name="Renews",type=string,JSONPath=`.status.renewsAfter`
// +kubebuilder:printcolumn:name="Expires",type=string,JSONPath=`.status.expiresAfter`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// ServiceClient is the Schema for the ServiceClients API
type ServiceClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceClientSpec   `json:"spec,omitempty"`
	Status ServiceClientStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceClientList contains a list of ServiceClient
type ServiceClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceClient{}, &ServiceClientList{})
}

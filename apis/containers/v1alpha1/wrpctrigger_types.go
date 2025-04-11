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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
)

// +die
// +die:field:name=GenericContainerSpec,die=GenericContainerSpecDie

// WrpcTriggerSpec defines the desired state of WrpcTrigger
type WrpcTriggerSpec struct {
	GenericContainerSpec `json:",inline"`
}

// +die
// +die:field:name=GenericContainerStatus,die=GenericContainerStatusDie

// WrpcTriggerStatus defines the observed state of WrpcTrigger
type WrpcTriggerStatus struct {
	apis.Status            `json:",inline"`
	GenericContainerStatus `json:",inline"`
	URL                    string `json:"url,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-container
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// WrpcTrigger is the Schema for the components API
type WrpcTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WrpcTriggerSpec   `json:"spec,omitempty"`
	Status WrpcTriggerStatus `json:"status,omitempty"`
}

var _ GenericContainer = (*WrpcTrigger)(nil)

func (r *WrpcTrigger) GetGenericContainerSpec() *GenericContainerSpec {
	return &r.Spec.GenericContainerSpec
}

func (r *WrpcTrigger) GetGenericContainerStatus() *GenericContainerStatus {
	return &r.Status.GenericContainerStatus
}

var _ componentsv1alpha1.ComponentLike = (*WrpcTrigger)(nil)

func (r *WrpcTrigger) GetGenericComponentSpec() *componentsv1alpha1.GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *WrpcTrigger) GetGenericComponentStatus() *componentsv1alpha1.GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// WrpcTriggerList contains a list of WrpcTrigger
type WrpcTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WrpcTrigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WrpcTrigger{}, &WrpcTriggerList{})
}

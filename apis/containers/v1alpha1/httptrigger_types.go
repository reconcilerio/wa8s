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
	"reconciler.io/runtime/apis"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
)

// +die
// +die:field:name=GenericContainerSpec,die=GenericContainerSpecDie

// HttpTriggerSpec defines the desired state of HttpTrigger
type HttpTriggerSpec struct {
	GenericContainerSpec `json:",inline"`
}

// +die
// +die:field:name=GenericContainerStatus,die=GenericContainerStatusDie

// HttpTriggerStatus defines the observed state of HttpTrigger
type HttpTriggerStatus struct {
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

// HttpTrigger is the Schema for the components API
type HttpTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HttpTriggerSpec   `json:"spec,omitempty"`
	Status HttpTriggerStatus `json:"status,omitempty"`
}

var _ GenericContainer = (*HttpTrigger)(nil)

func (r *HttpTrigger) GetGenericContainerSpec() *GenericContainerSpec {
	return &r.Spec.GenericContainerSpec
}

func (r *HttpTrigger) GetGenericContainerStatus() *GenericContainerStatus {
	return &r.Status.GenericContainerStatus
}

var _ componentsv1alpha1.ComponentLike = (*HttpTrigger)(nil)

func (r *HttpTrigger) GetGenericComponentSpec() *componentsv1alpha1.GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *HttpTrigger) GetGenericComponentStatus() *componentsv1alpha1.GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// HttpTriggerList contains a list of HttpTrigger
type HttpTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HttpTrigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HttpTrigger{}, &HttpTriggerList{})
}

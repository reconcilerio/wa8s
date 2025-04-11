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
	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
)

// +die
// +die:field:name=GenericComponentSpec,die=GenericComponentSpecDie,package=reconciler.io/wa8s/apis/components/v1alpha1
// +die:field:name=Ref,die=ComponentReferenceDie,package=reconciler.io/wa8s/apis/components/v1alpha1
// +die:field:name=ServiceAccountRef,die=ServiceAccountReferenceDie,package=reconciler.io/wa8s/apis/registries/v1alpha1

// WasmtimeContainerSpec defines the desired state of WasmtimeContainer
type WasmtimeContainerSpec struct {
	componentsv1alpha1.GenericComponentSpec `json:",inline"`

	// BaseImage in an oci repository holding wasmtime
	BaseImage string `json:"baseImage,omitempty"`
	// Ref references the component to convert to an image
	Ref componentsv1alpha1.ComponentReference `json:"ref,omitempty"`
	// ServiceAccountRef references the service account holding image pull secrets for the image
	ServiceAccountRef registriesv1alpha1.ServiceAccountReference `json:"serviceAccountRef,omitempty"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie,package=reconciler.io/wa8s/apis/components/v1alpha1

// WasmtimeContainerStatus defines the observed state of WasmtimeContainer
type WasmtimeContainerStatus struct {
	apis.Status                               `json:",inline"`
	componentsv1alpha1.GenericComponentStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-container
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// WasmtimeContainer is the Schema for the components API
type WasmtimeContainer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WasmtimeContainerSpec   `json:"spec,omitempty"`
	Status WasmtimeContainerStatus `json:"status,omitempty"`
}

// TODO this isn't a component, but it might be close enough
var _ componentsv1alpha1.ComponentLike = (*WasmtimeContainer)(nil)

func (r *WasmtimeContainer) GetGenericComponentSpec() *componentsv1alpha1.GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *WasmtimeContainer) GetGenericComponentStatus() *componentsv1alpha1.GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// WasmtimeContainerList contains a list of WasmtimeContainer
type WasmtimeContainerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WasmtimeContainer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WasmtimeContainer{}, &WasmtimeContainerList{})
}

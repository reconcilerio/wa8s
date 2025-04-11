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
)

// +die
// +die:field:name=GenericComponentSpec,die=GenericComponentSpecDie
// +die:field:name=GenericCompositionSpec,die=GenericCompositionSpecDie

// CompositionSpec defines the desired state of Composition
type CompositionSpec struct {
	GenericComponentSpec   `json:",inline"`
	GenericCompositionSpec `json:",inline"`
}

// +die
// +die:field:name=Plug,die=CompositionPlugDie,pointer=true
// +die:field:name=Dependencies,die=CompositionDependencyDie,listType=map,listMapKey=Component

// CompositionSpec defines the desired state of Composition
type GenericCompositionSpec struct {
	WAC          string                  `json:"wac,omitempty"`
	Plug         *CompositionPlug        `json:"plug,omitempty"`
	Dependencies []CompositionDependency `json:"dependencies,omitempty"`
}

// +die
type CompositionPlug struct{}

// +die
// +die:field:name=Ref,die=ComponentReferenceDie,pointer=true
// +die:field:name=Config,die=GenericConfigStoreSpecDie,pointer=true
// +die:field:name=OCI,die=OCIReferenceDie,pointer=true
// +die:field:name=Composition,die=GenericCompositionSpecDie,pointer=true
type CompositionDependency struct {
	Component string                  `json:"component"`
	Ref       *ComponentReference     `json:"ref,omitempty"`
	Config    *GenericConfigStoreSpec `json:"config,omitempty"`
	OCI       *OCIReference           `json:"oci,omitempty"`
	// Composition is schema equivalent to CompositionSpec, but schemaless to breaking out of recursive type nesting.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Composition *GenericCompositionSpec `json:"composition,omitempty"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie
// +die:field:name=Dependencies,die=CompositionDependencyStatusDie,listType=map,listMapKey=Component
//
// CompositionStatus defines the observed state of Composition
type CompositionStatus struct {
	apis.Status            `json:",inline"`
	GenericComponentStatus `json:",inline"`
	Dependencies           []CompositionDependencyStatus `json:"dependencies,omitempty"`
}

// +die
// +die:field:name=WIT,die=WITDie
type CompositionDependencyStatus struct {
	Component string `json:"component"`
	Image     string `json:"image"`
	WIT       WIT    `json:"wit,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-component
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// Composition is the Schema for the Compositions API
type Composition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CompositionSpec   `json:"spec,omitempty"`
	Status CompositionStatus `json:"status,omitempty"`
}

var _ ComponentLike = (*Composition)(nil)

func (r *Composition) GetGenericComponentSpec() *GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *Composition) GetGenericComponentStatus() *GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// CompositionList contains a list of Composition
type CompositionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Composition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Composition{}, &CompositionList{})
}

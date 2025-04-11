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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"reconciler.io/runtime/apis"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
)

// +die
type ComponentReference struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
}

// +die
// +die:field:name=GenericComponentSpec,die=GenericComponentSpecDie
// +die:field:name=OCI,die=OCIReferenceDie,pointer=true
// +die:field:name=Ref,die=ComponentReferenceDie,pointer=true

// ComponentSpec defines the desired state of Component
type ComponentSpec struct {
	GenericComponentSpec `json:",inline"`

	// OCI image to pull component from
	OCI *OCIReference `json:"oci,omitempty"`

	// Ref to another component
	Ref *ComponentReference `json:"ref,omitempty"`
}

// +die
// +die:field:name=ServiceAccountRef,die=ServiceAccountReferenceDie,package=reconciler.io/wa8s/apis/registries/v1alpha1
type OCIReference struct {
	// Image in an oci repository holding a wasm component
	Image string `json:"image,omitempty"`
	// ServiceAccountRef references the service account holding image pull secrets for the image
	ServiceAccountRef registriesv1alpha1.ServiceAccountReference `json:"serviceAccountRef,omitempty"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie

// ComponentStatus defines the observed state of Component
type ComponentStatus struct {
	apis.Status            `json:",inline"`
	GenericComponentStatus `json:",inline"`
}

// +kubebuilder:object:generate=false

type GenericComponent interface {
	runtime.Object
	metav1.Object
	webhook.CustomDefaulter
	ComponentLike

	GetSpec() *ComponentSpec
	GetStatus() *ComponentStatus
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-component
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// Component is the Schema for the components API
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec   `json:"spec,omitempty"`
	Status ComponentStatus `json:"status,omitempty"`
}

var _ GenericComponent = (*Component)(nil)

func (r *Component) GetSpec() *ComponentSpec {
	return &r.Spec
}

func (r *Component) GetStatus() *ComponentStatus {
	return &r.Status
}

func (r *Component) GetGenericComponentSpec() *GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *Component) GetGenericComponentStatus() *GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// ComponentList contains a list of Component
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Component `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=wa8s;wa8s-component
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true,spec=DieComponentSpec,status=DieComponentStatus

// ClusterComponent is the Schema for the components API
type ClusterComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec   `json:"spec,omitempty"`
	Status ComponentStatus `json:"status,omitempty"`
}

var _ GenericComponent = (*ClusterComponent)(nil)

func (r *ClusterComponent) GetSpec() *ComponentSpec {
	return &r.Spec
}

func (r *ClusterComponent) GetStatus() *ComponentStatus {
	return &r.Status
}

func (r *ClusterComponent) GetGenericComponentSpec() *GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *ClusterComponent) GetGenericComponentStatus() *GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// ClusterComponentList contains a list of Component
type ClusterComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
	SchemeBuilder.Register(&ClusterComponent{}, &ClusterComponentList{})
}

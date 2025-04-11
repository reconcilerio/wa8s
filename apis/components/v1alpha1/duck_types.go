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
	"k8s.io/apimachinery/pkg/types"
	"reconciler.io/runtime/apis"

	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
)

// +die
// +die:field:name=RepositoryRef,die=RepositoryReferenceDie,package=reconciler.io/wa8s/apis/registries/v1alpha1

// GenericComponentSpec defines the desired state of GenericComponent
type GenericComponentSpec struct {
	RepositoryRef registriesv1alpha1.RepositoryReference `json:"repositoryRef,omitempty"`
}

// +die
// +die:field:name=WIT,die=WITDie,pointer=true
// +die:field:name=Trace,die=ComponentSpanDie,listType=atomic

// GenericComponentStatus defines the observed state of GenericComponent
type GenericComponentStatus struct {
	// Image resolved from an oci repository holding the wasm component
	Image string          `json:"image,omitempty"`
	WIT   *WIT            `json:"wit,omitempty"`
	Trace []ComponentSpan `json:"trace,omitempty"`
}

// +die
type WIT struct {
	Imports []string `json:"imports,omitempty"`
	Exports []string `json:"exports,omitempty"`
}

// +die
// +die:field:name=Trace,die=ComponentSpanDie,listType=atomic
type ComponentSpan struct {
	Digest    string    `json:"digest,omitempty"`
	UID       types.UID `json:"uid"`
	Group     string    `json:"group"`
	Kind      string    `json:"kind"`
	Namespace string    `json:"namespace,omitempty"`
	Name      string    `json:"name"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Trace        []ComponentSpan `json:"trace,omitempty"`
	CycleOmitted bool            `json:"cycleOmitted,omitempty"`
}

// +kubebuilder:object:generate=false

type ComponentLike interface {
	runtime.Object
	metav1.Object

	GetGenericComponentSpec() *GenericComponentSpec
	GetGenericComponentStatus() *GenericComponentStatus
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +die
// +die:field:name=GenericComponentSpec,die=GenericComponentSpecDie
type ComponentDuckSpec struct {
	GenericComponentSpec `json:",inline"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie
type ComponentDuckStatus struct {
	apis.Status            `json:",inline"`
	GenericComponentStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +die:object=true
type ComponentDuck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentDuckSpec   `json:"spec,omitempty"`
	Status ComponentDuckStatus `json:"status,omitempty"`
}

var _ ComponentLike = (*ComponentDuck)(nil)

func (r *ComponentDuck) GetGenericComponentSpec() *GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *ComponentDuck) GetGenericComponentStatus() *GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true
type ComponentDuckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentDuck `json:"items"`
}

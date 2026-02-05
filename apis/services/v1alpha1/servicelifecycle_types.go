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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
)

// +die

type ServiceLifecycleReference struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// +die
// +die:field:name=GenericContainerSpec,die=GenericContainerSpecDie,package=reconciler.io/wa8s/apis/containers/v1alpha1
// +die:field:name=ClientRef,die=ComponentReferenceDie,package=reconciler.io/wa8s/apis/components/v1alpha1

// ServiceLifecycleSpec defines the desired state of ServiceLifecycle
type ServiceLifecycleSpec struct {
	containersv1alpha1.GenericContainerSpec `json:",inline"`
	ClientRef                               componentsv1alpha1.ComponentReference `json:"clientRef"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie,package=reconciler.io/wa8s/apis/components/v1alpha1

// ServiceLifecycleStatus defines the observed state of ServiceLifecycle
type ServiceLifecycleStatus struct {
	apis.Status                               `json:",inline"`
	componentsv1alpha1.GenericComponentStatus `json:",inline"`
	URL                                       string `json:"url,omitempty"`
}

// +kubebuilder:object:generate=false

type GenericServiceLifecycle interface {
	runtime.Object
	metav1.Object
	reconcilers.Defaulter

	GetSpec() *ServiceLifecycleSpec
	GetStatus() *ServiceLifecycleStatus
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-services
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// ServiceLifecycle is the Schema for the ServiceLifecycles API
type ServiceLifecycle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLifecycleSpec   `json:"spec,omitempty"`
	Status ServiceLifecycleStatus `json:"status,omitempty"`
}

var _ GenericServiceLifecycle = (*ServiceLifecycle)(nil)

func (r *ServiceLifecycle) GetSpec() *ServiceLifecycleSpec {
	return &r.Spec
}

func (r *ServiceLifecycle) GetStatus() *ServiceLifecycleStatus {
	return &r.Status
}

var _ componentsv1alpha1.ComponentLike = (*ServiceLifecycle)(nil)

func (r *ServiceLifecycle) GetGenericComponentSpec() *componentsv1alpha1.GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *ServiceLifecycle) GetGenericComponentStatus() *componentsv1alpha1.GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// ServiceLifecycleList contains a list of ServiceLifecycle
type ServiceLifecycleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceLifecycle `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=wa8s;wa8s-services
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true,spec=DieServiceLifecycleSpec,status=DieServiceLifecycleStatus

// ClusterServiceLifecycle is the Schema for the ClusterRepositories API
type ClusterServiceLifecycle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLifecycleSpec   `json:"spec,omitempty"`
	Status ServiceLifecycleStatus `json:"status,omitempty"`
}

var _ GenericServiceLifecycle = (*ClusterServiceLifecycle)(nil)

func (r *ClusterServiceLifecycle) GetSpec() *ServiceLifecycleSpec {
	return &r.Spec
}

func (r *ClusterServiceLifecycle) GetStatus() *ServiceLifecycleStatus {
	return &r.Status
}

var _ componentsv1alpha1.ComponentLike = (*ClusterServiceLifecycle)(nil)

func (r *ClusterServiceLifecycle) GetGenericComponentSpec() *componentsv1alpha1.GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *ClusterServiceLifecycle) GetGenericComponentStatus() *componentsv1alpha1.GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// ClusterServiceLifecycleList contains a list of ServiceLifecycle
type ClusterServiceLifecycleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterServiceLifecycle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceLifecycle{}, &ServiceLifecycleList{})
	SchemeBuilder.Register(&ClusterServiceLifecycle{}, &ClusterServiceLifecycleList{})
}

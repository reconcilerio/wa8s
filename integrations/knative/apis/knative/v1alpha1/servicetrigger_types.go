/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/runtime"
	"reconciler.io/runtime/apis"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
	servingv1 "reconciler.io/wa8s/integrations/knative/apis/serving/v1"
)

// +die
// +die:field:name=GenericContainerSpec,die=GenericContainerSpecDie,package=reconciler.io/wa8s/apis/containers/v1alpha1
// +die:field:name=ConfigurationSpec,die=ConfigurationSpecDie,package=reconciler.io/wa8s/integrations/knative/apis/serving/v1
// +die:field:name=RouteSpec,die=RouteSpecDie,package=reconciler.io/wa8s/integrations/knative/apis/serving/v1

// ServiceTriggerSpec defines the desired state of ServiceTrigger
type ServiceTriggerSpec struct {
	containersv1alpha1.GenericContainerSpec `json:",inline"`

	// ServiceSpec inlines an unrestricted ConfigurationSpec.
	servingv1.ConfigurationSpec `json:",inline"`

	// ServiceSpec inlines RouteSpec and restricts/defaults its fields
	// via webhook.  In particular, this spec can only reference this
	// Service's configuration and revisions (which also influences
	// defaults).
	servingv1.RouteSpec `json:",inline"`
}

// +die
// +die:field:name=GenericContainerStatus,die=GenericContainerStatusDie,package=reconciler.io/wa8s/apis/containers/v1alpha1
// +die:field:name=ConfigurationStatusFields,die=ConfigurationStatusFieldsDie,package=reconciler.io/wa8s/integrations/knative/apis/serving/v1
// +die:field:name=RouteStatusFields,die=RouteStatusFieldsDie,package=reconciler.io/wa8s/integrations/knative/apis/serving/v1

// ServiceTriggerStatus defines the observed state of ServiceTrigger
type ServiceTriggerStatus struct {
	apis.Status                               `json:",inline"`
	containersv1alpha1.GenericContainerStatus `json:",inline"`

	// Annotations is additional Status fields for the Resource to save some
	// additional State as well as convey more information to the user. This is
	// roughly akin to Annotations on any k8s resource, just the reconciler conveying
	// richer information outwards.
	Annotations map[string]string `json:"annotations,omitempty"`

	// In addition to inlining ConfigurationSpec, we also inline the fields
	// specific to ConfigurationStatus.
	servingv1.ConfigurationStatusFields `json:",inline"`

	// In addition to inlining RouteSpec, we also inline the fields
	// specific to RouteStatus.
	servingv1.RouteStatusFields `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:categories=wa8s;wa8s-knative
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
//+kubebuilder:printcolumn:name="Latest-Created",type=string,JSONPath=`.status.latestCreatedRevisionName`
//+kubebuilder:printcolumn:name="Latest-Ready",type=string,JSONPath=`.status.latestReadyRevisionName`
//+kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
//+kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// ServiceTrigger is the Schema for the ServiceTriggers API
type ServiceTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceTriggerSpec   `json:"spec,omitempty"`
	Status ServiceTriggerStatus `json:"status,omitempty"`
}

var _ componentsv1alpha1.ComponentLike = (*ServiceTrigger)(nil)
var _ containersv1alpha1.GenericContainer = (*ServiceTrigger)(nil)

func (r *ServiceTrigger) GetGenericContainerSpec() *containersv1alpha1.GenericContainerSpec {
	return &r.Spec.GenericContainerSpec
}

func (r *ServiceTrigger) GetGenericContainerStatus() *containersv1alpha1.GenericContainerStatus {
	return &r.Status.GenericContainerStatus
}

func (r *ServiceTrigger) GetGenericComponentSpec() *componentsv1alpha1.GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *ServiceTrigger) GetGenericComponentStatus() *componentsv1alpha1.GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

func (r *ServiceTrigger) GetRepositoryReference() *registriesv1alpha1.RepositoryReference {
	return &r.Spec.RepositoryRef
}

//+kubebuilder:object:root=true

// ServiceTriggerList contains a list of ServiceTrigger
type ServiceTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceTrigger `json:"items"`
}

func init() {
	schemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &ServiceTrigger{}, &ServiceTriggerList{})
		return nil
	})
}

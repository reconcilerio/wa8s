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
)

// +die
// +die:field:name=InstanceNames,die=ServiceResourceDefinitionNamesDie
// +die:field:name=ClientNames,die=ServiceResourceDefinitionNamesDie
// +die:field:name=Lifecycle,die=ServiceLifecycleSpecDie

// ServiceResourceDefinitionSpec defines the desired state of ServiceResourceDefinition
type ServiceResourceDefinitionSpec struct {
	Group         string                         `json:"group"`
	InstanceNames ServiceResourceDefinitionNames `json:"instanceNames"`
	ClientNames   ServiceResourceDefinitionNames `json:"clientNames"`
	Lifecycle     ServiceLifecycleSpec           `json:"lifecycle"`
}

// +die

// ServiceResourceDefinitionNames indicates the names to serve this service as a CustomResourceDefinition.
//
// Derived from CustomResourceDefinitionNames
type ServiceResourceDefinitionNames struct {
	// plural is the plural name of the resource to serve.
	// The custom resources are served under `/apis/<group>/<version>/.../<plural>`.
	// Must be all lowercase.
	Plural string `json:"plural"`
	// shortNames are short names for the resource, exposed in API discovery documents,
	// and used by clients to support invocations like `kubectl get <shortname>`.
	// It must be all lowercase.
	// +optional
	// +listType=atomic
	ShortNames []string `json:"shortNames,omitempty"`
	// kind is the serialized kind of the resource. It is normally CamelCase and singular.
	// Custom resource instances will use this value as the `kind` attribute in API calls.
	Kind string `json:"kind"`
	// categories is a list of grouped resources this custom resource belongs to (e.g. 'all').
	// This is published in API discovery documents, and used by clients to support invocations like
	// `kubectl get all`. `wa8s` and `wa8s-services` are added by default.
	// +optional
	// +listType=atomic
	Categories []string `json:"categories,omitempty"`
}

// +die

// ServiceResourceDefinitionStatus defines the observed state of ServiceResourceDefinition
type ServiceResourceDefinitionStatus struct {
	apis.Status `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=wa8s;wa8s-services
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// ServiceResourceDefinition is the Schema for the ServiceResourceDefinitions API
type ServiceResourceDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceResourceDefinitionSpec   `json:"spec,omitempty"`
	Status ServiceResourceDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceResourceDefinitionList contains a list of ServiceResourceDefinition
type ServiceResourceDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceResourceDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceResourceDefinition{}, &ServiceResourceDefinitionList{})
}

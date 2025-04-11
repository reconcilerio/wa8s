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
// +die:field:name=GenericConfigStoreSpec,die=GenericConfigStoreSpecDie

// ConfigStoreSpec defines the desired state of ConfigStore
type ConfigStoreSpec struct {
	GenericComponentSpec   `json:",inline"`
	GenericConfigStoreSpec `json:",inline"`
}

// +die
// +die:field:name=Values,die=ValueDie,listType=map
// +die:field:name=ValuesFrom,die=ValuesFromDie,listType=map

// ConfigStoreSpec defines the desired state of ConfigStore
type GenericConfigStoreSpec struct {
	Values     []Value      `json:"values,omitempty"`
	ValuesFrom []ValuesFrom `json:"valuesFrom,omitempty"`
}

// +die
// +die:field:name=ValueFrom,die=ValueFromDie,pointer=true
type Value struct {
	Name      string     `json:"name"`
	Value     string     `json:"value,omitempty"`
	ValueFrom *ValueFrom `json:"valueFrom,omitempty"`
}

// +die
type ValueFrom struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// +die
type ValuesFrom struct {
	Name   string `json:"name"`
	Prefix string `json:"prefix,omitempty"`
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie
//
// ConfigStoreStatus defines the observed state of ConfigStore
type ConfigStoreStatus struct {
	apis.Status            `json:",inline"`
	GenericComponentStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-component
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// ConfigStore is the Schema for the ConfigStores API
type ConfigStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigStoreSpec   `json:"spec,omitempty"`
	Status ConfigStoreStatus `json:"status,omitempty"`
}

var _ ComponentLike = (*ConfigStore)(nil)

func (r *ConfigStore) GetGenericComponentSpec() *GenericComponentSpec {
	return &r.Spec.GenericComponentSpec
}

func (r *ConfigStore) GetGenericComponentStatus() *GenericComponentStatus {
	return &r.Status.GenericComponentStatus
}

// +kubebuilder:object:root=true

// ConfigStoreList contains a list of ConfigStore
type ConfigStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfigStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfigStore{}, &ConfigStoreList{})
}

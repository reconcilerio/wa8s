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
	_ "embed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	dieapiextensionsv1 "reconciler.io/dies/apis/apiextensions/v1"
	"reconciler.io/runtime/apis"
)

// +die
// +die:field:name=Requests,die=ServiceInstanceRequestDie,listType=map,listMapKey=Key

// ServiceInstanceDuckSpec defines the desired state of ServiceInstance
type ServiceInstanceDuckSpec struct {
	Type     string                   `json:"type"`
	Tier     string                   `json:"tier,omitempty"`
	Requests []ServiceInstanceRequest `json:"requests,omitempty"`
}

// +die

// ServiceInstanceDuckStatus defines the observed state of ServiceInstanceDuck
type ServiceInstanceDuckStatus struct {
	apis.Status `json:",inline"`

	ServiceInstanceId string `json:"serviceInstanceId,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-services
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Instance ID",type=string,JSONPath=`.status.serviceInstanceId`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// ServiceInstanceDuck is the Schema for the ServiceInstanceDucks API
type ServiceInstanceDuck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceInstanceDuckSpec   `json:"spec,omitempty"`
	Status ServiceInstanceDuckStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceInstanceDuckList contains a list of ServiceInstanceDuck
type ServiceInstanceDuckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceInstanceDuck `json:"items"`
}

//go:embed serviceinstanceducks.yaml
var serviceInstanceDuckCRD []byte
var ServiceInstanceDuckCRD *dieapiextensionsv1.CustomResourceDefinitionDie

func init() {
	ServiceInstanceDuckCRD = dieapiextensionsv1.CustomResourceDefinitionBlank.DieFeedYAML(serviceInstanceDuckCRD)
}

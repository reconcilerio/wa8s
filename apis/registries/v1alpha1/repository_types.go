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
)

// +die

type RepositoryReference struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

// +die
// +die:field:name=ServiceAccountRef,die=ServiceAccountReferenceDie

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	Template          string                  `json:"template"`
	ServiceAccountRef ServiceAccountReference `json:"serviceAccountRef,omitempty"`
}

// +die

type ServiceAccountReference struct {
	// Namespace containing the ServiceAccount, only allowed for ClusterRepository resources
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// +die

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	apis.Status `json:",inline"`
}

// +kubebuilder:object:generate=false

type GenericRepository interface {
	runtime.Object
	metav1.Object
	webhook.CustomDefaulter

	GetSpec() *RepositorySpec
	GetStatus() *RepositoryStatus
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-registry
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// Repository is the Schema for the Repositorys API
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

var _ GenericRepository = (*Repository)(nil)

func (r *Repository) GetSpec() *RepositorySpec {
	return &r.Spec
}

func (r *Repository) GetStatus() *RepositoryStatus {
	return &r.Status
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=wa8s;wa8s-registry
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true,spec=DieRepositorySpec,status=DieRepositoryStatus

// ClusterRepository is the Schema for the ClusterRepositories API
type ClusterRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

var _ GenericRepository = (*ClusterRepository)(nil)

func (r *ClusterRepository) GetSpec() *RepositorySpec {
	return &r.Spec
}

func (r *ClusterRepository) GetStatus() *RepositoryStatus {
	return &r.Status
}

// +kubebuilder:object:root=true

// ClusterRepositoryList contains a list of Repository
type ClusterRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
	SchemeBuilder.Register(&ClusterRepository{}, &ClusterRepositoryList{})
}

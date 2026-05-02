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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
)

// +die
type ImageReference struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

// +die
// +die:field:name=RepositoryRef,die=RepositoryReferenceDie
// +die:field:name=ServiceAccountRef,die=ServiceAccountReferenceDie

// ImageSpec defines the desired state of Image
type ImageSpec struct {
	// RepositoryRef defines the destination repository for the image
	RepositoryRef RepositoryReference `json:"repositoryRef,omitempty"`

	// Image in an oci repository to be copied
	Image string `json:"image,omitempty"`
	// ServiceAccountRef references the service account holding image pull secrets for the image source
	ServiceAccountRef ServiceAccountReference `json:"serviceAccountRef,omitempty"`
}

// +die

// ImageStatus defines the observed state of Image
type ImageStatus struct {
	apis.Status `json:",inline"`

	// Image resolved from an oci repository
	Image string `json:"image,omitempty"`
}

// +kubebuilder:object:generate=false

type GenericImage interface {
	runtime.Object
	metav1.Object
	reconcilers.Defaulter
	RepositoryReferencer
	ServiceAccountReferencer

	GetSpec() *ImageSpec
	GetStatus() *ImageStatus
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=wa8s;wa8s-registry
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true

// Image is the Schema for the images API
type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSpec   `json:"spec,omitempty"`
	Status ImageStatus `json:"status,omitempty"`
}

var _ GenericImage = (*Image)(nil)

func (r *Image) GetSpec() *ImageSpec {
	return &r.Spec
}

func (r *Image) GetStatus() *ImageStatus {
	return &r.Status
}

func (r *Image) GetRepositoryReference() *RepositoryReference {
	return &r.Spec.RepositoryRef
}

func (r *Image) GetServiceAccountReference() *ServiceAccountReference {
	return &r.Spec.ServiceAccountRef
}

// +kubebuilder:object:root=true

// ImageList contains a list of Image
type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Image `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=wa8s;wa8s-registry
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +die:object=true,spec=DieImageSpec,status=DieImageStatus

// ClusterImage is the Schema for the images API
type ClusterImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageSpec   `json:"spec,omitempty"`
	Status ImageStatus `json:"status,omitempty"`
}

var _ GenericImage = (*ClusterImage)(nil)

func (r *ClusterImage) GetSpec() *ImageSpec {
	return &r.Spec
}

func (r *ClusterImage) GetStatus() *ImageStatus {
	return &r.Status
}

func (r *ClusterImage) GetRepositoryReference() *RepositoryReference {
	return &r.Spec.RepositoryRef
}

func (r *ClusterImage) GetServiceAccountReference() *ServiceAccountReference {
	return &r.Spec.ServiceAccountRef
}

// +kubebuilder:object:root=true

// ClusterImageList contains a list of Image
type ClusterImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterImage `json:"items"`
}

func init() {
	schemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &Image{}, &ImageList{})
		s.AddKnownTypes(GroupVersion, &ClusterImage{}, &ClusterImageList{})
		return nil
	})
}

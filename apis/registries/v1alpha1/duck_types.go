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
)

// +kubebuilder:object:generate=false

type ImageReferencer interface {
	runtime.Object
	metav1.Object

	GetImageReference() *ImageReference
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +kubebuilder:object:generate=false

type RepositoryReferencer interface {
	runtime.Object
	metav1.Object

	GetRepositoryReference() *RepositoryReference
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +kubebuilder:object:generate=false

type ServiceAccountReferencer interface {
	runtime.Object
	metav1.Object

	GetServiceAccountReference() *ServiceAccountReference
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

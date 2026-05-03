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
	"k8s.io/apimachinery/pkg/types"
	"reconciler.io/runtime/apis"
)

const (
	ImageConditionReady           = apis.ConditionReady
	ImageConditionRepositoryReady = "RepositoryReady"
	ImageConditionCopied          = "ImageCopied"
)

func (s *Image) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ClusterImage) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *Image) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ClusterImage) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ImageStatus) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSetWithHappyReason(
		"Ready",
		ImageConditionRepositoryReady,
		ImageConditionCopied,
	)
}

func (s *Image) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ClusterImage) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ImageStatus) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.GetConditionSet().ManageWithContext(ctx, s)
}

func (s *ImageStatus) InitializeConditions(ctx context.Context) {
	s.GetConditionManager(ctx).InitializeConditions()
}

var _ apis.ConditionsAccessor = (*ImageStatus)(nil)

func (r *ImageReference) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

func (r *ImageReference) TypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: GroupVersion.String(),
		Kind:       r.Kind,
	}
}

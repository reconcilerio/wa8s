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
	"k8s.io/apimachinery/pkg/types"
	"reconciler.io/runtime/apis"
)

const (
	ComponentConditionReady           = apis.ConditionReady
	ComponentConditionRepositoryReady = "RepositoryReady"
	ComponentConditionCopied          = "ComponentCopied"
)

func (s *Component) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ClusterComponent) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *Component) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ClusterComponent) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ComponentStatus) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSetWithHappyReason(
		"Ready",
		ComponentConditionRepositoryReady,
		ComponentConditionCopied,
	)
}

func (s *Component) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ClusterComponent) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ComponentStatus) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.GetConditionSet().ManageWithContext(ctx, s)
}

func (s *ComponentStatus) InitializeConditions(ctx context.Context) {
	s.GetConditionManager(ctx).InitializeConditions()
}

var _ apis.ConditionsAccessor = (*ComponentStatus)(nil)

func (r *ComponentReference) NamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: r.Namespace,
		Name:      r.Name,
	}
}

func (r *ComponentReference) TypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: r.APIVersion,
		Kind:       r.Kind,
	}
}

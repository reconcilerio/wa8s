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

	"reconciler.io/runtime/apis"
)

const (
	ServiceLifecycleConditionReady          = apis.ConditionReady
	ServiceLifecycleConditionComponentReady = "ComponentReady"
	ServiceLifecycleConditionLifecycleReady = "LifecycleReady"
	ServiceLifecycleConditionClientReady    = "ClientReady"
	ServiceLifecycleConditionFinalizer      = "Finalizer"
)

func (s *ServiceLifecycle) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ClusterServiceLifecycle) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ServiceLifecycle) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ClusterServiceLifecycle) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ServiceLifecycleStatus) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSetWithHappyReason(
		"Ready",
		ServiceLifecycleConditionComponentReady,
		ServiceLifecycleConditionLifecycleReady,
		ServiceLifecycleConditionClientReady,
	)
}

func (s *ServiceLifecycle) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ClusterServiceLifecycle) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ServiceLifecycleStatus) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.GetConditionSet().ManageWithContext(ctx, s)
}

func (s *ServiceLifecycleStatus) InitializeConditions(ctx context.Context) {
	s.GetConditionManager(ctx).InitializeConditions()
}

var _ apis.ConditionsAccessor = (*ServiceLifecycleStatus)(nil)

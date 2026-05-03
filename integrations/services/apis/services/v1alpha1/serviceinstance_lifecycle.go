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
	ServiceInstanceConditionReady          = apis.ConditionReady
	ServiceInstanceConditionLifecycleReady = "LifecycleReady"
	ServiceInstanceConditionProvisioned    = "Provisioned"
	ServiceInstanceConditionFinalizer      = "Finalizer"
)

func (s *ServiceInstance) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ServiceInstance) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ServiceInstanceStatus) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSetWithHappyReason(
		"Ready",
		ServiceInstanceConditionLifecycleReady,
		ServiceInstanceConditionProvisioned,
	)
}

func (s *ServiceInstance) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ServiceInstanceStatus) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.GetConditionSet().ManageWithContext(ctx, s)
}

func (s *ServiceInstanceStatus) InitializeConditions(ctx context.Context) {
	s.GetConditionManager(ctx).InitializeConditions()
}

var _ apis.ConditionsAccessor = (*ServiceInstanceStatus)(nil)

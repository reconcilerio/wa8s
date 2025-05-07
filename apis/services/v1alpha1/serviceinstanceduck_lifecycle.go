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
	ServiceInstanceDuckConditionReady                = apis.ConditionReady
	ServiceInstanceDuckConditionServiceInstanceReady = "ServiceInstanceReady"
)

func (s *ServiceInstanceDuck) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ServiceInstanceDuck) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ServiceInstanceDuckStatus) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSetWithHappyReason(
		"Ready",
		ServiceInstanceDuckConditionServiceInstanceReady,
	)
}

func (s *ServiceInstanceDuck) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ServiceInstanceDuckStatus) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.GetConditionSet().ManageWithContext(ctx, s)
}

func (s *ServiceInstanceDuckStatus) InitializeConditions(ctx context.Context) {
	s.GetConditionManager(ctx).InitializeConditions()
}

var _ apis.ConditionsAccessor = (*ServiceInstanceDuckStatus)(nil)

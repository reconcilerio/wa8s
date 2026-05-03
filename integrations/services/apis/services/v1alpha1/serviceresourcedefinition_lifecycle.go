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
	ServiceResourceDefinitionConditionReady                    = apis.ConditionReady
	ServiceResourceDefinitionConditionLifecycleReady           = "LifecycleReady"
	ServiceResourceDefinitionConditionInstanceDefinitionReady  = "InstanceDefinitionReady"
	ServiceResourceDefinitionConditionClientDefinitionReady    = "ClientDefinitionReady"
	ServiceResourceDefinitionConditionClientComponentDuckReady = "ClientComponentDuckReady"
	ServiceResourceDefinitionConditionReconcilerRunning        = "ReconcilerRunning"
)

func (s *ServiceResourceDefinition) GetConditionsAccessor() apis.ConditionsAccessor {
	return &s.Status
}

func (s *ServiceResourceDefinition) GetConditionSet() apis.ConditionSet {
	return s.Status.GetConditionSet()
}

func (s *ServiceResourceDefinitionStatus) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSetWithHappyReason(
		"Ready",
		ServiceResourceDefinitionConditionLifecycleReady,
		ServiceResourceDefinitionConditionInstanceDefinitionReady,
		ServiceResourceDefinitionConditionClientDefinitionReady,
		ServiceResourceDefinitionConditionClientComponentDuckReady,
		ServiceResourceDefinitionConditionReconcilerRunning,
	)
}

func (s *ServiceResourceDefinition) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.Status.GetConditionManager(ctx)
}

func (s *ServiceResourceDefinitionStatus) GetConditionManager(ctx context.Context) apis.ConditionManager {
	return s.GetConditionSet().ManageWithContext(ctx, s)
}

func (s *ServiceResourceDefinitionStatus) InitializeConditions(ctx context.Context) {
	s.GetConditionManager(ctx).InitializeConditions()
}

var _ apis.ConditionsAccessor = (*ServiceResourceDefinitionStatus)(nil)

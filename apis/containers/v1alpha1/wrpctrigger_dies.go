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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	diemetav1 "reconciler.io/dies/apis/meta/v1"
)

var (
	WrpcTriggerConditionReadyBlank                  = diemetav1.ConditionBlank.Type(WrpcTriggerConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	WrpcTriggerConditionWasmtimeContainerReadyBlank = diemetav1.ConditionBlank.Type(WrpcTriggerConditionWasmtimeContainerReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	WrpcTriggerConditionDeploymentReadyBlank        = diemetav1.ConditionBlank.Type(WrpcTriggerConditionDeploymentReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	WrpcTriggerConditionServiceReadyBlank           = diemetav1.ConditionBlank.Type(WrpcTriggerConditionServiceReady).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *WrpcTriggerStatusDie) InitializeConditionsDie() *WrpcTriggerStatusDie {
	return d.DieStamp(func(r *WrpcTriggerStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *WrpcTriggerStatusDie) ObservedGeneration(v int64) *WrpcTriggerStatusDie {
	return d.DieStamp(func(r *WrpcTriggerStatus) {
		r.ObservedGeneration = v
	})
}

func (d *WrpcTriggerStatusDie) Conditions(v ...metav1.Condition) *WrpcTriggerStatusDie {
	return d.DieStamp(func(r *WrpcTriggerStatus) {
		r.Conditions = v
	})
}

func (d *WrpcTriggerStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *WrpcTriggerStatusDie {
	return d.DieStamp(func(r *WrpcTriggerStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

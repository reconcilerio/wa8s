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
	diemetav1 "reconciler.io/dies/apis/meta/v1"
)

var (
	HttpTriggerConditionReadyBlank                  = diemetav1.ConditionBlank.Type(HttpTriggerConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	HttpTriggerConditionWasmtimeContainerReadyBlank = diemetav1.ConditionBlank.Type(HttpTriggerConditionWasmtimeContainerReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	HttpTriggerConditionDeploymentReadyBlank        = diemetav1.ConditionBlank.Type(HttpTriggerConditionDeploymentReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	HttpTriggerConditionServiceReadyBlank           = diemetav1.ConditionBlank.Type(HttpTriggerConditionServiceReady).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *HttpTriggerStatusDie) InitializeConditionsDie() *HttpTriggerStatusDie {
	return d.DieStamp(func(r *HttpTriggerStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *HttpTriggerStatusDie) ObservedGeneration(v int64) *HttpTriggerStatusDie {
	return d.DieStamp(func(r *HttpTriggerStatus) {
		r.ObservedGeneration = v
	})
}

func (d *HttpTriggerStatusDie) Conditions(v ...metav1.Condition) *HttpTriggerStatusDie {
	return d.DieStamp(func(r *HttpTriggerStatus) {
		r.Conditions = v
	})
}

func (d *HttpTriggerStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *HttpTriggerStatusDie {
	return d.DieStamp(func(r *HttpTriggerStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

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
	CronTriggerConditionReadyBlank                  = diemetav1.ConditionBlank.Type(CronTriggerConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	CronTriggerConditionWasmtimeContainerReadyBlank = diemetav1.ConditionBlank.Type(CronTriggerConditionWasmtimeContainerReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	CronTriggerConditionCronJobReadyBlank           = diemetav1.ConditionBlank.Type(CronTriggerConditionCronJobReady).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *CronTriggerStatusDie) InitializeConditionsDie() *CronTriggerStatusDie {
	return d.DieStamp(func(r *CronTriggerStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *CronTriggerStatusDie) ObservedGeneration(v int64) *CronTriggerStatusDie {
	return d.DieStamp(func(r *CronTriggerStatus) {
		r.ObservedGeneration = v
	})
}

func (d *CronTriggerStatusDie) Conditions(v ...metav1.Condition) *CronTriggerStatusDie {
	return d.DieStamp(func(r *CronTriggerStatus) {
		r.Conditions = v
	})
}

func (d *CronTriggerStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *CronTriggerStatusDie {
	return d.DieStamp(func(r *CronTriggerStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

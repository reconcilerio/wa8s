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
	ComponentConditionReadyBlank           = diemetav1.ConditionBlank.Type(ComponentConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ComponentConditionRepositoryReadyBlank = diemetav1.ConditionBlank.Type(ComponentConditionRepositoryReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ComponentConditionCopiedBlank          = diemetav1.ConditionBlank.Type(ComponentConditionCopied).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *ComponentStatusDie) InitializeConditionsDie() *ComponentStatusDie {
	return d.DieStamp(func(r *ComponentStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *ComponentStatusDie) ObservedGeneration(v int64) *ComponentStatusDie {
	return d.DieStamp(func(r *ComponentStatus) {
		r.ObservedGeneration = v
	})
}

func (d *ComponentStatusDie) Conditions(v ...metav1.Condition) *ComponentStatusDie {
	return d.DieStamp(func(r *ComponentStatus) {
		r.Conditions = v
	})
}

func (d *ComponentStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *ComponentStatusDie {
	return d.DieStamp(func(r *ComponentStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

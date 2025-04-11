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
	CompositionConditionReadyBlank                = diemetav1.ConditionBlank.Type(CompositionConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	CompositionConditionRepositoryReadyBlank      = diemetav1.ConditionBlank.Type(CompositionConditionRepositoryReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	CompositionConditionDependenciesResolvedBlank = diemetav1.ConditionBlank.Type(CompositionConditionDependenciesResolved).Status(metav1.ConditionUnknown).Reason("Initializing")
	CompositionConditionPushedBlank               = diemetav1.ConditionBlank.Type(CompositionConditionPushed).Status(metav1.ConditionUnknown).Reason("Initializing")
	CompositionConditionChildComponentBlank       = diemetav1.ConditionBlank.Type(CompositionConditionChildComponent).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *CompositionStatusDie) ObservedGeneration(v int64) *CompositionStatusDie {
	return d.DieStamp(func(r *CompositionStatus) {
		r.ObservedGeneration = v
	})
}

func (d *CompositionStatusDie) InitializeConditionsDie() *CompositionStatusDie {
	return d.DieStamp(func(r *CompositionStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *CompositionStatusDie) Conditions(v ...metav1.Condition) *CompositionStatusDie {
	return d.DieStamp(func(r *CompositionStatus) {
		r.Conditions = v
	})
}

func (d *CompositionStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *CompositionStatusDie {
	return d.DieStamp(func(r *CompositionStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

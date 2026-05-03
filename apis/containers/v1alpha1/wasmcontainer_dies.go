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
	ComponentContainerImageConditionReadyBlank           = diemetav1.ConditionBlank.Type(ComponentContainerImageConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ComponentContainerImageConditionImageReadyBlank      = diemetav1.ConditionBlank.Type(ComponentContainerImageConditionImageReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ComponentContainerImageConditionRepositoryReadyBlank = diemetav1.ConditionBlank.Type(ComponentContainerImageConditionRepositoryReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ComponentContainerImageConditionPushedBlank          = diemetav1.ConditionBlank.Type(ComponentContainerImageConditionPushed).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *ComponentContainerImageStatusDie) InitializeConditionsDie() *ComponentContainerImageStatusDie {
	return d.DieStamp(func(r *ComponentContainerImageStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *ComponentContainerImageStatusDie) ObservedGeneration(v int64) *ComponentContainerImageStatusDie {
	return d.DieStamp(func(r *ComponentContainerImageStatus) {
		r.ObservedGeneration = v
	})
}

func (d *ComponentContainerImageStatusDie) Conditions(v ...metav1.Condition) *ComponentContainerImageStatusDie {
	return d.DieStamp(func(r *ComponentContainerImageStatus) {
		r.Conditions = v
	})
}

func (d *ComponentContainerImageStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *ComponentContainerImageStatusDie {
	return d.DieStamp(func(r *ComponentContainerImageStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

/*
Copyright 2026.

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
	ImageConditionReadyBlank           = diemetav1.ConditionBlank.Type(ImageConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ImageConditionRepositoryReadyBlank = diemetav1.ConditionBlank.Type(ImageConditionRepositoryReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ImageConditionCopiedBlank          = diemetav1.ConditionBlank.Type(ImageConditionCopied).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *ImageStatusDie) InitializeConditionsDie() *ImageStatusDie {
	return d.DieStamp(func(r *ImageStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *ImageStatusDie) ObservedGeneration(v int64) *ImageStatusDie {
	return d.DieStamp(func(r *ImageStatus) {
		r.ObservedGeneration = v
	})
}

func (d *ImageStatusDie) Conditions(v ...metav1.Condition) *ImageStatusDie {
	return d.DieStamp(func(r *ImageStatus) {
		r.Conditions = v
	})
}

func (d *ImageStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *ImageStatusDie {
	return d.DieStamp(func(r *ImageStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

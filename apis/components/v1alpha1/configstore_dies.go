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
	ConfigStoreConditionReadyBlank           = diemetav1.ConditionBlank.Type(ConfigStoreConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ConfigStoreConditionRepositoryReadyBlank = diemetav1.ConditionBlank.Type(ConfigStoreConditionRepositoryReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ConfigStoreConditionConfigResolvedBlank  = diemetav1.ConditionBlank.Type(ConfigStoreConditionConfigResolved).Status(metav1.ConditionUnknown).Reason("Initializing")
	ConfigStoreConditionPushedBlank          = diemetav1.ConditionBlank.Type(ConfigStoreConditionPushed).Status(metav1.ConditionUnknown).Reason("Initializing")
	ConfigStoreConditionChildComponentBlank  = diemetav1.ConditionBlank.Type(ConfigStoreConditionChildComponent).Status(metav1.ConditionUnknown).Reason("Initializing")
)

func (d *ConfigStoreStatusDie) ObservedGeneration(v int64) *ConfigStoreStatusDie {
	return d.DieStamp(func(r *ConfigStoreStatus) {
		r.ObservedGeneration = v
	})
}

func (d *ConfigStoreStatusDie) InitializeConditionsDie() *ConfigStoreStatusDie {
	return d.DieStamp(func(r *ConfigStoreStatus) {
		r.InitializeConditions(context.TODO())
	})
}

func (d *ConfigStoreStatusDie) Conditions(v ...metav1.Condition) *ConfigStoreStatusDie {
	return d.DieStamp(func(r *ConfigStoreStatus) {
		r.Conditions = v
	})
}

func (d *ConfigStoreStatusDie) ConditionsDie(v ...*diemetav1.ConditionDie) *ConfigStoreStatusDie {
	return d.DieStamp(func(r *ConfigStoreStatus) {
		r.Conditions = make([]metav1.Condition, len(v))
		for i := range v {
			r.Conditions[i] = v[i].DieRelease()
		}
	})
}

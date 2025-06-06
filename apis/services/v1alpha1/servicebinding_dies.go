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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	diemetav1 "reconciler.io/dies/apis/meta/v1"
)

var (
	ServiceBindingConditionReadyBlank         = diemetav1.ConditionBlank.Type(ServiceBindingConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceBindingConditionInstanceReadyBlank = diemetav1.ConditionBlank.Type(ServiceBindingConditionInstanceReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceBindingConditionSecretBlank        = diemetav1.ConditionBlank.Type(ServiceBindingConditionSecret).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceBindingConditionBoundBlank         = diemetav1.ConditionBlank.Type(ServiceBindingConditionBound).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceBindingConditionClientReadyBlank   = diemetav1.ConditionBlank.Type(ServiceBindingConditionClientReady).Status(metav1.ConditionUnknown).Reason("Initializing")
)

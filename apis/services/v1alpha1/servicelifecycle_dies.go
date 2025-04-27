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
	ServiceLifecycleConditionReadyBlank          = diemetav1.ConditionBlank.Type(ServiceLifecycleConditionReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceLifecycleConditionComponentReadyBlank = diemetav1.ConditionBlank.Type(ServiceLifecycleConditionComponentReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceLifecycleConditionLifecycleReadyBlank = diemetav1.ConditionBlank.Type(ServiceLifecycleConditionLifecycleReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceLifecycleConditionClientReadyBlank    = diemetav1.ConditionBlank.Type(ServiceLifecycleConditionClientReady).Status(metav1.ConditionUnknown).Reason("Initializing")
	ServiceLifecycleConditionFinalizerBlank      = diemetav1.ConditionBlank.Type(ServiceLifecycleConditionFinalizer).Status(metav1.ConditionUnknown).Reason("Initializing")
)

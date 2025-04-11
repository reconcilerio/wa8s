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

package validation

const (
	// duplicate values to avoid cyclic dependencies
	componentsv1alpha1 = "wa8s.reconciler.io/v1alpha1"
	containersv1alpha1 = "containers.wa8s.reconciler.io/v1alpha1"
	registriesv1alpha1 = "registries.wa8s.reconciler.io/v1alpha1"
	servicesv1alpha1   = "services.wa8s.reconciler.io/v1alpha1"
)

func DefaultApiVersionForKind(kind string) string {
	switch kind {
	case "Component":
		return componentsv1alpha1
	case "ClusterComponent":
		return componentsv1alpha1
	case "Composition":
		return componentsv1alpha1
	case "ConfigStore":
		return componentsv1alpha1
	case "CronTrigger":
		return containersv1alpha1
	case "HttpTrigger":
		return containersv1alpha1
	case "WasmtimeContainer":
		return containersv1alpha1
	case "WrpcTrigger":
		return containersv1alpha1
	case "Repository":
		return registriesv1alpha1
	case "ClusterRepository":
		return registriesv1alpha1
	case "ServiceClient":
		return servicesv1alpha1
	case "ServiceInstance":
		return servicesv1alpha1
	case "ServiceLifecycle":
		return servicesv1alpha1
	case "ClusterServiceLifecycle":
		return servicesv1alpha1
	default:
		return ""
	}
}

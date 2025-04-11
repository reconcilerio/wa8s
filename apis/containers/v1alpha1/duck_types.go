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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"reconciler.io/runtime/apis"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
)

// +die
// +die:field:name=GenericComponentSpec,die=GenericComponentSpecDie,package=reconciler.io/wa8s/apis/components/v1alpha1
// +die:field:name=Ref,die=ComponentReferenceDie,package=reconciler.io/wa8s/apis/components/v1alpha1
// +die:field:name=ServiceAccountRef,die=ServiceAccountReferenceDie,package=reconciler.io/wa8s/apis/registries/v1alpha1
// +die:field:name=HostCapabilities,die=HostCapabilitiesDie

// GenericContainerSpec defines the desired state of GenericContainer
type GenericContainerSpec struct {
	componentsv1alpha1.GenericComponentSpec `json:",inline"`

	// Ref references the component to convert to an image
	Ref componentsv1alpha1.ComponentReference `json:"ref,omitempty"`
	// ServiceAccountRef references the service account holding image pull secrets for the image
	ServiceAccountRef registriesv1alpha1.ServiceAccountReference `json:"serviceAccountRef,omitempty"`

	HostCapabilities HostCapabilities `json:"hostCapabilities,omitempty"`
}

// +die
// +die:field:name=Env,die=HostEnvDie,pointer=true
// +die:field:name=Config,die=HostConfigDie,pointer=true
// +die:field:name=Network,die=HostNetworkDie,pointer=true
type HostCapabilities struct {
	Env     *HostEnv     `json:"env,omitempty"`
	Config  *HostConfig  `json:"config,omitempty"`
	Network *HostNetwork `json:"network,omitempty"`
}

func (c *HostCapabilities) WrpcWasmtimeArgs() []string {
	args := []string{}

	// no capabilities are supported

	return args
}

func (c *HostCapabilities) WasmtimeArgs() []string {
	args := []string{}

	// TODO flesh out
	if c.Env != nil {
		args = append(args, c.Env.WasmtimeArgs()...)
	}
	if c.Config != nil {
		args = append(args, c.Config.WasmtimeArgs()...)
	}
	if c.Network != nil {
		args = append(args, c.Network.WasmtimeArgs()...)
	}

	return args
}

// +die
// +die:field:name=Vars,die=HostEnvVarDie,listType=map
type HostEnv struct {
	Inherit bool         `json:"inherit"`
	Vars    []HostEnvVar `json:"vars,omitempty"`
}

func (c *HostEnv) WasmtimeArgs() []string {
	args := []string{}

	if c.Inherit {
		args = append(args, "-Sinherit-env=y")
	}
	for i := range c.Vars {
		args = append(args, c.Vars[i].WasmtimeArgs()...)
	}

	return args
}

// +die
type HostEnvVar struct {
	Name  string  `json:"name"`
	Value *string `json:"value,omitempty"`
}

func (c *HostEnvVar) WasmtimeArgs() []string {
	args := []string{}

	if c.Value != nil {
		args = append(args, fmt.Sprintf("--env=%s=%s", c.Name, *c.Value))
	} else {
		args = append(args, fmt.Sprintf("--env=%s", c.Name))
	}

	return args
}

// +die
// +die:field:name=Vars,die=HostConfigVarDie,listType=map
type HostConfig struct {
	Vars []HostConfigVar `json:"vars,omitempty"`
}

func (c *HostConfig) WasmtimeArgs() []string {
	args := []string{}

	args = append(args, "-Sconfig=y")
	for i := range c.Vars {
		args = append(args, c.Vars[i].WasmtimeArgs()...)
	}

	return args
}

// +die
type HostConfigVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (c *HostConfigVar) WasmtimeArgs() []string {
	return []string{fmt.Sprintf("-Sconfig-var=%s=%s", c.Name, c.Value)}
}

// +die
type HostNetwork struct {
	Inherit      bool `json:"inherit,omitempty"`
	IPNameLookup bool `json:"ipNameLookup,omitempty"`
}

func (c *HostNetwork) WasmtimeArgs() []string {
	args := []string{}

	if c.Inherit {
		args = append(args, "-Sinherit-network=y")
	}
	if c.IPNameLookup {
		args = append(args, "-Sallow-ip-name-lookup=y")
	}

	return args
}

// +die
// +die:field:name=GenericComponentStatus,die=GenericComponentStatusDie,package=reconciler.io/wa8s/apis/components/v1alpha1

// GenericContainerStatus defines the observed state of GenericContainer
type GenericContainerStatus struct {
	componentsv1alpha1.GenericComponentStatus `json:",inline"`
}

// +die
type WIT struct {
	Imports []string `json:"imports,omitempty"`
	Exports []string `json:"exports,omitempty"`
}

// +kubebuilder:object:generate=false

type GenericContainer interface {
	runtime.Object
	metav1.Object

	GetGenericContainerSpec() *GenericContainerSpec
	GetGenericContainerStatus() *GenericContainerStatus
	GetConditionManager(ctx context.Context) apis.ConditionManager
}

// +die
// +die:field:name=GenericContainerSpec,die=GenericContainerSpecDie
type ContainerDuckSpec struct {
	GenericContainerSpec `json:",inline"`
}

// +die
// +die:field:name=GenericContainerStatus,die=GenericContainerStatusDie
type ContainerDuckStatus struct {
	apis.Status            `json:",inline"`
	GenericContainerStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +die:object=true
type ContainerDuck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContainerDuckSpec   `json:"spec,omitempty"`
	Status ContainerDuckStatus `json:"status,omitempty"`
}

var _ GenericContainer = (*ContainerDuck)(nil)

func (r *ContainerDuck) GetGenericContainerSpec() *GenericContainerSpec {
	return &r.Spec.GenericContainerSpec
}

func (r *ContainerDuck) GetGenericContainerStatus() *GenericContainerStatus {
	return &r.Status.GenericContainerStatus
}

// +kubebuilder:object:root=true
type ContainerDuckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerDuck `json:"items"`
}

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

package controllers

import (
	"errors"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"reconciler.io/runtime/reconcilers"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
	"reconciler.io/wa8s/components"
	"reconciler.io/wa8s/registry"
)

var (
	ConfigStoreStasher                 = reconcilers.NewStasher[map[string]string](reconcilers.StashKey("wa8s.reconciler.io/config-store"))
	ComponentStasher                   = reconcilers.NewStasher[[]byte](reconcilers.StashKey("wa8s.reconciler.io/component"))
	ComponentConfigStasher             = reconcilers.NewStasher[registry.WasmConfigFile](reconcilers.StashKey("wa8s.reconciler.io/component-config"))
	ComponentTraceStasher              = reconcilers.NewStasher[[]componentsv1alpha1.ComponentSpan](reconcilers.StashKey("wa8s.reconciler.io/component-trace"))
	CompositionDependenciesStasher     = reconcilers.NewStasher[[]components.ResolvedComponent](reconcilers.StashKey("wa8s.reconciler.io/composition-dependencies"))
	RepositoryKeychainStasher          = reconcilers.NewStasher[authn.Keychain](reconcilers.StashKey("wa8s.reconciler.io/repository-keychain"))
	RepositoryDigestStasher            = reconcilers.NewStasher[name.Digest](reconcilers.StashKey("wa8s.reconciler.io/repository-digest"))
	RepositoryTagStasher               = reconcilers.NewStasher[name.Tag](reconcilers.StashKey("wa8s.reconciler.io/repository-tag"))
	ServiceLifecycleCompositionStasher = reconcilers.NewStasher[string](reconcilers.StashKey("wa8s.reconciler.io/service-lifecycle-composition"))
	ServiceLifecycleAddressStasher     = reconcilers.NewStasher[string](reconcilers.StashKey("wa8s.reconciler.io/service-lifecycle-address"))
	ServiceLifecycleReferenceStasher   = reconcilers.NewStasher[servicesv1alpha1.ServiceLifecycleReference](reconcilers.StashKey("wa8s.reconciler.io/service-lifecycle-reference"))
	ServiceBindingIdStasher            = reconcilers.NewStasher[string](reconcilers.StashKey("wa8s.reconciler.io/service-binding-id"))
	ServiceInstanceIdStasher           = reconcilers.NewStasher[string](reconcilers.StashKey("wa8s.reconciler.io/service-instance-id"))
)

var (
	// ErrTransient captures an error that is of the moment, retrying the request may succeed. Meaningful state about the error has been captured on the status
	ErrTransient = errors.Join(reconcilers.ErrQuiet, reconcilers.ErrHaltSubReconcilers)
	// ErrDurable is permanent given the current state, the request should not be retried until the observed state has changed. Meaningful state about the error has been captured on the status
	ErrDurable = reconcilers.ErrDurable
	// ErrGenerationMismatch a referenced resource's .metadata.generation and .status.observedGeneration are out of sync. Treat as a transient error as this state is expected and we should avoid flapping
	ErrGenerationMismatch = errors.Join(ErrTransient, reconcilers.ErrSkipStatusUpdate)
	// ErrUpdateStatusBeforeContinuingReconcile halt this reconcile request and update the api server with the intermediate status
	ErrUpdateStatusBeforeContinuingReconcile = errors.Join(errors.New("UpdateStatusBeforeContinuingReconcile"), ErrDurable)
)

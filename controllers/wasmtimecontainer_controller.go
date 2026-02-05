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
	"context"
	"errors"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
	"reconciler.io/wa8s/registry"
)

// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=wasmtimecontainers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=wasmtimecontainers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=wasmtimecontainers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func WasmtimeContainerReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*containersv1alpha1.WasmtimeContainer] {
	return &reconcilers.ResourceReconciler[*containersv1alpha1.WasmtimeContainer]{
		Reconciler: &reconcilers.SuppressTransientErrors[*containersv1alpha1.WasmtimeContainer, *containersv1alpha1.WasmtimeContainerList]{
			Reconciler: reconcilers.Sequence[*containersv1alpha1.WasmtimeContainer]{
				reconcilers.Always[*containersv1alpha1.WasmtimeContainer]{
					ResolveComponent(),
					ResolveRepository[*containersv1alpha1.WasmtimeContainer](containersv1alpha1.WasmtimeContainerConditionRepositoryReady),
				},
				AppendComponent(),
			},
		},

		Config: c,
	}
}

func ResolveComponent() reconcilers.SubReconciler[*containersv1alpha1.WasmtimeContainer] {
	return &reconcilers.SyncReconciler[*containersv1alpha1.WasmtimeContainer]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.WatchesRawSource(ComponentDuckBroker.TrackedSource(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource *containersv1alpha1.WasmtimeContainer) error {
			component, err := ResolveComponentReference(ctx, resource.Spec.Ref)
			if err != nil {
				if errors.Is(err, ErrNotComponent) {
					resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "NotComponent", "%s %s is not a component", resource.Spec.Ref.APIVersion, resource.Spec.Ref.Kind)
					return reconcilers.ErrHaltSubReconcilers
				}
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "ComponentNotFound", "%s %s not found", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
					return ErrDurable
				}
				return err
			}

			trace := append(ComponentTraceStasher.RetrieveOrEmpty(ctx), SynthesizeSpan(ctx, component))
			ComponentTraceStasher.Store(ctx, trace)
			if hasCycle, sanitizedTrace := DetectTraceCycle(trace, resource); hasCycle {
				resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "CycleDetected", "components may not reference themselves directly or transitively")
				resource.Status.Trace = sanitizedTrace
				return ErrDurable
			}

			if err := component.Spec.Default(ctx); err != nil {
				return err
			}
			// avoid premature reconciliation, check generation and ready condition
			if component.Generation != component.Status.ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "Blocked", "waiting for %s %s to reconcile", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				return ErrGenerationMismatch
			}
			if ready := component.Status.GetCondition(componentsv1alpha1.ComponentDuckConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "NotReady", "%s %s is not ready", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "NotReady", "%s %s is not ready", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				}
				return ErrDurable
			}

			if component.Status.Image == "" {
				// should never be ready and missing an image, but ya know
				resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "ImageMissing", "%s %s is missing image", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				return ErrDurable
			}

			RepositoryKeychainStasher.Clear(ctx)
			if _, err := ResolveRepository[*componentsv1alpha1.ComponentDuck](containersv1alpha1.WasmtimeContainerConditionComponentPulled).Reconcile(ctx, component); err != nil {
				return err
			}
			keychain, err := RepositoryKeychainStasher.RetrieveOrError(ctx)
			if err != nil {
				return err
			}
			RepositoryKeychainStasher.Clear(ctx)

			ref, err := name.NewDigest(component.Status.Image, name.WeakValidation)
			if err != nil {
				return err
			}
			componentBytes, config, err := registry.Pull(ctx, ref, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				return err
			}

			ComponentStasher.Store(ctx, componentBytes)
			ComponentConfigStasher.Store(ctx, config)
			resource.GetConditionManager(ctx).MarkTrue(containersv1alpha1.WasmtimeContainerConditionComponentPulled, "Resolved", "")

			return nil
		},
	}
}

func AppendComponent() reconcilers.SubReconciler[*containersv1alpha1.WasmtimeContainer] {
	return &reconcilers.SyncReconciler[*containersv1alpha1.WasmtimeContainer]{
		Sync: func(ctx context.Context, resource *containersv1alpha1.WasmtimeContainer) error {
			keychain := RepositoryKeychainStasher.RetrieveOrDie(ctx)
			tagRef := RepositoryTagStasher.RetrieveOrDie(ctx)
			component := ComponentStasher.RetrieveOrDie(ctx)

			baseRef, err := name.ParseReference(resource.Spec.BaseImage, name.WeakValidation)
			if err != nil {
				return err
			}

			digestRef, err := registry.AppendComponent(ctx, baseRef, tagRef, component, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				return err
			}

			RepositoryDigestStasher.Store(ctx, digestRef)
			resource.GetConditionManager(ctx).MarkTrue(containersv1alpha1.WasmtimeContainerConditionPushed, "Pushed", "")
			resource.Status.Image = digestRef.Name()

			return nil
		},
	}
}

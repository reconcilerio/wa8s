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
	"reconciler.io/wa8s/controllers"
	"reconciler.io/wa8s/registry"
)

//+kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=componentcontainerimages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=componentcontainerimages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=componentcontainerimages/finalizers,verbs=update
//+kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ComponentContainerImageReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*containersv1alpha1.ComponentContainerImage] {
	return &reconcilers.ResourceReconciler[*containersv1alpha1.ComponentContainerImage]{
		Reconciler: &reconcilers.SuppressTransientErrors[*containersv1alpha1.ComponentContainerImage, *containersv1alpha1.ComponentContainerImageList]{
			Reconciler: reconcilers.Sequence[*containersv1alpha1.ComponentContainerImage]{
				reconcilers.Always[*containersv1alpha1.ComponentContainerImage]{
					ResolveComponent(),
					controllers.ResolveImage[*containersv1alpha1.ComponentContainerImage](containersv1alpha1.ComponentContainerImageConditionImageReady),
					controllers.ResolveRepository[*containersv1alpha1.ComponentContainerImage](containersv1alpha1.ComponentContainerImageConditionRepositoryReady),
				},
				AppendComponent(),
			},
		},

		Config: c,
	}
}

func ResolveComponent() reconcilers.SubReconciler[*containersv1alpha1.ComponentContainerImage] {
	return &reconcilers.SyncReconciler[*containersv1alpha1.ComponentContainerImage]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.WatchesRawSource(controllers.ComponentDuckBroker.TrackedSource(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource *containersv1alpha1.ComponentContainerImage) error {
			component, err := controllers.ResolveComponentReference(ctx, resource.Spec.Ref)
			if err != nil {
				if errors.Is(err, controllers.ErrNotComponent) {
					resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "NotComponent", "%s %s is not a component", resource.Spec.Ref.APIVersion, resource.Spec.Ref.Kind)
					return reconcilers.ErrHaltSubReconcilers
				}
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "ComponentNotFound", "%s %s not found", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
					return ErrDurable
				}
				return err
			}

			trace := append(controllers.ComponentTraceStasher.RetrieveOrEmpty(ctx), controllers.SynthesizeSpan(ctx, component))
			controllers.ComponentTraceStasher.Store(ctx, trace)
			if hasCycle, sanitizedTrace := controllers.DetectTraceCycle(trace, resource); hasCycle {
				resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "CycleDetected", "components may not reference themselves directly or transitively")
				resource.Status.Trace = sanitizedTrace
				return ErrDurable
			}

			if err := component.Spec.Default(ctx); err != nil {
				return err
			}
			// avoid premature reconciliation, check generation and ready condition
			if component.Generation != component.Status.ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "Blocked", "waiting for %s %s to reconcile", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				return ErrGenerationMismatch
			}
			if ready := component.Status.GetCondition(componentsv1alpha1.ComponentDuckConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "NotReady", "%s %s is not ready", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "NotReady", "%s %s is not ready", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				}
				return ErrDurable
			}

			if component.Status.Image == "" {
				// should never be ready and missing an image, but ya know
				resource.GetConditionManager(ctx).MarkFalse(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "ImageMissing", "%s %s is missing image", resource.Spec.Ref.Kind, resource.Spec.Ref.Name)
				return ErrDurable
			}

			controllers.RepositoryKeychainStasher.Clear(ctx)
			if _, err := controllers.ResolveRepository[*componentsv1alpha1.ComponentDuck](containersv1alpha1.ComponentContainerImageConditionComponentPulled).Reconcile(ctx, component); err != nil {
				return err
			}
			keychain, err := controllers.RepositoryKeychainStasher.RetrieveOrError(ctx)
			if err != nil {
				return err
			}
			controllers.RepositoryKeychainStasher.Clear(ctx)

			ref, err := name.NewDigest(component.Status.Image, name.WeakValidation)
			if err != nil {
				return err
			}
			componentBytes, config, err := registry.Pull(ctx, ref, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				return err
			}

			controllers.ComponentStasher.Store(ctx, componentBytes)
			controllers.ComponentConfigStasher.Store(ctx, config)
			resource.GetConditionManager(ctx).MarkTrue(containersv1alpha1.ComponentContainerImageConditionComponentPulled, "Resolved", "")

			return nil
		},
	}
}

func AppendComponent() reconcilers.SubReconciler[*containersv1alpha1.ComponentContainerImage] {
	return &reconcilers.SyncReconciler[*containersv1alpha1.ComponentContainerImage]{
		Sync: func(ctx context.Context, resource *containersv1alpha1.ComponentContainerImage) error {
			keychain := controllers.RepositoryKeychainStasher.RetrieveOrDie(ctx)
			tagRef := controllers.RepositoryTagStasher.RetrieveOrDie(ctx)
			component := controllers.ComponentStasher.RetrieveOrDie(ctx)
			image := controllers.RemoteImageStasher.RetrieveOrDie(ctx)

			digestRef, err := registry.AppendComponent(ctx, image, tagRef, component, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				return err
			}

			controllers.RepositoryDigestStasher.Store(ctx, digestRef)
			resource.GetConditionManager(ctx).MarkTrue(containersv1alpha1.ComponentContainerImageConditionPushed, "Pushed", "")
			resource.Status.Image = digestRef.Name()

			return nil
		},
	}
}

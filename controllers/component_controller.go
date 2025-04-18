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
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	"reconciler.io/wa8s/registry"
)

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=components,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=components/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=components/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func ComponentReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[componentsv1alpha1.GenericComponent] {
	return genericComponentReconciler(c, &componentsv1alpha1.Component{}, &componentsv1alpha1.ComponentList{})
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=clustercomponents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=clustercomponents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=clustercomponents/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func ClusterComponentReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[componentsv1alpha1.GenericComponent] {
	return genericComponentReconciler(c, &componentsv1alpha1.ClusterComponent{}, &componentsv1alpha1.ClusterComponentList{})
}

func genericComponentReconciler(c reconcilers.Config, t componentsv1alpha1.GenericComponent, lt client.ObjectList) *reconcilers.ResourceReconciler[componentsv1alpha1.GenericComponent] {
	return &reconcilers.ResourceReconciler[componentsv1alpha1.GenericComponent]{
		Type: t,

		Reconciler: &reconcilers.SuppressTransientErrors[componentsv1alpha1.GenericComponent, client.ObjectList]{
			ListType: lt,
			Reconciler: reconcilers.Sequence[componentsv1alpha1.GenericComponent]{
				reconcilers.Always[componentsv1alpha1.GenericComponent]{
					ResolveRepository[componentsv1alpha1.GenericComponent](componentsv1alpha1.ComponentConditionRepositoryReady),
					ResolveKeychain(),
				},
				CopyComponent(),
				ReflectComponentableStatus[componentsv1alpha1.GenericComponent](),
			},
		},

		Config: c,
	}
}

func ResolveKeychain() *reconcilers.SyncReconciler[componentsv1alpha1.GenericComponent] {
	return &reconcilers.SyncReconciler[componentsv1alpha1.GenericComponent]{
		Sync: func(ctx context.Context, resource componentsv1alpha1.GenericComponent) error {
			if resource.GetSpec().OCI == nil {
				return nil
			}

			keychain, err := registry.KeychainManager.CreateForServiceAccountRef(ctx, resource.GetSpec().OCI.ServiceAccountRef)
			if err != nil {
				if apierrs.IsNotFound(err) {
					status := err.(apierrs.APIStatus).Status()
					kind := status.Kind
					name := status.Details.Name
					resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.ComponentConditionCopied, fmt.Sprintf("%sNotFound", kind), "%s %s not found", kind, name)
					return ErrDurable
				}
				return err
			}

			if kc, err := RepositoryKeychainStasher.RetrieveOrError(ctx); err == nil {
				// merge with existing stashed keychain
				keychain = authn.NewMultiKeychain(keychain, kc)
			}
			RepositoryKeychainStasher.Store(ctx, keychain)
			return nil
		},
	}
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch

func CopyComponent() *reconcilers.SyncReconciler[componentsv1alpha1.GenericComponent] {
	return &reconcilers.SyncReconciler[componentsv1alpha1.GenericComponent]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&corev1.Secret{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&corev1.ServiceAccount{}, reconcilers.EnqueueTracked(ctx))
			bldr.WatchesRawSource(source.Channel(ComponentDuckBroker.Subscribe(ctx), reconcilers.EnqueueTracked(ctx)))

			return nil
		},
		Sync: func(ctx context.Context, resource componentsv1alpha1.GenericComponent) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)
			conditionManager := resource.GetConditionManager(ctx)
			log := logr.FromContextOrDiscard(ctx)

			keychain := RepositoryKeychainStasher.RetrieveOrDie(ctx)
			tagRef := RepositoryTagStasher.RetrieveOrDie(ctx)

			var source name.Digest
			if oci := resource.GetSpec().OCI; oci != nil {
				var err error
				source, err = registry.ResolveDigest(ctx, oci.Image, remote.WithAuthFromKeychain(keychain))
				if err != nil {
					return err
				}
			} else if ref := resource.GetSpec().Ref; ref != nil {
				component, err := ResolveComponentReference(ctx, *ref)
				if err != nil {
					if errors.Is(err, ErrNotComponent) {
						conditionManager.MarkFalse(componentsv1alpha1.ComponentConditionCopied, "NotComponent", "%s %s is not a component", ref.APIVersion, ref.Kind)
						return reconcilers.ErrHaltSubReconcilers
					}
					if apierrs.IsNotFound(err) {
						conditionManager.MarkUnknown(componentsv1alpha1.ComponentConditionCopied, "ComponentNotFound", "component %s %s not found", ref.Kind, ref.Name)
						return ErrDurable
					}
					return err
				}

				trace := append(ComponentTraceStasher.RetrieveOrEmpty(ctx), SynthesizeSpan(ctx, component))
				ComponentTraceStasher.Store(ctx, trace)
				if hasCycle, sanitizedTrace := DetectTraceCycle(trace, resource); hasCycle {
					conditionManager.MarkFalse(componentsv1alpha1.ComponentConditionCopied, "CycleDetected", "components may not reference themselves directly or transitively")
					resource.GetGenericComponentStatus().Trace = sanitizedTrace
					return ErrDurable
				}

				ready := component.Status.GetCondition(componentsv1alpha1.ComponentDuckConditionReady)
				if apis.ConditionIsFalse(ready) {
					conditionManager.MarkFalse(componentsv1alpha1.ComponentConditionCopied, "Blocked", "component %s %s not ready: %s %s", ref.Kind, ref.Name, ready.Reason, ready.Message)
					return ErrDurable
				}
				if apis.ConditionIsUnknown(ready) {
					conditionManager.MarkUnknown(componentsv1alpha1.ComponentConditionCopied, "Blocked", "component %s %s not ready", ref.Kind, ref.Name)
					return ErrDurable
				}
				if component.Status.Image == "" {
					// should never be ready and missing the image, but ya know
					conditionManager.MarkUnknown(componentsv1alpha1.ComponentConditionCopied, "Blocked", "component %s %s missing image", ref.Kind, ref.Name)
					return ErrDurable
				}
				source, err = name.NewDigest(component.Status.Image)
				if err != nil {
					conditionManager.MarkFalse(componentsv1alpha1.ComponentConditionCopied, "InvalidImage", "component %s %s has invalid image: %s", ref.Kind, ref.Name, component.Status.Image)
					return ErrDurable
				}
			} else {
				panic(fmt.Errorf("image or ref must be defined"))
			}

			digestRef, err := registry.Copy(ctx, source, tagRef, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				log.Error(err, "failed to copy component", "repository", tagRef.Name())
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "CopyFailed", "%s", err)
				conditionManager.MarkFalse(componentsv1alpha1.ComponentConditionCopied, "CopyFailed", "failed to copy component to %q", tagRef.Name())
				return err
			}

			config, err := registry.PullConfig(ctx, digestRef, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				log.Error(err, "failed to load component config", "repository", tagRef.Name())
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "CopyFailed", "%s", err)
				conditionManager.MarkFalse(componentsv1alpha1.ComponentConditionCopied, "CopyFailed", "failed to copy component to %q", tagRef.Name())
				return err
			}

			conditionManager.MarkTrue(componentsv1alpha1.ComponentConditionCopied, "Copied", "")

			RepositoryDigestStasher.Store(ctx, digestRef)
			ComponentConfigStasher.Store(ctx, config)

			return nil
		},
	}
}

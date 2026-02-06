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

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
	"reconciler.io/wa8s/services/lifecycle"
)

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ServiceInstanceReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceInstance] {
	return &reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceInstance]{
		SyncStatusDuringFinalization: true,
		Reconciler: &reconcilers.WithFinalizer[*servicesv1alpha1.ServiceInstance]{
			Finalizer: fmt.Sprintf("%s/reconciler", servicesv1alpha1.GroupVersion.Group),
			Reconciler: &reconcilers.SuppressTransientErrors[*servicesv1alpha1.ServiceInstance, *servicesv1alpha1.ServiceInstanceList]{
				Reconciler: reconcilers.Sequence[*servicesv1alpha1.ServiceInstance]{
					ResolveServiceLifecycle(),
					ManageServiceInstance(),
				},
			},
			ReadyToClearFinalizer: func(ctx context.Context, resource *servicesv1alpha1.ServiceInstance) bool {
				// block deletion while Finalizer condition exist
				return resource.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceInstanceConditionFinalizer) == nil
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicelifecycles,verbs=get;list;watch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles,verbs=get;list;watch

func ResolveServiceLifecycle() reconcilers.SubReconciler[*servicesv1alpha1.ServiceInstance] {
	return &reconcilers.SyncReconciler[*servicesv1alpha1.ServiceInstance]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&servicesv1alpha1.ClusterServiceLifecycle{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&servicesv1alpha1.ServiceLifecycle{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		SyncDuringFinalization: true,
		Sync: func(ctx context.Context, resource *servicesv1alpha1.ServiceInstance) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			lifecycleRef := resource.Spec.Ref
			var lifecycle servicesv1alpha1.GenericServiceLifecycle
			if lifecycleRef.Kind == "ClusterServiceLifecycle" {
				lifecycle = &servicesv1alpha1.ClusterServiceLifecycle{
					ObjectMeta: metav1.ObjectMeta{
						Name: lifecycleRef.Name,
					},
				}
			} else {
				lifecycle = &servicesv1alpha1.ServiceLifecycle{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: resource.Namespace,
						Name:      lifecycleRef.Name,
					},
				}
			}
			if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(lifecycle), lifecycle); err != nil {
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceConditionLifecycleReady, "ServiceLifecycleNotFound", "%s %s not found", lifecycleRef.Kind, lifecycleRef.Name)
					return ErrDurable
				}
				return err
			}

			// avoid premature reconciliation, check generation and ready condition
			if lifecycle.GetGeneration() != lifecycle.GetStatus().ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceInstanceConditionLifecycleReady, "Blocked", "waiting for %s %s to reconcile", lifecycleRef.Kind, lifecycleRef.Name)
				return ErrGenerationMismatch
			}
			lifecycle.GetStatus().InitializeConditions(ctx)
			if ready := lifecycle.GetStatus().GetCondition(servicesv1alpha1.ServiceLifecycleConditionReady); !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceConditionLifecycleReady, "ServiceLifecycleNotReady", "%s: %s", ready.Reason, ready.Message)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceInstanceConditionLifecycleReady, "ServiceLifecycleNotReady", "%s: %s", ready.Reason, ready.Message)
				}
				return ErrDurable
			}

			if err := lifecycle.Default(ctx); err != nil {
				return err
			}
			address := lifecycle.GetStatus().URL
			if address == "" {
				// should never be Ready and not have a URL, but ya know
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceInstanceConditionLifecycleReady, "MissingURL", "the %s status URL is required", lifecycleRef.Kind)
			}

			ServiceLifecycleAddressStasher.Store(ctx, address)
			resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceInstanceConditionLifecycleReady, "Ready", "")

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicebindings,verbs=get;list;watch

func ManageServiceInstance() reconcilers.SubReconciler[*servicesv1alpha1.ServiceInstance] {
	return &reconcilers.SyncReconciler[*servicesv1alpha1.ServiceInstance]{
		Setup: func(ctx context.Context, mgr controllerruntime.Manager, bldr *builder.Builder) error {
			bldr.Watches(&servicesv1alpha1.ServiceBinding{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},

		Sync: func(ctx context.Context, resource *servicesv1alpha1.ServiceInstance) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			if resource.Status.ServiceInstanceId != "" {
				// previously provisioned
				// TODO check for drift and update if needed
				resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceInstanceConditionProvisioned, "Provisioned", "")
				return nil
			}

			address := ServiceLifecycleAddressStasher.RetrieveOrDie(ctx)
			instanceId := string(resource.UID)
			type_ := resource.Spec.Type
			var tier *string
			if resource.Spec.Tier != "" {
				tier = &resource.Spec.Tier
			}
			requests := resource.Spec.Requests

			err := lifecycle.NewLifecycle(address).Provision(ctx, instanceId, type_, tier, requests)
			if err != nil {
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "ProvisionFailed", "%s", err)
				return err
			}
			c.Recorder.Eventf(resource, corev1.EventTypeNormal, "Provisioned", "")

			resource.Status.ServiceInstanceId = instanceId
			resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceInstanceConditionProvisioned, "Provisioned", "")

			return ErrUpdateStatusBeforeContinuingReconcile
		},
		Finalize: func(ctx context.Context, resource *servicesv1alpha1.ServiceInstance) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			if resource.Status.ServiceInstanceId == "" {
				// nothing to do
				return nil
			}

			// check for ServiceBindings dependent on this ServiceInstance
			// No need to check for ServiceClients as a client will implicitly create a binding
			serviceBindings := &servicesv1alpha1.ServiceBindingList{}
			if err := c.List(ctx, serviceBindings, client.InNamespace(resource.Namespace)); err != nil {
				return err
			}
			for _, serviceBinding := range serviceBindings.Items {
				if serviceBinding.Spec.Ref.Name == resource.Name {
					// track client to be deleted
					c.Tracker.TrackObject(&serviceBinding, resource)
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceConditionFinalizer, "ServiceBindingExists", "deletion blocked by ServiceBinding %s", serviceBinding.Name)
					return ErrDurable
				}
			}

			address := ServiceLifecycleAddressStasher.RetrieveOrDie(ctx)
			instanceId := resource.Status.ServiceInstanceId
			retain := false

			if err := lifecycle.NewLifecycle(address).Destroy(ctx, instanceId, &retain); err != nil {
				// TODO handle case where the instance doesn't exist
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "DestroyFailed", "%s", err)
				resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceConditionFinalizer, "DestroyFailed", "%s", err)
				return ErrDurable
			}
			resource.GetConditionManager(ctx).ClearCondition(servicesv1alpha1.ServiceInstanceConditionFinalizer)
			c.Recorder.Eventf(resource, corev1.EventTypeNormal, "Destroyed", "")

			return nil
		},
	}
}

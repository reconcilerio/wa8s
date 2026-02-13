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
	"errors"
	"fmt"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
	"reconciler.io/wa8s/internal/defaults"
)

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicelifecycles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicelifecycles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicelifecycles/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ServiceLifecycleReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	childLabelKey := fmt.Sprintf("%s/service-lifecycle", servicesv1alpha1.GroupVersion.Group)
	return genericServiceLifecycleReconciler(c, &servicesv1alpha1.ServiceLifecycle{}, &servicesv1alpha1.ServiceLifecycleList{}, childLabelKey)
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ClusterServiceLifecycleReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	childLabelKey := fmt.Sprintf("%s/cluster-service-lifecycle", servicesv1alpha1.GroupVersion.Group)
	return genericServiceLifecycleReconciler(c, &servicesv1alpha1.ClusterServiceLifecycle{}, &servicesv1alpha1.ClusterServiceLifecycleList{}, childLabelKey)
}

func genericServiceLifecycleReconciler(c reconcilers.Config, t servicesv1alpha1.GenericServiceLifecycle, lt client.ObjectList, childLabelKey string) *reconcilers.ResourceReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	return &reconcilers.ResourceReconciler[servicesv1alpha1.GenericServiceLifecycle]{
		Type: t,

		SyncStatusDuringFinalization: true,
		Reconciler: &reconcilers.WithFinalizer[servicesv1alpha1.GenericServiceLifecycle]{
			Finalizer: fmt.Sprintf("%s/reconciler", servicesv1alpha1.GroupVersion.Group),
			Reconciler: &reconcilers.SuppressTransientErrors[servicesv1alpha1.GenericServiceLifecycle, client.ObjectList]{
				ListType: lt,
				Reconciler: reconcilers.Sequence[servicesv1alpha1.GenericServiceLifecycle]{
					CheckForInstancesBeforeFinalizing(),
					&reconcilers.IfThen[servicesv1alpha1.GenericServiceLifecycle]{
						If: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) bool {
							return resource.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceLifecycleConditionFinalizer) == nil
						},
						Then: reconcilers.Sequence[servicesv1alpha1.GenericServiceLifecycle]{
							ServiceLifecycleCompositionChildReconciler(childLabelKey),
							HttpTriggerChildReconciler(childLabelKey),
							ClientComponentReconciler(),
						},
					},
				},
			},
			ReadyToClearFinalizer: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) bool {
				// block deletion while Finalizer condition exist
				return resource.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceLifecycleConditionFinalizer) == nil
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances,verbs=get;list;watch

func CheckForInstancesBeforeFinalizing() reconcilers.SubReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	return &reconcilers.SyncReconciler[servicesv1alpha1.GenericServiceLifecycle]{
		Setup: func(ctx context.Context, mgr controllerruntime.Manager, bldr *builder.Builder) error {
			bldr.Watches(&servicesv1alpha1.ServiceInstance{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) error {
			// nothing to do
			return nil
		},
		Finalize: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			// check for ServiceInstances dependent on this (Cluster)ServiceLifecycle
			gvk, err := c.GroupVersionKindFor(resource)
			if err != nil {
				return err
			}
			namespaced, err := c.IsObjectNamespaced(resource)
			if err != nil {
				return err
			}

			serviceInstances := &servicesv1alpha1.ServiceInstanceList{}
			if err := c.List(ctx, serviceInstances, client.InNamespace(resource.GetNamespace())); err != nil {
				return err
			}
			for _, serviceInstance := range serviceInstances.Items {
				if err := serviceInstance.Default(ctx); err != nil {
					return err
				}

				if serviceInstance.Spec.Ref.Name == resource.GetName() && serviceInstance.Spec.Ref.Kind == gvk.Kind {
					if !namespaced || serviceInstance.Spec.Ref.Namespace == resource.GetNamespace() {
						// track client to be deleted
						if err := c.Tracker.TrackObject(&serviceInstance, resource); err != nil {
							return err
						}
						if resource.GetNamespace() == "" {
							resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionFinalizer, "ServiceInstanceExists", "deletion blocked by ServiceInstance %s/%s", serviceInstance.Namespace, serviceInstance.Name)
						} else {
							resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionFinalizer, "ServiceInstanceExists", "deletion blocked by ServiceInstance %s", serviceInstance.Name)
						}
						return nil
					}
				}
			}

			if err := resource.GetConditionManager(ctx).ClearCondition(servicesv1alpha1.ServiceLifecycleConditionFinalizer); err != nil {
				return err
			}

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=compositions,verbs=get;list;watch;create;update;patch;delete

func ServiceLifecycleCompositionChildReconciler(childLabelKey string) reconcilers.SubReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	return &reconcilers.ChildReconciler[servicesv1alpha1.GenericServiceLifecycle, *componentsv1alpha1.Composition, *componentsv1alpha1.CompositionList]{
		DesiredChild: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) (*componentsv1alpha1.Composition, error) {
			child := &componentsv1alpha1.Composition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    resource.GetNamespace(),
					GenerateName: fmt.Sprintf("%s-", resource.GetName()),
					Labels: reconcilers.MergeMaps(
						resource.GetLabels(),
						map[string]string{
							childLabelKey: resource.GetName(),
						},
					),
				},
				Spec: componentsv1alpha1.CompositionSpec{
					GenericCompositionSpec: componentsv1alpha1.GenericCompositionSpec{
						Dependencies: []componentsv1alpha1.CompositionDependency{
							{
								Component: "componentized:services-host",
								Ref: &componentsv1alpha1.ComponentReference{
									Kind: "ClusterComponent",
									Name: "wa8s-services-lifecycle-host-http",
								},
							},
							{
								Component: "componentized:services-lifecycle",
								Composition: &componentsv1alpha1.GenericCompositionSpec{
									Dependencies: []componentsv1alpha1.CompositionDependency{
										{
											Component: "componentized:services-lifecycle",
											Ref:       &resource.GetSpec().Ref,
										},
										{
											Component: "componentized:services-credential-admin",
											Ref: &componentsv1alpha1.ComponentReference{
												Kind: "ClusterComponent",
												Name: "wa8s-services-credential-admin",
											},
										},
									},
								},
							},
							{
								Component: "componentized:logging",
								Ref: &componentsv1alpha1.ComponentReference{
									Kind: "ClusterComponent",
									Name: "wa8s-services-logging",
								},
							},
						},
					},
				},
			}

			if child.Namespace == "" { // TODO parameterize
				child.Namespace = defaults.Namespace()
			}

			return child, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*componentsv1alpha1.Composition]{
			MergeBeforeUpdate: func(current, desired *componentsv1alpha1.Composition) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent servicesv1alpha1.GenericServiceLifecycle, child *componentsv1alpha1.Composition, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "Unknown", "")
				}
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "Missing", "")
				return ErrDurable
			}
			ServiceLifecycleCompositionStasher.Store(ctx, child.Name)

			// avoid premature reconciliation, check generation and ready condition
			if child.Generation != child.Status.ObservedGeneration {
				parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "Blocked", "waiting for Composition %s to reconcile", child.Name)
				return ErrGenerationMismatch
			}

			if ready := child.Status.GetCondition(componentsv1alpha1.CompositionConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "NotReady", "child Composition %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "NotReady", "child Composition %s is not ready", child.Name)
				}
				return ErrDurable
			}

			if child.Status.Image == "" {
				// should never be ready and missing an image, but ya know
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "ImageMissing", "child Composition %s is missing image", child.Name)
				return ErrDurable
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "Ready", "")

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=httptriggers,verbs=get;list;watch;create;update;patch;delete

func HttpTriggerChildReconciler(childLabelKey string) reconcilers.SubReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	return &reconcilers.ChildReconciler[servicesv1alpha1.GenericServiceLifecycle, *containersv1alpha1.HttpTrigger, *containersv1alpha1.HttpTriggerList]{
		DesiredChild: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) (*containersv1alpha1.HttpTrigger, error) {
			child := &containersv1alpha1.HttpTrigger{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: resource.GetNamespace(),
					Name:      fmt.Sprintf("%s-lifecycle", resource.GetName()),
					Labels: reconcilers.MergeMaps(
						resource.GetLabels(),
						map[string]string{
							childLabelKey: resource.GetName(),
						},
					),
				},
				Spec: containersv1alpha1.HttpTriggerSpec{
					GenericContainerSpec: containersv1alpha1.GenericContainerSpec{
						GenericComponentSpec: resource.GetSpec().GenericComponentSpec,
						Ref: componentsv1alpha1.ComponentReference{
							Kind: "Composition",
							Name: ServiceLifecycleCompositionStasher.RetrieveOrDie(ctx),
						},
						ServiceAccountRef: resource.GetSpec().ServiceAccountRef,
						HostCapabilities:  resource.GetSpec().HostCapabilities,
					},
				},
			}

			if child.Namespace == "" {
				// TODO parameterize
				child.Namespace = defaults.Namespace()
			}

			return child, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*containersv1alpha1.HttpTrigger]{
			MergeBeforeUpdate: func(current, desired *containersv1alpha1.HttpTrigger) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent servicesv1alpha1.GenericServiceLifecycle, child *containersv1alpha1.HttpTrigger, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "AlreadyExists", "%s", apierrs.ReasonForError(err))
					return ErrDurable
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "Unknown", "")
				}
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "Missing", "")
				return ErrDurable
			}

			// avoid premature reconciliation, check generation and ready condition
			if child.Generation != child.Status.ObservedGeneration {
				parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "Blocked", "waiting for HttpTrigger %s to reconcile", child.Name)
				return ErrGenerationMismatch
			}

			if ready := child.Status.GetCondition(containersv1alpha1.HttpTriggerConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "NotReady", "child HttpTrigger %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "NotReady", "child HttpTrigger %s is not ready", child.Name)
				}
				return ErrDurable
			}

			if child.Status.URL == "" {
				// should never be ready and missing an image, but ya know
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "URLMissing", "HttpTrigger %s is missing url", child.Name)
				return ErrDurable
			}

			parent.GetStatus().URL = child.Status.URL
			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceLifecycleConditionLifecycleReady, "Ready", "")

			return nil
		},
	}
}

func ClientComponentReconciler() reconcilers.SubReconciler[servicesv1alpha1.GenericServiceLifecycle] {
	return &reconcilers.SyncReconciler[servicesv1alpha1.GenericServiceLifecycle]{
		Setup: func(ctx context.Context, mgr controllerruntime.Manager, bldr *builder.Builder) error {
			bldr.WatchesRawSource(ComponentDuckBroker.TrackedSource(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource servicesv1alpha1.GenericServiceLifecycle) error {
			ref := resource.GetSpec().ClientRef
			component, err := ResolveComponentReference(ctx, ref)
			if err != nil {
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionClientReady, "ComponentNotFound", "%s %s not found", ref.Kind, ref.Name)
					return ErrDurable
				}
				return err
			}
			if err := component.Spec.Default(ctx); err != nil {
				return err
			}
			// avoid premature reconciliation, check generation and ready condition
			if component.Generation != component.Status.ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionClientReady, "Blocked", "waiting for %s %s to reconcile", ref.Kind, ref.Name)
				return ErrGenerationMismatch
			}
			if ready := component.Status.GetCondition(componentsv1alpha1.ComponentDuckConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionClientReady, "NotReady", "%s %s is not ready", ref.Kind, ref.Name)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceLifecycleConditionClientReady, "NotReady", "%s %s is not ready", ref.Kind, ref.Name)
				}
				return ErrDurable
			}

			resource.GetStatus().GenericComponentStatus = component.Status.GenericComponentStatus
			if resource.GetStatus().Image == "" {
				// should never be ready and missing an image, but ya know
				resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceLifecycleConditionComponentReady, "ImageMissing", "%s %s is missing image", ref.Kind, ref.Name)
			} else {
				resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceLifecycleConditionClientReady, "Ready", "")
			}

			return nil
		},
	}
}

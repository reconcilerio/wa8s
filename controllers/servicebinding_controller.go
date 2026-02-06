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

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	rtime "reconciler.io/runtime/time"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
	"reconciler.io/wa8s/services/lifecycle"
)

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicebindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicebindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ServiceBindingReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceBinding] {
	childLabelKey := fmt.Sprintf("%s/service-binding", servicesv1alpha1.GroupVersion.Group)

	return &reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceBinding]{
		Reconciler: &reconcilers.WithFinalizer[*servicesv1alpha1.ServiceBinding]{
			Finalizer: fmt.Sprintf("%s/reconciler", servicesv1alpha1.GroupVersion.Group),
			Reconciler: &reconcilers.SuppressTransientErrors[*servicesv1alpha1.ServiceBinding, *servicesv1alpha1.ServiceBindingList]{
				Reconciler: reconcilers.Sequence[*servicesv1alpha1.ServiceBinding]{
					ResolveServiceInstanceId(),
					ServiceBindingSecret(),
					ManageServiceBinding(),
					ConfigureClientComponent(),
					ExpirationRequeue(),
					ComponentChildReconciler[*servicesv1alpha1.ServiceBinding](servicesv1alpha1.ServiceBindingConditionChildComponent, childLabelKey, nil),
				},
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances,verbs=get;list;watch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicelifecycles,verbs=get;list;watch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles,verbs=get;list;watch

func ResolveServiceInstanceId() reconcilers.SubReconciler[*servicesv1alpha1.ServiceBinding] {
	return &reconcilers.SyncReconciler[*servicesv1alpha1.ServiceBinding]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&servicesv1alpha1.ServiceInstance{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&servicesv1alpha1.ServiceLifecycle{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&servicesv1alpha1.ClusterServiceLifecycle{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		SyncDuringFinalization: true,
		Sync: func(ctx context.Context, resource *servicesv1alpha1.ServiceBinding) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			instance := &servicesv1alpha1.ServiceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: resource.Namespace,
					Name:      resource.Spec.Ref.Name,
				},
			}
			if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionInstanceReady, "NotFound", "ServiceInstance %s not found", instance.Name)
					return ErrDurable
				}
				return err
			}

			// avoid premature reconciliation, check generation and ready condition
			if instance.GetGeneration() != instance.Status.ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionInstanceReady, "Blocked", "waiting for ServiceInstance %s to reconcile", instance.Name)
				return ErrGenerationMismatch
			}
			instance.Status.InitializeConditions(ctx)
			if ready := instance.Status.GetCondition(servicesv1alpha1.ServiceInstanceConditionReady); !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionInstanceReady, "NotReady", "%s: %s", ready.Reason, ready.Message)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionInstanceReady, "NotReady", "%s: %s", ready.Reason, ready.Message)
				}
				return ErrDurable
			}

			if err := instance.Default(ctx); err != nil {
				return err
			}
			instanceId := instance.Status.ServiceInstanceId
			if instanceId == "" {
				// should never be Ready and not have a ServiceInstanceId, but ya know
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionInstanceReady, "MissingServiceInstanceId", "the ServiceInstance status ServiceInstanceId is required")
			}

			ServiceInstanceIdStasher.Store(ctx, instanceId)
			resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceBindingConditionInstanceReady, "Ready", "")

			// resolve ServiceLifecycle URL
			lifecycleRef := instance.Spec.Ref
			ServiceLifecycleReferenceStasher.Store(ctx, lifecycleRef)
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
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionInstanceReady, "NotFound", "%s %s not found", lifecycleRef.Kind, lifecycleRef.Name)
					return ErrDurable
				}
				return err
			}

			// avoid premature reconciliation, check generation and ready condition
			if lifecycle.GetGeneration() != lifecycle.GetStatus().ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionInstanceReady, "Blocked", "waiting for %s %s to reconcile", lifecycleRef.Kind, lifecycleRef.Name)
				return ErrGenerationMismatch
			}
			address := lifecycle.GetStatus().URL
			if address == "" {
				// should never be Ready and not have a URL, but ya know
				resource.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionInstanceReady, "MissingURL", "the %s status URL is required", lifecycleRef.Kind)
			}
			ServiceLifecycleAddressStasher.Store(ctx, address)

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

func ServiceBindingSecret() reconcilers.SubReconciler[*servicesv1alpha1.ServiceBinding] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceBinding, *corev1.Secret, *corev1.SecretList]{
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceBinding) (*corev1.Secret, error) {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: resource.Namespace,
					Name:      fmt.Sprintf("service-binding-%s", resource.UID),
					Labels: map[string]string{
						"services.wa8s.reconciler.io/type": "binding",
						"services.wa8s.reconciler.io/id":   string(resource.UID),
					},
					// finalizer is cleared by the credential-admin
					Finalizers: []string{ServiceCredentialFinalizer},
				},
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*corev1.Secret]{
			MergeBeforeUpdate: func(current, desired *corev1.Secret) {
				current.Labels = desired.Labels
				// data values are set externally, don't overwrite
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceBinding, child *corev1.Secret, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionSecret, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionSecret, "AlreadyExists", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionSecret, "Unknown", "")
				}
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionSecret, "Missing", "")
				return ErrDurable
			}

			ServiceBindingIdStasher.Store(ctx, string(parent.UID))
			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceBindingConditionSecret, "Created", "")

			return nil
		},
	}
}

func ManageServiceBinding() reconcilers.SubReconciler[*servicesv1alpha1.ServiceBinding] {
	return &reconcilers.SyncReconciler[*servicesv1alpha1.ServiceBinding]{
		Sync: func(ctx context.Context, resource *servicesv1alpha1.ServiceBinding) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			if resource.Status.ServiceBindingId != "" {
				// previously bound

				if resource.Status.Expired {
					// previously expired
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionBound, "Expired", "")
					return nil
				}

				if resource.CreationTimestamp.Time.Before(rtime.RetrieveNow(ctx).Add(-1 * resource.Spec.Duration.Duration)) {
					// newly expired, unbind

					address := ServiceLifecycleAddressStasher.RetrieveOrDie(ctx)
					bindingId := ServiceBindingIdStasher.RetrieveOrDie(ctx)
					instanceId := ServiceInstanceIdStasher.RetrieveOrDie(ctx)

					if err := lifecycle.NewLifecycle(address).Unbind(ctx, bindingId, instanceId); err != nil {
						// TODO handle case where the binding doesn't exist
						c.Recorder.Eventf(resource, corev1.EventTypeWarning, "ExpireFailed", "%s", err)
						return err
					}
					c.Recorder.Eventf(resource, corev1.EventTypeNormal, "Expired", "")
					resource.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionBound, "Expired", "")
					resource.Status.Expired = true

					return nil
				}

				resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceBindingConditionBound, "Bound", "")

				return nil
			}

			address := ServiceLifecycleAddressStasher.RetrieveOrDie(ctx)
			bindingId := ServiceBindingIdStasher.RetrieveOrDie(ctx)
			instanceId := ServiceInstanceIdStasher.RetrieveOrDie(ctx)
			scopes := resource.Spec.Scopes

			if err := lifecycle.NewLifecycle(address).Bind(ctx, bindingId, instanceId, scopes); err != nil {
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "BindingFailed", "%s", err)
				return err
			}
			c.Recorder.Eventf(resource, corev1.EventTypeNormal, "Bound", "")

			resource.Status.Binding.Name = fmt.Sprintf("service-binding-%s", bindingId)
			resource.Status.ServiceBindingId = bindingId
			resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceBindingConditionBound, "Bound", "")

			return ErrUpdateStatusBeforeContinuingReconcile
		},
		Finalize: func(ctx context.Context, resource *servicesv1alpha1.ServiceBinding) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			if resource.Status.ServiceBindingId == "" || resource.Status.Expired {
				// nothing to do
				return nil
			}

			address := ServiceLifecycleAddressStasher.RetrieveOrDie(ctx)
			bindingId := resource.Status.ServiceBindingId
			instanceId := resource.Status.ServiceBindingId

			if err := lifecycle.NewLifecycle(address).Unbind(ctx, bindingId, instanceId); err != nil {
				// TODO handle case where the binding doesn't exist
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "UnbindFailed", "%s", err)
				return err
			}
			c.Recorder.Eventf(resource, corev1.EventTypeNormal, "Unbound", "")

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=compositions,verbs=get;list;watch;create;update;patch;delete

func ConfigureClientComponent() reconcilers.SubReconciler[*servicesv1alpha1.ServiceBinding] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceBinding, *componentsv1alpha1.Composition, *componentsv1alpha1.CompositionList]{
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceBinding) (*componentsv1alpha1.Composition, error) {
			lifecycleRef := ServiceLifecycleReferenceStasher.RetrieveOrDie(ctx)

			return &componentsv1alpha1.Composition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    resource.Namespace,
					GenerateName: fmt.Sprintf("%s-", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							fmt.Sprintf("%s/service-binding", servicesv1alpha1.GroupVersion.Group): resource.Name,
						},
					),
				},
				Spec: componentsv1alpha1.CompositionSpec{
					GenericCompositionSpec: componentsv1alpha1.GenericCompositionSpec{
						Dependencies: []componentsv1alpha1.CompositionDependency{
							{
								Component: "componentized:service-lifecycle-client",
								Ref: &componentsv1alpha1.ComponentReference{
									APIVersion: servicesv1alpha1.GroupVersion.String(),
									Kind:       lifecycleRef.Kind,
									Namespace:  lifecycleRef.Namespace,
									Name:       lifecycleRef.Name,
								},
							},
							{
								Component: "componentized:service-credentials",
								Composition: &componentsv1alpha1.GenericCompositionSpec{
									Dependencies: []componentsv1alpha1.CompositionDependency{
										{
											Component: "componentized:credential-config",
											Ref: &componentsv1alpha1.ComponentReference{
												Kind: "ClusterComponent",
												Name: "wa8s-services-credential-config",
											},
										},
										{
											// TODO the credenial-store should be late bound, and left as an import here
											Component: "componentized:credential-store",
											Ref: &componentsv1alpha1.ComponentReference{
												Kind: "ClusterComponent",
												Name: "wa8s-services-credential-store",
											},
										},
										{
											Component: "componentized:binding-id",
											Config: &componentsv1alpha1.GenericConfigStoreSpec{
												Values: []componentsv1alpha1.Value{
													{
														Name:  "binding-id",
														Value: resource.Status.ServiceBindingId,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*componentsv1alpha1.Composition]{
			MergeBeforeUpdate: func(current, desired *componentsv1alpha1.Composition) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceBinding, child *componentsv1alpha1.Composition, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionClientReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionClientReady, "Unknown", "")
				}
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionClientReady, "Missing", "")
				return ErrDurable
			}

			// avoid premature reconciliation, check generation and ready condition
			if child.Generation != child.Status.ObservedGeneration {
				parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionClientReady, "Blocked", "waiting for client Composition %s to reconcile", child.Name)
				return ErrGenerationMismatch
			}

			if ready := child.Status.GetCondition(componentsv1alpha1.CompositionConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionClientReady, "NotReady", "client Composition %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceBindingConditionClientReady, "NotReady", "client Composition %s is not ready", child.Name)
				}
				return ErrDurable
			}

			if child.Status.Image == "" {
				// should never be ready and missing an image, but ya know
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceBindingConditionClientReady, "ImageMissing", "client Composition %s is missing image", child.Name)
				return ErrDurable
			}

			parent.Status.GenericComponentStatus = child.Status.GenericComponentStatus
			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceBindingConditionClientReady, "Ready", "")

			return nil
		},
	}
}

func ExpirationRequeue() reconcilers.SubReconciler[*servicesv1alpha1.ServiceBinding] {
	return &reconcilers.SyncReconciler[*servicesv1alpha1.ServiceBinding]{
		SyncWithResult: func(ctx context.Context, resource *servicesv1alpha1.ServiceBinding) (reconcilers.Result, error) {
			if resource.Status.Expired {
				return reconcile.Result{}, nil
			}

			now := rtime.RetrieveNow(ctx)
			expiration := resource.CreationTimestamp.Time.Add(resource.Spec.Duration.Duration)
			resource.Status.ExpiresAfter = metav1.NewTime(expiration)

			if after := expiration.Sub(now); after > 0 {
				return reconcile.Result{RequeueAfter: after}, nil
			}

			// should have already expired
			return reconcile.Result{Requeue: true}, nil
		},
	}
}

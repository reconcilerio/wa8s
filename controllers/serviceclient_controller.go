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
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	rtime "reconciler.io/runtime/time"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
)

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceclients/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func ServiceClientReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceClient] {
	return &reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceClient]{
		Reconciler: &reconcilers.SuppressTransientErrors[*servicesv1alpha1.ServiceClient, *servicesv1alpha1.ServiceClientList]{
			Reconciler: reconcilers.Sequence[*servicesv1alpha1.ServiceClient]{
				StampServiceBinding(),
				RenewRequeue(),
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=servicebindings,verbs=get;list;watch;create;update;patch;delete

func StampServiceBinding() reconcilers.SubReconciler[*servicesv1alpha1.ServiceClient] {
	return &reconcilers.ChildSetReconciler[*servicesv1alpha1.ServiceClient, *servicesv1alpha1.ServiceBinding, *servicesv1alpha1.ServiceBindingList]{
		DesiredChildren: func(ctx context.Context, resource *servicesv1alpha1.ServiceClient) ([]*servicesv1alpha1.ServiceBinding, error) {
			now := rtime.RetrieveNow(ctx)

			var current *servicesv1alpha1.ServiceBinding
			desired := []*servicesv1alpha1.ServiceBinding{}
			for _, child := range reconcilers.RetrieveKnownChildren[*servicesv1alpha1.ServiceBinding](ctx) {
				child.Labels = reconcilers.MergeMaps(
					resource.Labels,
					map[string]string{
						fmt.Sprintf("%s/service-client", servicesv1alpha1.GroupVersion.Group): resource.Name,
					},
				)

				// prune expired children, or children whose duration is significantly longer than the current duration
				if child.Status.Expired || child.CreationTimestamp.Add(resource.Spec.Duration.Duration+5*time.Minute).Before(now) {
					continue
				}

				if current == nil || current.CreationTimestamp.Before(&child.CreationTimestamp) {
					current = child
				}

				desired = append(desired, child)
			}

			if current != nil {
				if equality.Semantic.DeepEqual(current.Spec.Ref, resource.Spec.Ref) && equality.Semantic.DeepEqual(current.Spec.Scopes, resource.Spec.Scopes) && equality.Semantic.DeepEqual(current.Spec.Duration, resource.Spec.Duration) {
					// in sync
					if current.CreationTimestamp.Add(resource.Spec.Duration.Duration - resource.Spec.RenewBefore.Duration).After(now) {
						return desired, nil
					}
				}
			}

			current = &servicesv1alpha1.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    resource.Namespace,
					GenerateName: fmt.Sprintf("%s-", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							fmt.Sprintf("%s/service-client", servicesv1alpha1.GroupVersion.Group): resource.Name,
						},
					),
				},
				Spec: servicesv1alpha1.ServiceBindingSpec{
					Ref:      resource.Spec.Ref,
					Scopes:   resource.Spec.Scopes,
					Duration: resource.Spec.Duration,
				},
			}

			return append(desired, current), nil
		},
		IdentifyChild: func(child *servicesv1alpha1.ServiceBinding) string {
			if child.CreationTimestamp.IsZero() {
				return "current"
			}
			return string(child.UID)
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*servicesv1alpha1.ServiceBinding]{
			MergeBeforeUpdate: func(current, desired *servicesv1alpha1.ServiceBinding) {
				current.Labels = desired.Labels
				// will never update an existing ServiceBinding spec
			},
		},
		ReflectChildrenStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceClient, result reconcilers.ChildSetResult[*servicesv1alpha1.ServiceBinding]) error {
			if err := result.AggregateError(); err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientConditionBound, "Invalid", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceClientConditionBound, "Unknown", "")
				}
				return errors.Join(err, ErrTransient)
			}
			var current *servicesv1alpha1.ServiceBinding
			for _, childResult := range result.Children {
				child := childResult.Child
				if child != nil && (current == nil || current.CreationTimestamp.Before(&child.CreationTimestamp)) {
					current = child
				}
			}

			if current == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientConditionBound, "Missing", "")
				return ErrDurable
			}

			parent.Status.RenewsAfter = metav1.NewTime(current.CreationTimestamp.Add(parent.Spec.Duration.Duration - parent.Spec.RenewBefore.Duration))

			// avoid premature reconciliation, check generation and ready condition
			if current.Generation != current.Status.ObservedGeneration {
				parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceClientConditionBound, "Blocked", "waiting for ServiceBinding %s to reconcile", current.Name)
				return ErrGenerationMismatch
			}

			if ready := current.Status.GetCondition(servicesv1alpha1.ServiceBindingConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientConditionBound, "NotReady", "ServiceBinding %s is not ready", current.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceClientConditionBound, "NotReady", "ServiceBinding %s is not ready", current.Name)
				}
				return ErrDurable
			}

			if current.Status.GenericComponentStatus.Image == "" {
				// should never be ready and missing an image, but ya know
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientConditionBound, "ImageMissing", "ServiceBinding %s is missing image", current.Name)
				return ErrDurable
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceClientConditionBound, "Ready", "")

			parent.Status.GenericComponentStatus = current.Status.GenericComponentStatus
			parent.Status.ServiceBindingId = current.Status.ServiceBindingId
			parent.Status.ExpiresAfter = current.Status.ExpiresAfter

			return nil
		},
	}
}

func RenewRequeue() reconcilers.SubReconciler[*servicesv1alpha1.ServiceClient] {
	return &reconcilers.SyncReconciler[*servicesv1alpha1.ServiceClient]{
		SyncWithResult: func(ctx context.Context, resource *servicesv1alpha1.ServiceClient) (reconcilers.Result, error) {
			now := rtime.RetrieveNow(ctx)
			renewsAt := resource.Status.RenewsAfter

			if after := renewsAt.Sub(now); after > 0 {
				return reconcile.Result{RequeueAfter: after}, nil
			}

			// should have already renewed
			return reconcile.Result{Requeue: true}, nil
		},
	}
}

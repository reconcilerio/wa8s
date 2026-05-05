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
	"regexp"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	diecorev1 "reconciler.io/dies/apis/core/v1"
	"reconciler.io/runtime/reconcilers"

	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
	"reconciler.io/wa8s/controllers"
	knativev1alpha1 "reconciler.io/wa8s/integrations/knative/apis/knative/v1alpha1"
	servingv1 "reconciler.io/wa8s/integrations/knative/apis/serving/v1"
)

//+kubebuilder:rbac:groups=knative.wa8s.reconciler.io,resources=servicetriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=knative.wa8s.reconciler.io,resources=servicetriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=knative.wa8s.reconciler.io,resources=servicetriggers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ServiceTriggerReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*knativev1alpha1.ServiceTrigger] {
	childLabelKey := fmt.Sprintf("%s/service-trigger", knativev1alpha1.GroupVersion.Group)
	imageRef := registriesv1alpha1.ImageReference{
		Kind: "ClusterImage",
		Name: "wasmtime",
	}

	return &reconcilers.ResourceReconciler[*knativev1alpha1.ServiceTrigger]{
		Reconciler: &reconcilers.SuppressTransientErrors[*knativev1alpha1.ServiceTrigger, *knativev1alpha1.ServiceTriggerList]{
			Reconciler: reconcilers.Sequence[*knativev1alpha1.ServiceTrigger]{
				controllers.ComponentContainerImageChildReconciler[*knativev1alpha1.ServiceTrigger](knativev1alpha1.ServiceTriggerConditionComponentContainerImageReady, childLabelKey, imageRef),
				KnativeServiceChildReconciler(childLabelKey),
			},
		},

		Config: c,
	}
}

//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete

func KnativeServiceChildReconciler(childLabelKey string) reconcilers.SubReconciler[*knativev1alpha1.ServiceTrigger] {
	return &reconcilers.ChildReconciler[*knativev1alpha1.ServiceTrigger, *servingv1.Service, *servingv1.ServiceList]{
		DesiredChild: func(ctx context.Context, resource *knativev1alpha1.ServiceTrigger) (*servingv1.Service, error) {
			if resource.Status.Image == "" {
				return nil, nil
			}

			service := servingv1.ServiceBlank.DieFeed(servingv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   resource.Namespace,
					Name:        resource.Name,
					Annotations: filterMap(resource.Annotations, "([^/]*\\.)?knative\\.dev(/.*)?"),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							childLabelKey: resource.Name,
						},
					),
				},
				Spec: servingv1.ServiceSpec{
					ConfigurationSpec: resource.Spec.ConfigurationSpec,
					RouteSpec:         resource.Spec.RouteSpec,
				},
			}).
				SpecDie(func(d *servingv1.ServiceSpecDie) {
					d.ConfigurationSpecDie(func(d *servingv1.ConfigurationSpecDie) {
						d.TemplateDie(func(d *servingv1.RevisionTemplateSpecDie) {
							d.SpecDie(func(d *servingv1.RevisionSpecDie) {
								d.PodSpecDie(func(d *diecorev1.PodSpecDie) {
									d.ContainerDie("", func(d *diecorev1.ContainerDie) {
										port := int32(8080)

										d.Name("user-container")
										d.Image(resource.Status.Image)
										d.Command("wasmtime")
										args := []string{"serve"}
										args = append(args, resource.Spec.HostCapabilities.WasmtimeArgs()...)
										args = append(args,
											// TODO configure cli capability
											"-Scli",
											"-Opooling-allocator=n",
											fmt.Sprintf("--addr=0.0.0.0:%d", port),
											"/component.wasm",
										)
										d.Args(args...)
										d.Ports(corev1.ContainerPort{ContainerPort: port})
									})
								})
							})
						})
					})
				})

			return service.DieReleasePtr(), nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*servingv1.Service]{
			HarmonizeImmutableFields: func(current, desired *servingv1.Service) {
				desired.Annotations["serving.knative.dev/creator"] = current.Annotations["serving.knative.dev/creator"]
				desired.Annotations["serving.knative.dev/lastModifier"] = current.Annotations["serving.knative.dev/lastModifier"]
			},
			MergeBeforeUpdate: func(current, desired *servingv1.Service) {
				current.Annotations = desired.Annotations
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},

		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *knativev1alpha1.ServiceTrigger, child *servingv1.Service, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, "Invalid", "%s", err)
					return errors.Join(err, ErrDurable)
				} else if apierrs.IsAlreadyExists(err) {
					parent.GetConditionManager(ctx).MarkFalse(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, "AlreadyExists", "another Service already exists with name %s", parent.GetName())
					return errors.Join(err, ErrDurable)
				}

				parent.GetConditionManager(ctx).MarkUnknown(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, "Unknown", "")
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, "Missing", "")
				return ErrDurable
			}

			// avoid premature reconciliation, check generation and ready condition
			if child.Generation != child.Status.ObservedGeneration {
				parent.GetConditionManager(ctx).MarkUnknown(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, "Blocked", "waiting for Deployment %s to reconcile", child.Name)
				return ErrGenerationMismatch
			}

			ready := child.Status.GetCondition(servingv1.ServiceConditionReady)
			if ready == nil {
				ready = &metav1.Condition{Reason: "Unknown"}
			}
			// Knative conditions allow reason to be empty, while metav1.Condition requires a value
			reason := ready.Reason
			switch ready.Status {
			case metav1.ConditionTrue:
				if reason == "" {
					reason = "Ready"
				}
				parent.GetConditionManager(ctx).MarkTrue(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, reason, "%s", ready.Message)
			case metav1.ConditionFalse:
				if reason == "" {
					reason = "NotReady"
				}
				parent.GetConditionManager(ctx).MarkFalse(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, reason, "%s", ready.Message)
			default:
				if reason == "" {
					reason = "Unknown"
				}
				parent.GetConditionManager(ctx).MarkUnknown(knativev1alpha1.ServiceTriggerConditionComponentServiceReady, reason, "%s", ready.Message)
			}

			parent.Status.Annotations = child.Status.Annotations
			parent.Status.ConfigurationStatusFields = child.Status.ConfigurationStatusFields
			parent.Status.RouteStatusFields = child.Status.RouteStatusFields

			return nil
		},
	}
}

func filterMap(data map[string]string, patterns ...string) map[string]string {
	compiledPatterns := []*regexp.Regexp{}
	for _, pattern := range patterns {
		compiledPatterns = append(compiledPatterns, regexp.MustCompile(pattern))
	}

	filtered := map[string]string{}
	for k, v := range data {
		for _, pattern := range compiledPatterns {
			if pattern.MatchString(k) {
				filtered[k] = v
				continue
			}
		}
	}
	return filtered
}

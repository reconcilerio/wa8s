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
	"slices"
	"strings"
	"time"

	"github.com/stoewer/go-strcase"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dieapiextensionsv1 "reconciler.io/dies/apis/apiextensions/v1"
	diemetav1 "reconciler.io/dies/apis/meta/v1"
	duckv1 "reconciler.io/ducks/api/v1"
	duckreconcilers "reconciler.io/ducks/reconcilers"
	"reconciler.io/runtime/apis"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	servicesv1alpha1 "reconciler.io/wa8s/apis/services/v1alpha1"
	"reconciler.io/wa8s/internal/defaults"
)

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceresourcedefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceresourcedefinitions/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func ServiceResourceDefinitionReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceResourceDefinition] {
	return &reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceResourceDefinition]{
		Reconciler: &reconcilers.SuppressTransientErrors[*servicesv1alpha1.ServiceResourceDefinition, *servicesv1alpha1.ServiceResourceDefinitionList]{
			Reconciler: &reconcilers.WithFinalizer[*servicesv1alpha1.ServiceResourceDefinition]{
				Finalizer: fmt.Sprintf("%s/reconciler", servicesv1alpha1.GroupVersion.Group),
				Reconciler: reconcilers.Sequence[*servicesv1alpha1.ServiceResourceDefinition]{
					ServiceResourceDefinitionLifecyceReconciler(),
					ServiceResourceDefinitionInstanceResourceDefintionReconciler(),
					ServiceResourceDefinitionClientResourceDefintionReconciler(),
					ServiceResourceDefinitionClientComponentDuckReconciler(),
					ServiceResourceDefinitionDuckSubReconciler(),
				},
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles,verbs=get;list;watch;create;update;patch;delete

func ServiceResourceDefinitionLifecyceReconciler() reconcilers.SubReconciler[*servicesv1alpha1.ServiceResourceDefinition] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceResourceDefinition, *servicesv1alpha1.ClusterServiceLifecycle, *servicesv1alpha1.ClusterServiceLifecycleList]{
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) (*servicesv1alpha1.ClusterServiceLifecycle, error) {
			return &servicesv1alpha1.ClusterServiceLifecycle{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    defaults.Namespace(),
					GenerateName: fmt.Sprintf("%s-", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							"services.wa8s.reconciler.io/service-resource-definition": resource.Name,
						},
					),
				},
				Spec: resource.Spec.Lifecycle,
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*servicesv1alpha1.ClusterServiceLifecycle]{
			MergeBeforeUpdate: func(current, desired *servicesv1alpha1.ClusterServiceLifecycle) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceResourceDefinition, child *servicesv1alpha1.ClusterServiceLifecycle, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionLifecycleReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionLifecycleReady, "Unknown", "")
					// retry reconcile request
					return reconcilers.ErrQuiet
				}
				return nil
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionLifecycleReady, "Missing", "")
				return nil
			}

			ready := child.GetConditionManager(ctx).GetCondition(duckv1.DuckConditionReady)
			if !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionLifecycleReady, "NotReady", "child ServiceLifecycle %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionLifecycleReady, "NotReady", "child ServiceLifecycle %s is not ready", child.Name)
				}
				return nil
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceResourceDefinitionConditionLifecycleReady, "Ready", "")

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete

func ServiceResourceDefinitionInstanceResourceDefintionReconciler() reconcilers.SubReconciler[*servicesv1alpha1.ServiceResourceDefinition] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceResourceDefinition, *apiextensionsv1.CustomResourceDefinition, *apiextensionsv1.CustomResourceDefinitionList]{
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			child := servicesv1alpha1.ServiceInstanceDuckCRD.
				MetadataDie(func(d *diemetav1.ObjectMetaDie) {
					d.Name(fmt.Sprintf("%s.%s", resource.Spec.InstanceNames.Plural, resource.Spec.Group))
					d.AddLabel("services.wa8s.reconciler.io/service-resource-definition", resource.Name)
					d.AddLabel("services.wa8s.reconciler.io/service-resource-definition-type", "instance")
				}).
				SpecDie(func(d *dieapiextensionsv1.CustomResourceDefinitionSpecDie) {
					d.Group(resource.Spec.Group)
					d.Names(apiextensionsv1.CustomResourceDefinitionNames{
						Plural:     resource.Spec.InstanceNames.Plural,
						Singular:   strings.ToLower(resource.Spec.InstanceNames.Kind),
						Kind:       resource.Spec.InstanceNames.Kind,
						ListKind:   fmt.Sprintf("%sList", resource.Spec.InstanceNames.Kind),
						Categories: resource.Spec.InstanceNames.Categories,
						ShortNames: resource.Spec.InstanceNames.ShortNames,
					})
					d.Conversion(&apiextensionsv1.CustomResourceConversion{
						Strategy: apiextensionsv1.NoneConverter,
					})
				}).
				DieReleasePtr()

			return child, nil
		},
		OurChild: func(resource *servicesv1alpha1.ServiceResourceDefinition, child *apiextensionsv1.CustomResourceDefinition) bool {
			if child.Labels == nil {
				return false
			}
			return child.Labels["services.wa8s.reconciler.io/service-resource-definition"] == resource.Name && child.Labels["services.wa8s.reconciler.io/service-resource-definition-type"] == "instance"
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*apiextensionsv1.CustomResourceDefinition]{
			MergeBeforeUpdate: func(current, desired *apiextensionsv1.CustomResourceDefinition) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceResourceDefinition, child *apiextensionsv1.CustomResourceDefinition, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					details := err.(apierrs.APIStatus).Status().Details
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "AlreadyExists", "another %s already exists with name %s", details.Kind, details.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "Unknown", "")
					// retry reconcile request
					return reconcilers.ErrQuiet
				}
				return nil
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "Missing", "")
				return nil
			}

			idx := slices.IndexFunc(child.Status.Conditions, func(c apiextensionsv1.CustomResourceDefinitionCondition) bool {
				return c.Type == apiextensionsv1.Established
			})
			established := apiextensionsv1.CustomResourceDefinitionCondition{Reason: "Initializing"}
			if idx >= 0 {
				established = child.Status.Conditions[idx]
			}
			if established.Status != apiextensionsv1.ConditionTrue {
				if established.Status == apiextensionsv1.ConditionFalse {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "NotEstablished", "child CustomResourceDefinition %s is not established", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "NotEstablished", "child CustomResourceDefinition %s is not established", child.Name)
				}
				return nil
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady, "Established", "")

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete

func ServiceResourceDefinitionClientResourceDefintionReconciler() reconcilers.SubReconciler[*servicesv1alpha1.ServiceResourceDefinition] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceResourceDefinition, *apiextensionsv1.CustomResourceDefinition, *apiextensionsv1.CustomResourceDefinitionList]{
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) (*apiextensionsv1.CustomResourceDefinition, error) {
			child := servicesv1alpha1.ServiceClientDuckCRD.
				MetadataDie(func(d *diemetav1.ObjectMetaDie) {
					d.Name(fmt.Sprintf("%s.%s", resource.Spec.ClientNames.Plural, resource.Spec.Group))
					d.AddLabel("services.wa8s.reconciler.io/service-resource-definition", resource.Name)
					d.AddLabel("services.wa8s.reconciler.io/service-resource-definition-type", "client")
				}).
				SpecDie(func(d *dieapiextensionsv1.CustomResourceDefinitionSpecDie) {
					d.Group(resource.Spec.Group)
					d.Names(apiextensionsv1.CustomResourceDefinitionNames{
						Plural:     resource.Spec.ClientNames.Plural,
						Singular:   strings.ToLower(resource.Spec.ClientNames.Kind),
						Kind:       resource.Spec.ClientNames.Kind,
						ListKind:   fmt.Sprintf("%sList", resource.Spec.ClientNames.Kind),
						Categories: resource.Spec.ClientNames.Categories,
						ShortNames: resource.Spec.ClientNames.ShortNames,
					})
					d.Conversion(&apiextensionsv1.CustomResourceConversion{
						Strategy: apiextensionsv1.NoneConverter,
					})
				}).
				DieReleasePtr()

			return child, nil
		},
		OurChild: func(resource *servicesv1alpha1.ServiceResourceDefinition, child *apiextensionsv1.CustomResourceDefinition) bool {
			if child.Labels == nil {
				return false
			}
			return child.Labels["services.wa8s.reconciler.io/service-resource-definition"] == resource.Name && child.Labels["services.wa8s.reconciler.io/service-resource-definition-type"] == "client"
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*apiextensionsv1.CustomResourceDefinition]{
			MergeBeforeUpdate: func(current, desired *apiextensionsv1.CustomResourceDefinition) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceResourceDefinition, child *apiextensionsv1.CustomResourceDefinition, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					details := err.(apierrs.APIStatus).Status().Details
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "AlreadyExists", "another %s already exists with name %s", details.Kind, details.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "Unknown", "")
					// retry reconcile request
					return reconcilers.ErrQuiet
				}
				return nil
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "Missing", "")
				return nil
			}

			idx := slices.IndexFunc(child.Status.Conditions, func(c apiextensionsv1.CustomResourceDefinitionCondition) bool {
				return c.Type == apiextensionsv1.Established
			})
			established := apiextensionsv1.CustomResourceDefinitionCondition{Reason: "Initializing"}
			if idx >= 0 {
				established = child.Status.Conditions[idx]
			}
			if established.Status != apiextensionsv1.ConditionTrue {
				if established.Status == apiextensionsv1.ConditionFalse {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "NotEstablished", "child CustomResourceDefinition %s is not established", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "NotEstablished", "child CustomResourceDefinition %s is not established", child.Name)
				}
				return nil
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady, "Established", "")

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=componentducks,verbs=get;list;watch;create;update;patch;delete

func ServiceResourceDefinitionClientComponentDuckReconciler() reconcilers.SubReconciler[*servicesv1alpha1.ServiceResourceDefinition] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceResourceDefinition, *duckv1.Duck, *duckv1.DuckList]{
		ChildType: &duckv1.Duck{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "wa8s.reconciler.io/v1",
				Kind:       "ComponentDuck",
			},
		},
		ChildListType: &duckv1.DuckList{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "wa8s.reconciler.io/v1",
				Kind:       "ComponentDuckList",
			},
		},
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) (*duckv1.Duck, error) {
			return &duckv1.Duck{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "wa8s.reconciler.io/v1",
					Kind:       "ComponentDuck",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s.%s", resource.Spec.ClientNames.Plural, resource.Spec.Group),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							"services.wa8s.reconciler.io/service-resource-definition": resource.Name,
						},
					),
				},
				Spec: duckv1.DuckSpec{
					Group:   resource.Spec.Group,
					Version: "v1alpha1",
					Kind:    resource.Spec.ClientNames.Kind,
				},
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*duckv1.Duck]{
			Type: &duckv1.Duck{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "wa8s.reconciler.io/v1",
					Kind:       "ComponentDuck",
				},
			},
			DangerouslyAllowDuckTypes: true,
			MergeBeforeUpdate: func(current, desired *duckv1.Duck) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceResourceDefinition, child *duckv1.Duck, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					details := err.(apierrs.APIStatus).Status().Details
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "AlreadyExists", "another %s already exists with name %s", details.Kind, details.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "Unknown", "")
					// retry reconcile request
					return reconcilers.ErrQuiet
				}
				return nil
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "Missing", "")
				return nil
			}

			ready := child.GetConditionManager(ctx).GetCondition(duckv1.DuckConditionReady)
			if !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "NotReady", "child ComponentDuck %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "NotReady", "child ComponentDuck %s is not ready", child.Name)
				}
				return nil
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceResourceDefinitionConditionClientComponentDuckReady, "Ready", "")

			return nil
		},
	}
}

func ServiceResourceDefinitionDuckSubReconciler() reconcilers.SubReconciler[*servicesv1alpha1.ServiceResourceDefinition] {
	syncPeriod := 10 * time.Hour
	return reconcilers.Sequence[*servicesv1alpha1.ServiceResourceDefinition]{
		&reconcilers.SyncReconciler[*servicesv1alpha1.ServiceResourceDefinition]{
			Sync: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) error {
				// require CRDs to be established before continuing
				if !apis.ConditionIsTrue(resource.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceResourceDefinitionConditionInstanceDefinitionReady)) {
					return ErrUpdateStatusBeforeContinuingReconcile
				}
				if !apis.ConditionIsTrue(resource.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceResourceDefinitionConditionClientDefinitionReady)) {
					return ErrUpdateStatusBeforeContinuingReconcile
				}

				return nil
			},
		},
		&duckreconcilers.SubManagerReconciler[*servicesv1alpha1.ServiceResourceDefinition]{
			SyncPeriod:      &syncPeriod,
			AssertFinalizer: fmt.Sprintf("%s/reconciler", servicesv1alpha1.GroupVersion.Group),
			LocalTypes: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) ([]schema.GroupKind, error) {
				return []schema.GroupKind{
					{Group: resource.Spec.Group, Kind: resource.Spec.InstanceNames.Kind},
					{Group: resource.Spec.Group, Kind: fmt.Sprintf("%sList", resource.Spec.InstanceNames.Kind)},
					{Group: resource.Spec.Group, Kind: resource.Spec.ClientNames.Kind},
					{Group: resource.Spec.Group, Kind: fmt.Sprintf("%sList", resource.Spec.ClientNames.Kind)},
				}, nil
			},
			SetupWithSubManager: func(ctx context.Context, mgr ctrl.Manager, resource *servicesv1alpha1.ServiceResourceDefinition) error {
				config := reconcilers.NewConfig(mgr, nil, syncPeriod)

				instanceTypeMeta := metav1.TypeMeta{
					APIVersion: schema.GroupVersion{Group: resource.Spec.Group, Version: "v1alpha1"}.String(),
					Kind:       resource.Spec.InstanceNames.Kind,
				}
				if err := ServiceInstanceDuckReconciler(config.WithTracker(), instanceTypeMeta, resource.Name).SetupWithManager(ctx, mgr); err != nil {
					return err
				}

				clientTypeMeta := metav1.TypeMeta{
					APIVersion: schema.GroupVersion{Group: resource.Spec.Group, Version: "v1alpha1"}.String(),
					Kind:       resource.Spec.ClientNames.Kind,
				}
				if err := ServiceClientDuckReconciler(config.WithTracker(), clientTypeMeta, resource.Name).SetupWithManager(ctx, mgr); err != nil {
					return err
				}

				return nil
			},
		},
		&reconcilers.SyncReconciler[*servicesv1alpha1.ServiceResourceDefinition]{
			Sync: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) error {
				resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceResourceDefinitionConditionReconcilerRunning, "Running", "")

				return nil
			},
			Finalize: func(ctx context.Context, resource *servicesv1alpha1.ServiceResourceDefinition) error {
				resource.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceResourceDefinitionConditionReconcilerRunning, "Shutdown", "")

				return nil
			},
		},
	}
}

func ServiceInstanceDuckReconciler(c reconcilers.Config, typeMeta metav1.TypeMeta, serviceResourceDefinitionName string) *reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceInstanceDuck] {
	return &reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceInstanceDuck]{
		Type: &servicesv1alpha1.ServiceInstanceDuck{
			TypeMeta: typeMeta,
		},
		Setup: func(ctx context.Context, mgr ctrl.Manager, bldr *builder.Builder) error {
			bldr.Named(strings.ToLower(schema.FromAPIVersionAndKind(typeMeta.APIVersion, typeMeta.Kind).GroupKind().String()))

			return nil
		},

		Reconciler: reconcilers.Sequence[*servicesv1alpha1.ServiceInstanceDuck]{
			ServiceInstanceDuckChildServiceInstanceReconciler(serviceResourceDefinitionName),
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles,verbs=get;list;watch

func ServiceInstanceDuckChildServiceInstanceReconciler(serviceResourceDefinitionName string) reconcilers.SubReconciler[*servicesv1alpha1.ServiceInstanceDuck] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceInstanceDuck, *servicesv1alpha1.ServiceInstance, *servicesv1alpha1.ServiceInstanceList]{
		Setup: func(ctx context.Context, mgr ctrl.Manager, bldr *builder.Builder) error {
			bldr.Watches(&servicesv1alpha1.ClusterServiceLifecycle{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceInstanceDuck) (*servicesv1alpha1.ServiceInstance, error) {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			// remap ref from custom instance type to ServiceInstance
			lifecycles := &servicesv1alpha1.ClusterServiceLifecycleList{}
			if err := c.TrackAndList(ctx, lifecycles, client.MatchingLabels{
				"services.wa8s.reconciler.io/service-resource-definition": serviceResourceDefinitionName,
			}); err != nil {
				return nil, err
			}
			if len(lifecycles.Items) != 1 {
				return nil, fmt.Errorf("failed to find ClusterServiceLifecycle")
			}
			lifecycleName := lifecycles.Items[0].Name

			gvk := schema.FromAPIVersionAndKind(resource.APIVersion, resource.Kind)

			return &servicesv1alpha1.ServiceInstance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    resource.Namespace,
					GenerateName: fmt.Sprintf("%s-", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							"services.wa8s.reconciler.io/service-resource-definition":    serviceResourceDefinitionName,
							"services.wa8s.reconciler.io/service-instance-duck":          resource.Name,
							fmt.Sprintf("%s/%s", gvk.Group, strcase.KebabCase(gvk.Kind)): resource.Name,
						},
					),
				},
				Spec: servicesv1alpha1.ServiceInstanceSpec{
					Ref: servicesv1alpha1.ServiceLifecycleReference{
						Kind: "ClusterServiceLifecycle",
						Name: lifecycleName,
					},
					Type:     resource.Spec.Type,
					Tier:     resource.Spec.Tier,
					Requests: resource.Spec.Requests,
				},
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*servicesv1alpha1.ServiceInstance]{
			MergeBeforeUpdate: func(current, desired *servicesv1alpha1.ServiceInstance) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceInstanceDuck, child *servicesv1alpha1.ServiceInstance, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					details := err.(apierrs.APIStatus).Status().Details
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "AlreadyExists", "another %s already exists with name %s", details.Kind, details.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "Unknown", "")
					// retry reconcile request
					return reconcilers.ErrQuiet
				}
				return nil
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "Missing", "")
				return nil
			}

			ready := child.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceInstanceConditionReady)
			if !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "NotReady", "child ServiceInstance %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "NotReady", "child ServiceInstance %s is not ready", child.Name)
				}
				return nil
			}

			parent.Status.ServiceInstanceId = child.Status.ServiceInstanceId

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceInstanceDuckConditionServiceInstanceReady, "Ready", "")

			return nil
		},
	}
}

func ServiceClientDuckReconciler(c reconcilers.Config, typeMeta metav1.TypeMeta, serviceResourceDefinitionName string) *reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceClientDuck] {
	gvk := schema.FromAPIVersionAndKind(typeMeta.APIVersion, typeMeta.Kind)
	childLabelKey := fmt.Sprintf("%s/%s", gvk.Group, strcase.KebabCase(gvk.Kind))

	return &reconcilers.ResourceReconciler[*servicesv1alpha1.ServiceClientDuck]{
		Type: &servicesv1alpha1.ServiceClientDuck{
			TypeMeta: typeMeta,
		},
		Setup: func(ctx context.Context, mgr ctrl.Manager, bldr *builder.Builder) error {
			bldr.Named(strings.ToLower(schema.FromAPIVersionAndKind(typeMeta.APIVersion, typeMeta.Kind).GroupKind().String()))

			return nil
		},

		Reconciler: reconcilers.Sequence[*servicesv1alpha1.ServiceClientDuck]{
			ServiceClientDuckChildServiceClientReconciler(serviceResourceDefinitionName),
			ComponentChildReconciler[*servicesv1alpha1.ServiceClientDuck](servicesv1alpha1.ServiceClientDuckConditionChildComponent, childLabelKey, nil),
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.wa8s.reconciler.io,resources=serviceinstances,verbs=get;list;watch

func ServiceClientDuckChildServiceClientReconciler(serviceResourceDefinitionName string) reconcilers.SubReconciler[*servicesv1alpha1.ServiceClientDuck] {
	return &reconcilers.ChildReconciler[*servicesv1alpha1.ServiceClientDuck, *servicesv1alpha1.ServiceClient, *servicesv1alpha1.ServiceClientList]{
		Setup: func(ctx context.Context, mgr ctrl.Manager, bldr *builder.Builder) error {
			bldr.Watches(&servicesv1alpha1.ServiceInstance{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		DesiredChild: func(ctx context.Context, resource *servicesv1alpha1.ServiceClientDuck) (*servicesv1alpha1.ServiceClient, error) {
			c := reconcilers.RetrieveConfigOrDie(ctx)

			// remap ref from custom instance type to ServiceInstance
			instances := &servicesv1alpha1.ServiceInstanceList{}
			if err := c.TrackAndList(ctx, instances, client.InNamespace(resource.Namespace), client.MatchingLabels{
				"services.wa8s.reconciler.io/service-resource-definition": serviceResourceDefinitionName,
				"services.wa8s.reconciler.io/service-instance-duck":       resource.Spec.Ref.Name,
			}); err != nil {
				return nil, err
			}
			if len(instances.Items) != 1 {
				return nil, fmt.Errorf("failed to find ServiceInstance")
			}
			resource = resource.DeepCopy()
			resource.Spec.Ref.Name = instances.Items[0].Name

			gvk := schema.FromAPIVersionAndKind(resource.APIVersion, resource.Kind)

			return &servicesv1alpha1.ServiceClient{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    resource.Namespace,
					GenerateName: fmt.Sprintf("%s-", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							"services.wa8s.reconciler.io/service-client-duck":            resource.Name,
							fmt.Sprintf("%s/%s", gvk.Group, strcase.KebabCase(gvk.Kind)): resource.Name,
						},
					),
				},
				Spec: resource.Spec,
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*servicesv1alpha1.ServiceClient]{
			MergeBeforeUpdate: func(current, desired *servicesv1alpha1.ServiceClient) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *servicesv1alpha1.ServiceClientDuck, child *servicesv1alpha1.ServiceClient, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					details := err.(apierrs.APIStatus).Status().Details
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "AlreadyExists", "another %s already exists with name %s", details.Kind, details.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "Unknown", "")
					// retry reconcile request
					return reconcilers.ErrQuiet
				}
				return nil
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "Missing", "")
				return nil
			}

			ready := child.GetConditionManager(ctx).GetCondition(servicesv1alpha1.ServiceClientConditionReady)
			if !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "NotReady", "child ServiceClient %s is not ready", child.Name)
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "NotReady", "child ServiceClient %s is not ready", child.Name)
				}
				return nil
			}

			parent.Status.GenericComponentStatus = child.Status.GenericComponentStatus
			parent.Status.ServiceBindingId = child.Status.ServiceBindingId
			parent.Status.Binding = child.Status.Binding
			parent.Status.RenewsAfter = child.Status.RenewsAfter
			parent.Status.ExpiresAfter = child.Status.ExpiresAfter

			parent.Status.Trace = []componentsv1alpha1.ComponentSpan{SynthesizeSpan(ctx, child)}
			if hasCycle, sanitizedTrace := DetectTraceCycle(parent.Status.Trace, parent); hasCycle {
				parent.GetConditionManager(ctx).MarkFalse(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "CycleDetected", "components may not reference themselves directly or transitively")
				parent.Status.Trace = sanitizedTrace
				return ErrDurable
			}

			parent.GetConditionManager(ctx).MarkTrue(servicesv1alpha1.ServiceClientDuckConditionServiceClientReady, "Ready", "")

			return nil
		},
	}
}

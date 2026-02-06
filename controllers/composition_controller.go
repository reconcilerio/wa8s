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
	"reconciler.io/wa8s/components"
	"reconciler.io/wa8s/registry"
)

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=compositions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=compositions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=compositions/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func CompositionReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*componentsv1alpha1.Composition] {
	childLabelKey := fmt.Sprintf("%s/composition", componentsv1alpha1.GroupVersion.Group)

	ourChild := func(resource componentsv1alpha1.ComponentLike, child *componentsv1alpha1.Component) bool {
		// check that this child is for a Composition as a whole, and not a Composition dependency
		if child.Annotations == nil {
			return true
		}
		_, ok := child.Annotations[fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group)]
		return !ok
	}

	return &reconcilers.ResourceReconciler[*componentsv1alpha1.Composition]{
		Reconciler: &reconcilers.SuppressTransientErrors[*componentsv1alpha1.Composition, *componentsv1alpha1.CompositionList]{
			Reconciler: reconcilers.Sequence[*componentsv1alpha1.Composition]{
				reconcilers.Always[*componentsv1alpha1.Composition]{
					ManageDependencies(childLabelKey),
					ResolveDependencies(),
					ReflectDependenciesStatus(),
					ResolveRepository[*componentsv1alpha1.Composition](componentsv1alpha1.CompositionConditionRepositoryReady),
					ComponentChildReconciler[*componentsv1alpha1.Composition](componentsv1alpha1.CompositionConditionChildComponent, childLabelKey, ourChild),
				},
				ComposeComponents(),
				PushComponent[*componentsv1alpha1.Composition](componentsv1alpha1.CompositionConditionPushed),
				ReflectComponentableStatus[*componentsv1alpha1.Composition](),
			},
		},

		Config: c,
	}
}

func ManageDependencies(childLabelKey string) reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.Always[*componentsv1alpha1.Composition]{
		ManageConfigStoreDependencies(childLabelKey),
		ManageOciDependencies(childLabelKey),
		ManageCompositionDependencies(childLabelKey),
		// expand to include other types
	}
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=configstores,verbs=get;list;watch;create;update;patch;delete

func ManageConfigStoreDependencies(childLabelKey string) reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.ChildSetReconciler[*componentsv1alpha1.Composition, *componentsv1alpha1.ConfigStore, *componentsv1alpha1.ConfigStoreList]{
		DesiredChildren: func(ctx context.Context, resource *componentsv1alpha1.Composition) ([]*componentsv1alpha1.ConfigStore, error) {
			children := []*componentsv1alpha1.ConfigStore{}

			for _, dependency := range resource.Spec.Dependencies {
				if dependency.Config == nil {
					continue
				}

				children = append(children, &componentsv1alpha1.ConfigStore{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:    resource.Namespace,
						GenerateName: fmt.Sprintf("%s-", resource.Name),
						Labels: reconcilers.MergeMaps(
							resource.Labels,
							map[string]string{
								childLabelKey: resource.GetName(),
							},
						),
						Annotations: map[string]string{
							fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group): dependency.Component,
						},
					},
					Spec: componentsv1alpha1.ConfigStoreSpec{
						GenericConfigStoreSpec: *dependency.Config,
					},
				})
			}

			return children, nil
		},
		IdentifyChild: func(child *componentsv1alpha1.ConfigStore) string {
			return child.Annotations[fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group)]
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*componentsv1alpha1.ConfigStore]{
			MergeBeforeUpdate: func(current, desired *componentsv1alpha1.ConfigStore) {
				current.Annotations = desired.Annotations
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildrenStatusOnParent: func(ctx context.Context, parent *componentsv1alpha1.Composition, results reconcilers.ChildSetResult[*componentsv1alpha1.ConfigStore]) {
			for _, result := range results.Children {
				for i := range parent.Spec.Dependencies {
					if result.Id != parent.Spec.Dependencies[i].Component {
						continue
					}
					if result.Child == nil {
						// TODO capture fault
						continue
					}
					// TODO move into a stashed value so we're not rewriting the spec
					parent.Spec.Dependencies[i].Ref = &componentsv1alpha1.ComponentReference{
						APIVersion: componentsv1alpha1.GroupVersion.String(),
						Kind:       "ConfigStore",
						Namespace:  result.Child.Namespace,
						Name:       result.Child.Name,
					}
				}
			}
		},
	}
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=components,verbs=get;list;watch;create;update;patch;delete

func ManageOciDependencies(childLabelKey string) reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.ChildSetReconciler[*componentsv1alpha1.Composition, *componentsv1alpha1.Component, *componentsv1alpha1.ComponentList]{
		DesiredChildren: func(ctx context.Context, resource *componentsv1alpha1.Composition) ([]*componentsv1alpha1.Component, error) {
			children := []*componentsv1alpha1.Component{}

			for _, dependency := range resource.Spec.Dependencies {
				if dependency.OCI == nil {
					continue
				}

				children = append(children, &componentsv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:    resource.Namespace,
						GenerateName: fmt.Sprintf("%s-", resource.Name),
						Labels: reconcilers.MergeMaps(
							resource.Labels,
							map[string]string{
								childLabelKey: resource.GetName(),
							},
						),
						Annotations: map[string]string{
							fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group): dependency.Component,
						},
					},
					Spec: componentsv1alpha1.ComponentSpec{
						OCI: dependency.OCI,
					},
				})
			}

			return children, nil
		},
		OurChild: func(resource *componentsv1alpha1.Composition, child *componentsv1alpha1.Component) bool {
			if child.Annotations == nil {
				return false
			}
			_, ok := child.Annotations[fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group)]
			return ok
		},
		IdentifyChild: func(child *componentsv1alpha1.Component) string {
			return child.Annotations[fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group)]
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*componentsv1alpha1.Component]{
			MergeBeforeUpdate: func(current, desired *componentsv1alpha1.Component) {
				current.Annotations = desired.Annotations
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildrenStatusOnParent: func(ctx context.Context, parent *componentsv1alpha1.Composition, results reconcilers.ChildSetResult[*componentsv1alpha1.Component]) {
			for _, result := range results.Children {
				for i := range parent.Spec.Dependencies {
					if result.Id != parent.Spec.Dependencies[i].Component {
						continue
					}
					if result.Child == nil {
						// TODO capture fault
						continue
					}
					// TODO move into a stashed value so we're not rewriting the spec
					parent.Spec.Dependencies[i].Ref = &componentsv1alpha1.ComponentReference{
						APIVersion: componentsv1alpha1.GroupVersion.String(),
						Kind:       "Component",
						Namespace:  result.Child.Namespace,
						Name:       result.Child.Name,
					}
				}
			}
		},
	}
}

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=compositions,verbs=get;list;watch;create;update;patch;delete

func ManageCompositionDependencies(childLabelKey string) reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.ChildSetReconciler[*componentsv1alpha1.Composition, *componentsv1alpha1.Composition, *componentsv1alpha1.CompositionList]{
		DesiredChildren: func(ctx context.Context, resource *componentsv1alpha1.Composition) ([]*componentsv1alpha1.Composition, error) {
			children := []*componentsv1alpha1.Composition{}

			for _, dependency := range resource.Spec.Dependencies {
				if dependency.Composition == nil {
					continue
				}

				children = append(children, &componentsv1alpha1.Composition{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:    resource.Namespace,
						GenerateName: fmt.Sprintf("%s-", resource.Name),
						Labels: reconcilers.MergeMaps(
							resource.Labels,
							map[string]string{
								childLabelKey: resource.GetName(),
							},
						),
						Annotations: map[string]string{
							fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group): dependency.Component,
						},
					},
					Spec: componentsv1alpha1.CompositionSpec{
						GenericCompositionSpec: *dependency.Composition,
					},
				})
			}

			return children, nil
		},
		IdentifyChild: func(child *componentsv1alpha1.Composition) string {
			return child.Annotations[fmt.Sprintf("%s/composition-dependency", componentsv1alpha1.GroupVersion.Group)]
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*componentsv1alpha1.Composition]{
			MergeBeforeUpdate: func(current, desired *componentsv1alpha1.Composition) {
				current.Annotations = desired.Annotations
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildrenStatusOnParent: func(ctx context.Context, parent *componentsv1alpha1.Composition, results reconcilers.ChildSetResult[*componentsv1alpha1.Composition]) {
			for _, result := range results.Children {
				for i := range parent.Spec.Dependencies {
					if result.Id != parent.Spec.Dependencies[i].Component {
						continue
					}
					if result.Child == nil {
						// TODO capture fault
						continue
					}
					// TODO move into a stashed value so we're not rewriting the spec
					parent.Spec.Dependencies[i].Ref = &componentsv1alpha1.ComponentReference{
						APIVersion: componentsv1alpha1.GroupVersion.String(),
						Kind:       "Composition",
						Namespace:  result.Child.Namespace,
						Name:       result.Child.Name,
					}
				}
			}
		},
	}
}

func ResolveDependencies() reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.ForEach[*componentsv1alpha1.Composition, componentsv1alpha1.CompositionDependency]{
		Items: func(ctx context.Context, resource *componentsv1alpha1.Composition) ([]componentsv1alpha1.CompositionDependency, error) {
			return resource.Spec.Dependencies, nil
		},
		Reconciler: ResolveDependency(),
	}
}

func ResolveDependency() reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.SyncReconciler[*componentsv1alpha1.Composition]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.WatchesRawSource(ComponentDuckBroker.TrackedSource(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource *componentsv1alpha1.Composition) error {
			iteration := reconcilers.CursorStasher[componentsv1alpha1.CompositionDependency]().RetrieveOrDie(ctx)

			component, err := ResolveComponentReference(ctx, *iteration.Item.Ref)
			if err != nil {
				if errors.Is(err, ErrNotComponent) {
					resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.CompositionConditionDependenciesResolved, "NotComponent", "%s %s is not a component", iteration.Item.Ref.APIVersion, iteration.Item.Ref.Kind)
					return reconcilers.ErrHaltSubReconcilers
				}
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.CompositionConditionDependenciesResolved, "ComponentNotFound", "%s %s not found (%d of %d)", iteration.Item.Ref.Kind, iteration.Item.Ref.Name, iteration.Index+1, iteration.Length)
					return ErrDurable
				}
				return err
			}

			trace := append(ComponentTraceStasher.RetrieveOrEmpty(ctx), SynthesizeSpan(ctx, component))
			ComponentTraceStasher.Store(ctx, trace)
			if hasCycle, sanitizedTrace := DetectTraceCycle(trace, resource); hasCycle {
				resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.CompositionConditionDependenciesResolved, "CycleDetected", "components may not reference themselves directly or transitively (%d of %d)", iteration.Index+1, iteration.Length)
				resource.GetGenericComponentStatus().Trace = sanitizedTrace

				return ErrDurable
			}

			if err := component.Spec.Default(ctx); err != nil {
				return err
			}
			// avoid premature reconciliation, check generation and ready condition
			if component.Generation != component.Status.ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(componentsv1alpha1.CompositionConditionDependenciesResolved, "Blocked", "waiting for %s %s to reconcile (%d of %d)", iteration.Item.Ref.Kind, iteration.Item.Ref.Name, iteration.Index+1, iteration.Length)
				return ErrGenerationMismatch
			}
			if ready := component.Status.GetCondition(componentsv1alpha1.ComponentDuckConditionReady); !apis.ConditionIsTrue(ready) {
				if ready == nil {
					ready = &metav1.Condition{Reason: "Initializing"}
				}
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.CompositionConditionDependenciesResolved, "NotReady", "%s %s is not ready (%d of %d)", iteration.Item.Ref.Kind, iteration.Item.Ref.Name, iteration.Index+1, iteration.Length)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(componentsv1alpha1.CompositionConditionDependenciesResolved, "NotReady", "%s %s is not ready (%d of %d)", iteration.Item.Ref.Kind, iteration.Item.Ref.Name, iteration.Index+1, iteration.Length)
				}
				return ErrDurable
			}

			if component.Status.Image == "" {
				// should never be ready and missing an image, but ya know
				resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.CompositionConditionDependenciesResolved, "ImageMissing", "%s %s is missing image (%d of %d)", iteration.Item.Ref.Kind, iteration.Item.Ref.Name, iteration.Index+1, iteration.Length)
				return ErrDurable
			}

			RepositoryKeychainStasher.Clear(ctx)
			if _, err := ResolveRepository[*componentsv1alpha1.ComponentDuck](componentsv1alpha1.CompositionConditionDependenciesResolved).Reconcile(ctx, component); err != nil {
				return err
			}
			keychain, err := RepositoryKeychainStasher.RetrieveOrError(ctx)
			if err != nil {
				return err
			}

			ref, err := name.NewDigest(component.Status.Image, name.WeakValidation)
			if err != nil {
				return err
			}
			componentBytes, _, err := registry.Pull(ctx, ref, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				return err
			}

			dependencies := CompositionDependenciesStasher.RetrieveOrEmpty(ctx)
			if dependencies == nil {
				dependencies = []components.ResolvedComponent{}
			}
			dependencies = append(dependencies, components.ResolvedComponent{
				Name:      iteration.Item.Component,
				Image:     ref,
				Component: componentBytes,
				WIT:       *component.Status.WIT,
			})
			CompositionDependenciesStasher.Store(ctx, dependencies)
			resource.GetConditionManager(ctx).MarkTrue(componentsv1alpha1.CompositionConditionDependenciesResolved, "Resolved", "resolved %d component dependencies", iteration.Index+1)

			return nil
		},
	}
}

func ComposeComponents() *reconcilers.SyncReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.SyncReconciler[*componentsv1alpha1.Composition]{
		Sync: func(ctx context.Context, resource *componentsv1alpha1.Composition) error {
			dependencies := CompositionDependenciesStasher.RetrieveOrDie(ctx)

			if resource.Spec.Plug != nil {
				composed, err := components.WACPlug(ctx, dependencies)
				if err != nil {
					return err
				}
				ComponentStasher.Store(ctx, composed)
			} else if resource.Spec.WAC != "" {
				composed, err := components.WACCompose(ctx, resource.Spec.WAC, dependencies)
				if err != nil {
					return err
				}
				ComponentStasher.Store(ctx, composed)
			} else {
				resource.GetConditionManager(ctx).MarkFalse(componentsv1alpha1.CompositionConditionPushed, "Invalid", "one of .spec[plug, wac] is required")
			}

			return nil
		},
	}
}

func ReflectDependenciesStatus() reconcilers.SubReconciler[*componentsv1alpha1.Composition] {
	return &reconcilers.SyncReconciler[*componentsv1alpha1.Composition]{
		Sync: func(ctx context.Context, resource *componentsv1alpha1.Composition) error {
			resource.Status.Dependencies = []componentsv1alpha1.CompositionDependencyStatus{}
			dependencies := CompositionDependenciesStasher.RetrieveOrEmpty(ctx)
			if dependencies == nil {
				return nil
			}

			for _, d := range dependencies {
				resource.Status.Dependencies = append(resource.Status.Dependencies,
					componentsv1alpha1.CompositionDependencyStatus{
						Component: d.Name,
						Image:     d.Image.Name(),
						WIT:       d.WIT,
					},
				)
			}

			return nil
		},
	}
}

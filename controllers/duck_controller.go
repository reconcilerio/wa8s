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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckclient "reconciler.io/ducks/client"
	"reconciler.io/runtime/duck"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"reconciler.io/wa8s/apis"
	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
	"reconciler.io/wa8s/registry"
)

var ComponentDuckBroker duckclient.Broker
var ErrNotComponent = errors.New("referenced apiVersion kind is not a component")

//+kubebuilder:rbac:groups=duck.reconciler.io,resources=ducktypes,verbs=get;list;watch
//+kubebuilder:rbac:groups=wa8s.reconciler.io,resources=componentducks,verbs=get;list;watch

func ResolveComponentReference(ctx context.Context, ref componentsv1alpha1.ComponentReference) (*componentsv1alpha1.ComponentDuck, error) {
	componentClient := duckclient.New(
		"componentducks.wa8s.reconciler.io",
		reconcilers.RetrieveConfigOrDie(ctx),
	)

	component := &componentsv1alpha1.ComponentDuck{
		TypeMeta: ref.TypeMeta(),
	}
	if err := componentClient.TrackAndGet(ctx, ref.NamespacedName(), component); err != nil {
		if errors.Is(err, duckclient.ErrUnknownDuck) {
			return nil, ErrNotComponent
		}
		return nil, err
	}
	return component, nil
}

//+kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=images,verbs=get;list;watch
//+kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterimages,verbs=get;list;watch

func ResolveImage[IR registriesv1alpha1.ImageReferencer](conditionType string) reconcilers.SubReconciler[IR] {
	return &reconcilers.SyncReconciler[IR]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&registriesv1alpha1.Image{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&registriesv1alpha1.ClusterImage{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource IR) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)
			conditionManager := resource.GetConditionManager(ctx)

			imageRef := resource.GetImageReference()
			var image registriesv1alpha1.GenericImage
			if imageRef.Kind == "ClusterImage" {
				image = &registriesv1alpha1.ClusterImage{
					ObjectMeta: metav1.ObjectMeta{
						Name: imageRef.Name,
					},
				}
			} else {
				image = &registriesv1alpha1.Image{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: resource.GetNamespace(),
						Name:      imageRef.Name,
					},
				}
			}
			if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(image), image); err != nil {
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(conditionType, "ImageNotFound", "%s %s not found", imageRef.Kind, imageRef.Name)
					return ErrDurable
				}
				return err
			}

			trace := append(ComponentTraceStasher.RetrieveOrEmpty(ctx), SynthesizeSpan(ctx, image))
			ComponentTraceStasher.Store(ctx, trace)
			if hasCycle, sanitizedTrace := DetectTraceCycle(trace, resource); hasCycle {
				conditionManager.MarkFalse(conditionType, "CycleDetected", "components may not reference themselves directly or transitively")
				if err := ReflectTrace(resource, sanitizedTrace); err != nil {
					panic(err)
				}
				return ErrDurable
			}

			// avoid premature reconciliation, check generation and ready condition
			if image.GetGeneration() != image.GetStatus().ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(conditionType, "Blocked", "waiting for %s %s to reconcile", imageRef.Kind, imageRef.Name)
				return ErrGenerationMismatch
			}
			image.GetStatus().InitializeConditions(ctx)
			if ready := image.GetStatus().GetCondition(registriesv1alpha1.ImageConditionReady); !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(conditionType, "ImageNotReady", "%s: %s", ready.Reason, ready.Message)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(conditionType, "ImageNotReady", "%s: %s", ready.Reason, ready.Message)
				}
				return ErrDurable
			}

			if err := image.Default(ctx); err != nil {
				return err
			}

			// get keychain for image from repository
			repositoryRef := image.GetSpec().RepositoryRef
			var repository registriesv1alpha1.GenericRepository
			if repositoryRef.Kind == "ClusterRepository" {
				repository = &registriesv1alpha1.ClusterRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name: repositoryRef.Name,
					},
				}
			} else {
				repository = &registriesv1alpha1.Repository{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: resource.GetNamespace(),
						Name:      repositoryRef.Name,
					},
				}
			}
			if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(repository), repository); err != nil {
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(conditionType, "RepositoryNotFound", "%s %s not found", repositoryRef.Kind, repositoryRef.Name)
					return ErrDurable
				}
				return err
			}

			// avoid premature reconciliation, check generation and ready condition
			if repository.GetGeneration() != repository.GetStatus().ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(conditionType, "Blocked", "waiting for %s %s to reconcile", repositoryRef.Kind, repositoryRef.Name)
				return ErrGenerationMismatch
			}
			repository.GetStatus().InitializeConditions(ctx)
			if ready := repository.GetStatus().GetCondition(registriesv1alpha1.RepositoryConditionReady); !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(conditionType, "RepositoryNotReady", "%s: %s", ready.Reason, ready.Message)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(conditionType, "RepositoryNotReady", "%s: %s", ready.Reason, ready.Message)
				}
				return ErrDurable
			}

			if err := repository.Default(ctx); err != nil {
				return err
			}

			keychain, err := registry.KeychainForRepo(ctx, repository)
			if err != nil {
				return errors.Join(err, ErrTransient)
			}
			if kc, err := RepositoryKeychainStasher.RetrieveOrError(ctx); err == nil {
				// merge with existing stashed keychain
				keychain = authn.NewMultiKeychain(keychain, kc)
			}
			RepositoryKeychainStasher.Store(ctx, keychain)

			imageDigest, err := name.NewDigest(image.GetStatus().Image, name.StrictValidation)
			if err != nil {
				return err
			}
			// TODO should we verify the image is accessible?
			RemoteImageStasher.Store(ctx, imageDigest)

			conditionManager.MarkTrue(conditionType, "Ready", "")

			return nil
		},
	}
}

//+kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=repositories,verbs=get;list;watch
//+kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterrepositories,verbs=get;list;watch

func ResolveRepository[RR registriesv1alpha1.RepositoryReferencer](conditionType string) reconcilers.SubReconciler[RR] {
	return &reconcilers.SyncReconciler[RR]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&registriesv1alpha1.Repository{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&registriesv1alpha1.ClusterRepository{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource RR) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)
			conditionManager := resource.GetConditionManager(ctx)

			repositoryRef := resource.GetRepositoryReference()
			var repository registriesv1alpha1.GenericRepository
			if repositoryRef.Kind == "ClusterRepository" {
				repository = &registriesv1alpha1.ClusterRepository{
					ObjectMeta: metav1.ObjectMeta{
						Name: repositoryRef.Name,
					},
				}
			} else {
				repository = &registriesv1alpha1.Repository{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: resource.GetNamespace(),
						Name:      repositoryRef.Name,
					},
				}
			}
			if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(repository), repository); err != nil {
				if apierrs.IsNotFound(err) {
					resource.GetConditionManager(ctx).MarkFalse(conditionType, "RepositoryNotFound", "%s %s not found", repositoryRef.Kind, repositoryRef.Name)
					return ErrDurable
				}
				return err
			}

			// avoid premature reconciliation, check generation and ready condition
			if repository.GetGeneration() != repository.GetStatus().ObservedGeneration {
				resource.GetConditionManager(ctx).MarkUnknown(conditionType, "Blocked", "waiting for %s %s to reconcile", repositoryRef.Kind, repositoryRef.Name)
				return ErrGenerationMismatch
			}
			repository.GetStatus().InitializeConditions(ctx)
			if ready := repository.GetStatus().GetCondition(registriesv1alpha1.RepositoryConditionReady); !apis.ConditionIsTrue(ready) {
				if apis.ConditionIsFalse(ready) {
					resource.GetConditionManager(ctx).MarkFalse(conditionType, "RepositoryNotReady", "%s: %s", ready.Reason, ready.Message)
				} else {
					resource.GetConditionManager(ctx).MarkUnknown(conditionType, "RepositoryNotReady", "%s: %s", ready.Reason, ready.Message)
				}
				return ErrDurable
			}

			if err := repository.Default(ctx); err != nil {
				return err
			}
			keychain, err := registry.KeychainForRepo(ctx, repository)
			if err != nil {
				return errors.Join(err, ErrTransient)
			}
			if kc, err := RepositoryKeychainStasher.RetrieveOrError(ctx); err == nil {
				// merge with existing stashed keychain
				keychain = authn.NewMultiKeychain(keychain, kc)
			}
			RepositoryKeychainStasher.Store(ctx, keychain)

			tagRef, err := registry.ApplyTemplate(ctx, repository.GetSpec().Template, resource)
			if err != nil {
				return err
			}
			RepositoryTagStasher.Store(ctx, tagRef)

			conditionManager.MarkTrue(conditionType, "Ready", "")

			return nil
		},
	}
}

func ResolveKeychain[SAR registriesv1alpha1.ServiceAccountReferencer](conditionType string) reconcilers.SubReconciler[SAR] {
	return &reconcilers.SyncReconciler[SAR]{
		Sync: func(ctx context.Context, resource SAR) error {
			ref := resource.GetServiceAccountReference()
			if ref == nil {
				return nil
			}

			keychain, err := registry.KeychainForServiceAccountRef(ctx, *ref)
			if err != nil {
				if apierrs.IsNotFound(err) {
					status := err.(apierrs.APIStatus).Status()
					kind := status.Kind
					name := status.Details.Name
					resource.GetConditionManager(ctx).MarkFalse(conditionType, fmt.Sprintf("%sNotFound", kind), "%s %s not found", kind, name)
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

func PushComponent[GC componentsv1alpha1.ComponentLike](conditionType string) reconcilers.SubReconciler[GC] {
	return &reconcilers.CastResource[GC, componentsv1alpha1.ComponentLike]{
		Reconciler: &reconcilers.SyncReconciler[componentsv1alpha1.ComponentLike]{
			Sync: func(ctx context.Context, resource componentsv1alpha1.ComponentLike) error {
				c := reconcilers.RetrieveConfigOrDie(ctx)
				log := logr.FromContextOrDiscard(ctx)
				conditionManager := resource.GetConditionManager(ctx)

				component := ComponentStasher.RetrieveOrDie(ctx)
				tagRef := RepositoryTagStasher.RetrieveOrDie(ctx)
				keychain := RepositoryKeychainStasher.RetrieveOrDie(ctx)

				digestRef, config, err := registry.Push(ctx, tagRef, component, remote.WithAuthFromKeychain(keychain))
				if err != nil {
					log.Error(err, "failed to push component", "repository", tagRef.Name())
					c.Recorder.Eventf(resource, corev1.EventTypeWarning, "PushFailed", "%s", err)
					conditionManager.MarkFalse(conditionType, "PushFailed", "failed to push component to %q", tagRef.Name())
					return err
				} else {
					conditionManager.MarkTrue(conditionType, "Pushed", "")
				}

				RepositoryDigestStasher.Store(ctx, digestRef)
				ComponentConfigStasher.Store(ctx, config)

				return nil
			},
		},
	}
}

//+kubebuilder:rbac:groups=wa8s.reconciler.io,resources=components,verbs=get;list;watch;create;update;patch;delete

func ComponentChildReconciler[GC componentsv1alpha1.ComponentLike](conditionType, childLabelKey string, ourChild func(resource componentsv1alpha1.ComponentLike, child *componentsv1alpha1.Component) bool) reconcilers.SubReconciler[GC] {
	return &reconcilers.CastResource[GC, componentsv1alpha1.ComponentLike]{
		Reconciler: &reconcilers.ChildReconciler[componentsv1alpha1.ComponentLike, *componentsv1alpha1.Component, *componentsv1alpha1.ComponentList]{
			DesiredChild: func(ctx context.Context, resource componentsv1alpha1.ComponentLike) (*componentsv1alpha1.Component, error) {
				if !ChildComponentShouldExist(ctx, resource) {
					return nil, nil
				}

				c := reconcilers.RetrieveConfigOrDie(ctx)
				gvk, err := c.GroupVersionKindFor(resource)
				if err != nil {
					return nil, err
				}

				componentSpec := resource.GetGenericComponentSpec()
				if componentSpec == nil {
					componentSpec = &componentsv1alpha1.GenericComponentSpec{}
				}

				return &componentsv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: resource.GetNamespace(),
						Name:      resource.GetName(),
						Labels: reconcilers.MergeMaps(
							resource.GetLabels(),
							map[string]string{
								childLabelKey: resource.GetName(),
							},
						),
					},
					Spec: componentsv1alpha1.ComponentSpec{
						GenericComponentSpec: *componentSpec.DeepCopy(),
						Ref: &componentsv1alpha1.ComponentReference{
							APIVersion: gvk.GroupVersion().String(),
							Kind:       gvk.Kind,
							Namespace:  resource.GetNamespace(),
							Name:       resource.GetName(),
						},
					},
				}, nil
			},
			OurChild: ourChild,
			ChildObjectManager: &reconcilers.UpdatingObjectManager[*componentsv1alpha1.Component]{
				MergeBeforeUpdate: func(current, desired *componentsv1alpha1.Component) {
					current.Labels = desired.Labels
					current.Spec = desired.Spec
				},
			},
			ReflectChildStatusOnParentWithError: func(ctx context.Context, parent componentsv1alpha1.ComponentLike, child *componentsv1alpha1.Component, err error) error {
				if err != nil {
					if apierrs.IsInvalid(err) {
						parent.GetConditionManager(ctx).MarkFalse(conditionType, "Invalid", "%s", apierrs.ReasonForError(err))
					} else if apierrs.IsAlreadyExists(err) {
						parent.GetConditionManager(ctx).MarkFalse(conditionType, "AlreadyExists", "another Component already exists with name %s", parent.GetName())
						return ErrDurable
					} else {
						parent.GetConditionManager(ctx).MarkUnknown(conditionType, "Unknown", "")
					}
					return errors.Join(err, ErrTransient)
				}

				if child == nil {
					if !ChildComponentShouldExist(ctx, parent) {
						parent.GetConditionManager(ctx).MarkTrue(conditionType, "NotNeeded", "")
						return nil
					}
					parent.GetConditionManager(ctx).MarkFalse(conditionType, "Missing", "")
					return ErrDurable
				}

				// don't reflect the child's status here to avoid reconciliation loops
				parent.GetConditionManager(ctx).MarkTrue(conditionType, "Exists", "")

				return nil
			},
		},
	}
}

func ChildComponentShouldExist(ctx context.Context, resource componentsv1alpha1.ComponentLike) bool {
	annotations := resource.GetAnnotations()
	if annotations != nil {
		createChild := annotations[apis.CreateChildComponentAnnotation]
		if createChild == apis.CreateChildComponentTrue {
			return true
		}
		if createChild == apis.CreateChildComponentFalse {
			return false
		}
	}
	return len(resource.GetOwnerReferences()) == 0
}

func ReflectComponentableStatus[GC componentsv1alpha1.ComponentLike]() reconcilers.SubReconciler[GC] {
	return &reconcilers.CastResource[GC, componentsv1alpha1.ComponentLike]{
		Reconciler: &reconcilers.SyncReconciler[componentsv1alpha1.ComponentLike]{
			Sync: func(ctx context.Context, resource componentsv1alpha1.ComponentLike) error {
				digestRef := RepositoryDigestStasher.RetrieveOrDie(ctx)
				resource.GetGenericComponentStatus().Image = digestRef.Name()

				trace := ComponentTraceStasher.RetrieveOrEmpty(ctx)
				resource.GetGenericComponentStatus().Trace = trace

				if config, err := ComponentConfigStasher.RetrieveOrError(ctx); err != nil {
					resource.GetGenericComponentStatus().WIT = nil
				} else {
					resource.GetGenericComponentStatus().WIT = &componentsv1alpha1.WIT{
						Imports: config.Component.Imports,
						Exports: config.Component.Exports,
					}
				}

				return nil
			},
		},
	}
}

func SynthesizeSpan(ctx context.Context, resource client.Object) componentsv1alpha1.ComponentSpan {
	c := reconcilers.RetrieveConfigOrDie(ctx)

	var image string
	var trace []componentsv1alpha1.ComponentSpan

	if component, ok := resource.(componentsv1alpha1.ComponentLike); ok {
		image = component.GetGenericComponentStatus().Image
		trace = component.GetGenericComponentStatus().Trace
	} else {
		// convert non-ComponentLike resources to a Component as a good faith attempt
		component := &componentsv1alpha1.ComponentDuck{}
		if err := duck.Convert(resource, component); err == nil {
			image = component.Status.Image
			trace = component.Status.Trace
		}
	}

	gvk, err := c.GroupVersionKindFor(resource)
	if err != nil {
		panic(err)
	}
	digestRef, _ := name.NewDigest(image)

	return componentsv1alpha1.ComponentSpan{
		Digest:    digestRef.DigestStr(),
		UID:       resource.GetUID(),
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: resource.GetNamespace(),
		Name:      resource.GetName(),
		Trace:     trace,
	}
}

func ReflectTrace(resource client.Object, trace []componentsv1alpha1.ComponentSpan) error {
	if component, ok := resource.(componentsv1alpha1.ComponentLike); ok {
		component.GetGenericComponentStatus().Trace = trace
		return nil
	}

	// patch the resource
	base := &componentsv1alpha1.ComponentDuck{
		ObjectMeta: metav1.ObjectMeta{
			// patch will fail if the generation doesn't match
			Generation: resource.GetGeneration(),
		},
	}
	withTrace := base.DeepCopy()
	withTrace.Status.Trace = trace

	patch, err := reconcilers.NewPatch(base, withTrace)
	if err != nil {
		return err
	}
	if err := patch.Apply(resource); err != nil {
		return err
	}

	return nil
}

func DetectTraceCycle(trace []componentsv1alpha1.ComponentSpan, component client.Object) (bool, []componentsv1alpha1.ComponentSpan) {
	if trace == nil {
		return false, nil
	}

	hasCycle := false
	sanitized := []componentsv1alpha1.ComponentSpan{}
	for _, s := range trace {
		s = *s.DeepCopy()
		if s.UID == component.GetUID() {
			hasCycle = true
			s.CycleOmitted = true
			s.Trace = nil
		}
		if hc, st := DetectTraceCycle(s.Trace, component); hc {
			hasCycle = true
			s.Trace = st
		}
		sanitized = append(sanitized, s)
	}
	return hasCycle, sanitized
}

//+kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=componentcontainerimages,verbs=get;list;watch;create;update;patch;delete

func ComponentContainerImageChildReconciler[GC containersv1alpha1.GenericContainer](conditionType, childLabelKey string, imageRef registriesv1alpha1.ImageReference) reconcilers.SubReconciler[GC] {
	return &reconcilers.CastResource[GC, containersv1alpha1.GenericContainer]{
		Reconciler: &reconcilers.ChildReconciler[containersv1alpha1.GenericContainer, *containersv1alpha1.ComponentContainerImage, *containersv1alpha1.ComponentContainerImageList]{
			DesiredChild: func(ctx context.Context, resource containersv1alpha1.GenericContainer) (*containersv1alpha1.ComponentContainerImage, error) {
				return &containersv1alpha1.ComponentContainerImage{
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
					Spec: containersv1alpha1.ComponentContainerImageSpec{
						Ref:                  resource.GetGenericContainerSpec().Ref,
						ImageRef:             imageRef,
						GenericComponentSpec: resource.GetGenericContainerSpec().GenericComponentSpec,
					},
				}, nil
			},
			ChildObjectManager: &reconcilers.UpdatingObjectManager[*containersv1alpha1.ComponentContainerImage]{
				MergeBeforeUpdate: func(current, desired *containersv1alpha1.ComponentContainerImage) {
					current.Labels = desired.Labels
					current.Spec = desired.Spec
				},
			},
			ReflectChildStatusOnParentWithError: func(ctx context.Context, parent containersv1alpha1.GenericContainer, child *containersv1alpha1.ComponentContainerImage, err error) error {
				if err != nil {
					if apierrs.IsInvalid(err) {
						parent.GetConditionManager(ctx).MarkFalse(conditionType, "Invalid", "%s", apierrs.ReasonForError(err))
					} else {
						parent.GetConditionManager(ctx).MarkUnknown(conditionType, "Unknown", "")
					}
					return errors.Join(err, ErrTransient)
				}

				if child == nil {
					parent.GetConditionManager(ctx).MarkFalse(conditionType, "Missing", "")
					return ErrDurable
				}

				// avoid premature reconciliation, check generation and ready condition
				if child.Generation != child.Status.ObservedGeneration {
					parent.GetConditionManager(ctx).MarkUnknown(conditionType, "Blocked", "waiting for ComponentContainerImage %s to reconcile", child.Name)
					return ErrGenerationMismatch
				}
				if ready := child.Status.GetCondition(containersv1alpha1.ComponentContainerImageConditionReady); !apis.ConditionIsTrue(ready) {
					if ready == nil {
						ready = &metav1.Condition{Reason: "Initializing"}
					}
					if apis.ConditionIsFalse(ready) {
						parent.GetConditionManager(ctx).MarkUnknown(conditionType, "NotReady", "ComponentContainerImage %s is not ready", child.Name)
					} else {
						parent.GetConditionManager(ctx).MarkUnknown(conditionType, "NotReady", "ComponentContainerImage %s is not ready", child.Name)
					}
					return ErrDurable
				}

				if child.Status.Image == "" {
					// should never be ready and missing an image, but ya know
					parent.GetConditionManager(ctx).MarkFalse(conditionType, "ImageMissing", "ComponentContainerImage %s is missing image", child.Name)
					return ErrDurable
				}

				parent.GetGenericContainerStatus().GenericComponentStatus = *child.Status.GenericComponentStatus.DeepCopy()
				parent.GetGenericContainerStatus().Trace = []componentsv1alpha1.ComponentSpan{SynthesizeSpan(ctx, child)}
				parent.GetConditionManager(ctx).MarkTrue(conditionType, "Ready", "")

				return nil
			},
		},
	}
}

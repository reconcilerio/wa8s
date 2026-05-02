/*
Copyright 2026.

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

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
	"reconciler.io/wa8s/registry"
)

// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=images,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=images/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=images/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ImageReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[registriesv1alpha1.GenericImage] {
	return genericImageReconciler(c, &registriesv1alpha1.Image{}, &registriesv1alpha1.ImageList{})
}

// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterimages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterimages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterimages/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func ClusterImageReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[registriesv1alpha1.GenericImage] {
	return genericImageReconciler(c, &registriesv1alpha1.ClusterImage{}, &registriesv1alpha1.ClusterImageList{})
}

func genericImageReconciler(c reconcilers.Config, t registriesv1alpha1.GenericImage, lt client.ObjectList) *reconcilers.ResourceReconciler[registriesv1alpha1.GenericImage] {
	return &reconcilers.ResourceReconciler[registriesv1alpha1.GenericImage]{
		Type: t,

		Reconciler: &reconcilers.SuppressTransientErrors[registriesv1alpha1.GenericImage, client.ObjectList]{
			ListType: lt,
			Reconciler: reconcilers.Sequence[registriesv1alpha1.GenericImage]{
				reconcilers.Always[registriesv1alpha1.GenericImage]{
					ResolveRepository[registriesv1alpha1.GenericImage](registriesv1alpha1.ImageConditionRepositoryReady),
					ResolveKeychain[registriesv1alpha1.GenericImage](registriesv1alpha1.ImageConditionCopied),
				},
				CopyImage(),
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch

func CopyImage() *reconcilers.SyncReconciler[registriesv1alpha1.GenericImage] {
	return &reconcilers.SyncReconciler[registriesv1alpha1.GenericImage]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&corev1.Secret{}, reconcilers.EnqueueTracked(ctx))
			bldr.Watches(&corev1.ServiceAccount{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource registriesv1alpha1.GenericImage) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)
			conditionManager := resource.GetConditionManager(ctx)
			log := logr.FromContextOrDiscard(ctx)

			keychain := RepositoryKeychainStasher.RetrieveOrDie(ctx)
			tagRef := RepositoryTagStasher.RetrieveOrDie(ctx)

			source, err := registry.ResolveDigest(ctx, resource.GetSpec().Image, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				return err
			}

			digestRef, err := registry.Copy(ctx, source, tagRef, remote.WithAuthFromKeychain(keychain))
			if err != nil {
				log.Error(err, "failed to copy image", "repository", tagRef.Name())
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "CopyFailed", "%s", err)
				conditionManager.MarkFalse(registriesv1alpha1.ImageConditionCopied, "CopyFailed", "failed to copy image to %q", tagRef.Name())
				return err
			}

			conditionManager.MarkTrue(registriesv1alpha1.ImageConditionCopied, "Copied", "")

			resource.GetStatus().Image = digestRef.Name()

			return nil
		},
	}
}

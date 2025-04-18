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

	"github.com/google/go-containerregistry/pkg/v1/remote"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
	"reconciler.io/wa8s/registry"
)

// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=repositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=repositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=repositories/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func RepositoryReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[registriesv1alpha1.GenericRepository] {
	return genericRepositoryReconciler(c, &registriesv1alpha1.Repository{}, &registriesv1alpha1.RepositoryList{})
}

// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=registries.wa8s.reconciler.io,resources=clusterrepositories/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func ClusterRepositoryReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[registriesv1alpha1.GenericRepository] {
	return genericRepositoryReconciler(c, &registriesv1alpha1.ClusterRepository{}, &registriesv1alpha1.ClusterRepositoryList{})
}

func genericRepositoryReconciler(c reconcilers.Config, t registriesv1alpha1.GenericRepository, lt client.ObjectList) *reconcilers.ResourceReconciler[registriesv1alpha1.GenericRepository] {
	return &reconcilers.ResourceReconciler[registriesv1alpha1.GenericRepository]{
		Type: t,

		Reconciler: &reconcilers.SuppressTransientErrors[registriesv1alpha1.GenericRepository, client.ObjectList]{
			ListType: lt,
			Reconciler: reconcilers.Sequence[registriesv1alpha1.GenericRepository]{
				RefreshKeychain(),
				CheckRepositoryAuthentication(),
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch

func RefreshKeychain() reconcilers.SubReconciler[registriesv1alpha1.GenericRepository] {
	return &reconcilers.WithFinalizer[registriesv1alpha1.GenericRepository]{
		Finalizer: fmt.Sprintf("%s/keychain", registriesv1alpha1.GroupVersion.Group),
		Reconciler: &reconcilers.SyncReconciler[registriesv1alpha1.GenericRepository]{
			Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
				bldr.Watches(&corev1.Secret{}, reconcilers.EnqueueTracked(ctx))
				bldr.Watches(&corev1.ServiceAccount{}, reconcilers.EnqueueTracked(ctx))

				return nil
			},
			Sync: func(ctx context.Context, resource registriesv1alpha1.GenericRepository) error {
				keychain, err := registry.KeychainManager.CreateForRepo(ctx, resource)
				if err != nil {
					if apierrs.IsNotFound(err) {
						status := err.(apierrs.APIStatus).Status()
						kind := status.Kind
						name := status.Details.Name
						resource.GetConditionManager(ctx).MarkFalse(registriesv1alpha1.RepositoryConditionCredentialsResolved, fmt.Sprintf("%sNotFound", kind), "%s %s not found", kind, name)
						return ErrDurable
					}
					return err
				}

				registry.KeychainManager.Set(resource, keychain)
				resource.GetConditionManager(ctx).MarkTrue(registriesv1alpha1.RepositoryConditionCredentialsResolved, "Resolved", "")

				return nil
			},
			Finalize: func(ctx context.Context, resource registriesv1alpha1.GenericRepository) error {
				registry.KeychainManager.Remove(resource)

				return nil
			},
		},
	}
}

func CheckRepositoryAuthentication() *reconcilers.SyncReconciler[registriesv1alpha1.GenericRepository] {
	return &reconcilers.SyncReconciler[registriesv1alpha1.GenericRepository]{
		Sync: func(ctx context.Context, resource registriesv1alpha1.GenericRepository) error {
			keychain, err := registry.KeychainManager.Get(resource)
			if err != nil {
				return errors.Join(err, ErrTransient)
			}

			ref, err := registry.ApplyTemplate(ctx, resource.GetSpec().Template, resource)
			if err != nil {
				resource.GetConditionManager(ctx).MarkFalse(registriesv1alpha1.RepositoryConditionAuthenticated, "InvalidTemplate", "%s", err)
				return ErrDurable
			}

			if err := remote.CheckPushPermission(ref, keychain, remote.DefaultTransport); err != nil {
				resource.GetConditionManager(ctx).MarkFalse(registriesv1alpha1.RepositoryConditionAuthenticated, "Unauthorized", "%s", err)
				return ErrDurable
			}

			resource.GetConditionManager(ctx).MarkTrue(registriesv1alpha1.RepositoryConditionAuthenticated, "Authenticated", "")
			return nil
		},
	}
}

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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	"reconciler.io/wa8s/components"
)

// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=configstores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=configstores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wa8s.reconciler.io,resources=configstores/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

func ConfigStoreReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*componentsv1alpha1.ConfigStore] {
	childLabelKey := fmt.Sprintf("%s/config-store", componentsv1alpha1.GroupVersion.Group)

	return &reconcilers.ResourceReconciler[*componentsv1alpha1.ConfigStore]{
		Reconciler: &reconcilers.SuppressTransientErrors[*componentsv1alpha1.ConfigStore, *componentsv1alpha1.ConfigStoreList]{
			Reconciler: reconcilers.Sequence[*componentsv1alpha1.ConfigStore]{
				reconcilers.Always[*componentsv1alpha1.ConfigStore]{
					CollectConfig(),
					ResolveRepository[*componentsv1alpha1.ConfigStore](componentsv1alpha1.ConfigStoreConditionRepositoryReady),
					ComponentChildReconciler[*componentsv1alpha1.ConfigStore](componentsv1alpha1.ConfigStoreConditionChildComponent, childLabelKey, nil),
				},
				ComponentizeConfig(),
				PushComponent[*componentsv1alpha1.ConfigStore](componentsv1alpha1.ConfigStoreConditionPushed),
				ReflectComponentableStatus[*componentsv1alpha1.ConfigStore](),
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch

func CollectConfig() reconcilers.SubReconciler[*componentsv1alpha1.ConfigStore] {
	return &reconcilers.SyncReconciler[*componentsv1alpha1.ConfigStore]{
		Setup: func(ctx context.Context, mgr manager.Manager, bldr *builder.TypedBuilder[reconcile.Request]) error {
			bldr.Watches(&corev1.ConfigMap{}, reconcilers.EnqueueTracked(ctx))

			return nil
		},
		Sync: func(ctx context.Context, resource *componentsv1alpha1.ConfigStore) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)
			conditionManager := resource.GetConditionManager(ctx)

			config := map[string]string{}

			for _, valuesFrom := range resource.Spec.GenericConfigStoreSpec.ValuesFrom {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: resource.Namespace,
						Name:      valuesFrom.Name,
					},
				}
				if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(configMap), configMap); err != nil {
					if apierrs.IsNotFound(err) {
						conditionManager.MarkFalse(componentsv1alpha1.ConfigStoreConditionConfigResolved, "ConfigMapNotFound", "ConfigMap %s not found", valuesFrom.Name)
						return ErrDurable
					}
					return err
				}
				for k, v := range configMap.Data {
					config[fmt.Sprintf("%s%s", valuesFrom.Prefix, k)] = v
				}
			}

			for _, values := range resource.Spec.GenericConfigStoreSpec.Values {
				if values.ValueFrom != nil {
					configMap := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: resource.Namespace,
							Name:      values.ValueFrom.Name,
						},
					}
					if err := c.TrackAndGet(ctx, client.ObjectKeyFromObject(configMap), configMap); err != nil {
						if apierrs.IsNotFound(err) {
							conditionManager.MarkFalse(componentsv1alpha1.ConfigStoreConditionConfigResolved, "ConfigMapNotFound", "ConfigMap %s not found", values.ValueFrom.Name)
							return ErrDurable
						}
						return err
					}
					if value, ok := configMap.Data[values.ValueFrom.Key]; ok {
						config[values.Name] = value
					} else {
						conditionManager.MarkFalse(componentsv1alpha1.ConfigStoreConditionConfigResolved, "KeyNotFound", "key %q not found in ConfigMap %s", values.ValueFrom.Key, values.ValueFrom.Name)
						return ErrDurable
					}
				} else {
					config[values.Name] = values.Value
				}
			}

			ConfigStoreStasher.Store(ctx, config)

			conditionManager.MarkTrue(componentsv1alpha1.ConfigStoreConditionConfigResolved, "Resolved", "")

			return nil
		},
	}
}

func ComponentizeConfig() reconcilers.SubReconciler[*componentsv1alpha1.ConfigStore] {
	return &reconcilers.SyncReconciler[*componentsv1alpha1.ConfigStore]{
		Sync: func(ctx context.Context, resource *componentsv1alpha1.ConfigStore) error {
			c := reconcilers.RetrieveConfigOrDie(ctx)
			config := ConfigStoreStasher.RetrieveOrDie(ctx)

			component, err := components.ComponentizeConfigStore(ctx, config)
			if err != nil {
				// not user recoverable
				logr.FromContextOrDiscard(ctx).Error(err, "componentize failed")
				c.Recorder.Eventf(resource, corev1.EventTypeWarning, "ComponentizeFailed", "%s", err)
				return err
			}

			ComponentStasher.Store(ctx, component)

			return nil
		},
	}
}

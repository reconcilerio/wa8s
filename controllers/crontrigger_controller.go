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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/reconcilers"

	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
)

// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=crontriggers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=crontriggers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=crontriggers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func CronTriggerReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*containersv1alpha1.CronTrigger] {
	childLabelKey := fmt.Sprintf("%s/cron-trigger", containersv1alpha1.GroupVersion.Group)
	baseImage := ""

	return &reconcilers.ResourceReconciler[*containersv1alpha1.CronTrigger]{
		Reconciler: &reconcilers.SuppressTransientErrors[*containersv1alpha1.CronTrigger, *containersv1alpha1.CronTriggerList]{
			Reconciler: reconcilers.Sequence[*containersv1alpha1.CronTrigger]{
				WasmContainerChildReconciler[*containersv1alpha1.CronTrigger](containersv1alpha1.CronTriggerConditionWasmtimeContainerReady, childLabelKey, baseImage),
				CronJobChildReconciler(childLabelKey),
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

func CronJobChildReconciler(childLabelKey string) reconcilers.SubReconciler[*containersv1alpha1.CronTrigger] {
	return &reconcilers.ChildReconciler[*containersv1alpha1.CronTrigger, *batchv1.CronJob, *batchv1.CronJobList]{
		DesiredChild: func(ctx context.Context, resource *containersv1alpha1.CronTrigger) (*batchv1.CronJob, error) {
			if resource.Status.Image == "" {
				return nil, nil
			}

			return &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    resource.Namespace,
					GenerateName: fmt.Sprintf("%s-", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							childLabelKey: resource.Name,
						},
					),
				},
				Spec: batchv1.CronJobSpec{
					Schedule: resource.Spec.Schedule,
					TimeZone: resource.Spec.TimeZone,
					JobTemplate: batchv1.JobTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: reconcilers.MergeMaps(
								resource.Labels,
								map[string]string{
									childLabelKey: resource.Name,
								},
							),
						},
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								ObjectMeta: metav1.ObjectMeta{
									Labels: reconcilers.MergeMaps(
										resource.Labels,
										map[string]string{
											childLabelKey: resource.Name,
										},
									),
								},
								Spec: corev1.PodSpec{
									RestartPolicy: resource.Spec.RestartPolicy,
									Containers: []corev1.Container{
										{
											Name:    "task",
											Image:   resource.Status.Image,
											Command: []string{"wasmtime"},
											Args: func() []string {
												args := []string{"run"}
												args = append(args, resource.Spec.HostCapabilities.WasmtimeArgs()...)
												args = append(args, "-Opooling-allocator=n", "/component.wasm")
												return args
											}(),
										},
									},
								},
							},
						},
					},
				},
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*batchv1.CronJob]{
			MergeBeforeUpdate: func(current, desired *batchv1.CronJob) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParent: func(ctx context.Context, parent *containersv1alpha1.CronTrigger, child *batchv1.CronJob, err error) {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.CronTriggerConditionCronJobReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.CronTriggerConditionCronJobReady, "Unknown", "")
				}
				return
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.CronTriggerConditionCronJobReady, "Missing", "")
				return
			}

			parent.GetConditionManager(ctx).MarkTrue(containersv1alpha1.CronTriggerConditionCronJobReady, "Created", "")
		},
	}
}

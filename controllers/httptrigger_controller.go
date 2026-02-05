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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reconciler.io/runtime/reconcilers"

	containersv1alpha1 "reconciler.io/wa8s/apis/containers/v1alpha1"
)

// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=httptriggers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=httptriggers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=containers.wa8s.reconciler.io,resources=httptriggers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core;events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

func HttpTriggerReconciler(c reconcilers.Config) *reconcilers.ResourceReconciler[*containersv1alpha1.HttpTrigger] {
	childLabelKey := fmt.Sprintf("%s/http-trigger", containersv1alpha1.GroupVersion.Group)
	baseImage := ""

	return &reconcilers.ResourceReconciler[*containersv1alpha1.HttpTrigger]{
		Reconciler: &reconcilers.SuppressTransientErrors[*containersv1alpha1.HttpTrigger, *containersv1alpha1.HttpTriggerList]{
			Reconciler: reconcilers.Sequence[*containersv1alpha1.HttpTrigger]{
				WasmContainerChildReconciler[*containersv1alpha1.HttpTrigger](containersv1alpha1.CronTriggerConditionWasmtimeContainerReady, childLabelKey, baseImage),
				HttpDeploymentChildReconciler(childLabelKey),
				HttpServiceChildReconciler(childLabelKey),
			},
		},

		Config: c,
	}
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func HttpDeploymentChildReconciler(childLabelKey string) reconcilers.SubReconciler[*containersv1alpha1.HttpTrigger] {
	return &reconcilers.ChildReconciler[*containersv1alpha1.HttpTrigger, *appsv1.Deployment, *appsv1.DeploymentList]{
		DesiredChild: func(ctx context.Context, resource *containersv1alpha1.HttpTrigger) (*appsv1.Deployment, error) {
			if resource.Status.Image == "" {
				return nil, nil
			}

			return &appsv1.Deployment{
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
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							childLabelKey: resource.Name,
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								childLabelKey: resource.Name,
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "http",
									Image:   resource.Status.Image,
									Command: []string{"wasmtime"},
									Args: func() []string {
										args := []string{"serve"}
										args = append(args, resource.Spec.HostCapabilities.WasmtimeArgs()...)
										args = append(args,
											// TODO configure cli capability
											"-Scli",
											"-Opooling-allocator=n",
											"--addr=0.0.0.0:8080",
											"/component.wasm",
										)
										return args
									}(),
									Ports: []corev1.ContainerPort{
										{
											Name:          "http",
											Protocol:      corev1.ProtocolTCP,
											ContainerPort: 8080,
										},
									},
									ReadinessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											TCPSocket: &corev1.TCPSocketAction{
												Port: intstr.FromString("http"),
											},
										},
									},
									LivenessProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											TCPSocket: &corev1.TCPSocketAction{
												Port: intstr.FromString("http"),
											},
										},
									},
									StartupProbe: &corev1.Probe{
										ProbeHandler: corev1.ProbeHandler{
											TCPSocket: &corev1.TCPSocketAction{
												Port: intstr.FromString("http"),
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
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*appsv1.Deployment]{
			MergeBeforeUpdate: func(current, desired *appsv1.Deployment) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *containersv1alpha1.HttpTrigger, child *appsv1.Deployment, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.HttpTriggerConditionDeploymentReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.HttpTriggerConditionDeploymentReady, "Unknown", "")
				}
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.HttpTriggerConditionDeploymentReady, "Missing", "")
				return ErrDurable
			}

			// avoid premature reconciliation, check generation and ready condition
			if child.Generation != child.Status.ObservedGeneration {
				parent.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.HttpTriggerConditionDeploymentReady, "Blocked", "waiting for Deployment %s to reconcile", child.Name)
				return ErrGenerationMismatch
			}

			if child.Status.ReadyReplicas > 0 && child.Status.UpdatedReplicas > 0 {
				parent.GetConditionManager(ctx).MarkTrue(containersv1alpha1.HttpTriggerConditionDeploymentReady, "Ready", "")
			} else {
				parent.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.HttpTriggerConditionDeploymentReady, "NotReady", "Deployment %s is not ready", child.Name)
			}

			return nil
		},
	}
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func HttpServiceChildReconciler(childLabelKey string) reconcilers.SubReconciler[*containersv1alpha1.HttpTrigger] {
	return &reconcilers.ChildReconciler[*containersv1alpha1.HttpTrigger, *corev1.Service, *corev1.ServiceList]{
		DesiredChild: func(ctx context.Context, resource *containersv1alpha1.HttpTrigger) (*corev1.Service, error) {
			if resource.Status.Image == "" {
				return nil, nil
			}

			return &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: resource.Namespace,
					Name:      fmt.Sprintf("%s-trigger", resource.Name),
					Labels: reconcilers.MergeMaps(
						resource.Labels,
						map[string]string{
							childLabelKey: resource.Name,
						},
					),
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Selector: map[string]string{
						childLabelKey: resource.Name,
					},
					Ports: []corev1.ServicePort{
						{
							Protocol:   corev1.ProtocolTCP,
							Port:       80,
							TargetPort: intstr.FromString("http"),
						},
					},
				},
			}, nil
		},
		ChildObjectManager: &reconcilers.UpdatingObjectManager[*corev1.Service]{
			HarmonizeImmutableFields: func(current, desired *corev1.Service) {
				desired.Spec.ClusterIP = current.Spec.ClusterIP
				desired.Spec.ClusterIPs = current.Spec.ClusterIPs
			},
			MergeBeforeUpdate: func(current, desired *corev1.Service) {
				current.Labels = desired.Labels
				current.Spec = desired.Spec
			},
		},
		ReflectChildStatusOnParentWithError: func(ctx context.Context, parent *containersv1alpha1.HttpTrigger, child *corev1.Service, err error) error {
			if err != nil {
				if apierrs.IsInvalid(err) {
					parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.HttpTriggerConditionServiceReady, "Invalid", "%s", apierrs.ReasonForError(err))
				} else if apierrs.IsAlreadyExists(err) {
					parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.HttpTriggerConditionServiceReady, "AlreadyExists", "%s", apierrs.ReasonForError(err))
					return ErrDurable
				} else {
					parent.GetConditionManager(ctx).MarkUnknown(containersv1alpha1.HttpTriggerConditionServiceReady, "Unknown", "%s", apierrs.ReasonForError(err))
				}
				return errors.Join(err, ErrTransient)
			}

			if child == nil {
				parent.GetConditionManager(ctx).MarkFalse(containersv1alpha1.HttpTriggerConditionServiceReady, "Missing", "")
				return ErrDurable
			}

			// Service has no meaningful status to inspect, assume it's ok

			parent.Status.URL = fmt.Sprintf("http://%s.%s.svc.cluster.local", child.Name, child.Namespace)
			parent.GetConditionManager(ctx).MarkTrue(containersv1alpha1.HttpTriggerConditionServiceReady, "Created", "")

			return nil
		},
	}
}

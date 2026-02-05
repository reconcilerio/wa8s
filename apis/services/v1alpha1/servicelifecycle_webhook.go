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

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-servicelifecycle,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=servicelifecycles,verbs=create;update,versions=v1alpha1,name=v1alpha1.servicelifecycles.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}
// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-clusterservicelifecycle,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=clusterservicelifecycles,verbs=create;update,versions=v1alpha1,name=v1alpha1.clusterservicelifecycles.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ServiceLifecycle) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

func (r *ClusterServiceLifecycle) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &ServiceLifecycle{}
var _ reconcilers.Defaulter = &ClusterServiceLifecycle{}

func (r *ServiceLifecycle) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ClusterServiceLifecycle) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceLifecycleSpec) Default(ctx context.Context) error {
	if err := r.GenericContainerSpec.Default(ctx); err != nil {
		return err
	}
	if err := r.ClientRef.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceLifecycleReference) Default(ctx context.Context) error {
	if r.Kind == "" {
		r.Kind = "ClusterServiceLifecycle"
	}

	if r.Namespace == "" {
		if r.Kind != "ClusterServiceLifecycle" {
			r.Namespace = validation.RetrieveResource(ctx).GetNamespace()
		}
	}

	return nil
}

func (r *ServiceLifecycleReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Kind == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("kind"), ""))
	}
	if r.Kind != "ServiceLifecycle" && r.Kind != "ClusterServiceLifecycle" {
		errs = append(errs, field.Invalid(fldPath.Child("kind"), r.Kind, "allowed values are ServiceLifecycle or ClusterServiceLifecycle"))
	}

	if r.Name == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

var _ admission.Validator[*ServiceLifecycle] = &ServiceLifecycle{}
var _ admission.Validator[*ClusterServiceLifecycle] = &ClusterServiceLifecycle{}

func (r *ServiceLifecycle) ValidateCreate(ctx context.Context, obj *ServiceLifecycle) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceLifecycle) ValidateUpdate(ctx context.Context, oldObj, newObj *ServiceLifecycle) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceLifecycle) ValidateDelete(ctx context.Context, obj *ServiceLifecycle) (warnings admission.Warnings, err error) {
	return
}

func (r *ServiceLifecycle) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ClusterServiceLifecycle) ValidateCreate(ctx context.Context, obj *ClusterServiceLifecycle) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterServiceLifecycle) ValidateUpdate(ctx context.Context, oldObj, newObj *ClusterServiceLifecycle) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterServiceLifecycle) ValidateDelete(ctx context.Context, obj *ClusterServiceLifecycle) (warnings admission.Warnings, err error) {
	return
}

func (r *ClusterServiceLifecycle) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ServiceLifecycleSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericContainerSpec.Validate(ctx, fldPath)...)
	errs = append(errs, r.ClientRef.Validate(ctx, fldPath.Child("clientRef"))...)

	return errs
}

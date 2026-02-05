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
	"fmt"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-serviceinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=serviceinstances;serviceinstances/status,verbs=create;update,versions=v1alpha1,name=v1alpha1.serviceinstances.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ServiceInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &ServiceInstance{}

func (r *ServiceInstance) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceInstanceSpec) Default(ctx context.Context) error {
	if err := r.Ref.Default(ctx); err != nil {
		return err
	}
	for i := range r.Requests {
		if err := r.Requests[i].Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *ServiceInstanceRequest) Default(ctx context.Context) error {
	return nil
}

func (r *ServiceInstanceReference) Default(ctx context.Context) error {
	return nil
}

func (r *ServiceInstanceReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Name == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

var _ admission.Validator[*ServiceInstance] = &ServiceInstance{}

func (r *ServiceInstance) ValidateCreate(ctx context.Context, obj *ServiceInstance) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceInstance) ValidateUpdate(ctx context.Context, oldObj, newObj *ServiceInstance) (warnings admission.Warnings, err error) {
	if err := oldObj.Default(ctx); err != nil {
		return nil, err
	}
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	// enforce immutability
	if !cmp.Equal(newObj.Spec, oldObj.Spec) {
		// TODO relax on a field by field basis
		return nil, fmt.Errorf(".spec updates are disallowed")
	}
	if oldObj.Status.ServiceInstanceId != "" && newObj.Status.ServiceInstanceId != oldObj.Status.ServiceInstanceId {
		return nil, fmt.Errorf(".status.serviceInstanceId is immutable once set")
	}

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceInstance) ValidateDelete(ctx context.Context, obj *ServiceInstance) (warnings admission.Warnings, err error) {
	return
}

func (r *ServiceInstance) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ServiceInstanceSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	if r.Type == "" {
		errs = append(errs, field.Required(fldPath.Child("type"), ""))
	}
	for i := range r.Requests {
		errs = append(errs, r.Requests[i].Validate(ctx, fldPath.Child("requests").Index(i))...)
	}

	return errs
}

func (r *ServiceInstanceRequest) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Key == "" {
		errs = append(errs, field.Required(fldPath.Child("key"), ""))
	}
	if r.Value == "" {
		errs = append(errs, field.Required(fldPath.Child("value"), ""))
	}

	return errs
}

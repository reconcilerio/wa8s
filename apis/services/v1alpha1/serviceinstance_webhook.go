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
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-serviceinstance,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=serviceinstances;serviceinstances/status,verbs=create;update,versions=v1alpha1,name=v1alpha1.serviceinstances.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ServiceInstance) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &ServiceInstance{}

func (r *ServiceInstance) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*ServiceInstance)
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

var _ webhook.CustomValidator = &ServiceInstance{}

func (r *ServiceInstance) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*ServiceInstance)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceInstance) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, oldObj); err != nil {
		return nil, err
	}
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*ServiceInstance)
	ctx = validation.StashResource(ctx, r)

	// enforce immutability
	old := oldObj.(*ServiceInstance)
	if !cmp.Equal(r.Spec, old.Spec) {
		// TODO relax on a field by field basis
		return nil, fmt.Errorf(".spec updates are disallowed")
	}
	if old.Status.ServiceInstanceId != "" && r.Status.ServiceInstanceId != old.Status.ServiceInstanceId {
		return nil, fmt.Errorf(".status.serviceInstanceId is immutable once set")
	}

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceInstance) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *ServiceInstance) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

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

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
	"time"

	"github.com/google/go-cmp/cmp"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-servicebinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=servicebindings;servicebindings/status,verbs=create;update,versions=v1alpha1,name=v1alpha1.servicebindings.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ServiceBinding) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &ServiceBinding{}

func (r *ServiceBinding) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*ServiceBinding)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceBindingSpec) Default(ctx context.Context) error {
	if err := r.Ref.Default(ctx); err != nil {
		return err
	}

	return nil
}

var _ webhook.CustomValidator = &ServiceBinding{}

func (r *ServiceBinding) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*ServiceBinding)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceBinding) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, oldObj); err != nil {
		return nil, err
	}
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*ServiceBinding)
	ctx = validation.StashResource(ctx, r)

	// enforce immutability
	old := oldObj.(*ServiceBinding)
	if !cmp.Equal(r.Spec, old.Spec) {
		return nil, fmt.Errorf(".spec updates are disallowed")
	}
	if old.Status.ServiceBindingId != "" && r.Status.ServiceBindingId != old.Status.ServiceBindingId {
		return nil, fmt.Errorf(".status.serviceBindingId is immutable once set")
	}

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceBinding) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *ServiceBinding) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ServiceBindingSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	for i, scope := range r.Scopes {
		if scope == "" {
			errs = append(errs, field.Invalid(fldPath.Child("scopes").Index(i), scope, "may not be empty"))
		}
	}

	if r.Duration.Duration < time.Hour {
		errs = append(errs, field.Invalid(fldPath.Child("duration"), r.Duration.Duration.String(), "minimum duration 1h"))
	}

	return errs
}

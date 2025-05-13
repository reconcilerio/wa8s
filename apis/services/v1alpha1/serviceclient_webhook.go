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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-serviceclient,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=serviceclients,verbs=create;update,versions=v1alpha1,name=v1alpha1.serviceclients.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ServiceClient) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &ServiceClient{}

func (r *ServiceClient) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*ServiceClient)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceClientSpec) Default(ctx context.Context) error {
	if err := r.Ref.Default(ctx); err != nil {
		return err
	}

	if r.Duration.Duration == 0 {
		r.Duration = metav1.Duration{
			Duration: 90 * 24 * time.Hour,
		}
	}
	if r.RenewBefore.Duration == 0 {
		if r.RenewBeforePercentage == 0 {
			r.RenewBeforePercentage = 33
		}
		r.RenewBefore = metav1.Duration{
			Duration: r.Duration.Duration * time.Duration(r.RenewBeforePercentage) / 100,
		}
		if r.RenewBefore.Duration < 5*time.Minute {
			r.RenewBefore = metav1.Duration{
				Duration: 5 * time.Minute,
			}
		}
	}

	return nil
}

var _ webhook.CustomValidator = &ServiceClient{}

func (r *ServiceClient) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*ServiceClient)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceClient) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*ServiceClient)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceClient) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *ServiceClient) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ServiceClientSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
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
	if r.RenewBefore.Duration < 5*time.Minute {
		errs = append(errs, field.Invalid(fldPath.Child("renewBefore"), r.RenewBefore.Duration.String(), "minimum duration 5m"))
	}
	if r.RenewBeforePercentage < 0 || r.RenewBeforePercentage > 100 {
		errs = append(errs, field.Invalid(fldPath.Child("renewBeforePercentage"), r.Duration.Duration.String(), "must be within 0-100"))
	}

	return errs
}

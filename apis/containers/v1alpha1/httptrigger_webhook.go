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

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/runtime/reconcilers"
	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

//+kubebuilder:webhook:path=/validate-containers-wa8s-reconciler-io-v1alpha1-httptrigger,mutating=false,failurePolicy=fail,sideEffects=None,groups=containers.wa8s.reconciler.io,resources=httptriggers,verbs=create;update,versions=v1alpha1,name=v1alpha1.httptriggers.containers.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *HttpTrigger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &HttpTrigger{}

func (r *HttpTrigger) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *HttpTriggerSpec) Default(ctx context.Context) error {
	if err := r.GenericContainerSpec.Default(ctx); err != nil {
		return err
	}

	return nil
}

var _ admission.Validator[*HttpTrigger] = &HttpTrigger{}

func (r *HttpTrigger) ValidateCreate(ctx context.Context, obj *HttpTrigger) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *HttpTrigger) ValidateUpdate(ctx context.Context, oldObj, newObj *HttpTrigger) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *HttpTrigger) ValidateDelete(ctx context.Context, obj *HttpTrigger) (warnings admission.Warnings, err error) {
	return
}

func (r *HttpTrigger) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *HttpTriggerSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericContainerSpec.Validate(ctx, fldPath)...)

	return errs
}

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
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/runtime/reconcilers"
	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-services-wa8s-reconciler-io-v1alpha1-serviceresourcedefinition,mutating=false,failurePolicy=fail,sideEffects=None,groups=services.wa8s.reconciler.io,resources=serviceresourcedefinitions,verbs=create;update,versions=v1alpha1,name=v1alpha1.serviceresourcedefinitions.services.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ServiceResourceDefinition) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &ServiceResourceDefinition{}

func (r *ServiceResourceDefinition) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceResourceDefinitionSpec) Default(ctx context.Context) error {
	if err := r.InstanceNames.Default(ctx); err != nil {
		return err
	}
	if err := r.ClientNames.Default(ctx); err != nil {
		return err
	}
	if err := r.Lifecycle.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceResourceDefinitionNames) Default(ctx context.Context) error {
	categories := sets.NewString(r.Categories...)
	if !categories.Has("wa8s") {
		r.Categories = append(r.Categories, "wa8s")
	}
	if !categories.Has("wa8s-services") {
		r.Categories = append(r.Categories, "services")
	}

	return nil
}

var _ admission.Validator[*ServiceResourceDefinition] = &ServiceResourceDefinition{}

func (r *ServiceResourceDefinition) ValidateCreate(ctx context.Context, obj *ServiceResourceDefinition) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceResourceDefinition) ValidateUpdate(ctx context.Context, oldObj, newObj *ServiceResourceDefinition) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ServiceResourceDefinition) ValidateDelete(ctx context.Context, obj *ServiceResourceDefinition) (warnings admission.Warnings, err error) {
	return
}

func (r *ServiceResourceDefinition) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ServiceResourceDefinitionSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.InstanceNames.Validate(ctx, fldPath.Child("instanceNames"))...)
	errs = append(errs, r.ClientNames.Validate(ctx, fldPath.Child("clientNames"))...)
	errs = append(errs, r.Lifecycle.Validate(ctx, fldPath.Child("lifecycle"))...)

	return errs
}

func (r *ServiceResourceDefinitionNames) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Kind == "" {
		errs = append(errs, field.Required(fldPath.Child("kind"), ""))
	}
	if strings.ToUpper(r.Kind[0:1]) != r.Kind[0:1] {
		errs = append(errs, field.Invalid(fldPath.Child("kind"), r.Kind, "must be camel case"))
	}
	if r.Plural == "" {
		errs = append(errs, field.Required(fldPath.Child("plural"), ""))
	}
	if strings.ToLower(r.Plural) != r.Plural {
		errs = append(errs, field.Invalid(fldPath.Child("kind"), r.Kind, "must be lower case"))
	}

	return errs
}

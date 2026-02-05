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

//+kubebuilder:webhook:path=/validate-wa8s-reconciler-io-v1alpha1-configstore,mutating=false,failurePolicy=fail,sideEffects=None,groups=wa8s.reconciler.io,resources=configstores,verbs=create;update,versions=v1alpha1,name=v1alpha1.configstores.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ConfigStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &ConfigStore{}

func (r *ConfigStore) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ConfigStoreSpec) Default(ctx context.Context) error {
	if err := r.GenericComponentSpec.Default(ctx); err != nil {
		return err
	}
	if err := r.GenericConfigStoreSpec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *GenericConfigStoreSpec) Default(ctx context.Context) error {
	for i := range r.ValuesFrom {
		if err := r.ValuesFrom[i].Default(ctx); err != nil {
			return err
		}
	}
	for i := range r.Values {
		if err := r.Values[i].Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *ValuesFrom) Default(ctx context.Context) error {
	if r.Name == "" {
		r.Name = validation.RetrieveResource(ctx).GetName()
	}

	return nil
}

func (r *Value) Default(ctx context.Context) error {
	if r.ValueFrom != nil {
		if err := r.ValueFrom.Default(ctx); err != nil {
			return nil
		}
	}

	return nil
}

func (r *ValueFrom) Default(ctx context.Context) error {
	if r.Name == "" {
		r.Name = validation.RetrieveResource(ctx).GetName()
	}

	return nil
}

var _ admission.Validator[*ConfigStore] = &ConfigStore{}

func (r *ConfigStore) ValidateCreate(ctx context.Context, obj *ConfigStore) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ConfigStore) ValidateUpdate(ctx context.Context, oldObj, newObj *ConfigStore) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ConfigStore) ValidateDelete(ctx context.Context, obj *ConfigStore) (warnings admission.Warnings, err error) {
	return
}

func (r *ConfigStore) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ConfigStoreSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericComponentSpec.Validate(ctx, fldPath)...)
	errs = append(errs, r.GenericConfigStoreSpec.Validate(ctx, fldPath)...)

	return errs
}

func (r *GenericConfigStoreSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	for i := range r.Values {
		errs = append(errs, r.Values[i].Validate(ctx, fldPath.Child("values").Index(i))...)
	}
	for i := range r.ValuesFrom {
		errs = append(errs, r.ValuesFrom[i].Validate(ctx, fldPath.Child("valuesFrom").Index(i))...)
	}

	return errs
}

func (r *ValuesFrom) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

func (r *Value) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}
	if r.Value != "" && r.ValueFrom != nil {
		errs = append(errs, field.Invalid(fldPath, nil, "value and valueFrom are mutually exclusive"))
	}
	if r.ValueFrom != nil {
		errs = append(errs, r.ValueFrom.Validate(ctx, fldPath.Child("valueFrom"))...)
	}

	return errs
}

func (r *ValueFrom) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}
	if r.Key == "" {
		errs = append(errs, field.Required(fldPath.Child("key"), ""))
	}

	return errs
}

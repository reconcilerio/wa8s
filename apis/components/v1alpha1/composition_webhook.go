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
	"fmt"
	"strings"

	runtime "k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/json"

	"reconciler.io/wa8s/validation"
)

//+kubebuilder:webhook:path=/validate-wa8s-reconciler-io-v1alpha1-composition,mutating=false,failurePolicy=fail,sideEffects=None,groups=wa8s.reconciler.io,resources=compositions,verbs=create;update,versions=v1alpha1,name=v1alpha1.compositions.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *Composition) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &Component{}

func (r *Composition) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*Composition)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *CompositionSpec) Default(ctx context.Context) error {
	if err := r.GenericComponentSpec.Default(ctx); err != nil {
		return err
	}
	if err := r.GenericCompositionSpec.Default(ctx); err != nil {
		return err
	}
	return nil
}

func (r *GenericCompositionSpec) Default(ctx context.Context) error {
	for i := range r.Dependencies {
		if err := r.Dependencies[i].Default(ctx); err != nil {
			return err
		}
	}

	if r.Plug == nil && r.WAC == "" {
		r.Plug = &CompositionPlug{}
	}

	return nil
}

func (r *CompositionDependency) Default(ctx context.Context) error {
	if r.Ref != nil {
		if err := r.Ref.Default(ctx); err != nil {
			return err
		}
	}
	if r.Config != nil {
		if err := r.Config.Default(ctx); err != nil {
			return err
		}
	}
	if r.OCI != nil {
		if err := r.OCI.Default(ctx); err != nil {
			return err
		}
	}
	if r.Composition != nil {
		if err := r.Composition.Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

var _ webhook.CustomValidator = &Composition{}

func (r *Composition) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.validateNoUnknownFields(ctx); err != nil {
		return nil, err
	}

	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*Composition)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Composition) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.validateNoUnknownFields(ctx); err != nil {
		return nil, err
	}

	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*Composition)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Composition) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *Composition) validateNoUnknownFields(ctx context.Context) error {
	req, err := admission.RequestFromContext(ctx)
	if err != nil {
		// not in an admission request, ignore
		return nil
	}
	strictErrors, err := json.UnmarshalStrict(req.Object.Raw, &Composition{}, json.DisallowUnknownFields)
	if err != nil {
		return err
	}
	return utilerrors.NewAggregate(strictErrors)

}

func (r *Composition) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}
func (r *CompositionSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericComponentSpec.Validate(ctx, fldPath)...)
	errs = append(errs, r.GenericCompositionSpec.Validate(ctx, fldPath)...)

	return errs
}

func (r *GenericCompositionSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	picked := sets.New[string]()
	notPicked := sets.New[string]()
	if r.Plug != nil {
		picked.Insert("plug")
		errs = append(errs, r.Plug.Validate(ctx, fldPath.Child("plug"))...)
	} else {
		notPicked.Insert("plug")
	}
	if r.WAC != "" {
		picked.Insert("wac")
	} else {
		notPicked.Insert("wac")
	}
	if picked.Len() == 0 {
		errs = append(errs, field.Required(fldPath.Child(fmt.Sprintf("[%s]", strings.Join(sets.List(notPicked), ", "))), "pick one"))
	}
	if picked.Len() > 1 {
		errs = append(errs, field.Invalid(fldPath.Child(fmt.Sprintf("[%s]", strings.Join(sets.List(picked), ", "))), nil, "pick one"))
	}

	if len(r.Dependencies) < 2 {
		errs = append(errs, field.Invalid(fldPath.Child("dependencies"), nil, "at least two dependencies are required"))
	}
	for i := range r.Dependencies {
		errs = append(errs, r.Dependencies[i].Validate(ctx, fldPath.Child("dependencies").Index(i))...)
	}

	return errs
}

func (r *CompositionPlug) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	// nothing to do

	return errs
}

func (r *CompositionDependency) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Component == "" {
		errs = append(errs, field.Required(fldPath.Child("component"), ""))
	}

	picked := sets.New[string]()
	notPicked := sets.New[string]()
	if r.Ref != nil {
		picked.Insert("ref")
		errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	} else {
		notPicked.Insert("ref")
	}
	if r.Config != nil {
		picked.Insert("config")
		errs = append(errs, r.Config.Validate(ctx, fldPath.Child("config"))...)
	} else {
		notPicked.Insert("config")
	}
	if r.OCI != nil {
		picked.Insert("oci")
		errs = append(errs, r.OCI.Validate(ctx, fldPath.Child("oci"))...)
	} else {
		notPicked.Insert("oci")
	}
	if r.Composition != nil {
		picked.Insert("composition")
		errs = append(errs, r.Composition.Validate(ctx, fldPath.Child("composition"))...)
	} else {
		notPicked.Insert("composition")
	}
	if picked.Len() == 0 {
		errs = append(errs, field.Required(fldPath.Child(fmt.Sprintf("[%s]", strings.Join(sets.List(notPicked), ", "))), "pick one"))
	}
	if picked.Len() > 1 {
		errs = append(errs, field.Invalid(fldPath.Child(fmt.Sprintf("[%s]", strings.Join(sets.List(picked), ", "))), nil, "pick one"))
	}

	return errs
}

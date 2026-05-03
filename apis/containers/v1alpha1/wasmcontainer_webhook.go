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
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
	"reconciler.io/wa8s/validation"
)

//+kubebuilder:webhook:path=/validate-containers-wa8s-reconciler-io-v1alpha1-componentcontainerimage,mutating=false,failurePolicy=fail,sideEffects=None,groups=containers.wa8s.reconciler.io,resources=componentcontainerimages,verbs=create;update,versions=v1alpha1,name=v1alpha1.componentcontainerimages.containers.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *ComponentContainerImage) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &ComponentContainerImage{}

func (r *ComponentContainerImage) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ComponentContainerImageSpec) Default(ctx context.Context) error {
	if err := r.GenericComponentSpec.Default(ctx); err != nil {
		return err
	}
	if err := r.Ref.Default(ctx); err != nil {
		return err
	}
	if r.ImageRef == (registriesv1alpha1.ImageReference{}) {
		r.ImageRef = registriesv1alpha1.ImageReference{
			Kind: "ClusterImage",
			Name: "wasmtime",
		}
	}
	if err := r.ImageRef.Default(ctx); err != nil {
		return err
	}

	return nil
}

var _ admission.Validator[*ComponentContainerImage] = &ComponentContainerImage{}

func (r *ComponentContainerImage) ValidateCreate(ctx context.Context, obj *ComponentContainerImage) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ComponentContainerImage) ValidateUpdate(ctx context.Context, oldObj, newObj *ComponentContainerImage) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ComponentContainerImage) ValidateDelete(ctx context.Context, obj *ComponentContainerImage) (warnings admission.Warnings, err error) {
	return
}

func (r *ComponentContainerImage) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ComponentContainerImageSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericComponentSpec.Validate(ctx, fldPath)...)
	errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	errs = append(errs, r.ImageRef.Validate(ctx, fldPath.Child("imageRef"))...)

	return errs
}

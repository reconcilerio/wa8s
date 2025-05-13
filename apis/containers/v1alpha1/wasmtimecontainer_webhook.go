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

	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/validation"
)

//+kubebuilder:webhook:path=/validate-containers-wa8s-reconciler-io-v1alpha1-wasmtimecontainer,mutating=false,failurePolicy=fail,sideEffects=None,groups=containers.wa8s.reconciler.io,resources=wasmtimecontainers,verbs=create;update,versions=v1alpha1,name=v1alpha1.wasmtimecontainers.containers.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *WasmtimeContainer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &WasmtimeContainer{}

func (r *WasmtimeContainer) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*WasmtimeContainer)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *WasmtimeContainerSpec) Default(ctx context.Context) error {
	if err := r.GenericComponentSpec.Default(ctx); err != nil {
		return err
	}
	if r.BaseImage == "" {
		// TODO sustainably build this image, for now check hack/wasmtime
		r.BaseImage = "ghcr.io/reconcilerio/wa8s/wasmtime:30.0.0@sha256:eb367f58c270307812abe30c62d4248a28cd1368a27fe4a7db201379a13c0f7a"
		// r.BaseImage = "ghcr.io/reconcilerio/wa8s/wasmtime:32.0.0@sha256:044ec10c535e78a93a51bbfc467eab5ec287ac0759fd16d771bc223edaac5dbb"
	}
	if err := r.Ref.Default(ctx); err != nil {
		return err
	}
	if err := r.ServiceAccountRef.Default(ctx); err != nil {
		return err
	}

	return nil
}

var _ webhook.CustomValidator = &WasmtimeContainer{}

func (r *WasmtimeContainer) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*WasmtimeContainer)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *WasmtimeContainer) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*WasmtimeContainer)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *WasmtimeContainer) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *WasmtimeContainer) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *WasmtimeContainerSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericComponentSpec.Validate(ctx, fldPath)...)
	if r.BaseImage == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("baseImage"), ""))
	}
	errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	errs = append(errs, r.ServiceAccountRef.Validate(ctx, fldPath.Child("serviceAccountRef"))...)

	return errs
}

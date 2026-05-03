/*
Copyright 2026.

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

	"k8s.io/apimachinery/pkg/util/validation/field"
	"reconciler.io/runtime/reconcilers"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/apis"
	"reconciler.io/wa8s/internal/defaults"
	"reconciler.io/wa8s/validation"
)

//+kubebuilder:webhook:path=/validate-registries-wa8s-reconciler-io-v1alpha1-image,mutating=false,failurePolicy=fail,sideEffects=None,groups=registries.wa8s.reconciler.io,resources=images,verbs=create;update,versions=v1alpha1,name=v1alpha1.images.registries.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1},serviceName=wa8s-manager-webhook
//+kubebuilder:webhook:path=/validate-registries-wa8s-reconciler-io-v1alpha1-clusterimage,mutating=false,failurePolicy=fail,sideEffects=None,groups=registries.wa8s.reconciler.io,resources=clusterimages,verbs=create;update,versions=v1alpha1,name=v1alpha1.clusterimages.registries.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1},serviceName=wa8s-manager-webhook

func (r *Image) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

func (r *ClusterImage) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

var _ reconcilers.Defaulter = &Image{}
var _ reconcilers.Defaulter = &ClusterImage{}

func (r *Image) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ClusterImage) Default(ctx context.Context) error {
	ctx = validation.StashResource(ctx, r)

	if r.Spec.ServiceAccountRef.Namespace == "" {
		r.Spec.ServiceAccountRef.Namespace = defaults.Namespace()
	}
	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ImageSpec) Default(ctx context.Context) error {
	if err := r.RepositoryRef.Default(ctx); err != nil {
		return err
	}
	if err := r.ServiceAccountRef.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ImageReference) Default(ctx context.Context) error {
	if r.Kind == "" {
		r.Kind = "Image"
	}
	if r.Namespace == "" && !strings.HasPrefix(r.Kind, "Cluster") {
		r.Namespace = validation.RetrieveResource(ctx).GetNamespace()
	}
	if r.Name == "" {
		r.Name = validation.RetrieveResource(ctx).GetName()
	}

	return nil
}

var _ admission.Validator[*Image] = &Image{}
var _ admission.Validator[*ClusterImage] = &ClusterImage{}

func (r *Image) ValidateCreate(ctx context.Context, obj *Image) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Image) ValidateUpdate(ctx context.Context, oldObj, newObj *Image) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Image) ValidateDelete(ctx context.Context, obj *Image) (warnings admission.Warnings, err error) {
	return
}

func (r *Image) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ClusterImage) ValidateCreate(ctx context.Context, obj *ClusterImage) (warnings admission.Warnings, err error) {
	if err := obj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, obj)

	return nil, obj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterImage) ValidateUpdate(ctx context.Context, oldObj, newObj *ClusterImage) (warnings admission.Warnings, err error) {
	if err := newObj.Default(ctx); err != nil {
		return nil, err
	}
	ctx = validation.StashResource(ctx, newObj)

	return nil, newObj.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterImage) ValidateDelete(ctx context.Context, obj *ClusterImage) (warnings admission.Warnings, err error) {
	return
}

func (r *ClusterImage) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, apis.ValidateCommonAnnotations(ctx, fldPath, r)...)
	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ImageSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.RepositoryRef.Validate(ctx, fldPath.Child("repositoryRef"))...)
	errs = append(errs, r.ServiceAccountRef.Validate(ctx, fldPath.Child("serviceAccountRef"))...)

	return errs
}

func (r *ImageReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Kind == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("kind"), ""))
	} else if r.Kind != "Image" && r.Kind != "ClusterImage" {
		errs = append(errs, field.Invalid(fldPath.Child("kind"), r.Kind, "must be one of Image or ClusterImage"))

	}
	if !strings.HasPrefix(r.Kind, "Cluster") {
		// TODO use a more robust checked for cluster scoped resources
		if r.Namespace == "" {
			// defaulted
			errs = append(errs, field.Required(fldPath.Child("namespace"), ""))
		} else if ns := validation.RetrieveResource(ctx).GetNamespace(); ns != "" && ns != r.Namespace && ns != defaults.Namespace() {
			errs = append(errs, field.Invalid(fldPath.Child("namespace"), r.Namespace, "cross namespace images are not allowed"))
		}
	}
	if r.Name == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

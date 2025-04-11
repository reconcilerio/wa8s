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

	"reconciler.io/wa8s/validation"
)

// +kubebuilder:webhook:path=/validate-registries-wa8s-reconciler-io-v1alpha1-repository,mutating=false,failurePolicy=fail,sideEffects=None,groups=registries.wa8s.reconciler.io,resources=repositories,verbs=create;update,versions=v1alpha1,name=v1alpha1.repositories.registries.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}
// +kubebuilder:webhook:path=/validate-registries-wa8s-reconciler-io-v1alpha1-clusterrepository,mutating=false,failurePolicy=fail,sideEffects=None,groups=registries.wa8s.reconciler.io,resources=clusterrepositories,verbs=create;update,versions=v1alpha1,name=v1alpha1.clusterrepositories.registries.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *Repository) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

func (r *ClusterRepository) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &Repository{}
var _ webhook.CustomDefaulter = &ClusterRepository{}

func (r *Repository) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*Repository)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ClusterRepository) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*ClusterRepository)
	ctx = validation.StashResource(ctx, r)

	if r.Spec.ServiceAccountRef.Namespace == "" {
		// TODO parameterize
		r.Spec.ServiceAccountRef.Namespace = "wa8s-system"
	}
	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}
func (r *RepositorySpec) Default(ctx context.Context) error {
	if err := r.ServiceAccountRef.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ServiceAccountReference) Default(ctx context.Context) error {
	if r.Namespace == "" {
		r.Namespace = validation.RetrieveResource(ctx).GetNamespace()
	}
	if r.Name == "" {
		r.Name = "default"
	}

	return nil
}

func (r *RepositoryReference) Default(ctx context.Context) error {
	if (*r == RepositoryReference{}) {
		r.Kind = "ClusterRepository"
		r.Name = "default"
	}

	if r.Kind != "ClusterRepository" {
		r.Kind = "Repository"
	}
	if r.Name == "" {
		r.Name = "default"
	}

	return nil
}

func (r *RepositoryReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Kind == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("kind"), ""))
	}
	if r.Kind != "Repository" && r.Kind != "ClusterRepository" {
		errs = append(errs, field.Invalid(fldPath.Child("kind"), r.Kind, "allowed values are Repository or ClusterRepository"))
	}

	if r.Name == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

var _ webhook.CustomValidator = &Repository{}
var _ webhook.CustomValidator = &ClusterRepository{}

func (r *Repository) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*Repository)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Repository) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*Repository)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Repository) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *Repository) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ClusterRepository) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*ClusterRepository)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterRepository) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*ClusterRepository)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterRepository) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *ClusterRepository) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *RepositorySpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Template == "" {
		errs = append(errs, field.Required(fldPath.Child("template"), ""))
	}
	errs = append(errs, r.ServiceAccountRef.Validate(ctx, fldPath.Child("serviceAccountRef"))...)

	return errs
}

func (r *ServiceAccountReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Namespace == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("namespace"), ""))
	} else if ns := validation.RetrieveResource(ctx).GetNamespace(); ns != "" && r.Namespace != ns {
		errs = append(errs, field.Invalid(fldPath.Child("namespace"), r.Namespace, "cross namespace service accounts are not allowed"))
	}
	if r.Name == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

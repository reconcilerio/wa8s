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
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"reconciler.io/wa8s/validation"
)

//+kubebuilder:webhook:path=/validate-wa8s-reconciler-io-v1alpha1-component,mutating=false,failurePolicy=fail,sideEffects=None,groups=wa8s.reconciler.io,resources=components,verbs=create;update,versions=v1alpha1,name=v1alpha1.components.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}
//+kubebuilder:webhook:path=/validate-wa8s-reconciler-io-v1alpha1-clustercomponent,mutating=false,failurePolicy=fail,sideEffects=None,groups=wa8s.reconciler.io,resources=clustercomponents,verbs=create;update,versions=v1alpha1,name=v1alpha1.clustercomponents.wa8s.reconciler.io,admissionReviewVersions={v1,v1beta1}

func (r *Component) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

func (r *ClusterComponent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

var _ webhook.CustomDefaulter = &Component{}
var _ webhook.CustomDefaulter = &ClusterComponent{}

func (r *Component) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*Component)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ClusterComponent) Default(ctx context.Context, obj runtime.Object) error {
	r = obj.(*ClusterComponent)
	ctx = validation.StashResource(ctx, r)

	if err := r.Spec.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *ComponentSpec) Default(ctx context.Context) error {
	if err := r.GenericComponentSpec.Default(ctx); err != nil {
		return err
	}
	if r.OCI != nil {
		if err := r.OCI.Default(ctx); err != nil {
			return err
		}
	}
	if r.Ref != nil {
		if err := r.Ref.Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *ComponentReference) Default(ctx context.Context) error {
	if r.Kind == "" {
		r.Kind = "Component"
	}
	if r.APIVersion == "" {
		r.APIVersion = validation.DefaultApiVersionForKind(r.Kind)
	}
	if r.Namespace == "" && !strings.HasPrefix(r.Kind, "Cluster") {
		r.Namespace = validation.RetrieveResource(ctx).GetNamespace()
	}
	if r.Name == "" {
		r.Name = validation.RetrieveResource(ctx).GetName()
	}

	return nil
}

func (r *OCIReference) Default(ctx context.Context) error {
	if err := r.ServiceAccountRef.Default(ctx); err != nil {
		return err
	}

	return nil
}

var _ webhook.CustomValidator = &Component{}
var _ webhook.CustomValidator = &ClusterComponent{}

func (r *Component) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*Component)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Component) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*Component)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *Component) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *Component) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ClusterComponent) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, obj); err != nil {
		return nil, err
	}
	r = obj.(*ClusterComponent)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterComponent) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	if err := r.Default(ctx, newObj); err != nil {
		return nil, err
	}
	r = newObj.(*ClusterComponent)
	ctx = validation.StashResource(ctx, r)

	return nil, r.Validate(ctx, field.NewPath("")).ToAggregate()
}

func (r *ClusterComponent) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	return
}

func (r *ClusterComponent) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.Spec.Validate(ctx, fldPath.Child("spec"))...)

	return errs
}

func (r *ComponentSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericComponentSpec.Validate(ctx, fldPath)...)

	picked := sets.New[string]()
	notPicked := sets.New[string]()
	if r.OCI != nil {
		picked.Insert("oci")
		errs = append(errs, r.OCI.Validate(ctx, fldPath.Child("oci"))...)
	} else {
		notPicked.Insert("oci")
	}
	if r.Ref != nil {
		picked.Insert("ref")
		errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	} else {
		notPicked.Insert("ref")
	}
	if picked.Len() == 0 {
		errs = append(errs, field.Required(fldPath.Child(fmt.Sprintf("[%s]", strings.Join(sets.List(notPicked), ", "))), "pick one"))
	}
	if picked.Len() > 1 {
		errs = append(errs, field.Invalid(fldPath.Child(fmt.Sprintf("[%s]", strings.Join(sets.List(picked), ", "))), nil, "pick one"))
	}

	return errs
}

func (r *OCIReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Image == "" {
		errs = append(errs, field.Required(fldPath.Child("image"), ""))
	}
	errs = append(errs, r.ServiceAccountRef.Validate(ctx, fldPath.Child("serviceAccountRef"))...)

	return errs
}

func (r *ComponentReference) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.APIVersion == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("apiVersion"), ""))
	}
	if r.Kind == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("kind"), ""))
	}
	if !strings.HasPrefix(r.Kind, "Cluster") {
		// TODO use a more robust checked for cluster scoped resources
		if r.Namespace == "" {
			// defaulted
			errs = append(errs, field.Required(fldPath.Child("namespace"), ""))
		} else if ns := validation.RetrieveResource(ctx).GetNamespace(); ns != "" && r.Namespace != ns {
			errs = append(errs, field.Invalid(fldPath.Child("namespace"), r.Namespace, "cross namespace components are not allowed"))
		}
	}
	if r.Name == "" {
		// defaulted
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

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
	registriesv1alpha1 "reconciler.io/wa8s/apis/registries/v1alpha1"
)

func (r *GenericContainerSpec) Default(ctx context.Context) error {
	if (r.RepositoryRef == registriesv1alpha1.RepositoryReference{}) {
		r.RepositoryRef = registriesv1alpha1.RepositoryReference{
			Kind: "ClusterRepository",
			Name: "external",
		}
	}

	if err := r.GenericComponentSpec.Default(ctx); err != nil {
		return err
	}
	if err := r.Ref.Default(ctx); err != nil {
		return err
	}
	if err := r.ServiceAccountRef.Default(ctx); err != nil {
		return err
	}
	if err := r.HostCapabilities.Default(ctx); err != nil {
		return err
	}

	return nil
}

func (r *HostCapabilities) Default(ctx context.Context) error {
	if r.Env != nil {
		if err := r.Env.Default(ctx); err != nil {
			return err
		}
	}
	if r.Config != nil {
		if err := r.Config.Default(ctx); err != nil {
			return err
		}
	}
	if r.Network != nil {
		if err := r.Network.Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *HostEnv) Default(ctx context.Context) error {
	for i := range r.Vars {
		if err := r.Vars[i].Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *HostEnvVar) Default(ctx context.Context) error {
	return nil
}

func (r *HostConfig) Default(ctx context.Context) error {
	for i := range r.Vars {
		if err := r.Vars[i].Default(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *HostConfigVar) Default(ctx context.Context) error {
	return nil
}

func (r *HostNetwork) Default(ctx context.Context) error {
	return nil
}

func (r *GenericContainerSpec) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	errs = append(errs, r.GenericComponentSpec.Validate(ctx, fldPath)...)
	errs = append(errs, r.Ref.Validate(ctx, fldPath.Child("ref"))...)
	errs = append(errs, r.ServiceAccountRef.Validate(ctx, fldPath.Child("serviceAccountRef"))...)
	errs = append(errs, r.HostCapabilities.Validate(ctx, fldPath.Child("hostCapabilities"))...)

	return errs
}

func (r *HostCapabilities) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Env != nil {
		errs = append(errs, r.Env.Validate(ctx, fldPath.Child("env"))...)
	}
	if r.Config != nil {
		errs = append(errs, r.Config.Validate(ctx, fldPath.Child("config"))...)
	}
	if r.Network != nil {
		errs = append(errs, r.Network.Validate(ctx, fldPath.Child("network"))...)
	}

	return errs
}

func (r *HostEnv) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	for i := range r.Vars {
		errs = append(errs, r.Vars[i].Validate(ctx, fldPath.Child("vars").Index(i))...)
	}

	return errs
}

func (r *HostEnvVar) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

func (r *HostConfig) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	for i := range r.Vars {
		errs = append(errs, r.Vars[i].Validate(ctx, fldPath.Child("vars").Index(i))...)

	}

	return errs
}

func (r *HostConfigVar) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if r.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("name"), ""))
	}

	return errs
}

func (r *HostNetwork) Validate(ctx context.Context, fldPath *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	return errs
}

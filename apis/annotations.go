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

package apis

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	CreateChildComponentAnnotation = "wa8s.reconciler.io/create-child-component"
	CreateChildComponentTrue       = "true"
	CreateChildComponentFalse      = "false"
	CreateChildComponentAuto       = "auto"
)

func ValidateCommonAnnotations(ctx context.Context, fldPath *field.Path, resource runtime.Object) field.ErrorList {
	errs := field.ErrorList{}

	annotations := resource.(metav1.Object).GetAnnotations()
	if annotations == nil {
		return errs
	}

	if createChild, ok := annotations[CreateChildComponentAnnotation]; ok {
		if createChild != CreateChildComponentTrue && createChild != CreateChildComponentFalse && createChild != CreateChildComponentAuto {
			errs = append(errs, field.Invalid(
				fldPath.Child("metadata", "annotations", CreateChildComponentAnnotation), createChild,
				fmt.Sprintf("must be one of: %s, %s, %s", CreateChildComponentTrue, CreateChildComponentFalse, CreateChildComponentAnnotation)),
			)
		}
	}

	return errs
}

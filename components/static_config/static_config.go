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

package static_config

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/samyfodil/wazy"
	"github.com/samyfodil/wazy/component"
)

var runtime wazy.Runtime
var compileCache *component.CompileCache

func init() {
	runtime = wazy.NewRuntime(context.Background())
	compileCache = component.NewCompileCache()
}

//go:generate go tool wazy-bindgen -wit static-config.wit -world root -package static_config -o ./zz_generated.wazy_bindgen.go

//go:embed static-config.wasm
var staticConfigWasm []byte

func Component(ctx context.Context) (*Root, error) {
	return Instantiate(ctx, runtime, staticConfigWasm, component.WithCompileCache(compileCache))
}

func FormatErrorCode(err ErrorCode) string {
	switch err.Case {
	case ErrorCodeCaseMalformed:
		malformed, _ := err.Malformed()
		if malformed.Name != nil {
			return fmt.Sprintf("malformed %s: %v", *malformed.Name, malformed.Message)
		}
		return fmt.Sprintf("malformed: %v", malformed.Message)
	case ErrorCodeCaseNotComponent:
		notComponent, _ := err.NotComponent()
		return formatOptionalMessage("not component", notComponent)
	case ErrorCodeCaseNotFound:
		notFound, _ := err.NotFound()
		return formatOptionalMessage("not found", notFound)
	case ErrorCodeCaseOther:
		other, _ := err.Other()
		return formatOptionalMessage("other", other)
	default:
		panic(fmt.Errorf("unknown error code %T", err.Case))
	}
}

func formatOptionalMessage(kind string, message *string) string {
	if message == nil {
		return kind
	}
	return fmt.Sprintf("%s: %v", kind, *message)
}

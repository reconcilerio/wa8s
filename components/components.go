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

package components

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/samyfodil/wazy"
	"github.com/samyfodil/wazy/component"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
)

var runtime wazy.Runtime
var compileCache *component.CompileCache

func init() {
	runtime = wazy.NewRuntime(context.Background())
	compileCache = component.NewCompileCache()
}

//go:embed wit-tools.wasm
var witToolsWasm []byte

func ExtractWIT(ctx context.Context, bytes []byte) (imports []string, exports []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling ExtractWIT: %s", r)
		}
	}()

	var inst *component.Instance
	// inst, err = component.Instantiate(ctx, runtime, witToolsWasm, component.WithCompileCache(compileCache))
	inst, err = component.Instantiate(ctx, runtime, witToolsWasm)
	if err != nil {
		return nil, nil, fmt.Errorf("instantiate wit-tools: %s", err)
	}
	defer func() {
		if closeErr := inst.Close(ctx); err == nil {
			err = closeErr
		}
	}()

	var got []component.Value
	// wit-source variant, case 0 is wasm(wasm)
	source := component.VariantValue{Disc: 0, Payload: bytes}
	got, err = inst.CallExport(ctx, "componentized:component/wit@0.1.0", "summarize", source)
	if err != nil {
		return nil, nil, fmt.Errorf("call wit-tools summarize: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, nil, fmt.Errorf("call wit-tools summarize: %s", formatErrorCode(result.Payload))
	}

	// summary record {imports: list<string>, exports: list<string>}
	summary := result.Payload.([]component.Value)
	if imports, err = component.ListOf[string](summary[0]); err != nil {
		return nil, nil, fmt.Errorf("call wit-tools summarize: %w", err)
	}
	if exports, err = component.ListOf[string](summary[1]); err != nil {
		return nil, nil, fmt.Errorf("call wit-tools summarize: %w", err)
	}

	return imports, exports, nil
}

//go:embed static-config.wasm
var staticConfigWasm []byte

func ComponentizeConfigStore(ctx context.Context, config map[string]string) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling ComponentizeConfigStore: %s", r)
		}
	}()

	var inst *component.Instance
	inst, err = component.Instantiate(ctx, runtime, staticConfigWasm, component.WithCompileCache(compileCache))
	if err != nil {
		return nil, fmt.Errorf("instantiate static-config: %s", err)
	}
	defer func() {
		if closeErr := inst.Close(ctx); err == nil {
			err = closeErr
		}
	}()

	values := []component.Value{}
	for key, value := range config {
		values = append(values, []component.Value{key, value})
	}

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:static-config/factory@0.3.0", "create", values)
	if err != nil {
		return nil, fmt.Errorf("call static-config create: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, fmt.Errorf("call static-config create: %s", formatErrorCode(result.Payload))
	}
	return component.ListOf[byte](result.Payload)
}

//go:embed wac.wasm
var wacWasm []byte

type CompositionDependency struct {
	Name      string
	Image     name.Digest
	Component []byte
	WIT       componentsv1alpha1.WIT
}

func WACCompose(ctx context.Context, wac string, dependencies []CompositionDependency) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling WACCompose: %s", r)
		}
	}()

	var inst *component.Instance
	inst, err = component.Instantiate(ctx, runtime, wacWasm, component.WithCompileCache(compileCache))
	if err != nil {
		return nil, fmt.Errorf("instantiate wac-loader: %s", err)
	}
	defer func() {
		if closeErr := inst.Close(ctx); err == nil {
			err = closeErr
		}
	}()

	plan := component.VariantValue{Disc: 0, Payload: wac}
	deps := []component.Value{}
	for _, dep := range dependencies {
		deps = append(deps, []component.Value{dep.Name, dep.Component})
	}

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:component/wac-loader@0.1.0", "compose", plan, deps)
	if err != nil {
		return nil, fmt.Errorf("call wac-loader compose: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, fmt.Errorf("call wac-loader compose: %s", formatErrorCode(result.Payload))
	}
	return component.ListOf[byte](result.Payload)
}

func WACPlug(ctx context.Context, dependencies []CompositionDependency) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling WACPlug: %s", r)
		}
	}()

	var inst *component.Instance
	inst, err = component.Instantiate(ctx, runtime, wacWasm, component.WithCompileCache(compileCache))
	if err != nil {
		return nil, fmt.Errorf("instantiate wac-loader: %s", err)
	}
	defer func() {
		if closeErr := inst.Close(ctx); err == nil {
			err = closeErr
		}
	}()

	socket := dependencies[0].Component
	plugs := []component.Value{}
	for _, plug := range dependencies[1:] {
		plugs = append(plugs, plug.Component)
	}

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:component/wac-loader@0.1.0", "plug", socket, plugs)
	if err != nil {
		return nil, fmt.Errorf("call wac-loader plug: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, fmt.Errorf("call wac-loader plug: %s", formatErrorCode(result.Payload))
	}
	return component.ListOf[byte](result.Payload)
}

// formatErrorCode renders a componentized:component/types@0.1.0 error-code variant
func formatErrorCode(v component.Value) string {
	code, ok := v.(component.VariantValue)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	switch code.Disc {
	case 0:
		// malformed(record {name: option<string>, message: string})
		malformed, ok := code.Payload.([]component.Value)
		if !ok || len(malformed) != 2 {
			return fmt.Sprintf("malformed: %v", code.Payload)
		}
		if name, ok := malformed[0].(string); ok {
			return fmt.Sprintf("malformed %s: %v", name, malformed[1])
		}
		return fmt.Sprintf("malformed: %v", malformed[1])
	case 1:
		return formatOptionalMessage("not component", code.Payload)
	case 2:
		return formatOptionalMessage("not found", code.Payload)
	case 3:
		return formatOptionalMessage("other", code.Payload)
	default:
		return fmt.Sprintf("unknown error code %d: %v", code.Disc, code.Payload)
	}
}

func formatOptionalMessage(kind string, message component.Value) string {
	if message == nil {
		return kind
	}
	return fmt.Sprintf("%s: %v", kind, message)
}

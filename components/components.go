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
		err = inst.Close(ctx)
	}()

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:component/wit@0.0.0-0", "summarize-world", bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("call wit-tools summarize-world: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, nil, fmt.Errorf("call wit-tools summarize-world: %s", result.Payload)
	}

	summary := result.Payload.([]component.Value)
	for _, i := range summary[0].([]component.Value) {
		imports = append(imports, i.(string))
	}
	for _, e := range summary[1].([]component.Value) {
		exports = append(exports, e.(string))
	}

	return
}

//go:embed static-config.wasm
var staticConfigWasm []byte

func ComponentizeConfigStore(ctx context.Context, config map[string]string) (_ []byte, err error) {
	var inst *component.Instance
	inst, err = component.Instantiate(ctx, runtime, staticConfigWasm, component.WithCompileCache(compileCache))
	if err != nil {
		panic(fmt.Errorf("instantiate static-config: %s", err))
	}
	defer func() {
		err = inst.Close(ctx)
	}()

	values := []component.Value{}
	for key, value := range config {
		values = append(values, []component.Value{key, value})
	}

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:config/factory", "build-component", values)
	if err != nil {
		panic(fmt.Errorf("call static-config: %w", err))
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, fmt.Errorf("call static-config: %s", result.Payload)
	}
	return result.Payload.([]byte), nil
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
		err = inst.Close(ctx)
	}()

	plan := component.VariantValue{Disc: 0, Payload: wac}
	deps := []component.Value{}
	for _, dep := range dependencies {
		deps = append(deps, []component.Value{dep.Name, dep.Component})
	}

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:component/wac-loader@0.0.0-0", "compose", plan, deps)
	if err != nil {
		return nil, fmt.Errorf("call wac-loader compose: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		return nil, fmt.Errorf("call wac-loader compose: %s", result.Payload)
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
		err = inst.Close(ctx)
	}()

	socket := dependencies[0].Component
	plugs := []component.Value{}
	for _, plug := range dependencies[1:] {
		plugs = append(plugs, plug.Component)
	}

	var got []component.Value
	got, err = inst.CallExport(ctx, "componentized:component/wac-loader@0.0.0-0", "plug", socket, plugs)
	if err != nil {
		return nil, fmt.Errorf("call wac-loader plug: %w", err)
	}
	result := got[0].(component.ResultValue)
	if result.IsErr {
		err = fmt.Errorf("call wac-loader plug: %s", result.Payload)
		return nil, err
	}
	return component.ListOf[byte](result.Payload)
}

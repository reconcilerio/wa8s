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
	"encoding/json"
	"fmt"
	"sort"

	extism "github.com/extism/go-sdk"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tetratelabs/wazero"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
)

//go:embed wit-tools.wasm
var witToolsWasm []byte
var witToolsPlugin *extism.Plugin

func ExtractWIT(ctx context.Context, component []byte) (_ string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling ExtractWIT: %s", r)
		}
	}()

	_, out, err := witToolsPlugin.CallWithContext(ctx, "extract", component)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

//go:embed static-config.wasm
var staticConfigWasm []byte
var staticConfigPlugin *extism.Plugin

func ComponentizeConfigStore(ctx context.Context, config map[string]string) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling ComponentizeConfigStore: %s", r)
		}
	}()

	c := [][]string{}

	for k, v := range config {
		c = append(c, []string{k, v})
	}
	sort.Slice(c, func(i, j int) bool {
		return c[i][0] < c[j][0]
	})

	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	_, component, err := staticConfigPlugin.CallWithContext(ctx, "build_component", bytes)
	if err != nil {
		return nil, err
	}

	return component, nil
}

//go:embed wac.wasm
var wacWasm []byte
var wacPlugin *extism.Plugin

type ResolvedComponent struct {
	Name      string
	Image     name.Digest
	Component []byte
	WIT       componentsv1alpha1.WIT
}

func WACCompose(ctx context.Context, wac string, dependencies []ResolvedComponent) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling WACCompose: %s", r)
		}
	}()

	type WACDependency struct {
		Name      string `json:"name"`
		Component []byte `json:"component"`
	}
	type WAC struct {
		Script       string          `json:"script"`
		Dependencies []WACDependency `json:"dependencies"`
	}

	input := WAC{
		Script:       wac,
		Dependencies: []WACDependency{},
	}
	for _, dependency := range dependencies {
		input.Dependencies = append(input.Dependencies, WACDependency{
			Name:      dependency.Name,
			Component: dependency.Component,
		})
	}

	inputJson, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	_, component, err := wacPlugin.CallWithContext(ctx, "compose", inputJson)
	if err != nil {
		return nil, err
	}

	return component, nil
}

func WACPlug(ctx context.Context, dependencies []ResolvedComponent) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic calling WACPlug: %s", r)
		}
	}()

	type WACDependency struct {
		Name      string `json:"name"`
		Component []byte `json:"component"`
	}
	type WAC struct {
		Script       string          `json:"script"`
		Dependencies []WACDependency `json:"dependencies"`
	}

	input := WAC{
		Script:       "",
		Dependencies: []WACDependency{},
	}
	for _, dependency := range dependencies {
		input.Dependencies = append(input.Dependencies, WACDependency{
			Name:      dependency.Name,
			Component: dependency.Component,
		})
	}

	inputJson, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	_, component, err := wacPlugin.CallWithContext(ctx, "plug", inputJson)
	if err != nil {
		return nil, err
	}

	return component, nil
}

func bootstrapPlugin(wasm []byte, name string) (*extism.Plugin, error) {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmData{
				Data: wasm,
				Name: name,
			},
		},
	}

	config := extism.PluginConfig{
		// EnableWasi:    true,
		RuntimeConfig: wazero.NewRuntimeConfig().WithCloseOnContextDone(true),
	}
	plugin, err := extism.NewPlugin(context.Background(), manifest, config, []extism.HostFunction{})
	if err != nil {
		return nil, err
	}
	return plugin, nil
}

func init() {
	plugin, err := bootstrapPlugin(witToolsWasm, "wit-tools.wasm")
	if err != nil {
		panic(err)
	}
	witToolsPlugin = plugin

	plugin, err = bootstrapPlugin(staticConfigWasm, "static-config.wasm")
	if err != nil {
		panic(err)
	}
	staticConfigPlugin = plugin

	plugin, err = bootstrapPlugin(wacWasm, "wac.wasm")
	if err != nil {
		panic(err)
	}
	wacPlugin = plugin
}

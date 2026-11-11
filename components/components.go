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
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/samyfodil/wazy/component/witgo"

	componentsv1alpha1 "reconciler.io/wa8s/apis/components/v1alpha1"
	"reconciler.io/wa8s/components/static_config"
	"reconciler.io/wa8s/components/wac"
	"reconciler.io/wa8s/components/wit_tools"
)

func ExtractWIT(ctx context.Context, bytes []byte) (imports []string, exports []string, err error) {
	component, err := wit_tools.Component(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("instantiate wit-tools: %s", err)
	}
	defer func() {
		err = errors.Join(err, component.Close(ctx))
	}()

	source := wit_tools.WitSourceWasm(bytes)

	var result witgo.Result[wit_tools.Summary, wit_tools.ErrorCode]
	result, err = component.Wit2().Summarize(ctx, source)
	if err != nil {
		return nil, nil, fmt.Errorf("call wit-tools summarize: %w", err)
	}
	if result.IsErr {
		return nil, nil, fmt.Errorf("call wit-tools summarize: %s", wit_tools.FormatErrorCode(result.Err))
	}
	summary := result.Ok
	return summary.Imports, summary.Exports, nil
}

func ComponentizeConfigStore(ctx context.Context, config map[string]string) (_ []byte, err error) {
	component, err := static_config.Component(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiate static-config: %s", err)
	}
	defer func() {
		err = errors.Join(err, component.Close(ctx))
	}()

	values := []witgo.Tuple2[string, string]{}
	for key, value := range config {
		values = append(values, witgo.Tuple2[string, string]{F0: key, F1: value})
	}

	result, err := component.Factory().Create(ctx, values)
	if err != nil {
		return nil, fmt.Errorf("call static-config create: %w", err)
	}
	if result.IsErr {
		return nil, fmt.Errorf("call static-config create: %s", static_config.FormatErrorCode(result.Err))
	}
	return result.Ok, nil
}

type CompositionDependency struct {
	Name      string
	Image     name.Digest
	Component []byte
	WIT       componentsv1alpha1.WIT
}

func WACCompose(ctx context.Context, wac_script string, dependencies []CompositionDependency) (_ []byte, err error) {
	component, err := wac.Component(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiate wac-loader: %s", err)
	}
	defer func() {
		err = errors.Join(err, component.Close(ctx))
	}()

	plan := wac.PlanWac(wac_script)
	deps := []wac.Dependency{}
	for _, dep := range dependencies {
		deps = append(deps, wac.Dependency{F0: dep.Name, F1: dep.Component})
	}

	var result witgo.Result[[]byte, wac.ErrorCode]
	result, err = component.WacLoader().Compose(ctx, plan, deps)
	if err != nil {
		return nil, fmt.Errorf("call wac-loader compose: %w", err)
	}
	if result.IsErr {
		return nil, fmt.Errorf("call wac-loader compose: %s", wac.FormatErrorCode(result.Err))
	}
	return result.Ok, nil
}

func WACPlug(ctx context.Context, dependencies []CompositionDependency) (_ []byte, err error) {
	component, err := wac.Component(ctx)
	if err != nil {
		return nil, fmt.Errorf("instantiate wac-loader: %s", err)
	}
	defer func() {
		err = errors.Join(err, component.Close(ctx))
	}()

	socket := dependencies[0].Component
	plugs := [][]byte{}
	for _, plug := range dependencies[1:] {
		plugs = append(plugs, plug.Component)
	}

	var result witgo.Result[[]byte, wac.ErrorCode]
	result, err = component.WacLoader().Plug(ctx, socket, plugs)
	if err != nil {
		return nil, fmt.Errorf("call wac-loader plug: %w", err)
	}

	if result.IsErr {
		return nil, fmt.Errorf("call wac-loader plug: %s", wac.FormatErrorCode(result.Err))
	}
	return result.Ok, nil
}

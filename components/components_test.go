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

package components_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"reconciler.io/wa8s/components"
)

// socket.wasm imports wasi:config/store and exports test:socket/greeter. Regenerate with:
//
//	wasm-tools component embed --dummy testdata/socket.wit -w socket | wasm-tools component new -o testdata/socket.wasm
//
//go:embed testdata/socket.wasm
var socketWasm []byte

//go:embed wit-tools.wasm
var witToolsWasm []byte

//go:embed wac.wasm
var wacWasm []byte

//go:embed static-config.wasm
var staticConfigWasm []byte

const configStoreInterface = "wasi:config/store@0.2.0-rc.1"

func TestExtractWIT(t *testing.T) {
	tests := []struct {
		name            string
		component       []byte
		expectedImports []string
		expectedExports []string
		expectedErr     string
	}{
		{
			name:            "wit-tools",
			component:       witToolsWasm,
			expectedImports: []string{"componentized:component/types@0.1.0"},
			expectedExports: []string{"componentized:component/wit@0.1.0"},
		},
		{
			name:            "wac-loader",
			component:       wacWasm,
			expectedImports: []string{"componentized:component/types@0.1.0"},
			expectedExports: []string{"componentized:component/wac-loader@0.1.0"},
		},
		{
			name:            "static-config",
			component:       staticConfigWasm,
			expectedImports: []string{"componentized:component/types@0.1.0"},
			expectedExports: []string{"componentized:static-config/factory@0.3.0"},
		},
		{
			name:            "socket",
			component:       socketWasm,
			expectedImports: []string{configStoreInterface},
			expectedExports: []string{"test:socket/greeter"},
		},
		{
			name:        "not a component",
			component:   []byte("not a component"),
			expectedErr: "call wit-tools summarize: not component",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			imports, exports, err := components.ExtractWIT(t.Context(), tc.component)
			if !assertErr(t, err, tc.expectedErr) {
				return
			}
			if diff := cmp.Diff(tc.expectedImports, imports, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("unexpected imports (-expected, +actual): %s", diff)
			}
			if diff := cmp.Diff(tc.expectedExports, exports, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("unexpected exports (-expected, +actual): %s", diff)
			}
		})
	}
}

func TestComponentizeConfigStore(t *testing.T) {
	ctx := t.Context()

	configStore, err := components.ComponentizeConfigStore(ctx, map[string]string{
		"greeting": "hello",
		"name":     "world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	assertWIT(t, configStore, nil, []string{configStoreInterface})
}

func TestWACPlug(t *testing.T) {
	ctx := t.Context()

	configStore, err := components.ComponentizeConfigStore(ctx, map[string]string{"greeting": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	tests := []struct {
		name            string
		dependencies    []components.CompositionDependency
		expectedImports []string
		expectedExports []string
		expectedErr     string
	}{
		{
			name: "plug config store into socket",
			dependencies: []components.CompositionDependency{
				{Name: "test:socket", Component: socketWasm},
				{Name: "test:config", Component: configStore},
			},
			expectedImports: nil,
			expectedExports: []string{"test:socket/greeter"},
		},
		{
			name: "plug does not match socket",
			dependencies: []components.CompositionDependency{
				{Name: "test:config", Component: configStore},
				{Name: "test:socket", Component: socketWasm},
			},
			expectedErr: "call wac-loader plug: other: the socket component had no matching imports",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			composed, err := components.WACPlug(ctx, tc.dependencies)
			if !assertErr(t, err, tc.expectedErr) {
				return
			}
			assertWIT(t, composed, tc.expectedImports, tc.expectedExports)
		})
	}
}

func TestWACCompose(t *testing.T) {
	ctx := t.Context()

	configStore, err := components.ComponentizeConfigStore(ctx, map[string]string{"greeting": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	dependencies := []components.CompositionDependency{
		{Name: "test:socket", Component: socketWasm},
		{Name: "test:config", Component: configStore},
	}

	tests := []struct {
		name            string
		wac             string
		expectedImports []string
		expectedExports []string
		expectedErr     string
	}{
		{
			name: "compose socket with config store",
			wac: `package test:composition;

let config = new test:config {};
let socket = new test:socket { "wasi:config/store@0.2.0-rc.1": config["wasi:config/store@0.2.0-rc.1"] };

export socket["test:socket/greeter"];
`,
			expectedImports: nil,
			expectedExports: []string{"test:socket/greeter"},
		},
		{
			name: "missing dependency",
			wac: `package test:composition;

let missing = new test:missing {};
`,
			expectedErr: "call wac-loader compose: not found: test:missing",
		},
		{
			name:        "malformed wac",
			wac:         "not wac",
			expectedErr: "call wac-loader compose: malformed: malformed wac script",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			composed, err := components.WACCompose(ctx, tc.wac, dependencies)
			if !assertErr(t, err, tc.expectedErr) {
				return
			}
			assertWIT(t, composed, tc.expectedImports, tc.expectedExports)
		})
	}
}

// assertErr checks err against the expected error substring, returning true when the test should continue
func assertErr(t *testing.T, err error, expectedErr string) bool {
	t.Helper()
	if expectedErr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		return true
	}
	if err == nil {
		t.Fatalf("expected error containing %q", expectedErr)
	}
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected error containing %q, got %q", expectedErr, err)
	}
	return false
}

func assertWIT(t *testing.T, component []byte, expectedImports, expectedExports []string) {
	t.Helper()
	imports, exports, err := components.ExtractWIT(t.Context(), component)
	if err != nil {
		t.Fatalf("unexpected error extracting WIT: %s", err)
	}
	if diff := cmp.Diff(expectedImports, imports, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("unexpected imports (-expected, +actual): %s", diff)
	}
	if diff := cmp.Diff(expectedExports, exports, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("unexpected exports (-expected, +actual): %s", diff)
	}
}

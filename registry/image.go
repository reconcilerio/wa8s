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

package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reconciler.io/runtime/reconcilers"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"reconciler.io/wa8s/wit"
)

const (
	ImageManifestMediaType      types.MediaType = "application/vnd.oci.image.manifest.v1+json"
	WasmManifestConfigMediaType types.MediaType = "application/vnd.wasm.config.v0+json"
	WasmLayerMediaType          types.MediaType = "application/wasm"
	WasmArchitecture                            = "wasm"
	WasmModuleOS                                = "wasip1"
	WasmComponentOS                             = "wasip2"
	Author                                      = "wa8s"
)

func newWasmImage(ctx context.Context, component []byte) (v1.Image, WasmConfigFile, error) {
	w, err := wit.Extract(ctx, component)
	if err != nil {
		return nil, WasmConfigFile{}, err
	}
	imports, exports := wit.ImportsExports(w)
	h, _, err := v1.SHA256(bytes.NewReader(component))
	if err != nil {
		return nil, WasmConfigFile{}, err
	}

	config := WasmConfigFile{
		Created:      v1.Time{Time: time.Unix(1, 0)},
		Author:       Author,
		Architecture: WasmArchitecture,
		OS:           WasmComponentOS,
		LayerDigests: []string{
			h.String(),
		},
		Component: WasmConfigFileComponent{
			Imports: imports,
			Exports: exports,
			Target:  nil,
		},
	}

	return &wasmImage{
		component: component,
		layer:     static.NewLayer(component, WasmLayerMediaType),
		config:    config,
	}, config, nil
}

var _ v1.Image = (*wasmImage)(nil)

type wasmImage struct {
	component []byte
	layer     v1.Layer
	config    WasmConfigFile
}

// ConfigFile implements v1.Image.
func (w *wasmImage) ConfigFile() (*v1.ConfigFile, error) {
	config, err := w.RawConfigFile()
	if err != nil {
		return nil, err
	}
	var configFile v1.ConfigFile
	if err := json.Unmarshal(config, &configFile); err != nil {
		return nil, err
	}
	return &configFile, nil
}

// ConfigName implements v1.Image.
func (w *wasmImage) ConfigName() (v1.Hash, error) {
	config, err := w.RawConfigFile()
	if err != nil {
		return v1.Hash{}, err
	}
	digest, _, err := v1.SHA256(bytes.NewReader(config))
	if err != nil {
		return v1.Hash{}, err
	}
	return digest, nil
}

// Digest implements v1.Image.
func (w *wasmImage) Digest() (v1.Hash, error) {
	manifest, err := w.RawManifest()
	if err != nil {
		return v1.Hash{}, err
	}
	digest, _, err := v1.SHA256(bytes.NewReader(manifest))
	if err != nil {
		return v1.Hash{}, err
	}
	return digest, nil
}

// LayerByDiffID implements v1.Image.
func (w *wasmImage) LayerByDiffID(diffId v1.Hash) (v1.Layer, error) {
	layerDiffId, err := w.layer.DiffID()
	if err != nil {
		return nil, err
	}
	if diffId != layerDiffId {
		return nil, fmt.Errorf("no layer found")
	}
	return w.layer, nil
}

// LayerByDigest implements v1.Image.
func (w *wasmImage) LayerByDigest(digest v1.Hash) (v1.Layer, error) {
	layerDigest, err := w.layer.Digest()
	if err != nil {
		return nil, err
	}
	if digest != layerDigest {
		return nil, fmt.Errorf("no layer found")
	}
	return w.layer, nil
}

// Layers implements v1.Image.
func (w *wasmImage) Layers() ([]v1.Layer, error) {
	return []v1.Layer{w.layer}, nil
}

// Manifest implements v1.Image.
func (w *wasmImage) Manifest() (*v1.Manifest, error) {
	config, err := w.RawConfigFile()
	if err != nil {
		return nil, err
	}
	configDigest, configSize, err := v1.SHA256(bytes.NewReader(config))
	if err != nil {
		return nil, err
	}

	layerDigest, err := w.layer.Digest()
	if err != nil {
		return nil, err
	}
	layerMediaType, err := w.layer.MediaType()
	if err != nil {
		return nil, err
	}
	layerSize, err := w.layer.Size()
	if err != nil {
		return nil, err
	}

	return &v1.Manifest{
		SchemaVersion: 2,
		MediaType:     ImageManifestMediaType,
		Config: v1.Descriptor{
			Digest:    configDigest,
			MediaType: WasmManifestConfigMediaType,
			Size:      configSize,
		},
		Layers: []v1.Descriptor{
			{
				Digest:    layerDigest,
				MediaType: layerMediaType,
				Size:      layerSize,
			},
		},
	}, nil
}

// MediaType implements v1.Image.
func (w *wasmImage) MediaType() (types.MediaType, error) {
	return ImageManifestMediaType, nil
}

// RawConfigFile implements v1.Image.
func (w *wasmImage) RawConfigFile() ([]byte, error) {
	return json.Marshal(&w.config)
}

// RawManifest implements v1.Image.
func (w *wasmImage) RawManifest() ([]byte, error) {
	manifest, err := w.Manifest()
	if err != nil {
		return nil, err
	}
	return json.Marshal(manifest)
}

// Size implements v1.Image.
func (w *wasmImage) Size() (int64, error) {
	manifest, err := w.RawManifest()
	if err != nil {
		return 0, err
	}
	return int64(len(manifest)), nil
}

type WasmConfigFile struct {
	Created      v1.Time                 `json:"created,omitempty"`
	Author       string                  `json:"author,omitempty"`
	Architecture string                  `json:"architecture"`
	OS           string                  `json:"os"`
	LayerDigests []string                `json:"layerDigests,omitempty"`
	Component    WasmConfigFileComponent `json:"component,omitempty"`
}

type WasmConfigFileComponent struct {
	Exports         []string `json:"exports,omitempty"`
	Imports         []string `json:"imports,omitempty"`
	OptionalImports []string `json:"optional_imports,omitempty"`
	Target          *string  `json:"target"`
}

func ApplyTemplate(ctx context.Context, imageTemplate string, obj client.Object) (name.Tag, error) {
	template, err := template.New("repository").Parse(imageTemplate)
	if err != nil {
		return name.Tag{}, err
	}

	gvk, _ := reconcilers.RetrieveConfigOrDie(ctx).GroupVersionKindFor(obj)
	data := newTemplateData(obj, gvk)
	var image bytes.Buffer
	if err := template.Execute(&image, data); err != nil {
		return name.Tag{}, err
	}

	ref, err := name.NewTag(image.String(), name.WeakValidation)
	if err != nil {
		return name.Tag{}, err
	}

	return ref, nil
}

func newTemplateData(obj client.Object, gvk schema.GroupVersionKind) map[string]string {
	return map[string]string{
		"Namespace":       defaultValue(obj.GetNamespace(), "unset-namespace"),
		"Name":            defaultValue(obj.GetName(), "unset-name"),
		"UID":             defaultValue(string(obj.GetUID()), "unset-uid"),
		"Generation":      defaultValue(fmt.Sprintf("%d", obj.GetGeneration()), "unset-generation"),
		"ResourceVersion": defaultValue(obj.GetResourceVersion(), "unset-resource-version"),
		"Group":           defaultValue(gvk.Group, "unset-group"),
		"Kind":            defaultValue(strings.ToLower(gvk.Kind), "unset-kind"),
	}
}

func defaultValue(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}

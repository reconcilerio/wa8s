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
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func ResolveDigest(ctx context.Context, image string, opts ...remote.Option) (name.Digest, error) {
	transport, err := CustomTransport(ctx)
	if err != nil {
		return name.Digest{}, err
	}
	opts = append(opts, remote.WithContext(ctx), remote.WithTransport(transport))

	if ref, err := name.NewDigest(image, name.WeakValidation); err == nil {
		return ref, nil
	}

	tag, err := name.NewTag(image, name.WeakValidation)
	if err != nil {
		return name.Digest{}, fmt.Errorf("failed to parse image name %q into a tag: %w", image, err)
	}

	desc, err := remote.Head(tag, opts...)
	if err != nil {
		return name.Digest{}, err
	}
	return name.NewDigest(fmt.Sprintf("%s@%s", tag.Repository.String(), desc.Digest), name.WeakValidation)
}

func Push(ctx context.Context, ref name.Reference, component []byte, opts ...remote.Option) (name.Digest, WasmConfigFile, error) {
	transport, err := CustomTransport(ctx)
	if err != nil {
		return name.Digest{}, WasmConfigFile{}, err
	}
	opts = append(opts, remote.WithContext(ctx), remote.WithTransport(transport))

	img, config, err := newWasmImage(ctx, component)
	if err != nil {
		return name.Digest{}, WasmConfigFile{}, err
	}
	digest, err := img.Digest()
	if err != nil {
		return name.Digest{}, WasmConfigFile{}, err
	}
	if err := remote.Push(ref, img, opts...); err != nil {
		return name.Digest{}, WasmConfigFile{}, err
	}
	published, err := name.NewDigest(fmt.Sprintf("%s@%s", ref, digest), name.WeakValidation)
	if err != nil {
		return name.Digest{}, WasmConfigFile{}, err
	}
	return published, config, nil
}

func Pull(ctx context.Context, ref name.Digest, opts ...remote.Option) ([]byte, WasmConfigFile, error) {
	transport, err := CustomTransport(ctx)
	if err != nil {
		return nil, WasmConfigFile{}, err
	}
	opts = append(opts, remote.WithContext(ctx), remote.WithTransport(transport))

	d, err := remote.Get(ref, opts...)
	if err != nil {
		return nil, WasmConfigFile{}, err
	}

	image, err := d.Image()
	if err != nil {
		return nil, WasmConfigFile{}, err
	}

	if mediaType, err := image.MediaType(); err != nil {
		return nil, WasmConfigFile{}, err
	} else if !mediaType.IsImage() {
		// TODO handle image indexes?
		return nil, WasmConfigFile{}, fmt.Errorf("expected image, found %q", mediaType)
	}

	layers, err := image.Layers()
	if err != nil {
		return nil, WasmConfigFile{}, err
	}
	if len(layers) != 1 {
		return nil, WasmConfigFile{}, fmt.Errorf("must be exactly 1 layer")
	}

	layer := layers[0]
	if mediaType, err := layer.MediaType(); err != nil {
		return nil, WasmConfigFile{}, err
	} else if mediaType != WasmLayerMediaType {
		return nil, WasmConfigFile{}, fmt.Errorf("layer must be of media type %q, found %q", WasmLayerMediaType, mediaType)
	}

	componentRaw, err := layer.Uncompressed()
	if err != nil {
		return nil, WasmConfigFile{}, err
	}
	component, err := io.ReadAll(componentRaw)
	if err != nil {
		return nil, WasmConfigFile{}, err
	}

	config := WasmConfigFile{}
	if rawConfig, err := image.RawConfigFile(); err != nil {
		return nil, WasmConfigFile{}, err
	} else if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, WasmConfigFile{}, err
	}

	return component, config, nil
}

func PullConfig(ctx context.Context, ref name.Digest, opts ...remote.Option) (WasmConfigFile, error) {
	transport, err := CustomTransport(ctx)
	if err != nil {
		return WasmConfigFile{}, err
	}
	opts = append(opts, remote.WithContext(ctx), remote.WithTransport(transport))

	d, err := remote.Get(ref, opts...)
	if err != nil {
		return WasmConfigFile{}, err
	}

	image, err := d.Image()
	if err != nil {
		return WasmConfigFile{}, err
	}

	if mediaType, err := image.MediaType(); err != nil {
		return WasmConfigFile{}, err
	} else if !mediaType.IsImage() {
		// TODO handle image indexes?
		return WasmConfigFile{}, fmt.Errorf("expected image, found %q", mediaType)
	}

	config := WasmConfigFile{}
	if rawConfig, err := image.RawConfigFile(); err != nil {
		return WasmConfigFile{}, err
	} else if err := json.Unmarshal(rawConfig, &config); err != nil {
		return WasmConfigFile{}, err
	}

	return config, nil
}

func Copy(ctx context.Context, from name.Reference, to name.Tag, opts ...remote.Option) (name.Digest, error) {
	transport, err := CustomTransport(ctx)
	if err != nil {
		return name.Digest{}, err
	}
	opts = append(opts, remote.WithContext(ctx), remote.WithTransport(transport))

	pusher, err := remote.NewPusher(opts...)
	if err != nil {
		return name.Digest{}, err
	}
	puller, err := remote.NewPuller(opts...)
	if err != nil {
		return name.Digest{}, err
	}
	digest, err := ResolveDigest(ctx, from.Name(), opts...)
	if err != nil {
		return name.Digest{}, err
	}
	desc, err := puller.Get(ctx, digest)
	if err != nil {
		return name.Digest{}, err
	}
	if err := pusher.Push(ctx, to, desc); err != nil {
		return name.Digest{}, err
	}
	return name.NewDigest(fmt.Sprintf("%s@%s", to.Repository, digest.DigestStr()))
}

func componentAsLayer(component []byte) (v1.Layer, error) {
	buf := bytes.NewBuffer([]byte{})
	tarWriter := tar.NewWriter(buf)
	defer tarWriter.Close()
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "component.wasm",
		Size: int64(len(component)),
		Mode: 0444,

		ModTime: time.Unix(60, 0),
	}); err != nil {
		return nil, err
	}
	if _, err := io.Copy(tarWriter, bytes.NewReader(component)); err != nil {
		return nil, err
	}
	layer := stream.NewLayer(io.NopCloser(buf), stream.WithMediaType(types.OCILayer))
	return layer, nil
}

func AppendComponent(ctx context.Context, base name.Reference, target name.Tag, component []byte, opts ...remote.Option) (name.Digest, error) {
	transport, err := CustomTransport(ctx)
	if err != nil {
		return name.Digest{}, err
	}
	opts = append(opts, remote.WithContext(ctx), remote.WithTransport(transport))

	// copy base image into the target repository
	relocatedBase, err := Copy(ctx, base, target, opts...)
	if err != nil {
		return name.Digest{}, err
	}

	// append component layer
	layer, err := componentAsLayer(component)
	if err != nil {
		return name.Digest{}, err
	}
	baseImg, err := remote.Image(relocatedBase, opts...)
	if err != nil {
		return name.Digest{}, err
	}
	// TODO what if the image is an index?
	img, err := mutate.AppendLayers(baseImg, layer)
	if err != nil {
		return name.Digest{}, err
	}

	// push appended image
	pusher, err := remote.NewPusher(opts...)
	if err != nil {
		return name.Digest{}, err
	}
	if err := pusher.Push(ctx, target, img); err != nil {
		return name.Digest{}, err
	}

	// resulting digested ref
	digest, err := img.Digest()
	if err != nil {
		return name.Digest{}, err
	}
	return name.NewDigest(fmt.Sprintf("%s@%s", target.Repository, digest))
}

func ParseReference(image string) (name.Reference, error) {
	if ref, err := name.NewDigest(image, name.WeakValidation); err == nil {
		return ref, nil
	}
	return name.NewTag(image, name.WeakValidation)
}

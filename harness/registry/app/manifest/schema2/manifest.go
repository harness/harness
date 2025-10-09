// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema2

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
)

const (
	// MediaTypeManifest specifies the mediaType for the current version.
	MediaTypeManifest = "application/vnd.docker.distribution.manifest.v2+json"

	// MediaTypeImageConfig specifies the mediaType for the image configuration.
	MediaTypeImageConfig = "application/vnd.docker.container.image.v1+json"

	// MediaTypePluginConfig specifies the mediaType for plugin configuration.
	MediaTypePluginConfig = "application/vnd.docker.plugin.v1+json"

	// MediaTypeLayer is the mediaType used for layers referenced by the
	// manifest.
	MediaTypeLayer = "application/vnd.docker.image.rootfs.diff.tar.gzip"

	// MediaTypeForeignLayer is the mediaType used for layers that must be
	// downloaded from foreign URLs.
	MediaTypeForeignLayer = "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip"

	// MediaTypeUncompressedLayer is the mediaType used for layers which
	// are not compressed.
	MediaTypeUncompressedLayer = "application/vnd.docker.image.rootfs.diff.tar"
)

// SchemaVersion provides a pre-initialized version structure for this
// packages version of the manifest.
var SchemaVersion = manifest.Versioned{
	SchemaVersion: 2,
	MediaType:     MediaTypeManifest,
}

func init() {
	schema2Func := func(b []byte) (manifest.Manifest, manifest.Descriptor, error) {
		m := new(DeserializedManifest)
		err := m.UnmarshalJSON(b)
		if err != nil {
			return nil, manifest.Descriptor{}, err
		}

		dgst := digest.FromBytes(b)
		return m, manifest.Descriptor{Digest: dgst, Size: int64(len(b)), MediaType: MediaTypeManifest}, err
	}
	err := manifest.RegisterManifestSchema(MediaTypeManifest, schema2Func)
	if err != nil {
		panic(fmt.Sprintf("Unable to register manifest: %s", err))
	}
}

// Manifest defines a schema2 manifest.
type Manifest struct {
	manifest.Versioned

	// Config references the image configuration as a blob.
	Config manifest.Descriptor `json:"config"`

	// Layers lists descriptors for the layers referenced by the
	// configuration.
	Layers []manifest.Descriptor `json:"layers"`
}

// References returns the descriptors of this manifests references.
func (m Manifest) References() []manifest.Descriptor {
	references := make([]manifest.Descriptor, 0, 1+len(m.Layers))
	references = append(references, m.Config)
	references = append(references, m.Layers...)
	return references
}

// Target returns the target of this manifest.
func (m Manifest) Target() manifest.Descriptor {
	return m.Config
}

// DeserializedManifest wraps Manifest with a copy of the original JSON.
// It satisfies the distribution.Manifest interface.
type DeserializedManifest struct {
	Manifest

	// canonical is the canonical byte representation of the Manifest.
	canonical []byte
}

// FromStruct takes a Manifest structure, marshals it to JSON, and returns a
// DeserializedManifest which contains the manifest and its JSON representation.
func FromStruct(m Manifest) (*DeserializedManifest, error) {
	var deserialized DeserializedManifest
	deserialized.Manifest = m

	var err error
	deserialized.canonical, err = json.MarshalIndent(&m, "", "   ")
	return &deserialized, err
}

// UnmarshalJSON populates a new Manifest struct from JSON data.
func (m *DeserializedManifest) UnmarshalJSON(b []byte) error {
	m.canonical = make([]byte, len(b))
	// store manifest in canonical
	copy(m.canonical, b)

	// Unmarshal canonical JSON into Manifest object
	var mfst Manifest
	if err := json.Unmarshal(m.canonical, &mfst); err != nil {
		return err
	}

	if mfst.MediaType != MediaTypeManifest {
		return fmt.Errorf(
			"mediaType in manifest should be '%s' not '%s'",
			MediaTypeManifest, mfst.MediaType,
		)
	}

	m.Manifest = mfst

	return nil
}

// MarshalJSON returns the contents of canonical. If canonical is empty,
// marshals the inner contents.
func (m *DeserializedManifest) MarshalJSON() ([]byte, error) {
	if len(m.canonical) > 0 {
		return m.canonical, nil
	}

	return nil, errors.New("JSON representation not initialized in DeserializedManifest")
}

func (m *DeserializedManifest) Version() manifest.Versioned   { return m.Versioned }
func (m *DeserializedManifest) Config() manifest.Descriptor   { return m.Target() }
func (m *DeserializedManifest) Layers() []manifest.Descriptor { return m.Manifest.Layers }
func (m *DeserializedManifest) DistributableLayers() []manifest.Descriptor {
	var ll []manifest.Descriptor
	for _, l := range m.Layers() {
		if l.MediaType != MediaTypeForeignLayer {
			ll = append(ll, l)
		}
	}
	return ll
}

func (m *DeserializedManifest) TotalSize() int64 {
	var layersSize int64
	for _, layer := range m.Layers() {
		layersSize += layer.Size
	}

	return layersSize + m.Config().Size + int64(len(m.canonical))
}

// Payload returns the raw content of the manifest. The contents can be used to
// calculate the content identifier.
func (m DeserializedManifest) Payload() (string, []byte, error) {
	return m.MediaType, m.canonical, nil
}

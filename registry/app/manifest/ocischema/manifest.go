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

package ocischema

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/harness/gitness/registry/app/manifest"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// SchemaVersion provides a pre-initialized version structure for OCI Image
// Manifests.
var SchemaVersion = manifest.Versioned{
	SchemaVersion: 2,
	MediaType:     v1.MediaTypeImageManifest,
}

func init() {
	ocischemaFunc := func(b []byte) (manifest.Manifest, manifest.Descriptor, error) {
		if err := validateManifest(b); err != nil {
			return nil, manifest.Descriptor{}, err
		}
		m := new(DeserializedManifest)
		err := m.UnmarshalJSON(b)
		if err != nil {
			return nil, manifest.Descriptor{}, err
		}

		dgst := digest.FromBytes(b)
		return m, manifest.Descriptor{
			MediaType:   v1.MediaTypeImageManifest,
			Digest:      dgst,
			Size:        int64(len(b)),
			Annotations: m.Annotations(),
		}, err
	}
	err := manifest.RegisterManifestSchema(v1.MediaTypeImageManifest, ocischemaFunc)
	if err != nil {
		panic(fmt.Sprintf("Unable to register manifest: %s", err))
	}
}

// Manifest defines a ocischema manifest.
type Manifest struct {
	manifest.Versioned

	// This OPTIONAL property contains the type of an artifact when the
	// manifest is used for an artifact. This MUST be set when
	// config.mediaType is set to the empty value.
	ArtifactType string `json:"artifactType,omitempty"`

	// Config references the image configuration as a blob.
	Config manifest.Descriptor `json:"config"`

	// Layers lists descriptors for the layers referenced by the
	// configuration.
	Layers []manifest.Descriptor `json:"layers"`

	// This OPTIONAL property specifies a descriptor of another manifest.
	// This value, used by the referrers API, indicates a relationship to
	// the specified manifest.
	Subject *manifest.Descriptor `json:"subject,omitempty"`

	// Annotations contains arbitrary metadata for the image manifest.
	Annotations map[string]string `json:"annotations,omitempty"`
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

	if mfst.MediaType != "" && mfst.MediaType != v1.MediaTypeImageManifest {
		return fmt.Errorf(
			"if present, mediaType in manifest should be '%s' not '%s'",
			v1.MediaTypeImageManifest, mfst.MediaType,
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

// Payload returns the raw content of the manifest. The contents can be used to
// calculate the content identifier.
func (m DeserializedManifest) Payload() (string, []byte, error) {
	return v1.MediaTypeImageManifest, m.canonical, nil
}

// validateManifest returns an error if the byte slice is invalid JSON or if it
// contains fields that belong to a index.
func validateManifest(b []byte) error {
	var doc struct {
		Manifests any `json:"manifests,omitempty"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	if doc.Manifests != nil {
		return errors.New("ocimanifest: expected manifest but found index")
	}
	return nil
}

func (m *DeserializedManifest) Version() manifest.Versioned {
	// Media type can be either Docker (`application/vnd.docker.distribution.manifest.v2+json`) or OCI (empty).
	// We need to make it explicit if empty, otherwise we're not able to distinguish between media types.
	if m.Versioned.MediaType == "" {
		m.Versioned.MediaType = v1.MediaTypeImageManifest
	}

	return m.Versioned
}

func (m *DeserializedManifest) Config() manifest.Descriptor   { return m.Target() }
func (m *DeserializedManifest) Layers() []manifest.Descriptor { return m.Manifest.Layers }
func (m *DeserializedManifest) DistributableLayers() []manifest.Descriptor {
	var ll []manifest.Descriptor
	for _, l := range m.Layers() {
		switch l.MediaType {
		case v1.MediaTypeImageLayerNonDistributable, v1.MediaTypeImageLayerNonDistributableGzip:
			continue
		}
		ll = append(ll, l)
	}
	return ll
}
func (m *DeserializedManifest) ArtifactType() string { return m.Manifest.ArtifactType }
func (m *DeserializedManifest) Subject() manifest.Descriptor {
	if m.Manifest.Subject == nil {
		return manifest.Descriptor{}
	}
	return *m.Manifest.Subject
}

func (m *DeserializedManifest) Annotations() map[string]string {
	if m.Manifest.Annotations == nil {
		return map[string]string{}
	}
	return m.Manifest.Annotations
}

func (m *DeserializedManifest) TotalSize() int64 {
	var layersSize int64
	for _, layer := range m.Layers() {
		layersSize += layer.Size
	}

	return layersSize + m.Config().Size + int64(len(m.canonical))
}

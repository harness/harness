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

// IndexSchemaVersion provides a pre-initialized version structure for OCI Image
// Indices.
var IndexSchemaVersion = manifest.Versioned{
	SchemaVersion: 2,
	MediaType:     v1.MediaTypeImageIndex,
}

func init() {
	imageIndexFunc := func(b []byte) (manifest.Manifest, manifest.Descriptor, error) {
		if err := validateIndex(b); err != nil {
			return nil, manifest.Descriptor{}, err
		}
		m := new(DeserializedImageIndex)
		err := m.UnmarshalJSON(b)
		if err != nil {
			return nil, manifest.Descriptor{}, err
		}

		if m.MediaType != "" && m.MediaType != v1.MediaTypeImageIndex {
			err = fmt.Errorf(
				"if present, mediaType in image index should be '%s' not '%s'",
				v1.MediaTypeImageIndex, m.MediaType,
			)

			return nil, manifest.Descriptor{}, err
		}

		dgst := digest.FromBytes(b)
		return m, manifest.Descriptor{
			MediaType:   v1.MediaTypeImageIndex,
			Digest:      dgst,
			Size:        int64(len(b)),
			Annotations: m.Annotations(),
		}, err
	}
	err := manifest.RegisterManifestSchema(v1.MediaTypeImageIndex, imageIndexFunc)
	if err != nil {
		panic(fmt.Sprintf("Unable to register OCI Image Index: %s", err))
	}
}

// ImageIndex references manifests for various platforms.
type ImageIndex struct {
	manifest.Versioned

	// This OPTIONAL property contains the type of an artifact when the
	// manifest is used for an artifact. This MUST be set when
	// config.mediaType is set to the empty value.
	ArtifactType string `json:"artifactType,omitempty"`

	// Manifests references a list of manifests
	Manifests []manifest.Descriptor `json:"manifests"`

	// Annotations is an optional field that contains arbitrary metadata for the
	// image index
	Annotations map[string]string `json:"annotations,omitempty"`

	// This OPTIONAL property specifies a descriptor of another manifest.
	// This value, used by the referrers API, indicates a relationship to
	// the specified manifest.
	Subject *manifest.Descriptor `json:"subject,omitempty"`
}

// References returns the distribution descriptors for the referenced image
// manifests.
func (ii ImageIndex) References() []manifest.Descriptor {
	return ii.Manifests
}

// DeserializedImageIndex wraps ManifestList with a copy of the original
// JSON.
type DeserializedImageIndex struct {
	ImageIndex

	// canonical is the canonical byte representation of the Manifest.
	canonical []byte
}

// FromDescriptors takes a slice of descriptors and a map of annotations, and
// returns a DeserializedManifestList which contains the resulting manifest list
// and its JSON representation. If annotations is nil or empty then the
// annotations property will be omitted from the JSON representation.
func FromDescriptors(descriptors []manifest.Descriptor, annotations map[string]string) (
	*DeserializedImageIndex, error,
) {
	return fromDescriptorsWithMediaType(descriptors, annotations, v1.MediaTypeImageIndex)
}

// fromDescriptorsWithMediaType is for testing purposes,
// it's useful to be able to specify the media type explicitly.
func fromDescriptorsWithMediaType(
	descriptors []manifest.Descriptor,
	annotations map[string]string,
	mediaType string,
) (_ *DeserializedImageIndex, err error) {
	m := ImageIndex{
		Versioned: manifest.Versioned{
			SchemaVersion: IndexSchemaVersion.SchemaVersion,
			MediaType:     mediaType,
		},
		Annotations: annotations,
	}

	m.Manifests = make([]manifest.Descriptor, len(descriptors))
	copy(m.Manifests, descriptors)

	deserialized := DeserializedImageIndex{
		ImageIndex: m,
	}

	deserialized.canonical, err = json.MarshalIndent(&m, "", "   ")
	return &deserialized, err
}

// UnmarshalJSON populates a new ManifestList struct from JSON data.
func (m *DeserializedImageIndex) UnmarshalJSON(b []byte) error {
	m.canonical = make([]byte, len(b))
	// store manifest list in canonical
	copy(m.canonical, b)

	// Unmarshal canonical JSON into ManifestList object
	var manifestList ImageIndex
	if err := json.Unmarshal(m.canonical, &manifestList); err != nil {
		return err
	}

	m.ImageIndex = manifestList

	return nil
}

// MarshalJSON returns the contents of canonical. If canonical is empty,
// marshals the inner contents.
func (m *DeserializedImageIndex) MarshalJSON() ([]byte, error) {
	if len(m.canonical) > 0 {
		return m.canonical, nil
	}

	return nil, errors.New("JSON representation not initialized in DeserializedImageIndex")
}

// Payload returns the raw content of the manifest list. The contents can be
// used to calculate the content identifier.
func (m DeserializedImageIndex) Payload() (string, []byte, error) {
	mediaType := m.MediaType
	if m.MediaType == "" {
		mediaType = v1.MediaTypeImageIndex
	}

	return mediaType, m.canonical, nil
}

// validateIndex returns an error if the byte slice is invalid JSON or if it
// contains fields that belong to a manifest.
func validateIndex(b []byte) error {
	var doc struct {
		Config any `json:"config,omitempty"`
		Layers any `json:"layers,omitempty"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	if doc.Config != nil || doc.Layers != nil {
		return errors.New("index: expected index but found manifest")
	}
	return nil
}
func (m *DeserializedImageIndex) ArtifactType() string { return m.ImageIndex.ArtifactType }
func (m *DeserializedImageIndex) Subject() manifest.Descriptor {
	if m.ImageIndex.Subject == nil {
		return manifest.Descriptor{}
	}
	return *m.ImageIndex.Subject
}

func (m *DeserializedImageIndex) Annotations() map[string]string {
	if m.ImageIndex.Annotations == nil {
		return map[string]string{}
	}
	return m.ImageIndex.Annotations
}

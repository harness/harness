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

package manifest

import (
	"context"
	"fmt"
	"mime"

	"github.com/opencontainers/go-digest"
)

// Manifest represents a registry object specifying a set of
// references and an optional target.
type Manifest interface {
	// References returns a list of objects which make up this manifest.
	// A reference is anything which can be represented by a
	// distribution.Descriptor. These can consist of layers, resources or other
	// manifests.
	//
	// While no particular order is required, implementations should return
	// them from highest to lowest priority. For example, one might want to
	// return the base layer before the top layer.
	References() []Descriptor

	// Payload provides the serialized format of the manifest, in addition to
	// the media type.
	Payload() (mediaType string, payload []byte, err error)
}

type ManifestV2 interface {
	Manifest
	Version() Versioned
	Config() Descriptor
	Layers() []Descriptor

	// TotalSize is the sum of the size of the manifest payload, layer and config
	// blob sizes.
	TotalSize() int64
	// DistributableLayers is a slice of distributable image layers. This is a subset of Layers, excluding items with
	// media types that are known to identify non-distributable layers.
	DistributableLayers() []Descriptor
}

// ManifestOCI extends ManifestV2 with property descriptions from the
// OCI Image Manifest specification (v1.1).
// https://github.com/opencontainers/image-spec/blob/main/manifest.md#image-manifest-property-descriptions
type ManifestOCI interface {
	ManifestV2

	// This OPTIONAL property contains the type of an artifact when the
	// manifest is used for an artifact.
	ArtifactType() string

	// This OPTIONAL property specifies a descriptor of another manifest.
	// This value, used by the referrers API, indicates a relationship
	// to the specified manifest.
	Subject() Descriptor

	Annotations() map[string]string
}

// ManifestBuilder creates a manifest allowing one to include dependencies.
// Instances can be obtained from a version-specific manifest package.  Manifest
// specific data is passed into the function which creates the builder.
type ManifestBuilder interface {
	// Build creates the manifest from his builder.
	Build(ctx context.Context) (Manifest, error)

	// References returns a list of objects which have been added to this
	// builder. The dependencies are returned in the order they were added,
	// which should be from base to head.
	References() []Descriptor

	// AppendReference includes the given object in the manifest after any
	// existing dependencies. If the add fails, such as when adding an
	// unsupported dependency, an error may be returned.
	//
	// The destination of the reference is dependent on the manifest type and
	// the dependency type.
	AppendReference(dependency Describable) error
}

// ManifestEnumerator enables iterating over manifests.
type ManifestEnumerator interface {
	// Enumerate calls ingester for each manifest.
	Enumerate(ctx context.Context, ingester func(digest.Digest) error) error
}

// Describable is an interface for descriptors.
type Describable interface {
	Descriptor() Descriptor
}

// ManifestMediaTypes returns the supported media types for manifests.
func ManifestMediaTypes() (mediaTypes []string) {
	for t := range mappings {
		if t != "" {
			mediaTypes = append(mediaTypes, t)
		}
	}
	return
}

// UnmarshalFunc implements manifest unmarshalling a given MediaType.
type UnmarshalFunc func([]byte) (Manifest, Descriptor, error)

var mappings = make(map[string]UnmarshalFunc)

// UnmarshalManifest looks up manifest unmarshal functions based on
// MediaType.
func UnmarshalManifest(ctHeader string, p []byte) (Manifest, Descriptor, error) {
	// Need to look up by the actual media type, not the raw contents of
	// the header. Strip semicolons and anything following them.
	var mediaType string
	if ctHeader != "" {
		var err error
		mediaType, _, err = mime.ParseMediaType(ctHeader)
		if err != nil {
			return nil, Descriptor{}, err
		}
	}

	unmarshalFunc, ok := mappings[mediaType]
	if !ok {
		unmarshalFunc, ok = mappings[""]
		if !ok {
			return nil, Descriptor{}, fmt.Errorf(
				"unsupported manifest media type and no default available: %s",
				mediaType,
			)
		}
	}

	return unmarshalFunc(p)
}

// RegisterManifestSchema registers an UnmarshalFunc for a given schema type.  This
// should be called from specific.
func RegisterManifestSchema(mediaType string, u UnmarshalFunc) error {
	if _, ok := mappings[mediaType]; ok {
		return fmt.Errorf("manifest media type registration would overwrite existing: %s", mediaType)
	}
	mappings[mediaType] = u
	return nil
}

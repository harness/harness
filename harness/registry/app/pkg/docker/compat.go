// Source: https://gitlab.com/gitlab-org/container-registry

// Copyright 2019 Gitlab Inc.
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

package docker

import (
	"errors"
	"fmt"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/ocischema"
	"github.com/harness/gitness/registry/app/manifest/schema2"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// MediaTypeManifest specifies the mediaType for the current version. Note
	// that for schema version 1, the the media is optionally "application/json".
	MediaTypeManifest = "application/vnd.docker.distribution.manifest.v1+json"
	// MediaTypeSignedManifest specifies the mediatype for current SignedManifest version.
	MediaTypeSignedManifest = "application/vnd.docker.distribution.manifest.v1+prettyjws"
)

// MediaTypeBuildxCacheConfig is the mediatype associated with buildx
// cache config blobs. This should be unique to buildx.
var MediaTypeBuildxCacheConfig = "application/vnd.buildkit.cacheconfig.v0"

// SplitReferences contains two lists of manifest list references broken down
// into either blobs or manifests. The result of appending these two lists
// together should include all of the descriptors returned by
// ManifestList.References with no duplicates, additions, or omissions.
type SplitReferences struct {
	Manifests []manifest.Descriptor
	Blobs     []manifest.Descriptor
}

// References returns the references of the DeserializedManifestList split into
// manifests and layers based on the mediatype of the standard list of
// descriptors. Only known manifest mediatypes will be sorted into the manifests
// array while everything else will be sorted into blobs. Helm chart manifests
// do not include a mediatype at the time of this commit, but they are unlikely
// to be included within a manifest list.
func References(ml *manifestlist.DeserializedManifestList) SplitReferences {
	var (
		manifests = make([]manifest.Descriptor, 0)
		blobs     = make([]manifest.Descriptor, 0)
	)

	for _, r := range ml.References() {
		switch r.MediaType {
		case schema2.MediaTypeManifest,
			manifestlist.MediaTypeManifestList,
			v1.MediaTypeImageManifest,
			MediaTypeSignedManifest,
			MediaTypeManifest:

			manifests = append(manifests, r)
		default:
			blobs = append(blobs, r)
		}
	}

	return SplitReferences{Manifests: manifests, Blobs: blobs}
}

// LikelyBuildxCache returns true if the manifest list is likely a buildx cache
// manifest based on the unique buildx config mediatype.
func LikelyBuildxCache(ml *manifestlist.DeserializedManifestList) bool {
	blobs := References(ml).Blobs

	for _, desc := range blobs {
		if desc.MediaType == MediaTypeBuildxCacheConfig {
			return true
		}
	}

	return false
}

// ContainsBlobs returns true if the manifest list contains any blobs.
func ContainsBlobs(ml *manifestlist.DeserializedManifestList) bool {
	return len(References(ml).Blobs) > 0
}

func OCIManifestFromBuildkitIndex(ml *manifestlist.DeserializedManifestList) (*ocischema.DeserializedManifest, error) {
	refs := References(ml)
	if len(refs.Manifests) > 0 {
		return nil, errors.New("buildkit index has unexpected manifest references")
	}

	// set "config" and "layer" references apart.
	var cfg *manifest.Descriptor
	var layers []manifest.Descriptor
	for _, ref := range refs.Blobs {
		refCopy := ref
		if refCopy.MediaType == MediaTypeBuildxCacheConfig {
			cfg = &refCopy
		} else {
			layers = append(layers, refCopy)
		}
	}

	// make sure they were found.
	if cfg == nil {
		return nil, errors.New("buildkit index has no config reference")
	}

	m, err := ocischema.FromStruct(
		ocischema.Manifest{
			Versioned: ocischema.SchemaVersion,
			Config:    *cfg,
			Layers:    layers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("building manifest from buildkit index: %w", err)
	}

	return m, nil
}

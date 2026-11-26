//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	repostore "github.com/harness/gitness/registry/app/store/database"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

func rpfDigest(seed string) digest.Digest {
	return digest.FromString(seed)
}

func rpfDescriptor(mediaType, seed string) manifestlist.ManifestDescriptor {
	return manifestlist.ManifestDescriptor{
		Descriptor: manifest.Descriptor{MediaType: mediaType, Digest: rpfDigest(seed), Size: 10},
		Platform:   manifestlist.PlatformSpec{Architecture: "amd64", OS: "linux"},
	}
}

func rpfIndex(t *testing.T, descriptors ...manifestlist.ManifestDescriptor) *manifestlist.DeserializedManifestList {
	t.Helper()
	ml, err := manifestlist.FromDescriptors(descriptors)
	require.NoError(t, err)
	return ml
}

func TestRpfReferencesSplitsManifestsFromBlobs(t *testing.T) {
	ml := rpfIndex(t,
		rpfDescriptor(schema2.MediaTypeManifest, "m1"),
		rpfDescriptor(v1.MediaTypeImageManifest, "m2"),
		rpfDescriptor(manifestlist.MediaTypeManifestList, "m3"),
		rpfDescriptor(MediaTypeSignedManifest, "m4"),
		rpfDescriptor(MediaTypeManifest, "m5"),
		rpfDescriptor("application/octet-stream", "b1"),
	)
	refs := References(ml)
	require.Len(t, refs.Manifests, 5)
	require.Len(t, refs.Blobs, 1)
	require.Equal(t, "application/octet-stream", refs.Blobs[0].MediaType)
	require.True(t, ContainsBlobs(ml))

	onlyManifests := rpfIndex(t, rpfDescriptor(schema2.MediaTypeManifest, "m1"))
	require.False(t, ContainsBlobs(onlyManifests))
	require.False(t, LikelyBuildxCache(onlyManifests))
}

func TestRpfLikelyBuildxCache(t *testing.T) {
	buildx := rpfIndex(t,
		rpfDescriptor(MediaTypeBuildxCacheConfig, "cfg"),
		rpfDescriptor("application/vnd.oci.image.layer.v1.tar", "layer"),
	)
	require.True(t, LikelyBuildxCache(buildx))

	plain := rpfIndex(t, rpfDescriptor("application/octet-stream", "b"))
	require.False(t, LikelyBuildxCache(plain))
}

func TestRpfOCIManifestFromBuildkitIndex(t *testing.T) {
	index := rpfIndex(t,
		rpfDescriptor(MediaTypeBuildxCacheConfig, "cfg"),
		rpfDescriptor("application/vnd.oci.image.layer.v1.tar", "layer1"),
		rpfDescriptor("application/vnd.oci.image.layer.v1.tar", "layer2"),
	)
	m, err := OCIManifestFromBuildkitIndex(index)
	require.NoError(t, err)
	require.Equal(t, MediaTypeBuildxCacheConfig, m.Config().MediaType)
	require.Len(t, m.Layers(), 2)

	_, err = OCIManifestFromBuildkitIndex(rpfIndex(t,
		rpfDescriptor(schema2.MediaTypeManifest, "m1"), rpfDescriptor(MediaTypeBuildxCacheConfig, "cfg")))
	require.ErrorContains(t, err, "unexpected manifest references")

	_, err = OCIManifestFromBuildkitIndex(rpfIndex(t, rpfDescriptor("application/octet-stream", "b")))
	require.ErrorContains(t, err, "no config reference")
}

func TestRpfCreateLinkEntry(t *testing.T) {
	const orig = "https://registry.test/v2/acme/repo/tags/list?n=5#frag"

	t.Run("no cursors yields no link", func(t *testing.T) {
		got, err := CreateLinkEntry(orig, registrytypes.FilterParams{MaxEntries: 5}, "", "")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("next link", func(t *testing.T) {
		got, err := CreateLinkEntry(orig, registrytypes.FilterParams{MaxEntries: 5, LastEntry: "v5", Name: "v"}, "", "")
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(got, "<https://registry.test/v2/acme/repo/tags/list?"))
		require.True(t, strings.HasSuffix(got, `rel="next"`))
		require.Contains(t, got, "last=v5")
		require.Contains(t, got, "n=5")
		require.Contains(t, got, "name=v")
		require.NotContains(t, got, "frag")
	})

	t.Run("previous and next links are combined", func(t *testing.T) {
		filters := registrytypes.FilterParams{
			MaxEntries: 5, BeforeEntry: "v1", LastEntry: "v5",
			OrderBy: PublishedAtQueryParamKey, SortOrder: repostore.OrderDesc,
		}
		got, err := CreateLinkEntry(orig, filters, "2024-01-01", "2024-02-02")
		require.NoError(t, err)
		parts := strings.Split(got, ", ")
		require.Len(t, parts, 2)
		require.Contains(t, parts[0], `rel="previous"`)
		require.Contains(t, parts[0], "before="+encodeForQuery(EncodeFilter("2024-01-01", "v1")))
		require.Contains(t, parts[1], `rel="next"`)
		require.Contains(t, parts[1], "last="+encodeForQuery(EncodeFilter("2024-02-02", "v5")))
		require.Contains(t, parts[1], "sort=-published_at")
	})

	t.Run("published-at ordering without cursors values keeps plain entries", func(t *testing.T) {
		filters := registrytypes.FilterParams{
			MaxEntries: 5, LastEntry: "v5", OrderBy: PublishedAtQueryParamKey,
		}
		got, err := CreateLinkEntry(orig, filters, "", "")
		require.NoError(t, err)
		require.Contains(t, got, "last=v5")
		require.Contains(t, got, "sort=published_at")
	})

	t.Run("invalid urls are rejected", func(t *testing.T) {
		_, err := CreateLinkEntry("://bad", registrytypes.FilterParams{LastEntry: "x"}, "", "")
		require.Error(t, err)
		_, err = CreateLinkEntry("://bad", registrytypes.FilterParams{BeforeEntry: "x"}, "", "")
		require.Error(t, err)
	})
}

func encodeForQuery(v string) string {
	return strings.NewReplacer("+", "%2B", "/", "%2F", "=", "%3D").Replace(v)
}

func TestRpfEncodeFilter(t *testing.T) {
	require.Equal(t, "MjAyNC0wMS0wMXx2MQ==", EncodeFilter("2024-01-01", "v1"))
}

func TestRpfResponses(t *testing.T) {
	boom := errors.New("boom")
	responses := map[string]Response{
		"get manifest":    &GetManifestResponse{Errors: []error{errors.New("old"), errors.New("older")}},
		"put manifest":    &PutManifestResponse{},
		"delete manifest": &DeleteManifestResponse{Errors: []error{errors.New("old")}},
		"get blob":        &GetBlobResponse{},
	}
	for name, r := range responses {
		t.Run(name, func(t *testing.T) {
			r.SetError(boom)
			require.Equal(t, []error{boom}, r.GetErrors())
		})
	}
}

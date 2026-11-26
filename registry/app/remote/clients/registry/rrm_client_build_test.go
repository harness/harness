// Copyright 2023 Harness, Inc.
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

package registry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRrmBuildPingURL(t *testing.T) {
	require.Equal(t, "http://registry.example/v2/", buildPingURL("http://registry.example"))
}

func TestRrmBuildCatalogURL(t *testing.T) {
	got := buildCatalogURL("https://reg.io")
	require.Equal(t, "https://reg.io/v2/_catalog?n=1000", got)
}

func TestRrmBuildTagListURL(t *testing.T) {
	got := buildTagListURL("http://host", "library/nginx")
	require.Equal(t, "http://host/v2/library/nginx/tags/list", got)
}

func TestRrmBuildManifestURL(t *testing.T) {
	got := buildManifestURL("http://host", "repo/img", "latest")
	require.Equal(t, "http://host/v2/repo/img/manifests/latest", got)
}

func TestRrmBuildBlobURL(t *testing.T) {
	dgst := "sha256:abc123"
	got := buildBlobURL("http://host", "repo/img", dgst)
	require.Equal(t, "http://host/v2/repo/img/blobs/"+dgst, got)
}

func TestRrmBuildMountBlobURL(t *testing.T) {
	dgst := "sha256:abc"
	got := buildMountBlobURL("http://host", "dst/repo", dgst, "src/repo")
	require.Contains(t, got, "mount="+dgst)
	require.Contains(t, got, "from=src/repo")
}

func TestRrmBuildInitiateBlobUploadURL(t *testing.T) {
	got := buildInitiateBlobUploadURL("http://host", "repo/x")
	require.Equal(t, "http://host/v2/repo/x/blobs/uploads/", got)
}

func TestRrmBuildChunkBlobUploadURL(t *testing.T) {
	t.Run("absolute location with digest", func(t *testing.T) {
		got, err := buildChunkBlobUploadURL("http://host", "http://host/v2/r/blobs/u/uuid", "sha256:abc", true)
		require.NoError(t, err)
		require.Contains(t, got, "digest=sha256")
	})

	t.Run("relative location", func(t *testing.T) {
		got, err := buildChunkBlobUploadURL("http://host", "/v2/r/blobs/u/uuid", "sha256:abc", false)
		require.NoError(t, err)
		require.Equal(t, "http://host/v2/r/blobs/u/uuid", got)
	})

	t.Run("invalid location", func(t *testing.T) {
		_, err := buildChunkBlobUploadURL("http://host", "://bad", "sha256:abc", false)
		require.Error(t, err)
	})
}

func TestRrmBuildMonolithicBlobUploadURL(t *testing.T) {
	got, err := buildMonolithicBlobUploadURL("http://host", "/v2/r/blobs/u/uuid", "sha256:abc")
	require.NoError(t, err)
	require.Contains(t, got, "digest=sha256")
}

func TestRrmNextLinkHeader(t *testing.T) {
	t.Run("parses next rel", func(t *testing.T) {
		link := `<http://next.page/catalog>; rel="next", <http://prev.page>; rel="prev"`
		require.Equal(t, "http://next.page/catalog", next(link))
	})

	t.Run("empty when no next", func(t *testing.T) {
		require.Empty(t, next(`<http://only.one>; rel="prev"`))
	})

	t.Run("empty header", func(t *testing.T) {
		require.Empty(t, next(""))
	})
}

func TestRrmParseContentRange(t *testing.T) {
	t.Run("valid range", func(t *testing.T) {
		start, end, err := parseContentRange("0-99")
		require.NoError(t, err)
		require.Equal(t, int64(0), start)
		require.Equal(t, int64(99), end)
	})

	t.Run("invalid format", func(t *testing.T) {
		_, _, err := parseContentRange("invalid")
		require.Error(t, err)
	})

	t.Run("non-numeric start", func(t *testing.T) {
		_, _, err := parseContentRange("x-99")
		require.Error(t, err)
	})
}

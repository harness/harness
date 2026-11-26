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

package registry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/remote/clients/registry/auth/null"

	"github.com/stretchr/testify/require"
)

const (
	w2rLayerDigest  = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	w2rConfigDigest = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	w2rManifestBody = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",` +
		`"config":{"mediaType":"application/vnd.oci.image.config.v1+json","size":2,"digest":"` + w2rConfigDigest + `"},` +
		`"layers":[{"mediaType":"application/vnd.oci.image.layer.v1.tar+gzip","size":3,"digest":"` + w2rLayerDigest + `"}]}`
)

type w2rRegistry struct {
	mu       sync.Mutex
	requests []string
	handle   func(w http.ResponseWriter, r *http.Request) bool
}

func (f *w2rRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()
	if f.handle != nil && f.handle(w, r) {
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (f *w2rRegistry) seen(entry string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.requests {
		if r == entry {
			return true
		}
	}
	return false
}

func w2rClient(t *testing.T, reg *w2rRegistry) (Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(reg)
	t.Cleanup(srv.Close)
	return NewClientWithAuthorizer(srv.URL, null.NewAuthorizer(), true), srv
}

func TestW2rClientPullPushAndDeleteManifest(t *testing.T) {
	ctx := context.Background()
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.Contains(r.URL.Path, "/manifests/") {
			return false
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", "sha256:abc")
			_, _ = io.WriteString(w, w2rManifestBody)
		case http.MethodHead:
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", w2rLayerDigest)
			w.Header().Set("Content-Length", "10")
		case http.MethodPut:
			w.Header().Set("Docker-Content-Digest", "sha256:pushed")
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
		}
		return true
	}}
	c, _ := w2rClient(t, reg)

	m, dgst, err := c.PullManifest(ctx, "lib/app", "latest")
	require.NoError(t, err)
	require.Equal(t, "sha256:abc", dgst)
	require.Len(t, m.References(), 2)

	_, _, err = c.PullManifest(ctx, "lib/app", "latest", "application/vnd.oci.image.manifest.v1+json")
	require.NoError(t, err)

	pushed, err := c.PushManifest(ctx, "lib/app", "v1", "application/vnd.oci.image.manifest.v1+json",
		[]byte(w2rManifestBody))
	require.NoError(t, err)
	require.Equal(t, "sha256:pushed", pushed)

	// A digest reference is deleted directly, a tag is resolved first.
	require.NoError(t, c.DeleteManifest(ctx, "lib/app", w2rLayerDigest))
	require.NoError(t, c.DeleteManifest(ctx, "lib/app", "latest"))
	require.True(t, reg.seen("DELETE /v2/lib/app/manifests/"+w2rLayerDigest))
}

func TestW2rClientDeleteManifestMissingTag(t *testing.T) {
	c, _ := w2rClient(t, &w2rRegistry{})
	require.Error(t, c.DeleteManifest(context.Background(), "lib/app", "missing"))
}

func TestW2rClientPullBlobAndChunk(t *testing.T) {
	ctx := context.Background()
	var gotRange string
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method != http.MethodGet || !strings.Contains(r.URL.Path, "/blobs/") {
			return false
		}
		gotRange = r.Header.Get("Range")
		w.Header().Set("Content-Length", "5")
		_, _ = io.WriteString(w, "hello")
		return true
	}}
	c, _ := w2rClient(t, reg)

	size, body, err := c.PullBlob(ctx, "lib/app", w2rLayerDigest)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
	require.Equal(t, int64(5), size)
	require.Equal(t, "hello", string(data))

	size, body, err = c.PullBlobChunk(ctx, "lib/app", w2rLayerDigest, 5, 0, 4)
	require.NoError(t, err)
	require.NoError(t, body.Close())
	require.Equal(t, int64(5), size)
	require.Equal(t, "bytes=0-4", gotRange)
}

func TestW2rClientPullBlobErrors(t *testing.T) {
	c, _ := w2rClient(t, &w2rRegistry{})
	_, _, err := c.PullBlob(context.Background(), "lib/app", w2rLayerDigest)
	require.Error(t, err)
	_, _, err = c.PullBlobChunk(context.Background(), "lib/app", w2rLayerDigest, 5, 0, 4)
	require.Error(t, err)
}

func TestW2rClientPushBlobMonolithic(t *testing.T) {
	ctx := context.Background()
	var uploaded string
	var srvURL string
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/blobs/uploads/"):
			w.Header().Set("Location", srvURL+"/v2/lib/app/blobs/uploads/u1")
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/blobs/uploads/u1"):
			data, _ := io.ReadAll(r.Body)
			uploaded = string(data) + "|" + r.URL.Query().Get("digest")
			w.WriteHeader(http.StatusCreated)
		default:
			return false
		}
		return true
	}}
	c, srv := w2rClient(t, reg)
	srvURL = srv.URL

	require.NoError(t, c.PushBlob(ctx, "lib/app", w2rLayerDigest, 5, strings.NewReader("hello")))
	require.Equal(t, "hello|"+w2rLayerDigest, uploaded)
}

func TestW2rClientPushBlobFailsWithoutUploadEndpoint(t *testing.T) {
	c, _ := w2rClient(t, &w2rRegistry{})
	require.Error(t, c.PushBlob(context.Background(), "lib/app", w2rLayerDigest, 1, strings.NewReader("x")))
}

func TestW2rClientPushBlobChunks(t *testing.T) {
	ctx := context.Background()
	var srvURL string
	var contentRanges []string
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/blobs/uploads/"):
			w.Header().Set("Location", srvURL+"/v2/lib/app/blobs/uploads/u2")
			w.WriteHeader(http.StatusAccepted)
		case strings.Contains(r.URL.Path, "/blobs/uploads/u2"):
			contentRanges = append(contentRanges, r.Method+" "+r.Header.Get("Content-Range"))
			w.Header().Set("Location", srvURL+"/v2/lib/app/blobs/uploads/u2")
			w.WriteHeader(http.StatusAccepted)
		default:
			return false
		}
		return true
	}}
	c, srv := w2rClient(t, reg)
	srvURL = srv.URL

	location, end, err := c.PushBlobChunk(ctx, "lib/app", w2rLayerDigest, 6, strings.NewReader("abc"), 0, 2, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), end)
	require.Contains(t, location, "/blobs/uploads/u2")

	_, end, err = c.PushBlobChunk(ctx, "lib/app", w2rLayerDigest, 6, strings.NewReader("def"), 3, 5, location)
	require.NoError(t, err)
	require.Equal(t, int64(5), end)
	require.Equal(t, []string{"PATCH 0-2", "PUT 3-5"}, contentRanges)
}

func TestW2rClientPushBlobChunkReportsUploadStatusOnFailure(t *testing.T) {
	var srvURL string
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		if !strings.Contains(r.URL.Path, "/blobs/uploads/u3") {
			return false
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Range", "0-1")
			w.Header().Set("Location", srvURL+"/v2/lib/app/blobs/uploads/u3")
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		w.WriteHeader(http.StatusInternalServerError)
		return true
	}}
	c, srv := w2rClient(t, reg)
	srvURL = srv.URL

	loc, end, err := c.PushBlobChunk(context.Background(), "lib/app", w2rLayerDigest, 10,
		strings.NewReader("cd"), 2, 3, srv.URL+"/v2/lib/app/blobs/uploads/u3")
	require.Error(t, err)
	require.Equal(t, int64(1), end, "the resume offset comes from the server")
	require.Contains(t, loc, "/blobs/uploads/u3")
}

func TestW2rClientMountAndDeleteBlob(t *testing.T) {
	ctx := context.Background()
	var mountQuery string
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/blobs/uploads/"):
			mountQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/blobs/"):
			w.WriteHeader(http.StatusAccepted)
		default:
			return false
		}
		return true
	}}
	c, _ := w2rClient(t, reg)

	require.NoError(t, c.MountBlob(ctx, "src/app", w2rLayerDigest, "dst/app"))
	require.Contains(t, mountQuery, "from=src/app")
	require.Contains(t, mountQuery, "mount=")
	require.NoError(t, c.DeleteBlob(ctx, "dst/app", w2rLayerDigest))

	empty, _ := w2rClient(t, &w2rRegistry{})
	require.Error(t, empty.DeleteBlob(ctx, "dst/app", w2rLayerDigest))
	require.Error(t, empty.MountBlob(ctx, "src/app", w2rLayerDigest, "dst/app"))
}

func TestW2rClientCopy(t *testing.T) {
	ctx := context.Background()
	manifestExists := false
	reg := &w2rRegistry{handle: func(w http.ResponseWriter, r *http.Request) bool {
		path := r.URL.Path
		switch {
		case strings.Contains(path, "/src/app/manifests/") && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", "sha256:source")
			_, _ = io.WriteString(w, w2rManifestBody)
		case strings.Contains(path, "/dst/app/manifests/") && r.Method == http.MethodHead:
			if !manifestExists {
				w.WriteHeader(http.StatusNotFound)
				return true
			}
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", "sha256:different")
		case strings.Contains(path, "/dst/app/blobs/") && r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		case strings.Contains(path, "/dst/app/blobs/uploads/") && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
		case strings.Contains(path, "/dst/app/manifests/") && r.Method == http.MethodPut:
			w.Header().Set("Docker-Content-Digest", "sha256:source")
			w.WriteHeader(http.StatusCreated)
		default:
			return false
		}
		return true
	}}
	c, _ := w2rClient(t, reg)

	require.NoError(t, c.Copy(ctx, "src/app", "latest", "dst/app", "latest", false))
	require.True(t, reg.seen("PUT /v2/dst/app/manifests/latest"))

	// A different artifact with the same name is rejected unless overriding.
	manifestExists = true
	require.Error(t, c.Copy(ctx, "src/app", "latest", "dst/app", "latest", false))
	require.NoError(t, c.Copy(ctx, "src/app", "latest", "dst/app", "latest", true))

	// A missing source fails.
	require.Error(t, c.Copy(ctx, "src/missing", "latest", "dst/app", "latest", false))
}

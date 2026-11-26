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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/remote/clients/registry/auth/null"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

var errRb5Boom = fmt.Errorf("rb5 boom")

const (
	rb5Layer      = "layer-data"
	rb5Config     = "{}"
	rb5IndexMedia = v1.MediaTypeImageIndex
)

type rb5Stored struct {
	mediaType string
	payload   []byte
}

// rb5Registry is a tiny in-memory OCI distribution server.
type rb5Registry struct {
	mu        sync.Mutex
	manifests map[string]rb5Stored
	blobs     map[string]string
	requests  []string
	// override handles a request first; returning true stops processing.
	override func(w http.ResponseWriter, r *http.Request) bool
}

func rb5NewRegistry() *rb5Registry {
	return &rb5Registry{manifests: map[string]rb5Stored{}, blobs: map[string]string{}}
}

func (g *rb5Registry) putManifest(repo, ref, mediaType string, payload []byte) string {
	dgst := digest.FromBytes(payload).String()
	g.mu.Lock()
	defer g.mu.Unlock()
	g.manifests[repo+"|"+ref] = rb5Stored{mediaType: mediaType, payload: payload}
	g.manifests[repo+"|"+dgst] = rb5Stored{mediaType: mediaType, payload: payload}
	return dgst
}

func (g *rb5Registry) putBlob(repo, dgst, data string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.blobs[repo+"|"+dgst] = data
}

func (g *rb5Registry) seen(sub string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, r := range g.requests {
		if strings.Contains(r, sub) {
			return true
		}
	}
	return false
}

func (g *rb5Registry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	g.requests = append(g.requests, r.Method+" "+r.URL.RequestURI())
	g.mu.Unlock()
	if g.override != nil && g.override(w, r) {
		return
	}
	p := strings.TrimPrefix(r.URL.Path, "/v2/")
	switch {
	case p == "":
		w.WriteHeader(http.StatusOK)
	case strings.Contains(p, "/manifests/"):
		g.serveManifest(w, r, p)
	case strings.Contains(p, "/blobs/uploads"):
		g.serveUpload(w, r, p)
	case strings.Contains(p, "/blobs/"):
		g.serveBlob(w, r, p)
	default:
		http.NotFound(w, r)
	}
}

func (g *rb5Registry) serveManifest(w http.ResponseWriter, r *http.Request, p string) {
	repo, ref, _ := strings.Cut(p, "/manifests/")
	key := repo + "|" + ref
	g.mu.Lock()
	stored, ok := g.manifests[key]
	g.mu.Unlock()
	switch r.Method {
	case http.MethodHead, http.MethodGet:
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", stored.mediaType)
		w.Header().Set("Docker-Content-Digest", digest.FromBytes(stored.payload).String())
		if r.Method == http.MethodGet {
			_, _ = w.Write(stored.payload)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		dgst := g.putManifest(repo, ref, r.Header.Get("Content-Type"), body)
		w.Header().Set("Docker-Content-Digest", dgst)
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		g.mu.Lock()
		delete(g.manifests, key)
		g.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (g *rb5Registry) serveBlob(w http.ResponseWriter, r *http.Request, p string) {
	repo, dgst, _ := strings.Cut(p, "/blobs/")
	g.mu.Lock()
	data, ok := g.blobs[repo+"|"+dgst]
	g.mu.Unlock()
	switch r.Method {
	case http.MethodHead:
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if rng := r.Header.Get("Range"); rng != "" {
			w.Header().Set("Content-Range", rng)
		}
		_, _ = w.Write([]byte(data))
	case http.MethodDelete:
		g.mu.Lock()
		delete(g.blobs, repo+"|"+dgst)
		g.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (g *rb5Registry) serveUpload(w http.ResponseWriter, r *http.Request, p string) {
	repo, rest, _ := strings.Cut(p, "/blobs/uploads")
	switch r.Method {
	case http.MethodPost:
		if mountDigest := r.URL.Query().Get("mount"); mountDigest != "" {
			g.putBlob(repo, mountDigest, "mounted")
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.Header().Set("Location", "/v2/"+repo+"/blobs/uploads/session-1")
		w.WriteHeader(http.StatusAccepted)
	case http.MethodPatch:
		w.Header().Set("Location", "/v2/"+repo+"/blobs/uploads/session-2")
		w.WriteHeader(http.StatusAccepted)
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		g.putBlob(repo, r.URL.Query().Get("digest"), string(body))
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		w.Header().Set("Range", "0-4")
		w.Header().Set("Location", "/v2/"+repo+"/blobs/uploads"+rest)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func rb5NewClient(t *testing.T, h http.Handler) (Client, string) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClientWithAuthorizer(srv.URL, null.NewAuthorizer(), true), srv.URL
}

func rb5ImageManifest(t *testing.T, mediaType string) ([]byte, string, string) {
	t.Helper()
	layer := digest.FromString(rb5Layer).String()
	cfg := digest.FromString(rb5Config).String()
	body, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     v1.MediaTypeImageManifest,
		"config":        map[string]any{"mediaType": v1.MediaTypeImageConfig, "digest": cfg, "size": len(rb5Config)},
		"layers": []map[string]any{
			{"mediaType": mediaType, "digest": layer, "size": len(rb5Layer)},
		},
	})
	require.NoError(t, err)
	return body, layer, cfg
}

func TestRb5NewClientPing(t *testing.T) {
	reg := rb5NewRegistry()
	srv := httptest.NewServer(reg)
	t.Cleanup(srv.Close)

	cli := NewClient(srv.URL, "user", "pass", true, true)
	require.NoError(t, cli.Ping(context.Background()))
	require.True(t, reg.seen("GET /v2/"))
}

func TestRb5PaginatedListings(t *testing.T) {
	ctx := context.Background()
	var server string
	cli, server := rb5NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		switch {
		case strings.HasSuffix(r.URL.Path, "/_catalog"):
			switch page {
			case "":
				w.Header().Set("Link", `</v2/_catalog?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode(map[string][]string{"repositories": {"a"}})
			case "2":
				w.Header().Set("Link", `<`+server+`/v2/_catalog?page=3>; rel="next"`)
				_ = json.NewEncoder(w).Encode(map[string][]string{"repositories": {"b"}})
			default:
				_ = json.NewEncoder(w).Encode(map[string][]string{"repositories": {"c"}})
			}
		case strings.HasSuffix(r.URL.Path, "/tags/list"):
			switch page {
			case "":
				w.Header().Set("Link", `</v2/lib/img/tags/list?page=2>; rel="next"`)
				_ = json.NewEncoder(w).Encode(map[string][]string{"tags": {"v1"}})
			default:
				_ = json.NewEncoder(w).Encode(map[string][]string{"tags": {"v2"}})
			}
		default:
			http.NotFound(w, r)
		}
	}))

	repos, err := cli.Catalog(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c"}, repos)

	tags, err := cli.ListTags(ctx, "lib/img")
	require.NoError(t, err)
	require.Equal(t, []string{"v1", "v2"}, tags)
}

func TestRb5ListingErrors(t *testing.T) {
	ctx := context.Background()

	broken, _ := rb5NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	_, err := broken.Catalog(ctx)
	require.Error(t, err)
	_, err = broken.ListTags(ctx, "lib/img")
	require.Error(t, err)

	failing, _ := rb5NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	_, err = failing.Catalog(ctx)
	require.Error(t, err)
	_, err = failing.ListTags(ctx, "lib/img")
	require.Error(t, err)
	_, _, err = failing.ManifestExist(ctx, "lib/img", "latest")
	require.Error(t, err)
	_, err = failing.BlobExist(ctx, "lib/img", "sha256:abc")
	require.Error(t, err)
	require.Error(t, failing.Ping(ctx))
}

func TestRb5ManifestOperations(t *testing.T) {
	ctx := context.Background()
	reg := rb5NewRegistry()
	cli, _ := rb5NewClient(t, reg)
	body, _, _ := rb5ImageManifest(t, v1.MediaTypeImageLayerGzip)

	pushed, err := cli.PushManifest(ctx, "lib/img", "v1", v1.MediaTypeImageManifest, body)
	require.NoError(t, err)
	require.Equal(t, digest.FromBytes(body).String(), pushed)

	m, dgst, err := cli.PullManifest(ctx, "lib/img", "v1")
	require.NoError(t, err)
	require.Equal(t, pushed, dgst)
	require.NotEmpty(t, m.References())

	_, _, err = cli.PullManifest(ctx, "lib/img", "v1", v1.MediaTypeImageManifest)
	require.NoError(t, err)

	_, _, err = cli.PullManifest(ctx, "lib/img", "missing")
	require.True(t, errors.IsErr(err, errors.NotFoundCode))

	// A manifest served with an unknown media type cannot be decoded.
	reg.putManifest("lib/img", "odd", "text/plain", []byte("garbage"))
	_, _, err = cli.PullManifest(ctx, "lib/img", "odd")
	require.Error(t, err)

	// Delete by tag resolves the digest first.
	require.NoError(t, cli.DeleteManifest(ctx, "lib/img", "v1"))
	require.True(t, reg.seen("DELETE /v2/lib/img/manifests/sha256:"))
	// Delete by digest goes straight to the delete call.
	require.NoError(t, cli.DeleteManifest(ctx, "lib/img", pushed))

	err = cli.DeleteManifest(ctx, "lib/img", "gone")
	require.True(t, errors.IsErr(err, errors.NotFoundCode))
}

func TestRb5ManifestFailures(t *testing.T) {
	ctx := context.Background()
	reg := rb5NewRegistry()
	body, _, _ := rb5ImageManifest(t, v1.MediaTypeImageLayerGzip)
	reg.putManifest("lib/img", "v1", v1.MediaTypeImageManifest, body)
	reg.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPut || r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		return false
	}
	cli, _ := rb5NewClient(t, reg)

	_, err := cli.PushManifest(ctx, "lib/img", "v2", v1.MediaTypeImageManifest, body)
	require.Error(t, err)
	require.Error(t, cli.DeleteManifest(ctx, "lib/img", "v1"))
	require.Error(t, cli.DeleteManifest(ctx, "lib/img", digest.FromBytes(body).String()))
}

func TestRb5BlobOperations(t *testing.T) {
	ctx := context.Background()
	reg := rb5NewRegistry()
	cli, _ := rb5NewClient(t, reg)
	dgst := digest.FromString(rb5Layer).String()
	reg.putBlob("lib/img", dgst, rb5Layer)

	size, rc, err := cli.PullBlob(ctx, "lib/img", dgst)
	require.NoError(t, err)
	require.Equal(t, int64(len(rb5Layer)), size)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, rb5Layer, string(data))

	size, rc, err = cli.PullBlobChunk(ctx, "lib/img", dgst, int64(len(rb5Layer)), 2, 5)
	require.NoError(t, err)
	require.Equal(t, int64(len(rb5Layer)), size)
	require.NoError(t, rc.Close())
	require.True(t, reg.seen("GET /v2/lib/img/blobs/"))

	_, _, err = cli.PullBlob(ctx, "lib/img", "sha256:missing")
	require.True(t, errors.IsErr(err, errors.NotFoundCode))
	_, _, err = cli.PullBlobChunk(ctx, "lib/img", "sha256:missing", 10, 0, 5)
	require.True(t, errors.IsErr(err, errors.NotFoundCode))

	require.NoError(t, cli.DeleteBlob(ctx, "lib/img", dgst))
	exist, err := cli.BlobExist(ctx, "lib/img", dgst)
	require.NoError(t, err)
	require.False(t, exist)

	require.NoError(t, cli.MountBlob(ctx, "lib/src", "sha256:mounted", "lib/dst"))
	exist, err = cli.BlobExist(ctx, "lib/dst", "sha256:mounted")
	require.NoError(t, err)
	require.True(t, exist)
}

func TestRb5BlobFailures(t *testing.T) {
	ctx := context.Background()
	cli, _ := rb5NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	require.Error(t, cli.DeleteBlob(ctx, "lib/img", "sha256:abc"))
	require.Error(t, cli.MountBlob(ctx, "lib/src", "sha256:abc", "lib/dst"))
	require.Error(t, cli.PushBlob(ctx, "lib/img", "sha256:abc", 3, strings.NewReader("abc")))
	_, _, err := cli.PullBlobChunk(ctx, "lib/img", "sha256:abc", 3, 0, 2)
	require.Error(t, err)
	_, _, err = cli.PushBlobChunk(ctx, "lib/img", "sha256:abc", 3, strings.NewReader("abc"), 0, 2, "")
	require.Error(t, err)
}

func TestRb5PushBlob(t *testing.T) {
	ctx := context.Background()
	reg := rb5NewRegistry()
	cli, _ := rb5NewClient(t, reg)
	dgst := digest.FromString(rb5Layer).String()

	require.NoError(t, cli.PushBlob(ctx, "lib/img", dgst, int64(len(rb5Layer)), strings.NewReader(rb5Layer)))
	exist, err := cli.BlobExist(ctx, "lib/img", dgst)
	require.NoError(t, err)
	require.True(t, exist)

	// The final upload call fails after the session was opened.
	reg.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		return false
	}
	require.Error(t, cli.PushBlob(ctx, "lib/img", "sha256:other", 3, strings.NewReader("abc")))
}

func TestRb5PushBlobChunk(t *testing.T) {
	ctx := context.Background()
	reg := rb5NewRegistry()
	cli, _ := rb5NewClient(t, reg)
	dgst := digest.FromString("0123456789").String()

	// First chunk opens the session and is sent with PATCH.
	location, end, err := cli.PushBlobChunk(ctx, "lib/img", dgst, 10, strings.NewReader("01234"), 0, 4, "")
	require.NoError(t, err)
	require.Equal(t, int64(4), end)
	require.Equal(t, "/v2/lib/img/blobs/uploads/session-2", location)
	require.True(t, reg.seen("PATCH /v2/lib/img/blobs/uploads/session-1"))

	// The last chunk is sent with PUT and carries the digest.
	_, end, err = cli.PushBlobChunk(ctx, "lib/img", dgst, 10, strings.NewReader("56789"), 5, 9, location)
	require.NoError(t, err)
	require.Equal(t, int64(9), end)
	require.True(t, reg.seen("PUT /v2/lib/img/blobs/uploads/session-2?digest="))

	// A failed chunk reports the server side upload status.
	reg.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		return false
	}
	newLocation, newEnd, err := cli.PushBlobChunk(ctx, "lib/img", dgst, 10, strings.NewReader("56"), 5, 6, location)
	require.Error(t, err)
	require.Equal(t, int64(4), newEnd)
	require.NotEmpty(t, newLocation)

	// When the status cannot be read either, the start offset is rewound.
	reg.override = func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == http.MethodPatch || r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return true
		}
		return false
	}
	_, newEnd, err = cli.PushBlobChunk(ctx, "lib/img", dgst, 10, strings.NewReader("56"), 5, 6, location)
	require.ErrorContains(t, err, "failed to get upload status")
	require.Equal(t, int64(4), newEnd)

	// An unparsable location is rejected before any request.
	_, _, err = cli.PushBlobChunk(ctx, "lib/img", dgst, 10, strings.NewReader("56"), 5, 6, "http://[::1")
	require.Error(t, err)
}

func TestRb5UploadStatus(t *testing.T) {
	ctx := context.Background()
	reg := rb5NewRegistry()
	cli, base := rb5NewClient(t, reg)
	impl, ok := cli.(*client)
	require.True(t, ok)

	location, end, err := impl.getUploadStatus(ctx, base+"/v2/lib/img/blobs/uploads/session-1")
	require.NoError(t, err)
	require.Equal(t, int64(4), end)
	require.Contains(t, location, "session-1")

	reg.override = func(w http.ResponseWriter, _ *http.Request) bool {
		w.Header().Set("Range", "garbage")
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	_, end, err = impl.getUploadStatus(ctx, base+"/v2/lib/img/blobs/uploads/session-1")
	require.Error(t, err)
	require.Equal(t, int64(-1), end)

	_, end, err = impl.getUploadStatus(ctx, "http://[::1")
	require.Error(t, err)
	require.Equal(t, int64(-1), end)
}

func TestRb5CopyArtifact(t *testing.T) {
	ctx := context.Background()
	body, layerDigest, cfgDigest := rb5ImageManifest(t, v1.MediaTypeImageLayerGzip)

	newEnv := func(t *testing.T) (*rb5Registry, Client, string) {
		t.Helper()
		reg := rb5NewRegistry()
		cli, _ := rb5NewClient(t, reg)
		dgst := reg.putManifest("lib/src", "v1", v1.MediaTypeImageManifest, body)
		return reg, cli, dgst
	}

	t.Run("copies manifest and mounts blobs", func(t *testing.T) {
		reg, cli, dgst := newEnv(t)
		require.NoError(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
		require.True(t, reg.seen("POST /v2/lib/dst/blobs/uploads/?mount="+layerDigest))
		require.True(t, reg.seen("POST /v2/lib/dst/blobs/uploads/?mount="+cfgDigest))
		exist, desc, err := cli.ManifestExist(ctx, "lib/dst", "v1")
		require.NoError(t, err)
		require.True(t, exist)
		require.Equal(t, dgst, string(desc.Digest))
	})

	t.Run("identical manifest is a no-op", func(t *testing.T) {
		reg, cli, dgst := newEnv(t)
		reg.putManifest("lib/dst", "v1", v1.MediaTypeImageManifest, body)
		require.Equal(t, dgst, digest.FromBytes(body).String())
		require.NoError(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
		require.False(t, reg.seen("PUT "))
	})

	t.Run("different digest needs override", func(t *testing.T) {
		reg, cli, _ := newEnv(t)
		reg.putManifest("lib/dst", "v1", v1.MediaTypeImageManifest, []byte(`{"schemaVersion":2}`))
		err := cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false)
		require.True(t, errors.IsErr(err, errors.PreconditionCode))
		require.NoError(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", true))
	})

	t.Run("existing blobs are not mounted again", func(t *testing.T) {
		reg, cli, _ := newEnv(t)
		reg.putBlob("lib/dst", layerDigest, rb5Layer)
		reg.putBlob("lib/dst", cfgDigest, rb5Config)
		require.NoError(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
		require.False(t, reg.seen("mount="))
	})

	t.Run("foreign layers are skipped", func(t *testing.T) {
		reg := rb5NewRegistry()
		cli, _ := rb5NewClient(t, reg)
		foreign, _, _ := rb5ImageManifest(t, "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip")
		reg.putManifest("lib/src", "v1", v1.MediaTypeImageManifest, foreign)
		require.NoError(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
		require.False(t, reg.seen("mount="+layerDigest))
	})

	t.Run("index copies child manifests", func(t *testing.T) {
		reg, cli, childDigest := newEnv(t)
		index, err := json.Marshal(map[string]any{
			"schemaVersion": 2,
			"mediaType":     rb5IndexMedia,
			"manifests": []map[string]any{
				{"mediaType": v1.MediaTypeImageManifest, "digest": childDigest, "size": len(body)},
			},
		})
		require.NoError(t, err)
		reg.putManifest("lib/src", "idx", rb5IndexMedia, index)
		require.NoError(t, cli.Copy(ctx, "lib/src", "idx", "lib/dst", "idx", false))
		exist, _, err := cli.ManifestExist(ctx, "lib/dst", childDigest)
		require.NoError(t, err)
		require.True(t, exist)
	})

	t.Run("source manifest missing", func(t *testing.T) {
		_, cli, _ := newEnv(t)
		err := cli.Copy(ctx, "lib/src", "missing", "lib/dst", "v1", false)
		require.True(t, errors.IsErr(err, errors.NotFoundCode))
	})

	t.Run("destination lookup fails", func(t *testing.T) {
		reg, cli, _ := newEnv(t)
		reg.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodHead && strings.Contains(r.URL.Path, "/lib/dst/manifests/") {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			}
			return false
		}
		require.Error(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
	})

	t.Run("blob lookup fails", func(t *testing.T) {
		reg, cli, _ := newEnv(t)
		reg.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodHead && strings.Contains(r.URL.Path, "/lib/dst/blobs/") {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			}
			return false
		}
		require.Error(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
	})

	t.Run("mount fails", func(t *testing.T) {
		reg, cli, _ := newEnv(t)
		reg.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			}
			return false
		}
		require.Error(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
	})

	t.Run("push fails", func(t *testing.T) {
		reg, cli, _ := newEnv(t)
		reg.override = func(w http.ResponseWriter, r *http.Request) bool {
			if r.Method == http.MethodPut {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			}
			return false
		}
		require.Error(t, cli.Copy(ctx, "lib/src", "v1", "lib/dst", "v1", false))
	})
}

type rb5Interceptor struct{ err error }

func (i rb5Interceptor) Intercept(req *http.Request) error {
	req.Header.Set("X-Rb5", "intercepted")
	return i.err
}

type rb5Authorizer struct{ err error }

func (a rb5Authorizer) Modify(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer rb5")
	return a.err
}

func TestRb5ClientDo(t *testing.T) {
	ctx := context.Background()
	var gotUA, gotAuth, gotIntercept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		gotIntercept = r.Header.Get("X-Rb5")
		switch r.URL.Path {
		case "/forbidden":
			w.WriteHeader(http.StatusForbidden)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/limited":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/unauthorized":
			w.WriteHeader(http.StatusUnauthorized)
		case "/other":
			w.WriteHeader(http.StatusTeapot)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	newReq := func(path string) *http.Request {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+path, nil)
		require.NoError(t, err)
		return req
	}

	cli := NewClientWithAuthorizer(srv.URL, rb5Authorizer{}, true, rb5Interceptor{})
	resp, err := cli.Do(newReq("/ok"))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, UserAgent, gotUA)
	require.Equal(t, "Bearer rb5", gotAuth)
	require.Equal(t, "intercepted", gotIntercept)

	codes := map[string]string{
		"/forbidden":    errors.ForbiddenCode,
		"/missing":      errors.NotFoundCode,
		"/limited":      errors.RateLimitCode,
		"/unauthorized": errors.UnAuthorizedCode,
		"/other":        errors.GeneralCode,
	}
	for path, code := range codes {
		err = rb5Send(cli, newReq(path))
		require.True(t, errors.IsErr(err, code), path)
	}

	failingInterceptor := NewClientWithAuthorizer(srv.URL, nil, true, rb5Interceptor{err: errRb5Boom})
	err = rb5Send(failingInterceptor, newReq("/ok"))
	require.ErrorIs(t, err, errRb5Boom)

	failingAuthorizer := NewClientWithAuthorizer(srv.URL, rb5Authorizer{err: errRb5Boom}, true)
	err = rb5Send(failingAuthorizer, newReq("/ok"))
	require.ErrorIs(t, err, errRb5Boom)

	// Without an authorizer the request is sent as is.
	anonymous := NewClientWithAuthorizer(srv.URL, nil, true)
	resp, err = anonymous.Do(newReq("/ok"))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	srv.Close()
	err = rb5Send(cli, newReq("/ok"))
	require.Error(t, err)
}

func TestRb5ClientFiles(t *testing.T) {
	ctx := context.Background()
	cli, base := rb5NewClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte("file body"))
		}
	}))

	headers, rc, err := cli.GetFileFromURL(ctx, base+"/some/file.txt")
	require.NoError(t, err)
	require.NotNil(t, headers)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, "file body", string(data))

	_, _, err = cli.GetFileFromURL(ctx, base+"/missing")
	require.True(t, errors.IsErr(err, errors.NotFoundCode))
	_, _, err = cli.GetFileFromURL(ctx, "http://[::1")
	require.Error(t, err)

	_, _, err = cli.GetFile(ctx, "/missing/file")
	require.True(t, errors.IsErr(err, errors.NotFoundCode))
	_, err = cli.HeadFile(ctx, "/missing/file")
	require.True(t, errors.IsErr(err, errors.NotFoundCode))
}

func TestRb5BuildFileURLInvalidEndpoint(t *testing.T) {
	require.Equal(t, "http://[::1/a/b", buildFileURL("http://[::1/", "/a/b"))
}

// rb5Send performs the request and closes the response body.
func rb5Send(c Client, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

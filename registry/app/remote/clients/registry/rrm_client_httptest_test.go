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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/remote/clients/registry/auth/null"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func rrmNewTestClient(t *testing.T, handler http.Handler) Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClientWithAuthorizer(srv.URL, null.NewAuthorizer(), true)
}

func TestRrmClientPing(t *testing.T) {
	cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	require.NoError(t, cli.Ping(context.Background()))
}

func TestRrmClientCatalog(t *testing.T) {
	cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/_catalog") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string][]string{
			"repositories": {"a/b", "c/d"},
		})
	}))
	repos, err := cli.Catalog(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"a/b", "c/d"}, repos)
}

func TestRrmClientListTags(t *testing.T) {
	cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/tags/list") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string][]string{"tags": {"v1", "latest"}})
	}))
	tags, err := cli.ListTags(context.Background(), "lib/img")
	require.NoError(t, err)
	require.Equal(t, []string{"v1", "latest"}, tags)
}

func TestRrmClientManifestExist(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Docker-Content-Digest", "sha256:abc")
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.Header().Set("Content-Length", "42")
			w.WriteHeader(http.StatusOK)
		}))
		exist, desc, err := cli.ManifestExist(context.Background(), "lib/img", "latest")
		require.NoError(t, err)
		require.True(t, exist)
		require.NotNil(t, desc)
		require.Equal(t, digest.Digest("sha256:abc"), desc.Digest)
	})

	t.Run("not found", func(t *testing.T) {
		cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		exist, desc, err := cli.ManifestExist(context.Background(), "lib/img", "missing")
		require.NoError(t, err)
		require.False(t, exist)
		require.Nil(t, desc)
	})
}

func TestRrmClientBlobExist(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		exist, err := cli.BlobExist(context.Background(), "lib/img", "sha256:abc")
		require.NoError(t, err)
		require.True(t, exist)
	})

	t.Run("missing", func(t *testing.T) {
		cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		exist, err := cli.BlobExist(context.Background(), "lib/img", "sha256:missing")
		require.NoError(t, err)
		require.False(t, exist)
	})
}

func TestRrmClientGetFileAndHeadFile(t *testing.T) {
	body := "file-payload"
	cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(body))
		case http.MethodHead:
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))

	ctx := context.Background()
	headers, rc, err := cli.GetFile(ctx, "path/to/file.txt")
	require.NoError(t, err)
	require.NotNil(t, headers)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, body, string(data))

	headHeaders, err := cli.HeadFile(ctx, "path/to/file.txt")
	require.NoError(t, err)
	require.NotNil(t, headHeaders)
}

func TestRrmClientGetURL(t *testing.T) {
	cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	url := cli.GetURL(context.Background(), "artifacts/pkg/1.0.0/file.tgz")
	require.Contains(t, url, "artifacts/pkg")
}

func TestRrmClientUnauthorizedError(t *testing.T) {
	cli := rrmNewTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[]}`))
	}))
	err := cli.Ping(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

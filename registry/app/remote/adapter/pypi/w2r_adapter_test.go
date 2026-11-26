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

package pypi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

const w2rSimplePage = `<!DOCTYPE html><html><head><meta name="pypi:repository-version" content="1.0">` +
	`<title>Links for demo</title></head><body>` +
	`<a href="../../files/demo-1.0.0.tar.gz#sha256=abc" data-requires-python="&gt;=3.8">demo-1.0.0.tar.gz</a>` +
	`<a href="https://cdn.example.com/demo-1.1.0-py3-none-any.whl">demo-1.1.0-py3-none-any.whl</a>` +
	`</body></html>`

func w2rNewAdapter(t *testing.T, handler http.HandlerFunc, suffix string) *adapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	reg := types.UpstreamProxy{
		RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypePYTHON,
	}
	if suffix != "" {
		reg.Config = &types.RegistryConfig{RemoteUrlSuffix: suffix}
	}
	a, err := newAdapter(context.Background(), refcache.SpaceFinder{}, reg, nil)
	require.NoError(t, err)
	typed, ok := a.(*adapter)
	require.True(t, ok)
	return typed
}

func w2rPyPIHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/simple/demo"), strings.HasPrefix(r.URL.Path, "/custom/demo"):
		_, _ = io.WriteString(w, w2rSimplePage)
	case strings.HasPrefix(r.URL.Path, "/files/"):
		_, _ = io.WriteString(w, "file:"+r.URL.Path)
	case r.URL.Path == "/pypi/demo/1.0.0/json":
		_, _ = io.WriteString(w, `{"info":{"name":"demo","version":"1.0.0","summary":"demo package"}}`)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestW2rPyPIGetMetadata(t *testing.T) {
	for _, suffix := range []string{"", "/custom/"} {
		a := w2rNewAdapter(t, w2rPyPIHandler, suffix)

		meta, err := a.GetMetadata(context.Background(), "demo")
		require.NoError(t, err, suffix)
		require.Equal(t, "Links for demo", meta.Title)
		require.Equal(t, "1.0", meta.Content)
		require.Len(t, meta.Packages, 2)
		require.Equal(t, "demo-1.0.0.tar.gz", meta.Packages[0].Name)
		require.Equal(t, ">=3.8", meta.Packages[0].RequiresPython())
	}

	a := w2rNewAdapter(t, w2rPyPIHandler, "")
	_, err := a.GetMetadata(context.Background(), "missing")
	require.Error(t, err)
}

func TestW2rPyPIGetMetadataRejectsInvalidPackages(t *testing.T) {
	a := w2rNewAdapter(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `<html><body><a href="demo-1.0.0.tar.gz"></a></body></html>`)
	}, "")
	_, err := a.GetMetadata(context.Background(), "demo")
	require.Error(t, err)
}

func TestW2rPyPIGetPackage(t *testing.T) {
	a := w2rNewAdapter(t, w2rPyPIHandler, "")

	rc, err := a.GetPackage(context.Background(), "demo", "demo-1.0.0.tar.gz")
	require.NoError(t, err)
	body, readErr := io.ReadAll(rc)
	require.NoError(t, readErr)
	require.NoError(t, rc.Close())
	require.True(t, strings.HasPrefix(string(body), "file:/files/demo-1.0.0.tar.gz"), string(body))

	_, err = a.GetPackage(context.Background(), "demo", "unknown.tar.gz")
	require.Error(t, err)
	_, err = a.GetPackage(context.Background(), "missing", "demo-1.0.0.tar.gz")
	require.Error(t, err)
}

func TestW2rPyPIGetJSON(t *testing.T) {
	a := w2rNewAdapter(t, w2rPyPIHandler, "")

	meta, err := a.GetJSON(context.Background(), "demo", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "demo", meta.Name)
	require.Equal(t, "demo package", meta.Summary)

	_, err = a.GetJSON(context.Background(), "demo", "9.9.9")
	require.Error(t, err)
}

func TestW2rPyPIParsers(t *testing.T) {
	meta, err := ParseMetadata(context.Background(), io.NopCloser(strings.NewReader(`{"info":{"name":"x"}}`)))
	require.NoError(t, err)
	require.Equal(t, "x", meta.Name)

	// Invalid JSON is tolerated and yields empty metadata.
	empty, err := ParseMetadata(context.Background(), io.NopCloser(strings.NewReader(`{`)))
	require.NoError(t, err)
	require.Empty(t, empty.Name)

	simple, err := ParsePyPISimple(io.NopCloser(strings.NewReader(w2rSimplePage)), "https://pypi.example.com/simple/demo/")
	require.NoError(t, err)
	require.Len(t, simple.Packages, 2)
	require.Equal(t, "https://pypi.example.com/files/demo-1.0.0.tar.gz#sha256=abc", simple.Packages[0].URL())
	require.Equal(t, "https://cdn.example.com/demo-1.1.0-py3-none-any.whl", simple.Packages[1].URL())

	require.NoError(t, validateMetadata(simple))
}

func TestW2rPyPIFactory(t *testing.T) {
	f := &factory{}
	got, err := f.Create(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: "https://pypi.example.invalid", RepoAuthType: string(artifact.AuthTypeAnonymous),
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	_, err = f.Create(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{RepoAuthType: "weird"}, nil)
	require.Error(t, err)
}

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

package npmjs

import (
	"context"
	"fmt"
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

func w2rNewAdapter(t *testing.T, status map[string]int) *adapter {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if code, ok := status[r.URL.Path]; ok {
			w.WriteHeader(code)
			return
		}
		switch r.URL.Path {
		case "/demo":
			_, _ = fmt.Fprintf(w, `{"name":"demo","versions":{"1.0.0":{"name":"demo","version":"1.0.0",`+
				`"dist":{"tarball":"%s/tarballs/demo-1.0.0.tgz"}}}}`, srv.URL)
		case "/broken":
			_, _ = io.WriteString(w, `{not json`)
		case "/tarballs/demo-1.0.0.tgz":
			_, _ = io.WriteString(w, "tarball-bytes")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	a, err := newAdapter(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeNPM,
	}, nil)
	require.NoError(t, err)
	typed, ok := a.(*adapter)
	require.True(t, ok)
	return typed
}

func TestW2rNPMGetPackageMetadata(t *testing.T) {
	a := w2rNewAdapter(t, nil)

	meta, err := a.GetPackageMetadata(context.Background(), "demo")
	require.NoError(t, err)
	require.Equal(t, "demo", meta.Name)
	require.Contains(t, meta.Versions, "1.0.0")

	_, err = a.GetPackageMetadata(context.Background(), "broken")
	require.Error(t, err)
}

func TestW2rNPMGetPackageMetadataMapsUpstreamStatus(t *testing.T) {
	a := w2rNewAdapter(t, map[string]int{"/hidden": http.StatusForbidden})

	_, err := a.GetPackageMetadata(context.Background(), "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get package metadata missing")

	_, err = a.GetPackageMetadata(context.Background(), "hidden")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get package metadata hidden")
}

func TestW2rNPMGetPackage(t *testing.T) {
	a := w2rNewAdapter(t, nil)

	rc, err := a.GetPackage(context.Background(), "demo", "1.0.0")
	require.NoError(t, err)
	body, readErr := io.ReadAll(rc)
	require.NoError(t, readErr)
	require.NoError(t, rc.Close())
	require.Equal(t, "tarball-bytes", string(body))

	_, err = a.GetPackage(context.Background(), "demo", "2.0.0")
	require.Error(t, err)
	_, err = a.GetPackage(context.Background(), "missing", "1.0.0")
	require.Error(t, err)
}

func TestW2rNPMGetPackageMapsTarballFailures(t *testing.T) {
	a := w2rNewAdapter(t, map[string]int{"/tarballs/demo-1.0.0.tgz": http.StatusNotFound})
	_, err := a.GetPackage(context.Background(), "demo", "1.0.0")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get package file")

	forbidden := w2rNewAdapter(t, map[string]int{"/tarballs/demo-1.0.0.tgz": http.StatusForbidden})
	_, err = forbidden.GetPackage(context.Background(), "demo", "1.0.0")
	require.Error(t, err)
}

func TestW2rNPMParseMetadataAndFactory(t *testing.T) {
	meta, err := ParseNPMMetadata(io.NopCloser(strings.NewReader(`{"name":"x","versions":{}}`)))
	require.NoError(t, err)
	require.Equal(t, "x", meta.Name)
	_, err = ParseNPMMetadata(io.NopCloser(strings.NewReader(`[`)))
	require.Error(t, err)

	f := &factory{}
	got, err := f.Create(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: "https://registry.npmjs.invalid", RepoAuthType: string(artifact.AuthTypeAnonymous),
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	_, err = f.Create(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{RepoAuthType: "weird"}, nil)
	require.Error(t, err)
}

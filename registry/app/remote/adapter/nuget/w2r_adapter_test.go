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

package nuget

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
	nugettypes "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func w2rNewAdapter(t *testing.T, handler http.HandlerFunc) (*adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	a, err := newAdapter(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL + "/v3/index.json", RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeNUGET,
	}, nil)
	require.NoError(t, err)
	typed, ok := a.(*adapter)
	require.True(t, ok)
	return typed, srv
}

func w2rBody(t *testing.T, rc io.ReadCloser, err error) string {
	t.Helper()
	require.NoError(t, err)
	defer rc.Close()
	b, readErr := io.ReadAll(rc)
	require.NoError(t, readErr)
	return string(b)
}

// w2rRead lets a call returning (reader, error) be passed straight in: w2rRead(t)(a.Call()).
func w2rRead(t *testing.T) func(io.ReadCloser, error) string {
	return func(rc io.ReadCloser, err error) string { return w2rBody(t, rc, err) }
}

func w2rNugetServer(srvURL *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v3/index.json":
			_, _ = fmt.Fprintf(w, `{"version":"3.0.0","resources":[`+
				`{"@id":"%[1]s/reg/","@type":"RegistrationsBaseUrl"},`+
				`{"@id":"%[1]s/reg-gz/","@type":"RegistrationsBaseUrl/3.6.0"},`+
				`{"@id":"%[1]s/flat/","@type":"PackageBaseAddress/3.0.0"},`+
				`{"@id":"%[1]s/search","@type":"SearchQueryService"}]}`, *srvURL)
		case strings.HasSuffix(r.URL.Path, "/$count"):
			_, _ = io.WriteString(w, " 42\n")
		default:
			_, _ = io.WriteString(w, "payload:"+r.URL.Path)
		}
	}
}

func TestW2rNugetAdapterV3Calls(t *testing.T) {
	var srvURL string
	a, srv := w2rNewAdapter(t, w2rNugetServer(&srvURL))
	srvURL = srv.URL
	ctx := context.Background()

	endpoints, err := a.GetServiceEndpoint(ctx)
	require.NoError(t, err)
	require.Len(t, endpoints.Resources, 4)

	body := w2rRead(t)(a.GetPackageMetadata(ctx, "pkg", ""))
	require.Equal(t, "payload:/reg-gz/pkg/index.json", body, "the highest resource type version wins")

	body = w2rRead(t)(a.GetPackageMetadata(ctx, "pkg", srv.URL+"/direct/meta.json"))
	require.Equal(t, "payload:/direct/meta.json", body)

	body = w2rRead(t)(a.GetPackage(ctx, "pkg", "1.0.0", "", "pkg.1.0.0.nupkg"))
	require.Equal(t, "payload:/flat/pkg/1.0.0/pkg.1.0.0.nupkg", body)
	body = w2rRead(t)(a.GetPackage(ctx, "pkg", "1.0.0", srv.URL+"/direct/file.nupkg", ""))
	require.Equal(t, "payload:/direct/file.nupkg", body)

	body = w2rRead(t)(a.ListPackageVersion(ctx, "pkg"))
	require.Equal(t, "payload:/flat/pkg/index.json", body)

	body = w2rRead(t)(a.SearchPackage(ctx, "term", 5, 10))
	require.Equal(t, "payload:/search", body)
}

func TestW2rNugetAdapterV2Calls(t *testing.T) {
	var srvURL string
	a, _ := w2rNewAdapter(t, w2rNugetServer(&srvURL))
	ctx := context.Background()

	for name, call := range map[string]func() (io.ReadCloser, error){
		"version metadata": func() (io.ReadCloser, error) { return a.GetPackageVersionMetadataV2(ctx, "pkg", "1.0.0") },
		"list versions":    func() (io.ReadCloser, error) { return a.ListPackageVersionV2(ctx, "pkg") },
		"search":           func() (io.ReadCloser, error) { return a.SearchPackageV2(ctx, "term", 5, 10) },
	} {
		rc, err := call()
		require.NoError(t, err, name)
		require.NotEmpty(t, w2rBody(t, rc, nil), name)
	}

	count, err := a.CountPackageV2(ctx, "term")
	require.NoError(t, err)
	require.Equal(t, int64(42), count)
	count, err = a.CountPackageVersionV2(ctx, "pkg")
	require.NoError(t, err)
	require.Equal(t, int64(42), count)
}

func TestW2rNugetAdapterCountParseFailure(t *testing.T) {
	a, _ := w2rNewAdapter(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "not-a-number") })
	_, err := a.CountPackageV2(context.Background(), "x")
	require.Error(t, err)
	_, err = a.CountPackageVersionV2(context.Background(), "x")
	require.Error(t, err)
}

func TestW2rNugetAdapterUpstreamFailures(t *testing.T) {
	a, _ := w2rNewAdapter(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	ctx := context.Background()

	_, err := a.GetServiceEndpoint(ctx)
	require.Error(t, err)
	_, err = a.GetPackageMetadata(ctx, "pkg", "")
	require.Error(t, err)
	_, err = a.GetPackageMetadata(ctx, "pkg", "http://127.0.0.1:1/unreachable")
	require.Error(t, err)
	_, err = a.GetPackage(ctx, "pkg", "1", "", "f")
	require.Error(t, err)
	_, err = a.ListPackageVersion(ctx, "pkg")
	require.Error(t, err)
	_, err = a.SearchPackage(ctx, "t", 1, 0)
	require.Error(t, err)
	_, err = a.GetPackageVersionMetadataV2(ctx, "pkg", "1")
	require.Error(t, err)
	_, err = a.ListPackageVersionV2(ctx, "pkg")
	require.Error(t, err)
	_, err = a.SearchPackageV2(ctx, "t", 1, 0)
	require.Error(t, err)
	_, err = a.CountPackageV2(ctx, "t")
	require.Error(t, err)
	_, err = a.CountPackageVersionV2(ctx, "pkg")
	require.Error(t, err)
}

func TestW2rNugetResourceLookup(t *testing.T) {
	endpoints := &nugettypes.ServiceEndpoint{Resources: []nugettypes.Resource{
		{ID: "a", Type: "PackageBaseAddress/3.0.0"}, {ID: "b", Type: "PackageBaseAddress/3.1.0"},
	}}
	got, err := getResourceByTypePrefix(endpoints, "PackageBaseAddress")
	require.NoError(t, err)
	require.Equal(t, "b", got)
	_, err = getResourceByTypePrefix(endpoints, "SearchQueryService")
	require.Error(t, err)

	parsed, err := ParseServiceEndpointResponse(io.NopCloser(strings.NewReader(`{"version":"3","resources":[]}`)))
	require.NoError(t, err)
	require.Equal(t, "3", parsed.Version)
	_, err = ParseServiceEndpointResponse(io.NopCloser(strings.NewReader(`{`)))
	require.Error(t, err)
}

func TestW2rNugetFactoryCreate(t *testing.T) {
	f := &factory{}
	got, err := f.Create(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: "https://example.invalid", RepoAuthType: string(artifact.AuthTypeAnonymous),
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, got)

	_, err = f.Create(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{RepoAuthType: "weird"}, nil)
	require.Error(t, err, "an unsupported auth type fails credential lookup")
}

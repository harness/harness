// Copyright 2026 Harness, Inc.
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
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	nugetctrl "github.com/harness/gitness/registry/app/api/controller/pkg/nuget"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	nugetpkg "github.com/harness/gitness/registry/app/pkg/nuget"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/request"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type rchPkgHandler struct {
	packages.Handler
	info pkg.ArtifactInfo
	err  error
}

func (h rchPkgHandler) GetArtifactInfo(_ *http.Request) (pkg.ArtifactInfo, error) {
	return h.info, h.err
}

func rchBasePackagesHandler() packages.Handler {
	return packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
}

type rchNugetCtrl struct {
	nugetctrl.Controller
	err      error
	redirect string
	noBody   bool
	lastType nugetpkg.FileBundleType
}

func (c *rchNugetCtrl) base() nugetctrl.BaseResponse {
	return nugetctrl.BaseResponse{
		Error:           c.err,
		ResponseHeaders: &commons.ResponseHeaders{Headers: map[string]string{"X-Rch": "1"}, Code: http.StatusOK},
	}
}

func (c *rchNugetCtrl) UploadPackage(
	_ context.Context, _ nugettype.ArtifactInfo, _ io.ReadCloser, t nugetpkg.FileBundleType,
) *nugetctrl.PutArtifactResponse {
	c.lastType = t
	return &nugetctrl.PutArtifactResponse{BaseResponse: c.base()}
}

func (c *rchNugetCtrl) DownloadPackage(_ context.Context, _ nugettype.ArtifactInfo) *nugetctrl.GetArtifactResponse {
	resp := &nugetctrl.GetArtifactResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
	if !c.noBody {
		resp.ReadCloser = io.NopCloser(strings.NewReader("nupkg"))
	}
	return resp
}

func (c *rchNugetCtrl) DeletePackage(_ context.Context, _ nugettype.ArtifactInfo) *nugetctrl.DeleteArtifactResponse {
	return &nugetctrl.DeleteArtifactResponse{BaseResponse: c.base()}
}

func (c *rchNugetCtrl) GetServiceEndpoint(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.GetServiceEndpointResponse {
	return &nugetctrl.GetServiceEndpointResponse{
		BaseResponse:    c.base(),
		ServiceEndpoint: &nugettype.ServiceEndpoint{Version: "3.0.0"},
	}
}

func (c *rchNugetCtrl) GetServiceEndpointV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.GetServiceEndpointV2Response {
	return &nugetctrl.GetServiceEndpointV2Response{
		BaseResponse:    c.base(),
		ServiceEndpoint: &nugettype.ServiceEndpointV2{Base: "http://base"},
	}
}

func (c *rchNugetCtrl) GetServiceMetadataV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.GetServiceMetadataV2Response {
	return &nugetctrl.GetServiceMetadataV2Response{
		BaseResponse:    c.base(),
		ServiceMetadata: &nugettype.ServiceMetadataV2{Version: "1.0"},
	}
}

func (c *rchNugetCtrl) ListPackageVersion(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.ListPackageVersionResponse {
	return &nugetctrl.ListPackageVersionResponse{
		BaseResponse:   c.base(),
		PackageVersion: &nugettype.PackageVersion{Versions: []string{"1.0.0"}},
	}
}

func (c *rchNugetCtrl) ListPackageVersionV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.ListPackageVersionV2Response {
	return &nugetctrl.ListPackageVersionV2Response{
		BaseResponse: c.base(),
		FeedResponse: &nugettype.FeedResponse{ID: "feed-list"},
	}
}

func (c *rchNugetCtrl) GetPackageMetadata(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.GetPackageMetadataResponse {
	return &nugetctrl.GetPackageMetadataResponse{
		BaseResponse:         c.base(),
		RegistrationResponse: &nugettype.RegistrationIndexResponse{Count: 2},
	}
}

func (c *rchNugetCtrl) GetPackageVersionMetadataV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.GetPackageVersionMetadataV2Response {
	return &nugetctrl.GetPackageVersionMetadataV2Response{
		BaseResponse:      c.base(),
		FeedEntryResponse: &nugettype.FeedEntryResponse{ID: "entry-id"},
	}
}

func (c *rchNugetCtrl) GetPackageVersionMetadata(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.GetPackageVersionMetadataResponse {
	return &nugetctrl.GetPackageVersionMetadataResponse{
		BaseResponse:             c.base(),
		RegistrationLeafResponse: &nugettype.RegistrationLeafResponse{Listed: true},
	}
}

func (c *rchNugetCtrl) CountPackageVersionV2(
	_ context.Context, _ nugettype.ArtifactInfo,
) *nugetctrl.EntityCountResponse {
	return &nugetctrl.EntityCountResponse{BaseResponse: c.base(), Count: 4}
}

func (c *rchNugetCtrl) CountPackageV2(
	_ context.Context, _ nugettype.ArtifactInfo, _ string,
) *nugetctrl.EntityCountResponse {
	return &nugetctrl.EntityCountResponse{BaseResponse: c.base(), Count: 7}
}

func (c *rchNugetCtrl) SearchPackage(
	_ context.Context, _ nugettype.ArtifactInfo, _ string, _, _ int,
) *nugetctrl.SearchPackageResponse {
	return &nugetctrl.SearchPackageResponse{
		BaseResponse:   c.base(),
		SearchResponse: &nugettype.SearchResultResponse{TotalHits: 5},
	}
}

func (c *rchNugetCtrl) SearchPackageV2(
	_ context.Context, _ nugettype.ArtifactInfo, _ string, _, _ int,
) *nugetctrl.SearchPackageV2Response {
	return &nugetctrl.SearchPackageV2Response{
		BaseResponse: c.base(),
		FeedResponse: &nugettype.FeedResponse{ID: "feed-search"},
	}
}

func (c *rchNugetCtrl) GetReadme(_ context.Context, _ nugettype.ArtifactInfo) *nugetctrl.GetReadmeResponse {
	return &nugetctrl.GetReadmeResponse{BaseResponse: c.base(), ReadmeContent: "# readme"}
}

func rchNewHandler(ctrl nugetctrl.Controller, baseErr error) Handler {
	return NewHandler(ctrl, rchPkgHandler{
		Handler: rchBasePackagesHandler(),
		info:    pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"},
		err:     baseErr,
	})
}

func rchNugetInfo() *nugettype.ArtifactInfo {
	return &nugettype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"},
		Filename:     "pkg.1.0.0.nupkg",
		Version:      "1.0.0",
	}
}

func rchNugetRequest(method, target string, body io.Reader, withInfo bool) *http.Request {
	req := httptest.NewRequest(method, target, body)
	if withInfo {
		req = req.WithContext(request.WithArtifactInfo(req.Context(), rchNugetInfo()))
	}
	return req
}

func rchMultipart(t *testing.T, field string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	part, err := mw.CreateFormFile(field, "pkg.nupkg")
	require.NoError(t, err)
	_, err = part.Write([]byte("package-bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

type rchNugetCall struct {
	name     string
	fn       func(h Handler) http.HandlerFunc
	okStatus int
	okBody   string
}

func rchNugetCalls() []rchNugetCall {
	return []rchNugetCall{
		{name: "delete", fn: func(h Handler) http.HandlerFunc { return h.DeletePackage }, okStatus: http.StatusOK},
		{name: "download", fn: func(h Handler) http.HandlerFunc { return h.DownloadPackage },
			okStatus: http.StatusOK, okBody: "nupkg"},
		{name: "download symbols v2", fn: func(h Handler) http.HandlerFunc {
			impl, ok := h.(*handler)
			if !ok {
				panic("unexpected handler type")
			}
			return impl.DownloadSymbolPackageV2
		}, okStatus: http.StatusOK, okBody: "nupkg"},
		{name: "list versions", fn: func(h Handler) http.HandlerFunc { return h.ListPackageVersion },
			okStatus: http.StatusOK, okBody: `"versions":["1.0.0"]`},
		{name: "list versions v2", fn: func(h Handler) http.HandlerFunc { return h.ListPackageVersionV2 },
			okStatus: http.StatusOK, okBody: "feed-list"},
		{name: "package metadata", fn: func(h Handler) http.HandlerFunc { return h.GetPackageMetadata },
			okStatus: http.StatusOK, okBody: `"count":2`},
		{name: "version metadata", fn: func(h Handler) http.HandlerFunc { return h.GetPackageVersionMetadata },
			okStatus: http.StatusOK, okBody: `"listed":true`},
		{name: "version metadata v2", fn: func(h Handler) http.HandlerFunc { return h.GetPackageVersionMetadataV2 },
			okStatus: http.StatusOK, okBody: "entry-id"},
		{name: "version count v2", fn: func(h Handler) http.HandlerFunc { return h.GetPackageVersionCountV2 },
			okStatus: http.StatusOK, okBody: ">4<"},
		{name: "count v2", fn: func(h Handler) http.HandlerFunc { return h.CountPackageV2 },
			okStatus: http.StatusOK, okBody: ">7<"},
		{name: "search", fn: func(h Handler) http.HandlerFunc { return h.SearchPackage },
			okStatus: http.StatusOK, okBody: `"totalHits":5`},
		{name: "search v2", fn: func(h Handler) http.HandlerFunc { return h.SearchPackageV2 },
			okStatus: http.StatusOK, okBody: "feed-search"},
		{name: "service endpoint", fn: func(h Handler) http.HandlerFunc { return h.GetServiceEndpoint },
			okStatus: http.StatusOK, okBody: `"version":"3.0.0"`},
		{name: "service endpoint v2", fn: func(h Handler) http.HandlerFunc { return h.GetServiceEndpointV2 },
			okStatus: http.StatusOK, okBody: "http://base"},
		{name: "service metadata v2", fn: func(h Handler) http.HandlerFunc { return h.GetServiceMetadataV2 },
			okStatus: http.StatusOK, okBody: "Version"},
	}
}

func TestRchNugetHandlersSuccess(t *testing.T) {
	for _, tc := range rchNugetCalls() {
		t.Run(tc.name, func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{}, nil)
			rec := httptest.NewRecorder()
			tc.fn(h)(rec, rchNugetRequest(http.MethodGet, "/x?q=a&skip=1&take=2&$skip=1&$top=3", nil, true))
			require.Equal(t, tc.okStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tc.okBody)
		})
	}
}

func TestRchNugetHandlersNoInfoInContext(t *testing.T) {
	for _, tc := range rchNugetCalls() {
		t.Run(tc.name, func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{}, nil)
			rec := httptest.NewRecorder()
			tc.fn(h)(rec, rchNugetRequest(http.MethodGet, "/x", nil, false))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestRchNugetHandlersControllerError(t *testing.T) {
	for _, tc := range rchNugetCalls() {
		t.Run(tc.name, func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{err: errors.New("boom")}, nil)
			rec := httptest.NewRecorder()
			tc.fn(h)(rec, rchNugetRequest(http.MethodGet, "/x", nil, true))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestRchNugetDownloadRedirect(t *testing.T) {
	for _, name := range []string{"download", "download symbols v2"} {
		for _, tc := range rchNugetCalls() {
			if tc.name != name {
				continue
			}
			t.Run(name, func(t *testing.T) {
				h := rchNewHandler(&rchNugetCtrl{redirect: "http://cdn.example/pkg"}, nil)
				rec := httptest.NewRecorder()
				tc.fn(h)(rec, rchNugetRequest(http.MethodGet, "/x", nil, true))
				require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
				require.Equal(t, "http://cdn.example/pkg", rec.Header().Get("Location"))
			})
		}
	}
}

func TestRchNugetDownloadNothingToServe(t *testing.T) {
	h := rchNewHandler(&rchNugetCtrl{noBody: true}, nil)
	rec := httptest.NewRecorder()
	h.DownloadPackage(rec, rchNugetRequest(http.MethodGet, "/x", nil, true))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRchNugetUpload(t *testing.T) {
	tests := []struct {
		name     string
		upload   func(h Handler) http.HandlerFunc
		wantType nugetpkg.FileBundleType
	}{
		{"package", func(h Handler) http.HandlerFunc { return h.UploadPackage }, nugetpkg.DependencyFile},
		{"symbols", func(h Handler) http.HandlerFunc { return h.UploadSymbolPackage }, nugetpkg.SymbolsFile},
	}
	for _, tc := range tests {
		t.Run(tc.name+" ok", func(t *testing.T) {
			ctrl := &rchNugetCtrl{}
			h := rchNewHandler(ctrl, nil)
			body, contentType := rchMultipart(t, "package")
			req := rchNugetRequest(http.MethodPut, "/x", body, true)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			tc.upload(h)(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, "1", rec.Header().Get("X-Rch"))
			require.Equal(t, tc.wantType, ctrl.lastType)
		})
		t.Run(tc.name+" missing file", func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{}, nil)
			body, contentType := rchMultipart(t, "other")
			req := rchNugetRequest(http.MethodPut, "/x", body, true)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			tc.upload(h)(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
		t.Run(tc.name+" not multipart", func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{}, nil)
			rec := httptest.NewRecorder()
			tc.upload(h)(rec, rchNugetRequest(http.MethodPut, "/x", strings.NewReader("raw"), true))
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
		t.Run(tc.name+" no info", func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{}, nil)
			body, contentType := rchMultipart(t, "package")
			req := rchNugetRequest(http.MethodPut, "/x", body, false)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			tc.upload(h)(rec, req)
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
		t.Run(tc.name+" controller error", func(t *testing.T) {
			h := rchNewHandler(&rchNugetCtrl{err: errors.New("boom")}, nil)
			body, contentType := rchMultipart(t, "package")
			req := rchNugetRequest(http.MethodPut, "/x", body, true)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			tc.upload(h)(rec, req)
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestRchNugetGetReadme(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		h := rchNewHandler(&rchNugetCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.GetReadme(rec, rchNugetRequest(http.MethodGet, "/x", nil, false))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "# readme", rec.Body.String())
		require.Equal(t, "1", rec.Header().Get("X-Rch"))
	})
	t.Run("controller error", func(t *testing.T) {
		h := rchNewHandler(&rchNugetCtrl{err: errors.New("boom")}, nil)
		rec := httptest.NewRecorder()
		h.GetReadme(rec, rchNugetRequest(http.MethodGet, "/x", nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("artifact info error", func(t *testing.T) {
		h := rchNewHandler(&rchNugetCtrl{}, errors.New("no registry"))
		rec := httptest.NewRecorder()
		h.GetReadme(rec, rchNugetRequest(http.MethodGet, "/x", nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestRchNugetGetPackageArtifactInfo(t *testing.T) {
	newReq := func(target string, params map[string]string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		req = req.WithContext(ctx)
		req.SetPathValue("*", "nested/path/")
		return req
	}

	h := rchNewHandler(&rchNugetCtrl{}, nil)
	got, err := h.GetPackageArtifactInfo(newReq("/x?proxy_endpoint=http://up", map[string]string{
		"id": "pkg", "filename": "pkg.nupkg", "version": "2.0.0",
	}))
	require.NoError(t, err)
	info, ok := got.(*nugettype.ArtifactInfo)
	require.True(t, ok)
	require.Equal(t, "pkg", info.Image)
	require.Equal(t, "pkg.nupkg", info.Filename)
	require.Equal(t, "2.0.0", info.Version)
	require.Equal(t, "http://up", info.ProxyEndpoint)
	require.Equal(t, "nested/path", info.NestedPath)

	// The id falls back to the quoted query parameter used by the V2 protocol.
	got, err = h.GetPackageArtifactInfo(newReq("/x?id='quoted'", nil))
	require.NoError(t, err)
	require.Equal(t, "quoted", got.GetImage())

	failing := rchNewHandler(&rchNugetCtrl{}, errors.New("no registry"))
	_, err = failing.GetPackageArtifactInfo(newReq("/x", nil))
	require.Error(t, err)
}

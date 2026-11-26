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

package npm_test

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/errors"
	npmctrl "github.com/harness/gitness/registry/app/api/controller/pkg/npm"
	npmhandler "github.com/harness/gitness/registry/app/api/handler/npm"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	npmmeta "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	npmtype "github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/request"

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

type rchNpmCtrl struct {
	npmctrl.Controller
	err      error
	redirect string
	noBody   bool
}

func (c *rchNpmCtrl) base() npmctrl.BaseResponse {
	return npmctrl.BaseResponse{
		Error:           c.err,
		ResponseHeaders: &commons.ResponseHeaders{Headers: map[string]string{"X-Rch": "1"}, Code: http.StatusOK},
	}
}

func (c *rchNpmCtrl) UploadPackageFile(
	_ context.Context, _ *npmtype.ArtifactInfo, _ io.ReadCloser,
) *npmctrl.PutArtifactResponse {
	return &npmctrl.PutArtifactResponse{Sha256: "abc", BaseResponse: c.base()}
}

func (c *rchNpmCtrl) DownloadPackageFile(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.GetArtifactResponse {
	resp := &npmctrl.GetArtifactResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
	if !c.noBody {
		resp.ReadCloser = io.NopCloser(strings.NewReader("tarball"))
	}
	return resp
}

func (c *rchNpmCtrl) GetPackageMetadata(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.GetMetadataResponse {
	return &npmctrl.GetMetadataResponse{
		BaseResponse:    c.base(),
		PackageMetadata: npmmeta.PackageMetadata{Name: "left-pad"},
	}
}

func (c *rchNpmCtrl) HeadPackageFileByName(
	_ context.Context, _ *npmtype.ArtifactInfo,
) *npmctrl.HeadMetadataResponse {
	return &npmctrl.HeadMetadataResponse{BaseResponse: c.base(), Exists: true}
}

func (c *rchNpmCtrl) ListTags(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.ListTagResponse {
	return &npmctrl.ListTagResponse{Tags: map[string]string{"latest": "1.0.0"}, BaseResponse: c.base()}
}

func (c *rchNpmCtrl) AddTag(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.ListTagResponse {
	return &npmctrl.ListTagResponse{Tags: map[string]string{"beta": "2.0.0"}, BaseResponse: c.base()}
}

func (c *rchNpmCtrl) DeleteTag(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.ListTagResponse {
	return &npmctrl.ListTagResponse{Tags: map[string]string{}, BaseResponse: c.base()}
}

func (c *rchNpmCtrl) DeleteVersion(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.DeleteEntityResponse {
	return &npmctrl.DeleteEntityResponse{Error: c.err}
}

func (c *rchNpmCtrl) DeletePackage(_ context.Context, _ *npmtype.ArtifactInfo) *npmctrl.DeleteEntityResponse {
	return &npmctrl.DeleteEntityResponse{Error: c.err}
}

func (c *rchNpmCtrl) SearchPackage(
	_ context.Context, _ *npmtype.ArtifactInfo, _ int, _ int,
) *npmctrl.SearchArtifactResponse {
	return &npmctrl.SearchArtifactResponse{BaseResponse: c.base(), Artifacts: &npmmeta.PackageSearch{Total: 2}}
}

type rchNpmCall struct {
	name string
	// fn picks the handler method under test.
	fn func(h npmhandler.Handler) http.HandlerFunc
	// noInfoStatus is the status expected when no npm artifact info is in the context.
	noInfoStatus int
	// errStatus is the status expected when the controller returns an error.
	errStatus int
	okStatus  int
	okBody    string
}

func rchNpmCalls() []rchNpmCall {
	return []rchNpmCall{
		{"upload", func(h npmhandler.Handler) http.HandlerFunc { return h.UploadPackage },
			http.StatusBadRequest, http.StatusInternalServerError, http.StatusOK, "Pushed.\nSha256: abc"},
		{"download", func(h npmhandler.Handler) http.HandlerFunc { return h.DownloadPackageFile },
			http.StatusInternalServerError, http.StatusInternalServerError, http.StatusOK, "tarball"},
		{"download by name", func(h npmhandler.Handler) http.HandlerFunc { return h.DownloadPackageFileByName },
			http.StatusInternalServerError, http.StatusInternalServerError, http.StatusOK, ""},
		{"metadata", func(h npmhandler.Handler) http.HandlerFunc { return h.GetPackageMetadata },
			http.StatusInternalServerError, http.StatusInternalServerError, http.StatusOK, `"name":"left-pad"`},
		{"head", func(h npmhandler.Handler) http.HandlerFunc { return h.HeadPackageFileByName },
			http.StatusInternalServerError, http.StatusOK, http.StatusOK, `"Exists":true`},
		{"list tags", func(h npmhandler.Handler) http.HandlerFunc { return h.ListPackageTag },
			http.StatusBadRequest, http.StatusInternalServerError, http.StatusOK, `{"latest":"1.0.0"}`},
		{"add tag", func(h npmhandler.Handler) http.HandlerFunc { return h.AddPackageTag },
			http.StatusBadRequest, http.StatusInternalServerError, http.StatusOK, `{"beta":"2.0.0"}`},
		{"delete tag", func(h npmhandler.Handler) http.HandlerFunc { return h.DeletePackageTag },
			http.StatusBadRequest, http.StatusInternalServerError, http.StatusOK, ""},
		{"delete version", func(h npmhandler.Handler) http.HandlerFunc { return h.DeleteVersion },
			http.StatusBadRequest, http.StatusInternalServerError, http.StatusNoContent, ""},
		{"delete package", func(h npmhandler.Handler) http.HandlerFunc { return h.DeletePackage },
			http.StatusBadRequest, http.StatusInternalServerError, http.StatusNoContent, ""},
		{"delete preview", func(h npmhandler.Handler) http.HandlerFunc { return h.DeletePreview },
			http.StatusBadRequest, http.StatusOK, http.StatusOK, ""},
		{"search", func(h npmhandler.Handler) http.HandlerFunc { return h.SearchPackage },
			http.StatusInternalServerError, http.StatusInternalServerError, http.StatusOK, `"total":2`},
	}
}

func rchNpmRequest(t *testing.T, withInfo bool) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/pkg/root/reg/npm/left-pad?text=pad&size=5&from=1", nil)
	ctx := req.Context()
	if withInfo {
		ctx = request.WithArtifactInfo(ctx, rchNpmInfo())
	}
	return req.WithContext(ctx)
}

func rchNpmInfo() *npmtype.ArtifactInfo {
	return &npmtype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"},
		Filename:     "left-pad-1.0.0.tgz",
		Version:      "1.0.0",
	}
}

func rchBasePackagesHandler() packages.Handler {
	return packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
}

func rchNpmHandler(ctrl npmctrl.Controller) npmhandler.Handler {
	return npmhandler.NewHandler(ctrl, rchPkgHandler{Handler: rchBasePackagesHandler()})
}

func TestRchNpmHandlersSuccess(t *testing.T) {
	for _, tc := range rchNpmCalls() {
		t.Run(tc.name, func(t *testing.T) {
			h := rchNpmHandler(&rchNpmCtrl{})
			rec := httptest.NewRecorder()
			tc.fn(h)(rec, rchNpmRequest(t, true))
			require.Equal(t, tc.okStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tc.okBody)
		})
	}
}

func TestRchNpmHandlersNoInfoInContext(t *testing.T) {
	for _, tc := range rchNpmCalls() {
		t.Run(tc.name, func(t *testing.T) {
			h := rchNpmHandler(&rchNpmCtrl{})
			rec := httptest.NewRecorder()
			tc.fn(h)(rec, rchNpmRequest(t, false))
			require.Equal(t, tc.noInfoStatus, rec.Code)
		})
	}
}

func TestRchNpmHandlersControllerError(t *testing.T) {
	for _, tc := range rchNpmCalls() {
		t.Run(tc.name, func(t *testing.T) {
			h := rchNpmHandler(&rchNpmCtrl{err: stderrors.New("boom")})
			rec := httptest.NewRecorder()
			tc.fn(h)(rec, rchNpmRequest(t, true))
			require.Equal(t, tc.errStatus, rec.Code)
		})
	}
}

func TestRchNpmDownloadVariants(t *testing.T) {
	t.Run("redirect", func(t *testing.T) {
		h := rchNpmHandler(&rchNpmCtrl{redirect: "http://cdn.example/pkg.tgz"})
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchNpmRequest(t, true))
		require.Equal(t, http.StatusFound, rec.Code)
		require.Equal(t, "http://cdn.example/pkg.tgz", rec.Header().Get("Location"))
	})
	t.Run("redirect by name", func(t *testing.T) {
		h := rchNpmHandler(&rchNpmCtrl{redirect: "http://cdn.example/pkg.tgz"})
		rec := httptest.NewRecorder()
		h.DownloadPackageFileByName(rec, rchNpmRequest(t, true))
		require.Equal(t, http.StatusFound, rec.Code)
	})
	t.Run("nothing to serve", func(t *testing.T) {
		h := rchNpmHandler(&rchNpmCtrl{noBody: true})
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchNpmRequest(t, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("content disposition", func(t *testing.T) {
		h := rchNpmHandler(&rchNpmCtrl{})
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchNpmRequest(t, true))
		require.Equal(t, "attachment; filename=left-pad-1.0.0.tgz", rec.Header().Get("Content-Disposition"))
	})
}

func TestRchNpmSearchNotFound(t *testing.T) {
	h := rchNpmHandler(&rchNpmCtrl{err: errors.NotFound("nothing")})
	rec := httptest.NewRecorder()
	h.SearchPackage(rec, rchNpmRequest(t, true))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func rchNpmPathRequest(method, path string, body string, values map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range values {
		req.SetPathValue(k, v)
	}
	return req
}

func TestRchNpmGetPackageArtifactInfo(t *testing.T) {
	tests := []struct {
		name      string
		req       *http.Request
		baseErr   error
		wantErr   bool
		wantImage string
		wantVer   string
		wantTags  []string
	}{
		{
			name: "scoped package with version",
			req: rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{
				"scope": "acme", "id": "lib", "version": "1.2.3",
			}),
			wantImage: "@acme/lib",
			wantVer:   "1.2.3",
			wantTags:  []string{""},
		},
		{
			name: "version from file name",
			req: rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{
				"id": "lib", "filename": "lib-2.0.0.tgz",
			}),
			wantImage: "lib",
			wantVer:   "2.0.0",
			wantTags:  []string{""},
		},
		{
			name: "invalid version",
			req: rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{
				"id": "lib", "version": "not-a-version",
			}),
			wantErr: true,
		},
		{
			name:    "base error",
			req:     rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{"id": "lib"}),
			baseErr: stderrors.New("no registry"),
			wantErr: true,
		},
		{
			name: "add tag payload",
			req: rchNpmPathRequest(http.MethodPut, "/-/package/lib/dist-tags/beta", `"3.0.0"`,
				map[string]string{"id": "lib", "tag": "beta"}),
			wantImage: "lib",
			wantVer:   "3.0.0",
			wantTags:  []string{"beta"},
		},
		{
			name: "rev path keeps request info",
			req: rchNpmPathRequest(http.MethodDelete, "/x/-rev/1", "payload",
				map[string]string{"id": "lib", "version": "1.0.0"}),
			wantImage: "lib",
			wantVer:   "1.0.0",
			wantTags:  []string{""},
		},
		{
			name: "other body returns base info only",
			req: rchNpmPathRequest(http.MethodPut, "/lib", `{"name":"lib"}`,
				map[string]string{"id": "lib", "version": "1.0.0"}),
			wantImage: "lib",
			wantVer:   "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := npmhandler.NewHandler(&rchNpmCtrl{}, rchPkgHandler{
				Handler: rchBasePackagesHandler(),
				info:    pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"},
				err:     tc.baseErr,
			})
			got, err := h.GetPackageArtifactInfo(tc.req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantImage, got.GetImage())
			require.Equal(t, tc.wantVer, got.GetVersion())
			if tc.wantTags != nil {
				info, ok := got.(*npmtype.ArtifactInfo)
				require.True(t, ok)
				require.Equal(t, tc.wantTags, info.DistTags)
			}
		})
	}
}

func TestRchNpmUtils(t *testing.T) {
	req := rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{"scope": "acme%2Fx", "id": "lib"})
	require.Equal(t, "@acme/x/lib", npmhandler.PackageNameFromParams(req))

	req = rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{"scope": "acme"})
	require.Equal(t, "@acme", npmhandler.PackageNameFromParams(req))

	req = rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{"id": "lib"})
	require.Equal(t, "lib", npmhandler.PackageNameFromParams(req))

	req = rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{"scope": "acme", "filename": "lib-1.0.0.tgz"})
	require.Equal(t, "@acme/lib-1.0.0.tgz", npmhandler.GetFileName(req))
	require.Equal(t, "1.0.0", npmhandler.GetVersionFromParams(req))

	req = rchNpmPathRequest(http.MethodGet, "/x", "", map[string]string{"filename": "lib.tgz"})
	require.Empty(t, npmhandler.GetVersionFromParams(req))
	require.Equal(t, "lib.tgz", npmhandler.GetFileName(req))

	_, err := npmhandler.VersionNameFromFileName("lib-1.0.0-beta.1.tgz")
	require.NoError(t, err)
	_, err = npmhandler.VersionNameFromFileName("broken")
	require.Error(t, err)

	req = rchNpmPathRequest(http.MethodPut, "/x", "body", nil)
	rc, err := npmhandler.GetNPMFile(req)
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "body", string(data))
}

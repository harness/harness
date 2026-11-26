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

package huggingface_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	hfctrl "github.com/harness/gitness/registry/app/api/controller/pkg/huggingface"
	hfhandler "github.com/harness/gitness/registry/app/api/handler/huggingface"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/request"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type rchPkgHandler struct {
	packages.Handler
	err error
}

func (h rchPkgHandler) GetArtifactInfo(*http.Request) (pkg.ArtifactInfo, error) {
	return pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"}, h.err
}

type rchHFCtrl struct {
	hfctrl.Controller
	err      error
	redirect string
	lastFile string
}

func (c *rchHFCtrl) base() hfctrl.BaseResponse {
	return hfctrl.BaseResponse{
		Error:           c.err,
		ResponseHeaders: &commons.ResponseHeaders{Code: http.StatusOK, Headers: map[string]string{"X-Rch": "1"}},
	}
}

func (c *rchHFCtrl) ValidateYaml(context.Context, hftype.ArtifactInfo, io.ReadCloser) *hfctrl.ValidateYamlResponse {
	return &hfctrl.ValidateYamlResponse{BaseResponse: c.base(), Response: &hftype.ValidateYamlResponse{}}
}

func (c *rchHFCtrl) PreUpload(context.Context, hftype.ArtifactInfo, io.ReadCloser) *hfctrl.PreUploadResponse {
	return &hfctrl.PreUploadResponse{BaseResponse: c.base(), Response: &hftype.PreUploadResponse{}}
}

func (c *rchHFCtrl) RevisionInfo(
	context.Context, hftype.ArtifactInfo, map[string][]string,
) *hfctrl.RevisionInfoResponse {
	return &hfctrl.RevisionInfoResponse{
		BaseResponse: c.base(), Response: &hftype.RevisionInfoResponse{XetEnabled: true},
	}
}

func (c *rchHFCtrl) LfsInfo(
	context.Context, hftype.ArtifactInfo, io.ReadCloser, string,
) *hfctrl.LfsInfoResponse {
	return &hfctrl.LfsInfoResponse{BaseResponse: c.base(), Response: &hftype.LfsInfoResponse{}}
}

func (c *rchHFCtrl) LfsVerify(context.Context, hftype.ArtifactInfo, io.ReadCloser) *hfctrl.LfsVerifyResponse {
	return &hfctrl.LfsVerifyResponse{BaseResponse: c.base(), Response: &hftype.LfsVerifyResponse{Success: true}}
}

func (c *rchHFCtrl) LfsUpload(context.Context, hftype.ArtifactInfo, io.ReadCloser) *hfctrl.LfsUploadResponse {
	return &hfctrl.LfsUploadResponse{BaseResponse: c.base(), Response: &hftype.LfsUploadResponse{Success: true}}
}

func (c *rchHFCtrl) CommitRevision(
	context.Context, hftype.ArtifactInfo, io.ReadCloser,
) *hfctrl.CommitRevisionResponse {
	return &hfctrl.CommitRevisionResponse{
		BaseResponse: c.base(), Response: &hftype.CommitRevisionResponse{OID: "abc", Success: true},
	}
}

func (c *rchHFCtrl) HeadFile(_ context.Context, _ hftype.ArtifactInfo, name string) *hfctrl.HeadFileResponse {
	c.lastFile = name
	return &hfctrl.HeadFileResponse{BaseResponse: c.base()}
}

func (c *rchHFCtrl) DownloadFile(
	_ context.Context, _ hftype.ArtifactInfo, name string,
) *hfctrl.DownloadFileResponse {
	c.lastFile = name
	return &hfctrl.DownloadFileResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
}

func rchHandler(ctrl hfctrl.Controller, baseErr error) hfhandler.Handler {
	base := packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
	return hfhandler.NewHandler(ctrl, rchPkgHandler{Handler: base, err: baseErr})
}

func rchRequest(method string, withInfo bool) *http.Request {
	req := httptest.NewRequest(method, "/x", strings.NewReader("{}"))
	if withInfo {
		info := &hftype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}}}
		req = req.WithContext(request.WithArtifactInfo(req.Context(), info))
	}
	return req
}

type rchHFCall struct {
	name   string
	call   func(h hfhandler.Handler) http.HandlerFunc
	okBody string
}

func rchHFCalls() []rchHFCall {
	return []rchHFCall{
		{name: "validate yaml", call: func(h hfhandler.Handler) http.HandlerFunc { return h.ValidateYAML },
			okBody: `"errors"`},
		{name: "pre upload", call: func(h hfhandler.Handler) http.HandlerFunc { return h.PreUpload },
			okBody: `"files"`},
		{name: "revision info", call: func(h hfhandler.Handler) http.HandlerFunc { return h.RevisionInfo },
			okBody: `"xetEnabled":true`},
		{name: "lfs info", call: func(h hfhandler.Handler) http.HandlerFunc { return h.LfsInfo },
			okBody: `"objects"`},
		{name: "lfs upload", call: func(h hfhandler.Handler) http.HandlerFunc { return h.LfsUpload },
			okBody: `"success":true`},
		{name: "lfs verify", call: func(h hfhandler.Handler) http.HandlerFunc { return h.LfsVerify },
			okBody: `"success":true`},
		{name: "commit revision", call: func(h hfhandler.Handler) http.HandlerFunc { return h.CommitRevision },
			okBody: `"commitOid":"abc"`},
	}
}

func TestRchHFJSONHandlers(t *testing.T) {
	for _, tc := range rchHFCalls() {
		t.Run(tc.name+" success", func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rchHandler(&rchHFCtrl{}, nil))(rec, rchRequest(http.MethodPost, true))
			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Body.String(), tc.okBody)
			require.Equal(t, "1", rec.Header().Get("X-Rch"))
		})
		t.Run(tc.name+" controller error", func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rchHandler(&rchHFCtrl{err: errors.New("boom")}, nil))(rec, rchRequest(http.MethodPost, true))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
		t.Run(tc.name+" no info in context", func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.call(rchHandler(&rchHFCtrl{}, nil))(rec, rchRequest(http.MethodPost, false))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestRchHFHeadFile(t *testing.T) {
	t.Run("decodes the file path", func(t *testing.T) {
		ctrl := &rchHFCtrl{}
		req := rchRequest(http.MethodHead, true)
		req.SetPathValue("*", "dir%2Fconfig.json")
		rec := httptest.NewRecorder()
		rchHandler(ctrl, nil).HeadFile(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "dir/config.json", ctrl.lastFile)
		require.Equal(t, "1", rec.Header().Get("X-Rch"))
	})
	t.Run("bad escape", func(t *testing.T) {
		req := rchRequest(http.MethodHead, true)
		req.SetPathValue("*", "%zz")
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{}, nil).HeadFile(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("controller error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{err: errors.New("boom")}, nil).HeadFile(rec, rchRequest(http.MethodHead, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{}, nil).HeadFile(rec, rchRequest(http.MethodHead, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestRchHFDownloadFile(t *testing.T) {
	t.Run("redirects", func(t *testing.T) {
		ctrl := &rchHFCtrl{redirect: "http://cdn.example/model.bin"}
		req := rchRequest(http.MethodGet, true)
		req.SetPathValue("*", "dir%2Fmodel.bin")
		rec := httptest.NewRecorder()
		rchHandler(ctrl, nil).DownloadFile(rec, req)
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		require.Equal(t, "http://cdn.example/model.bin", rec.Header().Get("Location"))
		require.Equal(t, "dir/model.bin", ctrl.lastFile)
	})
	t.Run("nothing to serve", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{}, nil).DownloadFile(rec, rchRequest(http.MethodGet, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("controller error is reported", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{err: errors.New("boom")}, nil).
			DownloadFile(rec, rchRequest(http.MethodGet, true))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("bad escape", func(t *testing.T) {
		req := rchRequest(http.MethodGet, true)
		req.SetPathValue("*", "%zz")
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{}, nil).DownloadFile(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchHFCtrl{}, nil).DownloadFile(rec, rchRequest(http.MethodGet, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func rchChiRequest(params map[string]string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestRchHFGetPackageArtifactInfo(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]string
		baseErr  error
		wantErr  bool
		wantRepo string
		wantRev  string
		wantType artifact.ArtifactType
		wantSHA  string
	}{
		{name: "defaults to a model on main", params: map[string]string{"repo": "org/model"},
			wantRepo: "org/model", wantRev: "main", wantType: artifact.ArtifactType("model")},
		{name: "plural dataset type and revision",
			params:   map[string]string{"repoType": "datasets", "repo": "org/data", "rev": "v1", "sha256": "abc"},
			wantRepo: "org/data", wantRev: "v1", wantType: artifact.ArtifactType("dataset"), wantSHA: "abc"},
		{name: "no repo", params: map[string]string{}, wantRepo: "", wantRev: "main",
			wantType: artifact.ArtifactType("model")},
		{name: "unsupported type", params: map[string]string{"repoType": "spaces", "repo": "org/x"}, wantErr: true},
		{name: "invalid repo id", params: map[string]string{"repo": "-bad-"}, wantErr: true},
		{name: "base info failure", params: map[string]string{"repo": "org/model"},
			baseErr: errors.New("no registry"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchHandler(&rchHFCtrl{}, tc.baseErr)
			got, err := h.GetPackageArtifactInfo(rchChiRequest(tc.params))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			info, ok := got.(*hftype.ArtifactInfo)
			require.True(t, ok)
			require.Equal(t, tc.wantRepo, info.Repo)
			require.Equal(t, tc.wantRepo, info.Image)
			require.Equal(t, tc.wantRev, info.Revision)
			require.Equal(t, tc.wantType, info.RepoType)
			require.Equal(t, tc.wantSHA, info.SHA256)
		})
	}
}

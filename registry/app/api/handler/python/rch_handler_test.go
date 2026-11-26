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

package python_test

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
	pythonctrl "github.com/harness/gitness/registry/app/api/controller/pkg/python"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	pythonhandler "github.com/harness/gitness/registry/app/api/handler/python"
	liberrors "github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
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

type rchPythonCtrl struct {
	pythonctrl.Controller
	err      error
	redirect string
	noBody   bool
	last     pythontype.ArtifactInfo
}

func (c *rchPythonCtrl) base() pythonctrl.BaseResponse {
	return pythonctrl.BaseResponse{
		Error:           c.err,
		ResponseHeaders: &commons.ResponseHeaders{Headers: map[string]string{"X-Rch": "1"}},
	}
}

func (c *rchPythonCtrl) GetPackageMetadata(
	_ context.Context, _ pythontype.ArtifactInfo,
) *pythonctrl.GetMetadataResponse {
	return &pythonctrl.GetMetadataResponse{
		BaseResponse: c.base(),
		PackageMetadata: pythontype.PackageMetadata{
			Name: "requests",
			Files: []pythontype.File{
				{FileURL: "http://files.example/requests-2.0.0.whl", Name: "requests-2.0.0.whl", RequiresPython: ">=3.8"},
			},
		},
	}
}

func (c *rchPythonCtrl) UploadPackageFile(
	_ context.Context, info pythontype.ArtifactInfo, _ multipart.File, _ *multipart.FileHeader,
) *pythonctrl.PutArtifactResponse {
	c.last = info
	return &pythonctrl.PutArtifactResponse{BaseResponse: c.base(), Sha256: "sha"}
}

func (c *rchPythonCtrl) DownloadPackageFile(
	context.Context, pythontype.ArtifactInfo,
) *pythonctrl.GetArtifactResponse {
	resp := &pythonctrl.GetArtifactResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
	if !c.noBody {
		resp.ReadCloser = io.NopCloser(strings.NewReader("wheel-bytes"))
	}
	return resp
}

func rchHandler(ctrl pythonctrl.Controller, baseErr error) pythonhandler.Handler {
	base := packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
	return pythonhandler.NewHandler(ctrl, rchPkgHandler{Handler: base, err: baseErr})
}

func rchRequest(method string, body io.Reader, withInfo bool) *http.Request {
	req := httptest.NewRequest(method, "/x", body)
	if withInfo {
		info := &pythontype.ArtifactInfo{
			ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}},
			Filename:     "requests-2.0.0.whl",
		}
		req = req.WithContext(request.WithArtifactInfo(req.Context(), info))
	}
	return req
}

func rchChiRequest(target string, params map[string]string, body io.Reader, contentType string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, target, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func rchFormBody(t *testing.T, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

func TestRchPythonGetPackageArtifactInfo(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]string
		fields       map[string]string
		baseErr      error
		wantErr      bool
		wantImage    string
		wantVersion  string
		wantFilename string
		wantHomePage string
	}{
		{name: "from the route", params: map[string]string{
			"image": "My_Package.Name", "version": "1.0.0", "filename": "f.whl",
		}, wantImage: "My-Package-Name", wantVersion: "1.0.0", wantFilename: "f.whl"},
		{name: "from the upload form", fields: map[string]string{"name": "my_pkg", "version": "2.1"},
			wantImage: "my-pkg", wantVersion: "2.1"},
		{name: "home page from project urls",
			fields: map[string]string{
				"name": "pkg", "version": "1.0", "project_urls": `{"Home-page":"https://example.com/home"}`,
			},
			wantImage: "pkg", wantVersion: "1.0", wantHomePage: "https://example.com/home"},
		{name: "home page falls back to the metadata field",
			fields: map[string]string{
				"name": "pkg", "version": "1.0", "home_page": "https://example.com/fallback",
			},
			wantImage: "pkg", wantVersion: "1.0", wantHomePage: "https://example.com/fallback"},
		{name: "invalid home page is dropped",
			fields:    map[string]string{"name": "pkg", "version": "1.0", "home_page": "not a url"},
			wantImage: "pkg", wantVersion: "1.0"},
		{name: "invalid version", params: map[string]string{"image": "pkg", "version": "not-a-version"},
			wantErr: true},
		{name: "invalid name", params: map[string]string{"image": "-pkg", "version": "1.0"}, wantErr: true},
		{name: "base info failure", baseErr: errors.New("no registry"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchHandler(&rchPythonCtrl{}, tc.baseErr)
			var body io.Reader
			contentType := ""
			if tc.fields != nil {
				buf, ct := rchFormBody(t, tc.fields)
				body, contentType = buf, ct
			}
			got, err := h.GetPackageArtifactInfo(rchChiRequest("/x", tc.params, body, contentType))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantImage, got.GetImage())
			require.Equal(t, tc.wantVersion, got.GetVersion())
			require.Equal(t, tc.wantFilename, got.GetFileName())
			info, ok := got.(*pythontype.ArtifactInfo)
			require.True(t, ok)
			require.Equal(t, tc.wantHomePage, info.Metadata.HomePage)
		})
	}
}

func TestRchPythonDownloadPackageFile(t *testing.T) {
	t.Run("serves the package", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{}, nil).DownloadPackageFile(rec, rchRequest(http.MethodGet, nil, true))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "wheel-bytes", rec.Body.String())
	})
	t.Run("redirects", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{redirect: "http://cdn.example/x.whl"}, nil).
			DownloadPackageFile(rec, rchRequest(http.MethodGet, nil, true))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	})
	t.Run("controller error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{err: errors.New("boom")}, nil).
			DownloadPackageFile(rec, rchRequest(http.MethodGet, nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{}, nil).DownloadPackageFile(rec, rchRequest(http.MethodGet, nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestRchPythonPackageMetadata(t *testing.T) {
	t.Run("renders the simple index", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{}, nil).PackageMetadata(rec, rchRequest(http.MethodGet, nil, true))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))
		body := rec.Body.String()
		require.Contains(t, body, "Links for requests")
		require.Contains(t, body, `data-requires-python="&gt;=3.8"`)
		require.Contains(t, body, "requests-2.0.0.whl</a>")
	})
	t.Run("not found", func(t *testing.T) {
		notFound := &liberrors.Error{Code: liberrors.NotFoundCode, Message: "nope"}
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{err: notFound}, nil).
			PackageMetadata(rec, rchRequest(http.MethodGet, nil, true))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("other errors", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{err: errors.New("boom")}, nil).
			PackageMetadata(rec, rchRequest(http.MethodGet, nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchPythonCtrl{}, nil).PackageMetadata(rec, rchRequest(http.MethodGet, nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestRchPythonUploadPackageFile(t *testing.T) {
	post := func(t *testing.T, ctrl *rchPythonCtrl, field string, withInfo bool) *httptest.ResponseRecorder {
		t.Helper()
		buf := &bytes.Buffer{}
		mw := multipart.NewWriter(buf)
		w, err := mw.CreateFormFile(field, "requests-2.0.0.whl")
		require.NoError(t, err)
		_, err = w.Write([]byte("wheel"))
		require.NoError(t, err)
		require.NoError(t, mw.Close())
		req := rchRequest(http.MethodPost, buf, withInfo)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		rec := httptest.NewRecorder()
		rchHandler(ctrl, nil).UploadPackageFile(rec, req)
		return rec
	}

	t.Run("uploads the package", func(t *testing.T) {
		ctrl := &rchPythonCtrl{}
		rec := post(t, ctrl, "content", true)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), "Sha256: sha")
		require.Equal(t, "requests-2.0.0.whl", ctrl.last.Filename)
	})
	t.Run("missing content part", func(t *testing.T) {
		require.GreaterOrEqual(t, post(t, &rchPythonCtrl{}, "other", true).Code, http.StatusBadRequest)
	})
	t.Run("no info in context", func(t *testing.T) {
		require.Equal(t, http.StatusInternalServerError, post(t, &rchPythonCtrl{}, "content", false).Code)
	})
	t.Run("controller error", func(t *testing.T) {
		rec := post(t, &rchPythonCtrl{err: errors.New("boom")}, "content", true)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

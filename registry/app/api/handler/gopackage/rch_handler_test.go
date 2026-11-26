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

package gopackage_test

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
	gopackagectrl "github.com/harness/gitness/registry/app/api/controller/pkg/gopackage"
	gopackagehandler "github.com/harness/gitness/registry/app/api/handler/gopackage"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	gotype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/request"

	"github.com/stretchr/testify/require"
)

type rchPkgHandler struct {
	packages.Handler
	err error
}

func (h rchPkgHandler) GetArtifactInfo(*http.Request) (pkg.ArtifactInfo, error) {
	return pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"}, h.err
}

type rchGoCtrl struct {
	gopackagectrl.Controller
	err      error
	redirect string
	noBody   bool
	last     *gotype.ArtifactInfo
}

func (c *rchGoCtrl) base() gopackagectrl.BaseResponse {
	return gopackagectrl.BaseResponse{
		Error:           c.err,
		ResponseHeaders: &commons.ResponseHeaders{Code: http.StatusOK, Headers: map[string]string{"X-Rch": "1"}},
	}
}

func (c *rchGoCtrl) UploadPackage(
	_ context.Context, info *gotype.ArtifactInfo, _ io.ReadCloser, _ io.ReadCloser,
) *gopackagectrl.UploadFileResponse {
	c.last = info
	return &gopackagectrl.UploadFileResponse{BaseResponse: c.base(), Status: "success"}
}

func (c *rchGoCtrl) DownloadPackageFile(
	context.Context, *gotype.ArtifactInfo,
) *gopackagectrl.DownloadFileResponse {
	resp := &gopackagectrl.DownloadFileResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
	if !c.noBody {
		resp.ReadCloser = io.NopCloser(strings.NewReader("module-zip"))
	}
	return resp
}

func (c *rchGoCtrl) RegeneratePackageIndex(
	_ context.Context, info *gotype.ArtifactInfo,
) *gopackagectrl.RegeneratePackageIndexResponse {
	c.last = info
	return &gopackagectrl.RegeneratePackageIndexResponse{BaseResponse: c.base(), Ok: true}
}

func (c *rchGoCtrl) RegeneratePackageMetadata(
	_ context.Context, info *gotype.ArtifactInfo,
) *gopackagectrl.RegeneratePackageMetadataResponse {
	c.last = info
	return &gopackagectrl.RegeneratePackageMetadataResponse{BaseResponse: c.base(), Ok: true}
}

func rchHandler(ctrl gopackagectrl.Controller, baseErr error) gopackagehandler.Handler {
	base := packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
	return gopackagehandler.NewHandler(ctrl, rchPkgHandler{Handler: base, err: baseErr})
}

func rchRequest(method, target string, body io.Reader, withInfo bool) *http.Request {
	req := httptest.NewRequest(method, target, body)
	if withInfo {
		info := &gotype.ArtifactInfo{
			ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, RegIdentifier: "reg"},
			FileName:     "v1.0.0.zip",
		}
		req = req.WithContext(request.WithArtifactInfo(req.Context(), info))
	}
	return req
}

func rchUploadBody(t *testing.T, parts map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	for _, name := range []string{"info", "mod", "zip", "extra"} {
		content, ok := parts[name]
		if !ok {
			continue
		}
		w, err := mw.CreateFormFile(name, name+".bin")
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

const (
	rchInfoJSON = `{"Version":"v1.2.3","Time":"2026-01-01T00:00:00Z"}`
	rchModFile  = "module example.com/hello\n\ngo 1.22\n"
)

func TestRchGoGetPackageArtifactInfo(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		baseErr   error
		wantErr   bool
		wantImage string
		wantVer   string
		wantFile  string
	}{
		{name: "name only", params: map[string]string{"name": "example.com/hello"}, wantImage: "example.com/hello"},
		{name: "latest", params: map[string]string{"*": "example.com/hello/@latest"},
			wantImage: "example.com/hello", wantVer: "@latest"},
		{name: "zip download", params: map[string]string{"*": "example.com/hello/@v/v1.0.2.zip"},
			wantImage: "example.com/hello", wantVer: "v1.0.2", wantFile: "v1.0.2.zip"},
		{name: "unsupported path", params: map[string]string{"*": "example.com/hello/other"}, wantErr: true},
		{name: "unparsable download path", params: map[string]string{"*": "@v/"}, wantErr: true},
		{name: "base info failure", baseErr: errors.New("no registry"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchHandler(&rchGoCtrl{}, tc.baseErr)
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			for k, v := range tc.params {
				req.SetPathValue(k, v)
			}
			got, err := h.GetPackageArtifactInfo(req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantImage, got.GetImage())
			require.Equal(t, tc.wantVer, got.GetVersion())
			require.Equal(t, tc.wantFile, got.GetFileName())
		})
	}
}

func TestRchGoDownloadPackageFile(t *testing.T) {
	t.Run("serves the module", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchRequest(http.MethodGet, "/x", nil, true))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "module-zip", rec.Body.String())
		require.Equal(t, "1", rec.Header().Get("X-Rch"))
	})
	t.Run("redirects", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{redirect: "http://cdn.example/mod.zip"}, nil)
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchRequest(http.MethodGet, "/x", nil, true))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	})
	t.Run("nothing to serve", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{noBody: true}, nil)
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchRequest(http.MethodGet, "/x", nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("controller error", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{err: errors.New("boom")}, nil)
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchRequest(http.MethodGet, "/x", nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.DownloadPackageFile(rec, rchRequest(http.MethodGet, "/x", nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestRchGoRegenerate(t *testing.T) {
	t.Run("index", func(t *testing.T) {
		ctrl := &rchGoCtrl{}
		h := rchHandler(ctrl, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageIndex(rec, rchRequest(http.MethodPost, "/x?image='example.com/hello'", nil, true))
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"ok":true}`, rec.Body.String())
		require.Equal(t, "example.com/hello", ctrl.last.Image)
	})
	t.Run("index requires an image", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageIndex(rec, rchRequest(http.MethodPost, "/x", nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("index controller error", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{err: errors.New("boom")}, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageIndex(rec, rchRequest(http.MethodPost, "/x?image=a", nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("index without info", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageIndex(rec, rchRequest(http.MethodPost, "/x?image=a", nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("metadata", func(t *testing.T) {
		ctrl := &rchGoCtrl{}
		h := rchHandler(ctrl, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageMetadata(rec, rchRequest(http.MethodPost, "/x?image=a&version='v1.0.0'", nil, true))
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"ok":true}`, rec.Body.String())
		require.Equal(t, "v1.0.0", ctrl.last.Version)
	})
	t.Run("metadata requires image and version", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageMetadata(rec, rchRequest(http.MethodPost, "/x?image=a", nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("metadata controller error", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{err: errors.New("boom")}, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageMetadata(rec, rchRequest(http.MethodPost, "/x?image=a&version=v1", nil, true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("metadata without info", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.RegeneratePackageMetadata(rec, rchRequest(http.MethodPost, "/x?image=a&version=v1", nil, false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestRchGoUploadPackage(t *testing.T) {
	full := map[string]string{"info": rchInfoJSON, "mod": rchModFile, "zip": "zip-bytes"}

	post := func(
		t *testing.T, h gopackagehandler.Handler, parts map[string]string, withInfo bool,
	) *httptest.ResponseRecorder {
		t.Helper()
		body, contentType := rchUploadBody(t, parts)
		req := rchRequest(http.MethodPut, "/x", body, withInfo)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		h.UploadPackage(rec, req)
		return rec
	}

	t.Run("uploads the package", func(t *testing.T) {
		ctrl := &rchGoCtrl{}
		rec := post(t, rchHandler(ctrl, nil), full, true)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"status":"success"`)
		require.Equal(t, "example.com/hello", ctrl.last.Image)
		require.Equal(t, "v1.2.3", ctrl.last.Version)
	})
	t.Run("unknown parts are skipped", func(t *testing.T) {
		parts := map[string]string{"extra": "x", "info": rchInfoJSON, "mod": rchModFile, "zip": "z"}
		rec := post(t, rchHandler(&rchGoCtrl{}, nil), parts, true)
		require.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("missing parts", func(t *testing.T) {
		for _, missing := range []string{"info", "mod", "zip"} {
			parts := map[string]string{}
			for k, v := range full {
				if k != missing {
					parts[k] = v
				}
			}
			rec := post(t, rchHandler(&rchGoCtrl{}, nil), parts, true)
			require.Equal(t, http.StatusInternalServerError, rec.Code, "missing %s", missing)
		}
	})
	t.Run("invalid info file", func(t *testing.T) {
		parts := map[string]string{"info": "not-json", "mod": rchModFile, "zip": "z"}
		rec := post(t, rchHandler(&rchGoCtrl{}, nil), parts, true)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("mod file without module name", func(t *testing.T) {
		parts := map[string]string{"info": rchInfoJSON, "mod": "go 1.22\n", "zip": "z"}
		rec := post(t, rchHandler(&rchGoCtrl{}, nil), parts, true)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("not multipart", func(t *testing.T) {
		h := rchHandler(&rchGoCtrl{}, nil)
		rec := httptest.NewRecorder()
		h.UploadPackage(rec, rchRequest(http.MethodPut, "/x", strings.NewReader("raw"), true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("controller error", func(t *testing.T) {
		rec := post(t, rchHandler(&rchGoCtrl{err: errors.New("boom")}, nil), full, true)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := post(t, rchHandler(&rchGoCtrl{}, nil), full, false)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

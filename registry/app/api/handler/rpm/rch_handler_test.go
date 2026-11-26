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

package rpm_test

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
	rpmctrl "github.com/harness/gitness/registry/app/api/controller/pkg/rpm"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	rpmhandler "github.com/harness/gitness/registry/app/api/handler/rpm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
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

type rchRPMCtrl struct {
	rpmctrl.Controller
	err      error
	redirect string
	noBody   bool
	last     rpmtype.ArtifactInfo
	lastFile string
}

func (c *rchRPMCtrl) base() rpmctrl.BaseResponse {
	return rpmctrl.BaseResponse{
		Error:           c.err,
		ResponseHeaders: &commons.ResponseHeaders{Code: http.StatusOK, Headers: map[string]string{"X-Rch": "1"}},
	}
}

func (c *rchRPMCtrl) UploadPackageFile(
	_ context.Context, info rpmtype.ArtifactInfo, _ multipart.Part, fileName string,
) *rpmctrl.PutArtifactResponse {
	c.last, c.lastFile = info, fileName
	return &rpmctrl.PutArtifactResponse{BaseResponse: c.base(), Sha256: "sha"}
}

func (c *rchRPMCtrl) DownloadPackageFile(_ context.Context, info rpmtype.ArtifactInfo) *rpmctrl.GetArtifactResponse {
	c.last = info
	resp := &rpmctrl.GetArtifactResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
	if !c.noBody {
		resp.ReadCloser = io.NopCloser(strings.NewReader("rpm-bytes"))
	}
	return resp
}

func (c *rchRPMCtrl) GetRepoData(
	_ context.Context, info rpmtype.ArtifactInfo, fileName string,
) *rpmctrl.GetRepoDataResponse {
	c.last, c.lastFile = info, fileName
	resp := &rpmctrl.GetRepoDataResponse{BaseResponse: c.base(), RedirectURL: c.redirect}
	if !c.noBody {
		resp.ReadCloser = io.NopCloser(strings.NewReader("<repomd/>"))
	}
	return resp
}

func rchHandler(ctrl rpmctrl.Controller, baseErr error) rpmhandler.Handler {
	base := packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
	return rpmhandler.NewHandler(ctrl, rchPkgHandler{Handler: base, err: baseErr})
}

func rchRequest(method, target string, body io.Reader, withInfo bool) *http.Request {
	req := httptest.NewRequest(method, target, body)
	if withInfo {
		info := &rpmtype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}}}
		req = req.WithContext(request.WithArtifactInfo(req.Context(), info))
	}
	return req
}

func rchSetPath(req *http.Request, values map[string]string) {
	for k, v := range values {
		req.SetPathValue(k, v)
	}
}

func TestRchRPMGetPackageArtifactInfo(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		baseErr   error
		wantErr   bool
		wantImage string
		wantVer   string
	}{
		{name: "name only", params: map[string]string{"name": "bash"}, wantImage: "bash"},
		{name: "escaped name and version",
			params:    map[string]string{"name": "my%2Dpkg", "version": "1.2", "architecture": "x86_64"},
			wantImage: "my-pkg", wantVer: "1.2.x86_64"},
		{name: "version needs architecture", params: map[string]string{"name": "bash", "version": "1.2"},
			wantImage: "bash"},
		{name: "bad name escape", params: map[string]string{"name": "%zz"}, wantErr: true},
		{name: "bad version escape",
			params:  map[string]string{"name": "bash", "version": "%zz", "architecture": "x86_64"},
			wantErr: true},
		{name: "base info failure", baseErr: errors.New("no registry"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchHandler(&rchRPMCtrl{}, tc.baseErr)
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			rchSetPath(req, tc.params)
			got, err := h.GetPackageArtifactInfo(req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantImage, got.GetImage())
			require.Equal(t, tc.wantVer, got.GetVersion())
		})
	}
}

func rchDownloadRequest(t *testing.T, withInfo bool, overrides map[string]string) *http.Request {
	t.Helper()
	req := rchRequest(http.MethodGet, "/x", nil, withInfo)
	params := map[string]string{
		"name": "bash", "version": "5.2", "architecture": "x86_64", "file": "bash.rpm", "*": "Packages/b",
	}
	for k, v := range overrides {
		params[k] = v
	}
	rchSetPath(req, params)
	return req
}

func TestRchRPMDownloadPackageFile(t *testing.T) {
	t.Run("serves the package", func(t *testing.T) {
		ctrl := &rchRPMCtrl{}
		rec := httptest.NewRecorder()
		rchHandler(ctrl, nil).DownloadPackageFile(rec, rchDownloadRequest(t, true, nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "rpm-bytes", rec.Body.String())
		require.Equal(t, "5.2.x86_64", ctrl.last.Version)
		require.Equal(t, "bash.rpm", ctrl.last.FileName)
		require.Equal(t, "Packages/b", ctrl.last.PackagePath)
		require.Equal(t, "x86_64", ctrl.last.Arch)
	})
	t.Run("redirects", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{redirect: "http://cdn.example/bash.rpm"}, nil).
			DownloadPackageFile(rec, rchDownloadRequest(t, true, nil))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	})
	t.Run("nothing to serve", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{noBody: true}, nil).DownloadPackageFile(rec, rchDownloadRequest(t, true, nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("controller error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{err: errors.New("boom")}, nil).
			DownloadPackageFile(rec, rchDownloadRequest(t, true, nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{}, nil).DownloadPackageFile(rec, rchDownloadRequest(t, false, nil))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	for _, key := range []string{"name", "version", "file", "*"} {
		t.Run("bad escape in "+key, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rchHandler(&rchRPMCtrl{}, nil).
				DownloadPackageFile(rec, rchDownloadRequest(t, true, map[string]string{key: "%zz"}))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestRchRPMGetRepoData(t *testing.T) {
	newReq := func(withInfo bool) *http.Request {
		req := rchRequest(http.MethodGet, "/x", nil, withInfo)
		req.SetPathValue("file", "repomd.xml")
		return req
	}
	t.Run("serves the repo data", func(t *testing.T) {
		ctrl := &rchRPMCtrl{}
		rec := httptest.NewRecorder()
		rchHandler(ctrl, nil).GetRepoData(rec, newReq(true))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "<repomd/>", rec.Body.String())
		require.Equal(t, "repomd.xml", ctrl.lastFile)
	})
	t.Run("redirects", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{redirect: "http://cdn.example/repomd.xml"}, nil).GetRepoData(rec, newReq(true))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	})
	t.Run("nothing to serve", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{noBody: true}, nil).GetRepoData(rec, newReq(true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("controller error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{err: errors.New("boom")}, nil).GetRepoData(rec, newReq(true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rchHandler(&rchRPMCtrl{}, nil).GetRepoData(rec, newReq(false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func rchRPMUpload(t *testing.T, field string) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	w, err := mw.CreateFormFile(field, "bash.rpm")
	require.NoError(t, err)
	_, err = w.Write([]byte("rpm-content"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

func TestRchRPMUploadPackageFile(t *testing.T) {
	post := func(t *testing.T, ctrl *rchRPMCtrl, field string, withInfo bool) *httptest.ResponseRecorder {
		t.Helper()
		body, contentType := rchRPMUpload(t, field)
		req := rchRequest(http.MethodPut, "/x", body, withInfo)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		rchHandler(ctrl, nil).UploadPackageFile(rec, req)
		return rec
	}

	t.Run("uploads the package", func(t *testing.T) {
		ctrl := &rchRPMCtrl{}
		rec := post(t, ctrl, "file", true)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), "Sha256: sha")
		require.Equal(t, "bash.rpm", ctrl.lastFile)
	})
	t.Run("missing file part", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, post(t, &rchRPMCtrl{}, "other", true).Code)
	})
	t.Run("no info in context", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, post(t, &rchRPMCtrl{}, "file", false).Code)
	})
	t.Run("controller error", func(t *testing.T) {
		require.Equal(t, http.StatusInternalServerError,
			post(t, &rchRPMCtrl{err: errors.New("boom")}, "file", true).Code)
	})
}

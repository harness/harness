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

package generic_test

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

	apirequest "github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/audit"
	genericctrl "github.com/harness/gitness/registry/app/api/controller/pkg/generic"
	generichandler "github.com/harness/gitness/registry/app/api/handler/generic"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	genericpkg "github.com/harness/gitness/registry/app/pkg/generic"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	generictype "github.com/harness/gitness/registry/app/pkg/types/generic"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/request"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const (
	rchParentID = int64(5)
	rchRegID    = int64(9)
)

type rchAuthorizer struct {
	authz.Authorizer
}

func (rchAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return true, nil
}

func (rchAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return true, nil
}

type rchAudit struct {
	audit.Service
}

func (rchAudit) Log(
	context.Context, types.Principal, audit.Resource, audit.Action, string, ...audit.Option,
) error {
	return nil
}

type rchSpaceIDCache struct{ space *types.SpaceCore }

func (c rchSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if c.space.ID != id {
		return nil, basestore.ErrResourceNotFound
	}
	return c.space, nil
}
func (rchSpaceIDCache) Stats() (int64, int64)            { return 0, 0 }
func (rchSpaceIDCache) Evict(_ context.Context, _ int64) {}

type rchSpacePathCache struct{ space *types.SpaceCore }

func (c rchSpacePathCache) Get(_ context.Context, path string) (*types.SpacePath, error) {
	if c.space.Path != path {
		return nil, basestore.ErrResourceNotFound
	}
	return &types.SpacePath{Value: path, IsPrimary: true, SpaceID: c.space.ID}, nil
}
func (rchSpacePathCache) Stats() (int64, int64)             { return 0, 0 }
func (rchSpacePathCache) Evict(_ context.Context, _ string) {}

func rchSpaceFinder() refcache.SpaceFinder {
	space := &types.SpaceCore{ID: rchParentID, Path: "root", Identifier: "root"}
	return refcache.NewSpaceFinder(
		rchSpaceIDCache{space: space}, rchSpacePathCache{space: space}, nil, storecache.Evictor[*types.SpaceCore]{},
	)
}

type rchRegistryDao struct {
	store.RegistryRepository
	reg *registrytypes.Registry
	err error
}

func (d *rchRegistryDao) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.reg, d.err
}

func (d *rchRegistryDao) GetByRootParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.reg, d.err
}

type rchRegistryFinder struct {
	refcache2.RegistryFinder
	reg *registrytypes.Registry
	err error
}

func (f rchRegistryFinder) FindByRootRef(
	_ context.Context, _ string, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return f.reg, f.err
}

type rchWrapper struct {
	interfaces.PackageWrapper
}

func (rchWrapper) IsFileOperationSupported(string) bool { return true }

type rchQuarantine struct {
	quarantine.Finder
}

func (rchQuarantine) CheckArtifactQuarantineStatus(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return nil
}

type rchFileManager struct {
	filemanager.FileManager
}

func (rchFileManager) UploadFileNoDBUpdate(
	_ context.Context, _ string, _ multipart.File, reader io.Reader, _ int64, _ int64,
) (registrytypes.FileInfo, error) {
	_, _ = io.Copy(io.Discard, reader)
	return registrytypes.FileInfo{Sha256: "sha-of-file"}, nil
}

func (rchFileManager) PostFileUpload(
	_ context.Context, _ string, _ int64, _ int64, _ string, _ registrytypes.FileInfo, _ int64,
) error {
	return nil
}

func (rchFileManager) DownloadFileByPath(
	_ context.Context, _ string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	return nil, 0, "http://redirect", nil
}

type rchImageDao struct {
	store.ImageRepository
}

func (rchImageDao) GetByName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Image, error) {
	return nil, basestore.ErrResourceNotFound
}

func (rchImageDao) CreateOrUpdate(_ context.Context, img *registrytypes.Image) error {
	img.ID = 3
	return nil
}

type rchArtifactDao struct {
	store.ArtifactRepository
}

func (rchArtifactDao) GetByName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Artifact, error) {
	return nil, basestore.ErrResourceNotFound
}

func (rchArtifactDao) CreateOrUpdate(context.Context, *registrytypes.Artifact) (int64, error) {
	return 1, nil
}

type rchTx struct{}

func (rchTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	return fn(ctx)
}

type rchGenericRegistry struct {
	genericpkg.Registry
	err      error
	redirect string
}

func (r *rchGenericRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *rchGenericRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeGENERIC}
}

func (r *rchGenericRegistry) PutFile(
	_ context.Context, _ generictype.ArtifactInfo, _ io.ReadCloser, _ string,
) (*commons.ResponseHeaders, string, error) {
	if r.err != nil {
		return nil, "", r.err
	}
	return &commons.ResponseHeaders{Headers: map[string]string{"X-Rch": "put"}}, "put-sha", nil
}

func (r *rchGenericRegistry) DownloadFile(
	_ context.Context, _ generictype.ArtifactInfo, _ string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	if r.err != nil {
		return nil, nil, nil, "", r.err
	}
	if r.redirect != "" {
		return &commons.ResponseHeaders{}, nil, nil, r.redirect, nil
	}
	return &commons.ResponseHeaders{Headers: map[string]string{"X-Rch": "get"}}, nil,
		io.NopCloser(strings.NewReader("file-content")), "", nil
}

func (r *rchGenericRegistry) DeleteFile(
	_ context.Context, _ generictype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &commons.ResponseHeaders{Code: http.StatusNoContent}, nil
}

func (r *rchGenericRegistry) HeadFile(
	_ context.Context, _ generictype.ArtifactInfo, _ string,
) (*commons.ResponseHeaders, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &commons.ResponseHeaders{Code: http.StatusOK, Headers: map[string]string{"X-Rch": "head"}}, nil
}

func rchGenericReg() *registrytypes.Registry {
	return &registrytypes.Registry{
		ID: rchRegID, Name: "reg", ParentID: rchParentID,
		PackageType: artifact.PackageTypeGENERIC, Type: artifact.RegistryTypeVIRTUAL,
	}
}

type rchBaseHandler struct {
	packages.Handler
	info pkg.ArtifactInfo
	err  error
}

func (h rchBaseHandler) GetArtifactInfo(*http.Request) (pkg.ArtifactInfo, error) {
	return h.info, h.err
}

type rchHandlerOpts struct {
	reg         *registrytypes.Registry
	daoErr      error
	finderErr   error
	registryErr error
	redirect    string
	baseErr     error
}

func rchSetup(t *testing.T, o rchHandlerOpts) *generichandler.Handler {
	t.Helper()
	if o.reg == nil {
		o.reg = rchGenericReg()
	}
	base.Register(&rchGenericRegistry{err: o.registryErr, redirect: o.redirect})
	t.Cleanup(func() { delete(base.TypeRegistry, "GENERIC:VIRTUAL") })

	dao := &rchRegistryDao{reg: o.reg, err: o.daoErr}
	dbStore := genericctrl.NewDBStore(dao, rchImageDao{}, rchArtifactDao{}, nil, nil)
	ctrl := genericctrl.NewController(
		nil, rchAuthorizer{}, rchFileManager{}, dbStore, rchTx{}, rchSpaceFinder(),
		nil, nil, rchQuarantine{}, nil, rchAudit{}, rchWrapper{},
	)
	realBase := packages.NewHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, refcache.SpaceFinder{}, nil, nil, nil, nil, nil, nil,
	)
	baseHandler := rchBaseHandler{
		Handler: realBase,
		info: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: rchParentID},
			RegIdentifier: "reg",
			RegistryID:    rchRegID,
		},
		err: o.baseErr,
	}
	return generichandler.NewGenericArtifactHandler(
		nil, ctrl, nil, nil, nil, nil, rchAuthorizer{}, baseHandler, rchSpaceFinder(),
		rchRegistryFinder{reg: o.reg, err: o.finderErr}, rchAudit{},
	)
}

func rchAuthedRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	session := &auth.Session{Principal: types.Principal{ID: 1}}
	return req.WithContext(apirequest.WithAuthSession(req.Context(), session))
}

func rchV2Info() generictype.ArtifactInfo {
	return generictype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: rchParentID},
			RegIdentifier: "reg",
			RegistryID:    rchRegID,
			Image:         "tool",
		},
		Version:  "1.0.0",
		FileName: "tool.bin",
		FilePath: "dir/tool.bin",
	}
}

func rchV2Request(method, target string, body io.Reader, withInfo bool) *http.Request {
	req := rchAuthedRequest(method, target, body)
	if withInfo {
		req = req.WithContext(request.WithArtifactInfo(req.Context(), rchV2Info()))
	}
	return req
}

func TestRchGenericExtractPathVars(t *testing.T) {
	tests := []struct {
		name                                string
		path                                string
		wantErr                             bool
		wantRoot, wantReg, wantArt, wantTag string
		wantFile                            string
	}{
		{name: "tag form", path: "/generic/root/reg/tool/1.0.0", wantRoot: "root", wantReg: "reg",
			wantArt: "tool", wantTag: "1.0.0"},
		{name: "file form", path: "/generic/root/reg/tool:1.0.0:dir/f.bin", wantRoot: "root", wantReg: "reg",
			wantArt: "tool", wantTag: "1.0.0", wantFile: "dir/f.bin"},
		{name: "wrong prefix", path: "/other/root/reg/tool/1", wantErr: true},
		{name: "missing registry", path: "/generic/root", wantErr: true},
		{name: "missing artifact", path: "/generic/root/reg", wantErr: true},
		{name: "colon without file", path: "/generic/root/reg/tool:1.0.0", wantErr: true},
		{name: "no tag", path: "/generic/root/reg/tool", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			root, reg, art, tag, file, _, err := generichandler.ExtractPathVars(req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{tc.wantRoot, tc.wantReg, tc.wantArt, tc.wantTag, tc.wantFile},
				[]string{root, reg, art, tag, file})
		})
	}
}

func TestRchGenericGetGenericArtifactInfo(t *testing.T) {
	blocked := rchGenericReg()
	blocked.BlockedPattern = pq.StringArray{"tool:1.0.0:.*"}
	notGeneric := rchGenericReg()
	notGeneric.PackageType = artifact.PackageTypeNPM
	orphan := rchGenericReg()
	orphan.ParentID = 777

	tests := []struct {
		name     string
		path     string
		opts     rchHandlerOpts
		wantErr  bool
		wantFile string
	}{
		{name: "tag path", path: "/generic/root/reg/tool/1.0.0", opts: rchHandlerOpts{}},
		{name: "file path", path: "/generic/root/reg/tool:1.0.0:dir/f.bin", opts: rchHandlerOpts{},
			wantFile: "dir/f.bin"},
		{name: "bad path", path: "/nope", wantErr: true},
		{name: "invalid registry identifier", path: "/generic/root/Bad_Reg!/tool/1.0.0", wantErr: true},
		{name: "invalid package name", path: "/generic/root/reg/-tool/1.0.0", wantErr: true},
		{name: "invalid version", path: "/generic/root/reg/tool/V1", wantErr: true},
		{name: "root not found", path: "/generic/unknown/reg/tool/1.0.0", wantErr: true},
		{name: "registry not found", path: "/generic/root/reg/tool/1.0.0",
			opts: rchHandlerOpts{daoErr: errors.New("missing")}, wantErr: true},
		{name: "not a generic registry", path: "/generic/root/reg/tool/1.0.0",
			opts: rchHandlerOpts{reg: notGeneric}, wantErr: true},
		{name: "parent not found", path: "/generic/root/reg/tool/1.0.0",
			opts: rchHandlerOpts{reg: orphan}, wantErr: true},
		{name: "blocked by pattern", path: "/generic/root/reg/tool:1.0.0:f.bin",
			opts: rchHandlerOpts{reg: blocked}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchSetup(t, tc.opts)
			info, err := h.GetGenericArtifactInfo(rchAuthedRequest(http.MethodGet, tc.path, nil))
			if tc.wantErr {
				require.False(t, commons.IsEmptyError(err))
				return
			}
			require.True(t, commons.IsEmptyError(err))
			require.Equal(t, "tool", info.Image)
			require.Equal(t, "1.0.0", info.Version)
			require.Equal(t, tc.wantFile, info.FileName)
			require.Equal(t, rchRegID, info.RegistryID)
		})
	}

	t.Run("info already in context", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		ctxInfo := pkg.GenericArtifactInfo{Version: "9.9.9"} //nolint:staticcheck
		req := httptest.NewRequest(http.MethodGet, "/anything", nil)
		req = req.WithContext(request.WithArtifactInfo(req.Context(), ctxInfo))
		info, err := h.GetGenericArtifactInfo(req)
		require.True(t, commons.IsEmptyError(err))
		require.Equal(t, "9.9.9", info.Version)
	})
}

func rchChiRequest(target string) *http.Request {
	req := rchAuthedRequest(http.MethodGet, target, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("rootIdentifier", "root")
	rctx.URLParams.Add("registryIdentifier", "reg")
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestRchGenericGetGenericArtifactInfoV2(t *testing.T) {
	blocked := rchGenericReg()
	blocked.AllowedPattern = pq.StringArray{"^nothing$"}
	npmReg := rchGenericReg()
	npmReg.PackageType = artifact.PackageTypeNPM

	tests := []struct {
		name     string
		target   string
		opts     rchHandlerOpts
		wantErr  bool
		wantInfo generictype.ArtifactInfo
	}{
		{name: "generic file", target: "/pkg/root/reg/files/tool/1.0.0/dir/tool.bin",
			wantInfo: generictype.ArtifactInfo{
				ArtifactInfo: pkg.ArtifactInfo{Image: "tool"}, Version: "1.0.0",
				FileName: "tool.bin", FilePath: "dir/tool.bin",
			}},
		{name: "no path", target: "/pkg/root/reg/files", wantErr: true},
		{name: "too few segments", target: "/pkg/root/reg/files/tool/1.0.0", wantErr: true},
		{name: "invalid package", target: "/pkg/root/reg/files/-tool/1.0.0/f.bin", wantErr: true},
		{name: "invalid file path", target: "/pkg/root/reg/files/tool/1.0.0/dir/..bad/../x", wantErr: true},
		{name: "blocked pattern", target: "/pkg/root/reg/files/tool/1.0.0/f.bin",
			opts: rchHandlerOpts{reg: blocked}, wantErr: true},
		{name: "registry not found", target: "/pkg/root/reg/files/tool/1.0.0/f.bin",
			opts: rchHandlerOpts{finderErr: errors.New("missing")}, wantErr: true},
		{name: "other package file", target: "/pkg/root/reg/files/a/b/c.txt", opts: rchHandlerOpts{reg: npmReg},
			wantInfo: generictype.ArtifactInfo{FileName: "c.txt", FilePath: "a/b/c.txt"}},
		{name: "other package no path", target: "/pkg/root/reg/files", opts: rchHandlerOpts{reg: npmReg},
			wantErr: true},
		{name: "other package bad path", target: "/pkg/root/reg/files/a/../b", opts: rchHandlerOpts{reg: npmReg},
			wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchSetup(t, tc.opts)
			info, err := h.GetGenericArtifactInfoV2(rchChiRequest(tc.target))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantInfo.Image, info.Image)
			require.Equal(t, tc.wantInfo.Version, info.Version)
			require.Equal(t, tc.wantInfo.FileName, info.FileName)
			require.Equal(t, tc.wantInfo.FilePath, info.FilePath)
			require.Equal(t, rchRegID, info.RegistryID)
		})
	}

	t.Run("root not found", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		req := rchAuthedRequest(http.MethodGet, "/pkg/other/reg/files/tool/1.0.0/f.bin", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("rootIdentifier", "other")
		rctx.URLParams.Add("registryIdentifier", "reg")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		_, err := h.GetGenericArtifactInfoV2(req)
		require.Error(t, err)
	})
}

func TestRchGenericGetPackageArtifactInfo(t *testing.T) {
	t.Run("files path delegates to V2", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		info, err := h.GetPackageArtifactInfo(rchChiRequest("/pkg/root/reg/files/tool/1.0.0/dir/tool.bin"))
		require.NoError(t, err)
		require.Equal(t, "tool", info.GetImage())
		require.Equal(t, "1.0.0", info.GetVersion())
	})
	t.Run("files path error is wrapped", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{finderErr: errors.New("missing")})
		_, err := h.GetPackageArtifactInfo(rchChiRequest("/pkg/root/reg/files/tool/1.0.0/dir/tool.bin"))
		require.Error(t, err)
	})
	t.Run("legacy path", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		req := rchAuthedRequest(http.MethodGet, "/generic/root/reg/tool/1.0.0", nil)
		req.SetPathValue("package", "tool")
		req.SetPathValue("version", "1.0.0")
		info, err := h.GetPackageArtifactInfo(req)
		require.NoError(t, err)
		require.Equal(t, "tool", info.GetImage())
		require.Equal(t, "1.0.0", info.GetVersion())
	})
	t.Run("legacy path with invalid version", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		req := rchAuthedRequest(http.MethodGet, "/generic/root/reg/tool/V1", nil)
		req.SetPathValue("package", "tool")
		req.SetPathValue("version", "V1")
		_, err := h.GetPackageArtifactInfo(req)
		require.Error(t, err)
	})
	t.Run("legacy path with base error", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{baseErr: errors.New("no registry")})
		_, err := h.GetPackageArtifactInfo(rchAuthedRequest(http.MethodGet, "/generic/root/reg/tool/1.0.0", nil))
		require.Error(t, err)
	})
}

func TestRchGenericV2FileHandlers(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		call     func(h *generichandler.Handler) http.HandlerFunc
		wantCode int
		wantBody string
		wantHdr  string
		body     string
	}{
		{name: "get", method: http.MethodGet, call: func(h *generichandler.Handler) http.HandlerFunc { return h.GetFile },
			wantCode: http.StatusOK, wantBody: "file-content", wantHdr: "get"},
		{name: "head", method: http.MethodHead, call: func(h *generichandler.Handler) http.HandlerFunc { return h.HeadFile },
			wantCode: http.StatusOK, wantHdr: "head"},
		{name: "put", method: http.MethodPut, call: func(h *generichandler.Handler) http.HandlerFunc { return h.PutFile },
			wantCode: http.StatusCreated, wantHdr: "put", body: "data"},
		{name: "delete", method: http.MethodDelete, call: func(h *generichandler.Handler) http.HandlerFunc {
			return h.DeleteFile
		}, wantCode: http.StatusNoContent},
	}
	for _, tc := range tests {
		t.Run(tc.name+" success", func(t *testing.T) {
			h := rchSetup(t, rchHandlerOpts{})
			rec := httptest.NewRecorder()
			tc.call(h)(rec, rchV2Request(tc.method, "/pkg/root/reg/files/tool/1.0.0/dir/tool.bin",
				strings.NewReader(tc.body), true))
			require.Equal(t, tc.wantCode, rec.Code)
			require.Contains(t, rec.Body.String(), tc.wantBody)
			if tc.wantHdr != "" {
				require.Equal(t, tc.wantHdr, rec.Header().Get("X-Rch"))
			}
		})
		t.Run(tc.name+" no info in context", func(t *testing.T) {
			h := rchSetup(t, rchHandlerOpts{})
			rec := httptest.NewRecorder()
			tc.call(h)(rec, rchV2Request(tc.method, "/x", strings.NewReader(tc.body), false))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
		t.Run(tc.name+" registry error", func(t *testing.T) {
			h := rchSetup(t, rchHandlerOpts{registryErr: errors.New("boom")})
			rec := httptest.NewRecorder()
			tc.call(h)(rec, rchV2Request(tc.method, "/x", strings.NewReader(tc.body), true))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
		})
	}
}

func TestRchGenericGetFileRedirectAndDisposition(t *testing.T) {
	h := rchSetup(t, rchHandlerOpts{redirect: "http://cdn.example/file"})
	rec := httptest.NewRecorder()
	h.GetFile(rec, rchV2Request(http.MethodGet, "/x", nil, true))
	require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	require.Equal(t, "http://cdn.example/file", rec.Header().Get("Location"))

	h = rchSetup(t, rchHandlerOpts{})
	rec = httptest.NewRecorder()
	h.GetFile(rec, rchV2Request(http.MethodGet, "/x", nil, true))
	require.Equal(t, `attachment; filename="tool.bin"`, rec.Header().Get("Content-Disposition"))
}

func rchPushBody(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	w, err := mw.CreateFormFile("file", "upload.bin")
	require.NoError(t, err)
	_, err = w.Write([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, mw.WriteField("filename", "tool.bin"))
	require.NoError(t, mw.WriteField("description", "d"))
	require.NoError(t, mw.Close())
	return buf, mw.FormDataContentType()
}

func TestRchGenericPushArtifact(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		body, contentType := rchPushBody(t)
		req := rchAuthedRequest(http.MethodPost, "/generic/root/reg/tool/1.0.0", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		h.PushArtifact(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)
		require.Contains(t, rec.Body.String(), "Sha256: sha-of-file")
	})
	t.Run("upload error", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		req := rchAuthedRequest(http.MethodPost, "/generic/root/reg/tool/1.0.0", strings.NewReader("raw"))
		rec := httptest.NewRecorder()
		h.PushArtifact(rec, req)
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		rec := httptest.NewRecorder()
		h.PushArtifact(rec, rchAuthedRequest(http.MethodPost, "/bad", nil))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestRchGenericPullArtifact(t *testing.T) {
	t.Run("redirect", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		rec := httptest.NewRecorder()
		h.PullArtifact(rec, rchAuthedRequest(http.MethodGet, "/generic/root/reg/tool:1.0.0:f.bin", nil))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		require.Equal(t, "http://redirect", rec.Header().Get("Location"))
		require.Equal(t, "attachment; filename=f.bin", rec.Header().Get("Content-Disposition"))
	})
	t.Run("filename from form value", func(t *testing.T) {
		h := rchSetup(t, rchHandlerOpts{})
		rec := httptest.NewRecorder()
		h.PullArtifact(rec, rchAuthedRequest(http.MethodGet, "/generic/root/reg/tool/1.0.0?filename=q.bin", nil))
		require.Equal(t, "attachment; filename=q.bin", rec.Header().Get("Content-Disposition"))
	})
}

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

package packages

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apirequest "github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	commonstype "github.com/harness/gitness/registry/app/pkg/types/commons"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/request"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type rchAuthorizer struct {
	authz.Authorizer
	allow bool
}

func (a rchAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.allow, nil
}

type rchAudit struct {
	audit.Service
	calls int
}

func (a *rchAudit) Log(
	context.Context, types.Principal, audit.Resource, audit.Action, string, ...audit.Option,
) error {
	a.calls++
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
	space := &types.SpaceCore{ID: 5, Path: "root", Identifier: "root"}
	return refcache.NewSpaceFinder(
		rchSpaceIDCache{space: space}, rchSpacePathCache{space: space}, nil, storecache.Evictor[*types.SpaceCore]{},
	)
}

type rchRegFinder struct {
	refcache2.RegistryFinder
	reg *registrytypes.Registry
	err error
}

func (f rchRegFinder) FindByRootParentID(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return f.reg, f.err
}

type rchWrapper struct {
	interfaces.PackageWrapper
}

func (rchWrapper) GetPackageTypeFromPathPackageType(pathType string) (string, error) {
	switch pathType {
	case "npm":
		return string(artifact.PackageTypeNPM), nil
	case "maven":
		return string(artifact.PackageTypeMAVEN), nil
	}
	return "", errors.New("unknown package type")
}

type rchStats struct {
	store.DownloadStatRepository
	err      error
	recorded int
}

func (s *rchStats) CreateByRegistryIDImageAndArtifactName(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	s.recorded++
	return s.err
}

type rchQuarantine struct {
	quarantine.Finder
	err error
}

func (q rchQuarantine) CheckArtifactQuarantineStatus(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return q.err
}

type rchURLProvider struct {
	urlprovider.Provider
}

func (rchURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, params ...string) string {
	return "http://pkg.example/" + regRef + "/" + pkgType + "/" + strings.Join(params, "/")
}

type rchHandlerOpts struct {
	reg        *registrytypes.Registry
	regErr     error
	allow      bool
	statsErr   error
	quarantine error
}

func rchNewHandler(o rchHandlerOpts) (*handler, *rchStats, *rchAudit) {
	if o.reg == nil {
		o.reg = &registrytypes.Registry{ID: 9, Name: "reg", ParentID: 5, PackageType: artifact.PackageTypeNPM}
	}
	stats := &rchStats{err: o.statsErr}
	auditSvc := &rchAudit{}
	h := NewHandler(
		nil, stats, nil, nil, nil, nil, nil, rchURLProvider{}, rchAuthorizer{allow: o.allow}, rchSpaceFinder(),
		rchRegFinder{reg: o.reg, err: o.regErr}, nil, rchQuarantine{err: o.quarantine}, rchWrapper{}, auditSvc, nil,
	)
	hnd, ok := h.(*handler)
	if !ok {
		panic("unexpected handler type")
	}
	return hnd, stats, auditSvc
}

func TestRchGetArtifactInfo(t *testing.T) {
	mavenReg := &registrytypes.Registry{ID: 9, Name: "reg", ParentID: 5, PackageType: artifact.PackageTypeMAVEN}
	orphanReg := &registrytypes.Registry{ID: 9, Name: "reg", ParentID: 99, PackageType: artifact.PackageTypeNPM}

	tests := []struct {
		name    string
		path    string
		opts    rchHandlerOpts
		wantErr bool
	}{
		{name: "ok", path: "/pkg/root/reg/npm/left-pad"},
		{name: "short path", path: "/pkg/root", wantErr: true},
		{name: "unknown package type", path: "/pkg/root/reg/cobol/x", wantErr: true},
		{name: "unknown root", path: "/pkg/nope/reg/npm/x", wantErr: true},
		{name: "unknown registry", path: "/pkg/root/reg/npm/x", opts: rchHandlerOpts{regErr: errors.New("missing")},
			wantErr: true},
		{name: "package type mismatch", path: "/pkg/root/reg/npm/x", opts: rchHandlerOpts{reg: mavenReg},
			wantErr: true},
		{name: "unknown parent space", path: "/pkg/root/reg/npm/x", opts: rchHandlerOpts{reg: orphanReg},
			wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := rchNewHandler(tc.opts)
			info, err := h.GetArtifactInfo(httptest.NewRequest(http.MethodGet, tc.path, nil))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "root", info.RootIdentifier)
			require.Equal(t, "reg", info.RegIdentifier)
			require.Equal(t, int64(9), info.RegistryID)
			require.Equal(t, int64(5), info.ParentID)
			require.Equal(t, artifact.PackageTypeNPM, info.PathPackageType)
		})
	}

	t.Run("target registry override from route", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{})
		req := httptest.NewRequest(http.MethodGet, "/pkg/root/reg/npm/x", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("targetRegistryIdentifier", "other")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		info, err := h.GetArtifactInfo(req)
		require.NoError(t, err)
		require.Equal(t, "other", info.RegIdentifier)
	})
}

func TestRchGetUtilityMethodArtifactInfo(t *testing.T) {
	orphanReg := &registrytypes.Registry{ID: 9, Name: "reg", ParentID: 99}
	tests := []struct {
		name    string
		path    string
		opts    rchHandlerOpts
		wantErr bool
	}{
		{name: "ok", path: "/pkg/root/reg/download/x"},
		{name: "short path", path: "/pkg/root", wantErr: true},
		{name: "unknown root", path: "/pkg/nope/reg/download/x", wantErr: true},
		{name: "unknown registry", path: "/pkg/root/reg/download/x",
			opts: rchHandlerOpts{regErr: errors.New("missing")}, wantErr: true},
		{name: "unknown parent space", path: "/pkg/root/reg/download/x", opts: rchHandlerOpts{reg: orphanReg},
			wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := rchNewHandler(tc.opts)
			info, err := h.GetUtilityMethodArtifactInfo(httptest.NewRequest(http.MethodGet, tc.path, nil))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "reg", info.RegIdentifier)
			require.Equal(t, int64(5), info.ParentID)
		})
	}

	t.Run("package artifact info wraps the utility info", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{})
		got, err := h.GetPackageArtifactInfo(httptest.NewRequest(http.MethodGet, "/pkg/root/reg/download/x", nil))
		require.NoError(t, err)
		require.Equal(t, int64(9), got.GetRegistryID())
		_, err = h.GetPackageArtifactInfo(httptest.NewRequest(http.MethodGet, "/pkg", nil))
		require.Error(t, err)
	})
}

func TestRchGetRegistryCheckAccess(t *testing.T) {
	ctxInfo := commonstype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{ParentID: 5},
			Registry: registrytypes.Registry{Name: "reg"},
		},
	}
	session := &auth.Session{Principal: types.Principal{ID: 1}}

	t.Run("allowed using info from context", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{allow: true})
		ctx := apirequest.WithAuthSession(request.WithArtifactInfo(context.Background(), ctxInfo), session)
		err := h.GetRegistryCheckAccess(ctx, httptest.NewRequest(http.MethodGet, "/x", nil),
			enum.PermissionArtifactsDownload)
		require.NoError(t, err)
	})
	t.Run("denied", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{allow: false})
		ctx := apirequest.WithAuthSession(request.WithArtifactInfo(context.Background(), ctxInfo), session)
		err := h.GetRegistryCheckAccess(ctx, httptest.NewRequest(http.MethodGet, "/x", nil),
			enum.PermissionArtifactsDownload)
		require.Error(t, err)
	})
	t.Run("info resolved from request when absent from context", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{allow: true})
		req := httptest.NewRequest(http.MethodGet, "/pkg/root/reg/npm/x", nil)
		ctx := apirequest.WithAuthSession(req.Context(), session)
		require.NoError(t, h.GetRegistryCheckAccess(ctx, req, enum.PermissionArtifactsDownload))
	})
	t.Run("info resolution failure", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{allow: true})
		err := h.GetRegistryCheckAccess(context.Background(), httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Error(t, err)
	})
}

func TestRchTrackDownloadStats(t *testing.T) {
	info := commonstype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{ParentID: 5}, RegistryID: 9, Image: "tool"},
		Version:      "1.0.0",
	}
	newReq := func() (*http.Request, context.Context) {
		ctx := request.WithArtifactInfo(context.Background(), info)
		session := &auth.Session{Principal: types.Principal{ID: 1}}
		ctx = apirequest.WithAuthSession(ctx, session)
		return httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx), ctx
	}

	t.Run("records stats and audits", func(t *testing.T) {
		h, stats, auditSvc := rchNewHandler(rchHandlerOpts{})
		req, ctx := newReq()
		require.NoError(t, h.TrackDownloadStats(ctx, req))
		require.Equal(t, 1, stats.recorded)
		require.Equal(t, 1, auditSvc.calls)
	})
	t.Run("store failure surfaces an internal error", func(t *testing.T) {
		h, _, _ := rchNewHandler(rchHandlerOpts{statsErr: errors.New("db")})
		req, ctx := newReq()
		require.ErrorIs(t, h.TrackDownloadStats(ctx, req), usererror.ErrInternal)
	})
}

func TestRchCheckQuarantineStatus(t *testing.T) {
	info := commonstype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{}, Image: "tool"},
		Version:      "1.0.0",
		FilePath:     "tool.bin",
	}
	ctx := request.WithArtifactInfo(context.Background(), info)

	tests := []struct {
		name    string
		finder  error
		wantErr error
	}{
		{name: "clean", finder: nil},
		{name: "quarantined", finder: usererror.ErrQuarantinedArtifact, wantErr: usererror.ErrQuarantinedArtifact},
		{name: "check failure is tolerated", finder: errors.New("cache down")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, _, _ := rchNewHandler(rchHandlerOpts{quarantine: tc.finder})
			err := h.CheckQuarantineStatus(ctx)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRchErrorHandling(t *testing.T) {
	h, _, _ := rchNewHandler(rchHandlerOpts{})
	ctx := context.Background()

	t.Run("HandleErrors2 writes the descriptor status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleErrors2(ctx, errcode.ErrCodeDenied.WithMessage("nope"), rec)
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "nope")
	})
	t.Run("HandleErrors2 ignores an empty error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleErrors2(ctx, errcode.Error{}, rec)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Body.String())
	})
	t.Run("HandleErrors uses the commons error status", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleErrors(ctx, errcode.Errors{&commons.Error{Status: http.StatusConflict}}, rec)
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	})
	t.Run("HandleErrors defaults to internal server error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleErrors(ctx, errcode.Errors{errors.New("boom")}, rec)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("HandleErrors ignores an empty list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleErrors(ctx, errcode.Errors{}, rec)
		require.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("HandleError translates user errors", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleError(ctx, rec, usererror.ErrQuarantinedArtifact)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("HandleError ignores nil", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.HandleError(ctx, rec, nil)
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestRchServeContentWithoutReader(t *testing.T) {
	h, _, _ := rchNewHandler(rchHandlerOpts{})
	rec := httptest.NewRecorder()
	h.ServeContent(rec, httptest.NewRequest(http.MethodGet, "/x", nil), nil, "file.txt")
	require.Equal(t, http.StatusOK, rec.Code)
	data, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	require.Empty(t, data)
	require.Nil(t, h.GetAuthenticator())
}

func TestRchDownloadFile(t *testing.T) {
	h, _, _ := rchNewHandler(rchHandlerOpts{})
	info := commonstype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{RootIdentifier: "root"}, RegIdentifier: "reg"},
	}

	newReq := func(target string, withInfo bool) *http.Request {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if withInfo {
			req = req.WithContext(request.WithArtifactInfo(req.Context(), info))
		}
		return req
	}

	t.Run("redirects to the files endpoint", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.DownloadFile(rec, newReq("/x?path=dir/file.bin", true))
		require.Equal(t, http.StatusPermanentRedirect, rec.Code)
		require.Equal(t, "http://pkg.example/root/reg//files/dir/file.bin", rec.Header().Get("Location"))
	})
	t.Run("path is required", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.DownloadFile(rec, newReq("/x", true))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("info is required", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.DownloadFile(rec, newReq("/x?path=a", false))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

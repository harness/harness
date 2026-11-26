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

package maven_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apirequest "github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	mavenhandler "github.com/harness/gitness/registry/app/api/handler/maven"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	liberrors "github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/request"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type rchAuthorizer struct {
	authz.Authorizer
	allow bool
}

func (a rchAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.allow, nil
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

type rchQuarantine struct {
	quarantine.Finder
	err error
}

func (q rchQuarantine) CheckArtifactQuarantineStatus(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return q.err
}

type rchMavenRegistry struct {
	maven.Registry
	errs     []error
	redirect string
}

func (r *rchMavenRegistry) GetMavenArtifactType() string { return "rch" }

func (r *rchMavenRegistry) HeadArtifact(
	context.Context, pkg.MavenArtifactInfo,
) (*commons.ResponseHeaders, []error) {
	return &commons.ResponseHeaders{Code: http.StatusOK, Headers: map[string]string{"X-Rch": "head"}}, r.errs
}

func (r *rchMavenRegistry) GetArtifact(
	context.Context, pkg.MavenArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, []error) {
	headers := &commons.ResponseHeaders{Headers: map[string]string{"X-Rch": "get"}}
	if r.errs != nil {
		return nil, nil, nil, "", r.errs
	}
	if r.redirect != "" {
		return headers, nil, nil, r.redirect, nil
	}
	return headers, nil, io.NopCloser(strings.NewReader("jar-bytes")), "", nil
}

type rchOpts struct {
	allow    bool
	reg      *registrytypes.Registry
	regErr   error
	errs     []error
	redirect string
	quar     error
}

func rchMavenReg() *registrytypes.Registry {
	return &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5,
		PackageType: artifact.PackageTypeMAVEN, Type: artifact.RegistryTypeVIRTUAL,
	}
}

func rchSetup(t *testing.T, o rchOpts) *mavenhandler.Handler {
	t.Helper()
	if o.reg == nil {
		o.reg = rchMavenReg()
	}
	dbStore := maven.NewDBStore(nil, nil, nil, nil, nil, nil, nil, nil)
	ctrl := maven.NewController(
		nil, nil, rchAuthorizer{allow: o.allow}, dbStore, rchSpaceFinder(), rchQuarantine{err: o.quar}, nil,
	)
	oldLocal := maven.TypeRegistry[maven.LocalRegistryType]
	maven.TypeRegistry[maven.LocalRegistryType] = &rchMavenRegistry{
		errs: o.errs, redirect: o.redirect,
	}
	t.Cleanup(func() { maven.TypeRegistry[maven.LocalRegistryType] = oldLocal })
	return mavenhandler.NewHandler(
		ctrl, nil, nil, nil, nil, rchAuthorizer{allow: o.allow}, rchSpaceFinder(),
		rchRegFinder{reg: o.reg, err: o.regErr}, nil, nil,
	)
}

func rchRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader("body"))
	session := &auth.Session{Principal: types.Principal{ID: 1}}
	ctx := apirequest.WithAuthSession(req.Context(), session)
	ctx = request.WithOriginalPath(ctx, "/maven/root/reg/io/example/app/1.0/app-1.0.jar")
	return req.WithContext(ctx)
}

const rchJarPath = "/maven/root/reg/io/example/app/1.0/app-1.0.jar"

func TestRchExtractPathVars(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    [6]string
		wantErr bool
	}{
		{name: "maven path", path: "/maven/root/reg/io/example/app/1.0/app-1.0.jar",
			want: [6]string{"root", "reg", "io.example", "app", "1.0", "app-1.0.jar"}},
		{name: "pkg prefixed path", path: "/pkg/root/reg/maven/io/example/app/1.0/app-1.0.jar",
			want: [6]string{"root", "reg", "io.example", "app", "1.0", "app-1.0.jar"}},
		{name: "artifact level metadata has no version", path: "/maven/root/reg/io/example/app/maven-metadata.xml",
			want: [6]string{"root", "reg", "io.example", "app", "", "maven-metadata.xml"}},
		{name: "snapshot metadata keeps the version", path: "/maven/root/reg/io/example/app/1.0-SNAPSHOT/maven-metadata.xml",
			want: [6]string{"root", "reg", "io.example", "app", "1.0-SNAPSHOT", "maven-metadata.xml"}},
		{name: "too short", path: "/maven/root/reg/file.jar", wantErr: true},
		{name: "missing group", path: "/maven/root/reg/app/1.0/app-1.0.jar", wantErr: true},
		{name: "illegal characters", path: "/maven/root/reg/io/ex:ample/app/1.0/app.jar", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, reg, group, art, version, file, err := mavenhandler.ExtractPathVars(tc.path)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, [6]string{root, reg, group, art, version, file})
		})
	}
}

func TestRchGetArtifactInfo(t *testing.T) {
	npmReg := rchMavenReg()
	npmReg.PackageType = artifact.PackageTypeNPM
	orphan := rchMavenReg()
	orphan.ParentID = 99
	blocked := rchMavenReg()
	blocked.BlockedPattern = pq.StringArray{"io.example:app:1.0"}
	upstream := rchMavenReg()
	upstream.Type = artifact.RegistryTypeUPSTREAM

	tests := []struct {
		name          string
		path          string
		opts          rchOpts
		remoteSupport bool
		wantErr       bool
	}{
		{name: "ok", path: rchJarPath, remoteSupport: true},
		{name: "invalid path", path: "/maven/root", wantErr: true},
		{name: "invalid registry identifier", path: "/maven/root/Bad_Reg!/io/example/app/1.0/a.jar", wantErr: true},
		{name: "unknown root", path: "/maven/nope/reg/io/example/app/1.0/a.jar", wantErr: true},
		{name: "unknown registry", path: rchJarPath, opts: rchOpts{regErr: errors.New("missing")}, wantErr: true},
		{name: "package type mismatch", path: rchJarPath, opts: rchOpts{reg: npmReg}, wantErr: true},
		{name: "unknown parent", path: rchJarPath, opts: rchOpts{reg: orphan}, wantErr: true},
		{name: "blocked by pattern", path: rchJarPath, opts: rchOpts{reg: blocked}, wantErr: true},
		{name: "upstream without remote support", path: rchJarPath, opts: rchOpts{reg: upstream}, wantErr: true},
		{name: "upstream with remote support", path: rchJarPath, opts: rchOpts{reg: upstream}, remoteSupport: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchSetup(t, tc.opts)
			info, err := h.GetArtifactInfo(rchRequest(http.MethodGet, tc.path), tc.remoteSupport)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "io.example:app", info.Image)
			require.Equal(t, "1.0", info.Version)
			require.Equal(t, "app-1.0.jar", info.FileName)
			require.Equal(t, int64(9), info.RegistryID)
			require.Equal(t, "root", info.RootIdentifier)
			require.Equal(t, "root", info.PathRoot)
		})
	}
}

func TestRchGetPackageArtifactInfo(t *testing.T) {
	h := rchSetup(t, rchOpts{})
	info, err := h.GetPackageArtifactInfo(rchRequest(http.MethodGet, rchJarPath))
	require.NoError(t, err)
	require.Equal(t, "1.0", info.GetVersion())
	require.Equal(t, artifact.PackageTypeMAVEN, info.BaseArtifactInfo().PathPackageType)

	_, err = h.GetPackageArtifactInfo(rchRequest(http.MethodGet, "/maven/root"))
	require.Error(t, err)
}

func TestRchGetArtifact(t *testing.T) {
	t.Run("serves content", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "jar-bytes", rec.Body.String())
		require.Equal(t, "get", rec.Header().Get("X-Rch"))
		require.Equal(t, "chunked", rec.Header().Get("Transfer-Encoding"))
	})
	t.Run("redirects", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, redirect: "http://cdn.example/app.jar"})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		require.Equal(t, "http://cdn.example/app.jar", rec.Header().Get("Location"))
	})
	t.Run("access denied", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("registry error with commons status", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, errs: []error{&commons.Error{Status: http.StatusConflict, Message: "x"}}})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.Equal(t, http.StatusConflict, rec.Code)
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	})
	t.Run("registry error with lib error code", func(t *testing.T) {
		libErr := liberrors.New(errors.New("missing")).WithCode(liberrors.NotFoundCode)
		h := rchSetup(t, rchOpts{allow: true, errs: []error{libErr}})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("plain registry error", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, errs: []error{errors.New("boom")}})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
	t.Run("quarantined artifact", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, quar: errors.New("quarantined")})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, rchJarPath))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetArtifact(rec, rchRequest(http.MethodGet, "/maven/root"))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
}

func TestRchHeadArtifact(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.HeadArtifact(rec, rchRequest(http.MethodHead, rchJarPath))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "head", rec.Header().Get("X-Rch"))
	})
	t.Run("denied", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.HeadArtifact(rec, rchRequest(http.MethodHead, rchJarPath))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("registry error", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, errs: []error{errors.New("boom")}})
		rec := httptest.NewRecorder()
		h.HeadArtifact(rec, rchRequest(http.MethodHead, rchJarPath))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.HeadArtifact(rec, rchRequest(http.MethodHead, "/maven/root"))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
}

func TestRchPutArtifactRejections(t *testing.T) {
	t.Run("denied", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.PutArtifact(rec, rchRequest(http.MethodPut, rchJarPath))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.PutArtifact(rec, rchRequest(http.MethodPut, "/maven/root"))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
}

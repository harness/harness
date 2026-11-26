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

package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	apirequest "github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/handler/packages"
	"github.com/harness/gitness/registry/app/api/middleware"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/docker"
	commonstype "github.com/harness/gitness/registry/app/pkg/types/commons"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/request"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const rchDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type rchNext struct {
	called bool
	status int
}

func (n *rchNext) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	n.called = true
	if n.status != 0 {
		w.WriteHeader(n.status)
	}
}

type rchPkgHandler struct {
	packages.Handler
	accessErr     error
	quarantineErr error
	statsErr      error
	statsCalls    int
}

func (h *rchPkgHandler) GetRegistryCheckAccess(context.Context, *http.Request, ...enum.Permission) error {
	return h.accessErr
}

func (h *rchPkgHandler) CheckQuarantineStatus(context.Context) error { return h.quarantineErr }

func (h *rchPkgHandler) TrackDownloadStats(context.Context, *http.Request) error {
	h.statsCalls++
	return h.statsErr
}

type rchProvider struct {
	info pkg.PackageArtifactInfo
	err  error
}

func (p rchProvider) GetPackageArtifactInfo(*http.Request) (pkg.PackageArtifactInfo, error) {
	return p.info, p.err
}

func rchInfo(fileName string) pkg.PackageArtifactInfo {
	return commonstype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{ParentID: 5}, RegIdentifier: "reg", Image: "pkg"},
		Version:      "1.0.0",
		FilePath:     fileName,
	}
}

func TestRchStatusWriter(t *testing.T) {
	t.Run("records the written status", func(t *testing.T) {
		sw := &middleware.StatusWriter{ResponseWriter: httptest.NewRecorder()}
		sw.WriteHeader(http.StatusCreated)
		require.Equal(t, http.StatusCreated, sw.StatusCode)
	})
	t.Run("a body write implies 200", func(t *testing.T) {
		rec := httptest.NewRecorder()
		sw := &middleware.StatusWriter{ResponseWriter: rec}
		n, err := sw.Write([]byte("hello"))
		require.NoError(t, err)
		require.Equal(t, 5, n)
		require.Equal(t, http.StatusOK, sw.StatusCode)
		require.Equal(t, "hello", rec.Body.String())
	})
	t.Run("an explicit status is kept on write", func(t *testing.T) {
		sw := &middleware.StatusWriter{ResponseWriter: httptest.NewRecorder()}
		sw.WriteHeader(http.StatusAccepted)
		_, err := sw.Write([]byte("x"))
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, sw.StatusCode)
	})
}

func TestRchStoreOriginalPath(t *testing.T) {
	var got string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = request.OriginalPathFrom(r.Context())
	})
	middleware.StoreOriginalPath(next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/a/b", nil))
	require.Equal(t, "/a/b", got)
}

func TestRchStoreArtifactInfo(t *testing.T) {
	t.Run("stores the info in the context", func(t *testing.T) {
		var got pkg.PackageArtifactInfo
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = request.ArtifactInfoFrom(r.Context())
		})
		mw := middleware.StoreArtifactInfo(rchProvider{info: rchInfo("f.bin")})
		mw(next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
		require.NotNil(t, got)
		require.Equal(t, "f.bin", got.GetFileName())
	})
	t.Run("translates provider errors", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		mw := middleware.StoreArtifactInfo(rchProvider{err: usererror.NotFound("missing")})
		mw(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.False(t, next.called)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestRchRequestPackageAccess(t *testing.T) {
	t.Run("allows when access is granted", func(t *testing.T) {
		next := &rchNext{}
		mw := middleware.RequestPackageAccess(&rchPkgHandler{}, enum.PermissionArtifactsDownload)
		mw(next).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
		require.True(t, next.called)
	})
	t.Run("rejects when access is denied", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		mw := middleware.RequestPackageAccess(&rchPkgHandler{accessErr: apiauth.ErrForbidden})
		mw(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.False(t, next.called)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRchRequestNugetPackageAccess(t *testing.T) {
	t.Run("allows when access is granted", func(t *testing.T) {
		next := &rchNext{}
		middleware.RequestNugetPackageAccess(&rchPkgHandler{})(next).
			ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
		require.True(t, next.called)
	})
	t.Run("challenges anonymous reads with basic auth", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		middleware.RequestNugetPackageAccess(&rchPkgHandler{accessErr: apiauth.ErrUnauthorized})(next).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), "Basic")
	})
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run("challenges anonymous "+method+" with an api key", func(t *testing.T) {
			rec := httptest.NewRecorder()
			middleware.RequestNugetPackageAccess(&rchPkgHandler{accessErr: apiauth.ErrUnauthorized})(&rchNext{}).
				ServeHTTP(rec, httptest.NewRequest(method, "/x", nil))
			require.Contains(t, rec.Header().Get("WWW-Authenticate"), "ApiKey")
		})
	}
	t.Run("other errors are translated", func(t *testing.T) {
		rec := httptest.NewRecorder()
		middleware.RequestNugetPackageAccess(&rchPkgHandler{accessErr: apiauth.ErrForbidden})(&rchNext{}).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRchCheckQuarantineStatus(t *testing.T) {
	t.Run("continues for clean artifacts", func(t *testing.T) {
		next := &rchNext{}
		middleware.CheckQuarantineStatus(&rchPkgHandler{})(next).
			ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
		require.True(t, next.called)
	})
	t.Run("blocks quarantined artifacts", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		middleware.CheckQuarantineStatus(&rchPkgHandler{quarantineErr: usererror.ErrQuarantinedArtifact})(next).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.False(t, next.called)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRchTrackDownloadStats(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantCalls int
	}{
		{"ok", http.StatusOK, 1},
		{"redirect", http.StatusTemporaryRedirect, 1},
		{"found", http.StatusFound, 1},
		{"not found", http.StatusNotFound, 0},
		{"server error", http.StatusInternalServerError, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &rchPkgHandler{}
			middleware.TrackDownloadStats(h)(&rchNext{status: tc.status}).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
			require.Equal(t, tc.wantCalls, h.statsCalls)
		})
	}
	t.Run("stat failures do not affect the response", func(t *testing.T) {
		h := &rchPkgHandler{statsErr: errors.New("db")}
		rec := httptest.NewRecorder()
		middleware.TrackDownloadStats(h)(&rchNext{status: http.StatusOK}).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, 1, h.statsCalls)
	})
}

func TestRchTrackDownloadStatsByLanguage(t *testing.T) {
	type mwFactory func(packages.Handler) func(http.Handler) http.Handler
	factories := map[string]struct {
		mw       mwFactory
		mainFile string
		sideFile string
	}{
		"go":     {middleware.TrackDownloadStatsForGoPackage, "v1.0.0.zip", "v1.0.0.mod"},
		"python": {middleware.TrackDownloadStatsForPythonPackage, "pkg-1.0.0.whl", "pkg-1.0.0.whl.asc"},
	}
	for name, f := range factories {
		serve := func(h *rchPkgHandler, status int, file string, withInfo bool) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if withInfo {
				req = req.WithContext(request.WithArtifactInfo(req.Context(), rchInfo(file)))
			}
			f.mw(h)(&rchNext{status: status}).ServeHTTP(httptest.NewRecorder(), req)
		}
		t.Run(name+" main artifact is tracked", func(t *testing.T) {
			h := &rchPkgHandler{}
			serve(h, http.StatusOK, f.mainFile, true)
			require.Equal(t, 1, h.statsCalls)
		})
		t.Run(name+" redirect is tracked", func(t *testing.T) {
			h := &rchPkgHandler{}
			serve(h, http.StatusTemporaryRedirect, f.mainFile, true)
			require.Equal(t, 1, h.statsCalls)
		})
		t.Run(name+" side files are ignored", func(t *testing.T) {
			h := &rchPkgHandler{}
			serve(h, http.StatusOK, f.sideFile, true)
			require.Zero(t, h.statsCalls)
		})
		t.Run(name+" failed requests are ignored", func(t *testing.T) {
			h := &rchPkgHandler{}
			serve(h, http.StatusNotFound, f.mainFile, true)
			require.Zero(t, h.statsCalls)
		})
		t.Run(name+" missing info is ignored", func(t *testing.T) {
			h := &rchPkgHandler{}
			serve(h, http.StatusOK, f.mainFile, false)
			require.Zero(t, h.statsCalls)
		})
		t.Run(name+" stat failures are tolerated", func(t *testing.T) {
			h := &rchPkgHandler{statsErr: errors.New("db")}
			serve(h, http.StatusOK, f.mainFile, true)
			require.Equal(t, 1, h.statsCalls)
		})
	}
}

func TestRchHeaderRewriters(t *testing.T) {
	t.Run("sig query becomes a bearer token", func(t *testing.T) {
		var got string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.Header.Get("Authorization") })
		for sig, want := range map[string]string{
			"abc":          "Bearer abc",
			"Bearer def":   "Bearer def",
			"Basic ghi123": "Bearer ghi123",
		} {
			middleware.CheckSig()(next).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x?sig="+url.QueryEscape(sig), nil))
			require.Equal(t, want, got)
		}
		got = "untouched"
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "keep")
		middleware.CheckSig()(next).ServeHTTP(httptest.NewRecorder(), req)
		require.Equal(t, "keep", got)
	})
	t.Run("api key headers are copied to authorization", func(t *testing.T) {
		var got string
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.Header.Get("Authorization") })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("x-api-key", "pat-token")
		middleware.CheckAuthHeader()(next).ServeHTTP(httptest.NewRecorder(), req)
		require.Equal(t, "pat-token", got)

		req = httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("x-nuget-apikey", "nuget-token")
		middleware.CheckNugetAPIKey()(next).ServeHTTP(httptest.NewRecorder(), req)
		require.Equal(t, "nuget-token", got)

		got = "untouched"
		middleware.CheckAuthHeader()(next).
			ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
		require.Empty(t, got)
	})
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

type rchSpaceFoldCache struct{ space *types.SpaceCore }

func (c rchSpaceFoldCache) Get(_ context.Context, ref string) (int64, error) {
	if c.space.Path != ref {
		return 0, basestore.ErrResourceNotFound
	}
	return c.space.ID, nil
}
func (rchSpaceFoldCache) Stats() (int64, int64)             { return 0, 0 }
func (rchSpaceFoldCache) Evict(_ context.Context, _ string) {}

func rchSpaceFinder() refcache.SpaceFinder {
	space := &types.SpaceCore{ID: 5, Path: "root", Identifier: "root"}
	return refcache.NewSpaceFinder(
		rchSpaceIDCache{space: space}, rchSpacePathCache{space: space}, rchSpaceFoldCache{space: space},
		storecache.Evictor[*types.SpaceCore]{},
	)
}

type rchPublicAccess struct {
	publicaccess.Service
	public bool
	err    error
}

func (p rchPublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return p.public, p.err
}

func TestRchCheckAuthWithChallenge(t *testing.T) {
	anonymous := &auth.Session{Principal: auth.AnonymousPrincipal}
	user := &auth.Session{Principal: types.Principal{ID: 1, UID: "user"}}
	newReq := func(session *auth.Session) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		return req.WithContext(apirequest.WithAuthSession(req.Context(), session))
	}
	run := func(
		provider rchProvider, access publicaccess.Service, session *auth.Session,
	) (*rchNext, *httptest.ResponseRecorder) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		middleware.CheckAuthWithChallenge(provider, rchSpaceFinder(), access)(next).ServeHTTP(rec, newReq(session))
		return next, rec
	}

	t.Run("authenticated users pass", func(t *testing.T) {
		next, _ := run(rchProvider{err: errors.New("unused")}, rchPublicAccess{}, user)
		require.True(t, next.called)
	})
	t.Run("anonymous users reach public registries", func(t *testing.T) {
		next, _ := run(rchProvider{info: rchInfo("")}, rchPublicAccess{public: true}, anonymous)
		require.True(t, next.called)
	})
	t.Run("anonymous users are challenged for private registries", func(t *testing.T) {
		next, rec := run(rchProvider{info: rchInfo("")}, rchPublicAccess{}, anonymous)
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), "Basic")
	})
	t.Run("public access lookup failure is challenged", func(t *testing.T) {
		next, rec := run(rchProvider{info: rchInfo("")}, rchPublicAccess{public: true, err: errors.New("db")}, anonymous)
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	t.Run("artifact info failure is challenged", func(t *testing.T) {
		next, rec := run(rchProvider{err: errors.New("bad path")}, rchPublicAccess{public: true}, anonymous)
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	t.Run("unknown space is challenged", func(t *testing.T) {
		info := commonstype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{ParentID: 99}}}
		next, rec := run(rchProvider{info: info}, rchPublicAccess{public: true}, anonymous)
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

type rchURLProvider struct {
	urlprovider.Provider
}

func (rchURLProvider) RegistryURL(context.Context, ...string) string {
	return "https://registry.example.com"
}

type rchRegFinder struct {
	refcache2.RegistryFinder
	reg *registrytypes.Registry
	err error
}

func (f rchRegFinder) FindByRootParentID(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return f.reg, f.err
}

func rchOCIHandler(public bool, publicErr error, reg *registrytypes.Registry, regErr error) *oci.Handler {
	return oci.NewHandler(
		nil, rchSpaceFinder(), nil, nil, nil, nil, rchURLProvider{}, nil, true,
		rchRegFinder{reg: reg, err: regErr}, rchPublicAccess{public: public, err: publicErr}, "", nil,
	)
}

func rchDockerRegistry() *registrytypes.Registry {
	return &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5,
		PackageType: artifact.PackageTypeDOCKER, Type: artifact.RegistryTypeVIRTUAL,
	}
}

func TestRchOciCheckAuth(t *testing.T) {
	anonymous := &auth.Session{Principal: auth.AnonymousPrincipal}
	user := &auth.Session{Principal: types.Principal{ID: 1, UID: "user"}}
	newReq := func(session *auth.Session, method, target string, authHeader string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		return req.WithContext(apirequest.WithAuthSession(req.Context(), session))
	}
	const manifestPath = "/v2/root/reg/app/manifests/latest"

	t.Run("authenticated users pass", func(t *testing.T) {
		next := &rchNext{}
		h := rchOCIHandler(false, nil, rchDockerRegistry(), nil)
		middleware.OciCheckAuth(h)(next).ServeHTTP(httptest.NewRecorder(), newReq(user, http.MethodGet, manifestPath, ""))
		require.True(t, next.called)
	})
	t.Run("anonymous pulls from public registries pass", func(t *testing.T) {
		next := &rchNext{}
		h := rchOCIHandler(true, nil, rchDockerRegistry(), nil)
		middleware.OciCheckAuth(h)(next).
			ServeHTTP(httptest.NewRecorder(), newReq(anonymous, http.MethodGet, manifestPath, "Basic abc"))
		require.True(t, next.called)
	})
	t.Run("anonymous pulls from private registries are challenged", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		h := rchOCIHandler(false, nil, rchDockerRegistry(), nil)
		middleware.OciCheckAuth(h)(next).
			ServeHTTP(rec, newReq(anonymous, http.MethodGet, manifestPath, "Basic abc"))
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), `service="gitness-registry"`)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), `scope="repository:root/reg:pull"`)
	})
	t.Run("anonymous writes are challenged even for public registries", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		h := rchOCIHandler(true, nil, rchDockerRegistry(), nil)
		middleware.OciCheckAuth(h)(next).
			ServeHTTP(rec, newReq(anonymous, http.MethodPut, manifestPath, "Basic abc"))
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	t.Run("requests without credentials are challenged", func(t *testing.T) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		h := rchOCIHandler(true, nil, rchDockerRegistry(), nil)
		middleware.OciCheckAuth(h)(next).ServeHTTP(rec, newReq(anonymous, http.MethodGet, manifestPath, ""))
		require.False(t, next.called)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})
	t.Run("the api root is challenged without a scope", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := rchOCIHandler(true, nil, rchDockerRegistry(), nil)
		req := newReq(anonymous, http.MethodGet, "/v2/", "Basic abc")
		req.RequestURI = "/v2/"
		middleware.OciCheckAuth(h)(&rchNext{}).ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.NotContains(t, rec.Header().Get("WWW-Authenticate"), "scope=")
	})
	t.Run("lookup failures are challenged", func(t *testing.T) {
		cases := map[string]*oci.Handler{
			"registry lookup": rchOCIHandler(true, nil, nil, errors.New("missing")),
			"public access":   rchOCIHandler(true, errors.New("db"), rchDockerRegistry(), nil),
		}
		for name, h := range cases {
			next := &rchNext{}
			rec := httptest.NewRecorder()
			middleware.OciCheckAuth(h)(next).
				ServeHTTP(rec, newReq(anonymous, http.MethodGet, manifestPath, "Basic abc"))
			require.False(t, next.called, name)
			require.Equal(t, http.StatusUnauthorized, rec.Code, name)
		}
	})
	t.Run("cross repository mounts request pull access on the source", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := rchOCIHandler(false, nil, rchDockerRegistry(), nil)
		req := newReq(anonymous, http.MethodPost, "/v2/root/reg/app/blobs/uploads/?from=src/repo", "Basic abc")
		middleware.OciCheckAuth(h)(&rchNext{}).ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), "src/repo")
	})
}

func TestRchBlockNonOciSourceToken(t *testing.T) {
	newReq := func(session *auth.Session) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/v2/root/reg/app/manifests/latest", nil)
		if session == nil {
			return req
		}
		return req.WithContext(apirequest.WithAuthSession(req.Context(), session))
	}
	user := types.Principal{ID: 1, UID: "user"}
	ociSession := &auth.Session{
		Principal: user,
		Metadata: &auth.AccessPermissionMetadata{
			AccessPermissions: &jwt.SubClaimsAccessPermissions{Source: jwt.OciSource},
		},
	}
	otherSession := &auth.Session{
		Principal: user,
		Metadata: &auth.AccessPermissionMetadata{
			AccessPermissions: &jwt.SubClaimsAccessPermissions{Source: "other"},
		},
	}
	tokenSession := &auth.Session{Principal: user, Metadata: &auth.TokenMetadata{TokenType: enum.TokenTypePAT}}

	tests := []struct {
		name        string
		session     *auth.Session
		wantPassing bool
	}{
		{"no session", nil, true},
		{"anonymous session", &auth.Session{Principal: auth.AnonymousPrincipal}, true},
		{"oci token", ociSession, true},
		{"other token source", otherSession, false},
		{"plain token without access permissions", tokenSession, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next := &rchNext{}
			rec := httptest.NewRecorder()
			middleware.BlockNonOciSourceToken(rchURLProvider{})(next).ServeHTTP(rec, newReq(tc.session))
			require.Equal(t, tc.wantPassing, next.called)
			if !tc.wantPassing {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
				require.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer realm=")
			}
		})
	}
}

type rchManifestDao struct {
	store.ManifestRepository
	digest registrytypes.Digest
	err    error
}

func (d rchManifestDao) FindManifestDigestByTagName(
	context.Context, int64, string, string,
) (registrytypes.Digest, error) {
	return d.digest, d.err
}

type rchQuarantineDao struct {
	store.QuarantineArtifactRepository
	found []*registrytypes.QuarantineArtifact
	err   error
}

func (d rchQuarantineDao) GetByFilePath(
	context.Context, string, int64, string, string, *artifact.ArtifactType,
) ([]*registrytypes.QuarantineArtifact, error) {
	return d.found, d.err
}

func rchDockerHandler(manifests store.ManifestRepository, quarantine store.QuarantineArtifactRepository) *oci.Handler {
	dbStore := docker.NewDBStore(nil, nil, nil, nil, nil, manifests, quarantine)
	ctrl := docker.NewController(nil, nil, nil, nil, nil, dbStore, rchSpaceFinder())
	return oci.NewHandler(
		ctrl, rchSpaceFinder(), nil, nil, nil, nil, rchURLProvider{}, nil, true,
		rchRegFinder{reg: rchDockerRegistry()}, rchPublicAccess{}, "", nil,
	)
}

func TestRchCheckQuarantineStatusOCI(t *testing.T) {
	stored, err := registrytypes.NewDigest(digest.Digest(rchDigest))
	require.NoError(t, err)
	quarantined := []*registrytypes.QuarantineArtifact{{}}
	manifestPath := "/v2/root/reg/app/manifests/latest"
	digestPath := "/v2/root/reg/app/manifests/" + rchDigest

	run := func(h *oci.Handler, method, target string) (*rchNext, *httptest.ResponseRecorder) {
		next := &rchNext{}
		rec := httptest.NewRecorder()
		middleware.CheckQuarantineStatusOCI(h)(next).ServeHTTP(rec, httptest.NewRequest(method, target, nil))
		return next, rec
	}

	t.Run("non manifest requests are not checked", func(t *testing.T) {
		next, _ := run(rchDockerHandler(nil, nil), http.MethodGet, "/v2/root/reg/app/tags/list")
		require.True(t, next.called)
	})
	t.Run("manifest writes are not checked", func(t *testing.T) {
		next, _ := run(rchDockerHandler(nil, nil), http.MethodPut, manifestPath)
		require.True(t, next.called)
	})
	t.Run("clean tag passes", func(t *testing.T) {
		h := rchDockerHandler(rchManifestDao{digest: stored}, rchQuarantineDao{})
		next, rec := run(h, http.MethodGet, manifestPath)
		require.True(t, next.called)
		require.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("quarantined tag is forbidden", func(t *testing.T) {
		h := rchDockerHandler(rchManifestDao{digest: stored}, rchQuarantineDao{found: quarantined})
		next, rec := run(h, http.MethodGet, manifestPath)
		require.False(t, next.called)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("quarantined digest is forbidden for head requests", func(t *testing.T) {
		h := rchDockerHandler(nil, rchQuarantineDao{found: quarantined})
		next, rec := run(h, http.MethodHead, digestPath)
		require.False(t, next.called)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("unknown tag passes", func(t *testing.T) {
		h := rchDockerHandler(rchManifestDao{err: basestore.ErrResourceNotFound}, rchQuarantineDao{})
		next, _ := run(h, http.MethodGet, manifestPath)
		require.True(t, next.called)
	})
	t.Run("lookup failures do not block the request", func(t *testing.T) {
		failing := map[string]*oci.Handler{
			"manifest lookup":   rchDockerHandler(rchManifestDao{err: errors.New("db")}, rchQuarantineDao{}),
			"quarantine lookup": rchDockerHandler(rchManifestDao{digest: stored}, rchQuarantineDao{err: errors.New("db")}),
			"bad stored digest": rchDockerHandler(rchManifestDao{digest: "garbage"}, rchQuarantineDao{}),
		}
		for name, h := range failing {
			next, _ := run(h, http.MethodGet, manifestPath)
			require.True(t, next.called, name)
		}
	})
	t.Run("registry info failures do not block the request", func(t *testing.T) {
		next, _ := run(rchDockerHandler(nil, nil), http.MethodGet, "/v2/unknown/reg/app/manifests/latest")
		require.True(t, next.called)
	})
}

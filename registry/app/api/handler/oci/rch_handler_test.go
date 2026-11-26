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

package oci

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
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/request"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type rchAuthorizer struct {
	authz.Authorizer
	allow bool
}

func (a rchAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.allow, nil
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

type rchSpaceFoldCache struct{ space *types.SpaceCore }

func (c rchSpaceFoldCache) Get(_ context.Context, ref string) (int64, error) {
	if !strings.EqualFold(c.space.Path, ref) {
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

func (q rchQuarantine) CheckOCIManifestQuarantineStatus(
	context.Context, int64, string, string, string,
) error {
	return q.err
}

type rchManifest struct{}

func (rchManifest) References() []manifest.Descriptor { return nil }

func (rchManifest) Payload() (string, []byte, error) {
	return "application/vnd.oci.image.manifest.v1+json", []byte(`{"schemaVersion":2}`), nil
}

type rchDockerRegistry struct {
	docker.Registry
	errs     []error
	redirect string
	status   int
	blob     string
	size     int64
}

func (r *rchDockerRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *rchDockerRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeDOCKER}
}

func (r *rchDockerRegistry) headers() *commons.ResponseHeaders {
	code := r.status
	if code == 0 {
		code = http.StatusOK
	}
	return &commons.ResponseHeaders{Code: code, Headers: map[string]string{"X-Rch": "yes"}}
}

func (r *rchDockerRegistry) GetBlob(
	context.Context, pkg.RegistryInfo,
) (*commons.ResponseHeaders, *storage.FileReader, int64, io.ReadCloser, string, []error) {
	if r.errs != nil {
		return nil, nil, 0, nil, "", r.errs
	}
	return r.headers(), nil, r.size, io.NopCloser(strings.NewReader(r.blob)), r.redirect, nil
}

func (r *rchDockerRegistry) ManifestExist(
	context.Context, pkg.RegistryInfo, []string, []string,
) (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error) {
	if r.errs != nil {
		return nil, manifest.Descriptor{}, nil, r.errs
	}
	return r.headers(), manifest.Descriptor{}, rchManifest{}, nil
}

func (r *rchDockerRegistry) PullManifest(
	ctx context.Context, _ pkg.RegistryInfo, _ []string, _ []string,
) (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error) {
	return r.ManifestExist(ctx, pkg.RegistryInfo{}, nil, nil)
}

type rchOpts struct {
	allow      bool
	reg        *registrytypes.Registry
	regErr     error
	registry   *rchDockerRegistry
	quarantine error
}

func rchDockerReg() *registrytypes.Registry {
	return &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5,
		PackageType: artifact.PackageTypeDOCKER, Type: artifact.RegistryTypeVIRTUAL,
	}
}

func rchSetup(t *testing.T, o rchOpts) *Handler {
	t.Helper()
	if o.reg == nil {
		o.reg = rchDockerReg()
	}
	if o.registry == nil {
		o.registry = &rchDockerRegistry{blob: "blob-bytes", size: int64(len("blob-bytes"))}
	}
	core := pkg.NewCoreController(nil, rchQuarantine{err: o.quarantine})
	ctrl := docker.NewController(nil, nil, core, nil, rchAuthorizer{allow: o.allow}, nil, rchSpaceFinder())
	oldLocal, oldRemote := pkg.TypeRegistry[pkg.LocalRegistry], pkg.TypeRegistry[pkg.RemoteRegistry]
	pkg.TypeRegistry[pkg.LocalRegistry] = o.registry
	t.Cleanup(func() {
		pkg.TypeRegistry[pkg.LocalRegistry] = oldLocal
		pkg.TypeRegistry[pkg.RemoteRegistry] = oldRemote
	})
	return NewHandler(
		ctrl, rchSpaceFinder(), nil, nil, nil, nil, nil, rchAuthorizer{allow: o.allow}, true,
		rchRegFinder{reg: o.reg, err: o.regErr}, nil, "anon-secret", nil,
	)
}

func rchRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader("body"))
	session := &auth.Session{Principal: types.Principal{ID: 1, UID: "user"}}
	ctx := apirequest.WithAuthSession(req.Context(), session)
	ctx = request.WithOriginalPath(ctx, "/v2/root/reg/app/manifests/latest")
	return req.WithContext(ctx)
}

const (
	rchManifestPath = "/v2/root/reg/app/manifests/latest"
	rchBlobPath     = "/v2/root/reg/app/blobs/sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func TestRchGetRouteType(t *testing.T) {
	tests := map[string]routeType{
		"/v2/root/reg/app/manifests/latest":         Manifests,
		"/v2/root/reg/app/blobs/sha256:abc":         Blobs,
		"/v2/root/reg/app/blobs/uploads/":           BlobsUploadsSession,
		"/v2/root/reg/app/blobs/uploads/uuid":       BlobsUploadsSession,
		"/v2/root/reg/app/tags/list":                Tags,
		"/v2/root/reg/app/referrers/sha256:abc":     Referrers,
		"/v2/root/reg/app/other/thing":              Invalid,
		"/v2/root/reg":                              Invalid,
		"/v2/root/reg/nested/app/manifests/latest/": Manifests,
	}
	for path, want := range tests {
		t.Run(path, func(t *testing.T) {
			require.Equal(t, want, getRouteType(path))
		})
	}
}

func TestRchExtractPathVars(t *testing.T) {
	digestRef := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tests := []struct {
		name                         string
		path                         string
		params                       map[string]string
		wantImage, wantRef, wantDgst string
		wantTag                      string
		wantRoot, wantReg            string
	}{
		{name: "manifest by tag", path: "/v2/root/reg/app/manifests/latest",
			wantRoot: "root", wantReg: "reg", wantImage: "app", wantRef: "latest", wantTag: "latest"},
		{name: "manifest by digest", path: "/v2/root/reg/team/app/manifests/" + digestRef,
			wantRoot: "root", wantReg: "reg", wantImage: "team/app", wantRef: digestRef, wantDgst: digestRef},
		{name: "blob", path: "/v2/root/reg/app/blobs/" + digestRef,
			wantRoot: "root", wantReg: "reg", wantImage: "app", wantDgst: digestRef},
		{name: "upload session", path: "/v2/root/reg/app/blobs/uploads/session-id",
			params:   map[string]string{"digest": "sha256:abc"},
			wantRoot: "root", wantReg: "reg", wantImage: "app", wantRef: "session-id", wantDgst: "sha256:abc"},
		{name: "upload start", path: "/v2/root/reg/app/blobs/uploads/",
			wantRoot: "root", wantReg: "reg", wantImage: "app"},
		{name: "tags list", path: "/v2/root/reg/app/tags/list",
			wantRoot: "root", wantReg: "reg", wantImage: "app"},
		{name: "referrers", path: "/v2/root/reg/app/referrers/" + digestRef,
			wantRoot: "root", wantReg: "reg", wantImage: "app", wantDgst: digestRef},
		{name: "invalid route", path: "/v2/root/reg/app/other/x",
			wantRoot: "root", wantReg: "reg", wantImage: "app"},
		{name: "too short", path: "/v2/root"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, reg, image, ref, dgst, tag := ExtractPathVars(context.Background(), tc.path, tc.params)
			require.Equal(t, []string{
				tc.wantRoot, tc.wantReg, tc.wantImage, tc.wantRef, tc.wantDgst, tc.wantTag,
			}, []string{root, reg, image, ref, dgst, tag})
		})
	}
}

func TestRchGetRegistryInfo(t *testing.T) {
	npmReg := rchDockerReg()
	npmReg.PackageType = artifact.PackageTypeNPM
	helmReg := rchDockerReg()
	helmReg.PackageType = artifact.PackageTypeHELM
	orphan := rchDockerReg()
	orphan.ParentID = 99
	blocked := rchDockerReg()
	blocked.BlockedPattern = pq.StringArray{"app:latest"}
	upstream := rchDockerReg()
	upstream.Type = artifact.RegistryTypeUPSTREAM

	tests := []struct {
		name          string
		path          string
		opts          rchOpts
		remoteSupport bool
		wantErr       bool
	}{
		{name: "docker registry", path: rchManifestPath, remoteSupport: false},
		{name: "helm registry", path: rchManifestPath, opts: rchOpts{reg: helmReg}},
		{name: "case insensitive root", path: "/v2/ROOT/reg/app/manifests/latest"},
		{name: "unknown root", path: "/v2/nope/reg/app/manifests/latest", wantErr: true},
		{name: "unknown registry", path: rchManifestPath, opts: rchOpts{regErr: errors.New("missing")},
			wantErr: true},
		{name: "non OCI registry", path: rchManifestPath, opts: rchOpts{reg: npmReg}, wantErr: true},
		{name: "unknown parent", path: rchManifestPath, opts: rchOpts{reg: orphan}, wantErr: true},
		{name: "blocked by pattern", path: rchManifestPath, opts: rchOpts{reg: blocked}, wantErr: true},
		{name: "upstream not supported", path: rchManifestPath, opts: rchOpts{reg: upstream}, wantErr: true},
		{name: "upstream supported", path: rchManifestPath, opts: rchOpts{reg: upstream}, remoteSupport: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := rchSetup(t, tc.opts)
			info, err := h.GetRegistryInfo(rchRequest(http.MethodGet, tc.path), tc.remoteSupport)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "app", info.Image)
			require.Equal(t, "latest", info.Tag)
			require.Equal(t, "reg", info.RegIdentifier)
			require.Equal(t, "root", info.PathRoot)
			require.NotNil(t, info.URLBuilder)
		})
	}
}

func TestRchHandleErrors(t *testing.T) {
	t.Run("serves the error as json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handleErrors(context.Background(), errcode.Errors{errcode.ErrCodeBlobUnknown}, rec)
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Contains(t, rec.Body.String(), "BLOB_UNKNOWN")
	})
	t.Run("nothing to serve for an empty list", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handleErrors(context.Background(), nil, rec)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Body.String())
	})
}

func TestRchAPIBase(t *testing.T) {
	rec := httptest.NewRecorder()
	(&Handler{}).APIBase(rec, httptest.NewRequest(http.MethodGet, "/v2/", nil))
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRchGetBlob(t *testing.T) {
	t.Run("serves the blob", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodGet, rchBlobPath))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "blob-bytes", rec.Body.String())
		require.Equal(t, "yes", rec.Header().Get("X-Rch"))
	})
	t.Run("head does not send a body", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodHead, rchBlobPath))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Body.String())
	})
	t.Run("redirects", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, registry: &rchDockerRegistry{redirect: "http://cdn.example/blob"}})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodGet, rchBlobPath))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code)
		require.Equal(t, "http://cdn.example/blob", rec.Header().Get("Location"))
	})
	t.Run("moved permanently is passed through", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, registry: &rchDockerRegistry{status: http.StatusMovedPermanently}})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodGet, rchBlobPath))
		require.Equal(t, http.StatusMovedPermanently, rec.Code)
	})
	t.Run("unknown blob", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, registry: &rchDockerRegistry{errs: []error{errors.New("gone")}}})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodGet, rchBlobPath))
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Contains(t, rec.Body.String(), "BLOB_UNKNOWN")
	})
	t.Run("access denied", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodGet, rchBlobPath))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetBlob(rec, rchRequest(http.MethodGet, "/v2/unknown/reg/app/blobs/sha256:abc"))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestRchGetManifest(t *testing.T) {
	t.Run("serves the manifest", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetManifest(rec, rchRequest(http.MethodGet, rchManifestPath))
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"schemaVersion":2}`, rec.Body.String())
		require.Equal(t, "yes", rec.Header().Get("X-Rch"))
	})
	t.Run("moved permanently has no body", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, registry: &rchDockerRegistry{status: http.StatusMovedPermanently}})
		rec := httptest.NewRecorder()
		h.GetManifest(rec, rchRequest(http.MethodGet, rchManifestPath))
		require.Equal(t, http.StatusMovedPermanently, rec.Code)
		require.Empty(t, rec.Body.String())
	})
	t.Run("unknown manifest", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, registry: &rchDockerRegistry{errs: []error{errors.New("gone")}}})
		rec := httptest.NewRecorder()
		h.GetManifest(rec, rchRequest(http.MethodGet, rchManifestPath))
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Contains(t, rec.Body.String(), "MANIFEST_UNKNOWN")
	})
	t.Run("quarantined manifest", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, quarantine: errors.New("quarantined")})
		rec := httptest.NewRecorder()
		h.GetManifest(rec, rchRequest(http.MethodGet, rchManifestPath))
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	})
	t.Run("access denied", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.GetManifest(rec, rchRequest(http.MethodGet, rchManifestPath))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetManifest(rec, rchRequest(http.MethodGet, "/v2/unknown/reg/app/manifests/latest"))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestRchHeadManifest(t *testing.T) {
	t.Run("returns headers only", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.HeadManifest(rec, rchRequest(http.MethodHead, rchManifestPath))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "yes", rec.Header().Get("X-Rch"))
		require.Empty(t, rec.Body.String())
	})
	t.Run("unknown manifest", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, registry: &rchDockerRegistry{errs: []error{errors.New("gone")}}})
		rec := httptest.NewRecorder()
		h.HeadManifest(rec, rchRequest(http.MethodHead, rchManifestPath))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("access denied", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.HeadManifest(rec, rchRequest(http.MethodHead, rchManifestPath))
		require.Equal(t, http.StatusForbidden, rec.Code)
	})
	t.Run("invalid path", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.HeadManifest(rec, rchRequest(http.MethodHead, "/v2/unknown/reg/app/manifests/latest"))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestRchAccessDeniedForWriteAndDeleteEndpoints(t *testing.T) {
	uploadPath := "/v2/root/reg/app/blobs/uploads/session-id"
	tests := []struct {
		name   string
		method string
		path   string
		call   func(h *Handler) http.HandlerFunc
	}{
		{"delete blob", http.MethodDelete, rchBlobPath, func(h *Handler) http.HandlerFunc { return h.DeleteBlob }},
		{"cancel upload", http.MethodDelete, uploadPath, func(h *Handler) http.HandlerFunc { return h.CancelBlobUpload }},
		{"delete manifest", http.MethodDelete, rchManifestPath,
			func(h *Handler) http.HandlerFunc { return h.DeleteManifest }},
		{"upload status", http.MethodGet, uploadPath,
			func(h *Handler) http.HandlerFunc { return h.GetUploadBlobStatus }},
		{"referrers", http.MethodGet, "/v2/root/reg/app/referrers/sha256:abc",
			func(h *Handler) http.HandlerFunc { return h.GetReferrers }},
		{"head blob", http.MethodHead, rchBlobPath, func(h *Handler) http.HandlerFunc { return h.HeadBlob }},
		{"initiate upload", http.MethodPost, "/v2/root/reg/app/blobs/uploads/",
			func(h *Handler) http.HandlerFunc { return h.InitiateUploadBlob }},
		{"complete upload", http.MethodPut, uploadPath,
			func(h *Handler) http.HandlerFunc { return h.CompleteBlobUpload }},
		{"put manifest", http.MethodPut, rchManifestPath, func(h *Handler) http.HandlerFunc { return h.PutManifest }},
	}
	for _, tc := range tests {
		t.Run(tc.name+" denied", func(t *testing.T) {
			h := rchSetup(t, rchOpts{allow: false})
			rec := httptest.NewRecorder()
			tc.call(h)(rec, rchRequest(tc.method, tc.path))
			require.Equal(t, http.StatusForbidden, rec.Code)
		})
		t.Run(tc.name+" unknown registry", func(t *testing.T) {
			h := rchSetup(t, rchOpts{allow: true, regErr: errors.New("missing")})
			rec := httptest.NewRecorder()
			tc.call(h)(rec, rchRequest(tc.method, tc.path))
			require.Equal(t, http.StatusNotFound, rec.Code)
		})
	}

	t.Run("tags are not listed without access", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.GetTags(rec, rchRequest(http.MethodGet, "/v2/root/reg/app/tags/list"))
		require.NotContains(t, rec.Body.String(), `"tags"`)
	})
	t.Run("tags with unknown registry", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, regErr: errors.New("missing")})
		rec := httptest.NewRecorder()
		h.GetTags(rec, rchRequest(http.MethodGet, "/v2/root/reg/app/tags/list"))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("patch upload with unknown registry", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true, regErr: errors.New("missing")})
		rec := httptest.NewRecorder()
		h.PatchBlobUpload(rec, rchRequest(http.MethodPatch, uploadPath))
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
	t.Run("catalog is not implemented but does not fail", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetCatalog(rec, rchRequest(http.MethodGet, "/v2/_catalog"))
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestRchGetRequestedResourceActions(t *testing.T) {
	got := GetRequestedResourceActions([]string{
		"", "registry", "repository:root/reg", "repository:root/reg/app:pull,push", "repository:a:b:c:",
	})
	require.Len(t, got, 4)
	require.Equal(t, &ResourceActions{Type: "registry", Actions: []string{}}, got[0])
	require.Equal(t, &ResourceActions{Type: "repository", Name: "root/reg", Actions: []string{}}, got[1])
	require.Equal(t, &ResourceActions{
		Type: "repository", Name: "root/reg/app", Actions: []string{"pull", "push"},
	}, got[2])
	require.Equal(t, &ResourceActions{Type: "repository", Name: "a:b:c", Actions: []string{}}, got[3])
	require.Empty(t, GetRequestedResourceActions(nil))
}

func TestRchGetScopes(t *testing.T) {
	target := "/token?scope=repository:a:pull%20repository:b:push&scope=registry:catalog:*"
	req := httptest.NewRequest(http.MethodGet, target, nil)
	require.Equal(t, []string{"repository:a:pull", "repository:b:push", "registry:catalog:*"}, getScopes(req.URL))
	require.Empty(t, getScopes(httptest.NewRequest(http.MethodGet, "/token", nil).URL))
}

func TestRchGetPermissionFromAction(t *testing.T) {
	ctx := context.Background()
	tests := map[string]enum.Permission{
		"pull":   enum.PermissionArtifactsDownload,
		"push":   enum.PermissionArtifactsUpload,
		"delete": enum.PermissionArtifactsDelete,
	}
	for action, want := range tests {
		got, err := getPermissionFromAction(ctx, action)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := getPermissionFromAction(ctx, "unknown")
	require.Error(t, err)
}

func TestRchGetToken(t *testing.T) {
	newReq := func(session *auth.Session, query string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/token?"+query, nil)
		return req.WithContext(apirequest.WithAuthSession(req.Context(), session))
	}
	anonymous := &auth.Session{Principal: auth.AnonymousPrincipal}

	t.Run("anonymous session gets a token", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetToken(rec, newReq(anonymous, "scope=repository:root/reg/app:pull,push,bogus"))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `"token"`)
	})
	t.Run("denied scopes are dropped but a token is still issued", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: false})
		rec := httptest.NewRecorder()
		h.GetToken(rec, newReq(anonymous, "scope=repository:root/reg/app:pull"))
		require.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("unknown space is skipped", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetToken(rec, newReq(anonymous, "scope=repository:nope/reg/app:pull"))
		require.NotEqual(t, 0, rec.Code)
	})
	t.Run("no scopes", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		rec := httptest.NewRecorder()
		h.GetToken(rec, newReq(anonymous, ""))
		require.Equal(t, http.StatusOK, rec.Code)
	})
	t.Run("non PAT token is forbidden", func(t *testing.T) {
		h := rchSetup(t, rchOpts{allow: true})
		session := &auth.Session{
			Principal: types.Principal{ID: 1, UID: "user"},
			Metadata:  &auth.TokenMetadata{TokenType: enum.TokenTypeSession},
		}
		rec := httptest.NewRecorder()
		h.GetToken(rec, newReq(session, ""))
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "only personal access token allowed")
	})
}

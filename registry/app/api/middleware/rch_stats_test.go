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
	"testing"

	apirequest "github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/audit"
	genericctrl "github.com/harness/gitness/registry/app/api/controller/pkg/generic"
	generichandler "github.com/harness/gitness/registry/app/api/handler/generic"
	mavenhandler "github.com/harness/gitness/registry/app/api/handler/maven"
	"github.com/harness/gitness/registry/app/api/handler/oci"
	"github.com/harness/gitness/registry/app/api/middleware"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/pkg/maven"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

type rchStatDaos struct {
	store.BlobRepository
	store.ImageRepository
	store.ArtifactRepository
	store.BandwidthStatRepository
	store.DownloadStatRepository
	store.RegistryRepository

	image        *registrytypes.Image
	imageErr     error
	artifact     *registrytypes.Artifact
	artifactErr  error
	blob         *registrytypes.Blob
	blobErr      error
	bandwidth    []*registrytypes.BandwidthStat
	bandwidthErr error
	downloads    []*registrytypes.DownloadStat
	downloadErr  error
	registry     *registrytypes.Registry
	registryErr  error
}

func (d *rchStatDaos) FindByDigestAndRootParentID(
	context.Context, digest.Digest, int64,
) (*registrytypes.Blob, error) {
	return d.blob, d.blobErr
}

func (d *rchStatDaos) GetByName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Image, error) {
	return d.image, d.imageErr
}

func (d *rchStatDaos) CreateBandwidth(stat *registrytypes.BandwidthStat) error {
	d.bandwidth = append(d.bandwidth, stat)
	return d.bandwidthErr
}

func (d *rchStatDaos) GetByParentIDAndName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.registry, d.registryErr
}

func (d *rchStatDaos) GetByRootParentIDAndName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.registry, d.registryErr
}

// Separate small adapters are needed because both stat DAOs expose a Create method.
type rchBandwidthAdapter struct{ d *rchStatDaos }

func (a rchBandwidthAdapter) Create(_ context.Context, stat *registrytypes.BandwidthStat) error {
	return a.d.CreateBandwidth(stat)
}

type rchDownloadAdapter struct {
	store.DownloadStatRepository
	d *rchStatDaos
}

func (a rchDownloadAdapter) Create(_ context.Context, stat *registrytypes.DownloadStat) error {
	a.d.downloads = append(a.d.downloads, stat)
	return a.d.downloadErr
}

type rchImageAdapter struct {
	store.ImageRepository
	d *rchStatDaos
}

func (a rchImageAdapter) GetByName(
	ctx context.Context, regID int64, name string, opts ...registrytypes.QueryOption,
) (*registrytypes.Image, error) {
	return a.d.GetByName(ctx, regID, name, opts...)
}

type rchArtifactAdapter struct {
	store.ArtifactRepository
	d *rchStatDaos
}

func (a rchArtifactAdapter) GetByName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Artifact, error) {
	return a.d.artifact, a.d.artifactErr
}

type rchBlobAdapter struct {
	store.BlobRepository
	d *rchStatDaos
}

func (a rchBlobAdapter) FindByDigestAndRootParentID(
	ctx context.Context, dgst digest.Digest, root int64,
) (*registrytypes.Blob, error) {
	return a.d.FindByDigestAndRootParentID(ctx, dgst, root)
}

type rchRegistryAdapter struct {
	store.RegistryRepository
	d *rchStatDaos
}

func (a rchRegistryAdapter) GetByParentIDAndName(
	ctx context.Context, parent int64, name string, opts ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return a.d.GetByParentIDAndName(ctx, parent, name, opts...)
}

func (a rchRegistryAdapter) GetByRootParentIDAndName(
	ctx context.Context, parent int64, name string, opts ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return a.d.GetByRootParentIDAndName(ctx, parent, name, opts...)
}

func (a rchRegistryAdapter) GetByIDIn(
	context.Context, []int64, ...registrytypes.QueryOption,
) (*[]registrytypes.Registry, error) {
	return &[]registrytypes.Registry{}, nil
}

func rchNewDaos() *rchStatDaos {
	return &rchStatDaos{
		image:    &registrytypes.Image{ID: 3},
		artifact: &registrytypes.Artifact{ID: 4, Metadata: []byte(`{"files":[{"size":10},{"size":5}]}`)},
		blob:     &registrytypes.Blob{Size: 42},
		registry: rchDockerRegistry(),
	}
}

type rchBandwidthDao struct {
	store.BandwidthStatRepository
	a rchBandwidthAdapter
}

func (b rchBandwidthDao) Create(ctx context.Context, stat *registrytypes.BandwidthStat) error {
	return b.a.Create(ctx, stat)
}

func rchOCIStatsHandler(d *rchStatDaos) *oci.Handler {
	dbStore := docker.NewDBStore(
		rchBlobAdapter{d: d}, rchImageAdapter{d: d}, rchArtifactAdapter{d: d},
		rchBandwidthDao{a: rchBandwidthAdapter{d: d}}, rchDownloadAdapter{d: d}, nil, nil,
	)
	core := pkg.NewCoreController(rchRegistryAdapter{d: d}, nil)
	ctrl := docker.NewController(nil, nil, core, nil, nil, dbStore, rchSpaceFinder())
	return oci.NewHandler(
		ctrl, rchSpaceFinder(), nil, nil, nil, nil, rchURLProvider{}, nil, true,
		rchRegFinder{reg: d.registry}, rchPublicAccess{}, "", rchAuditSvc{},
	)
}

func rchServe(mw func(http.Handler) http.Handler, method, target string, status int) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	session := &auth.Session{Principal: types.Principal{ID: 1, UID: "user"}}
	req = req.WithContext(apirequest.WithAuthSession(req.Context(), session))
	mw(&rchNext{status: status}).ServeHTTP(rec, req)
	return rec
}

func TestRchTrackDownloadStatOCI(t *testing.T) {
	manifestByDigest := "/v2/root/reg/app/manifests/" + rchDigest

	t.Run("records a download for a successful manifest pull", func(t *testing.T) {
		d := rchNewDaos()
		middlewareFn := middleware.TrackDownloadStat(rchOCIStatsHandler(d))
		rchServe(middlewareFn, http.MethodGet, manifestByDigest, http.StatusOK)
		require.Len(t, d.downloads, 1)
		require.Equal(t, int64(4), d.downloads[0].ArtifactID)
	})
	t.Run("falls back to the upstream proxies when the image is unknown", func(t *testing.T) {
		d := rchNewDaos()
		d.imageErr = basestore.ErrResourceNotFound
		rchServe(middleware.TrackDownloadStat(rchOCIStatsHandler(d)), http.MethodGet, manifestByDigest, http.StatusOK)
		require.Empty(t, d.downloads)
	})
	t.Run("lookup and write failures are tolerated", func(t *testing.T) {
		failures := map[string]func(d *rchStatDaos){
			"image":    func(d *rchStatDaos) { d.imageErr = errors.New("db") },
			"artifact": func(d *rchStatDaos) { d.artifactErr = errors.New("db") },
			"create":   func(d *rchStatDaos) { d.downloadErr = errors.New("db") },
		}
		for name, fail := range failures {
			d := rchNewDaos()
			fail(d)
			rec := rchServe(middleware.TrackDownloadStat(rchOCIStatsHandler(d)),
				http.MethodGet, manifestByDigest, http.StatusOK)
			require.Equal(t, http.StatusOK, rec.Code, name)
		}
	})
	t.Run("only successful manifest gets are tracked", func(t *testing.T) {
		cases := []struct {
			method string
			target string
			status int
		}{
			{http.MethodGet, manifestByDigest, http.StatusNotFound},
			{http.MethodHead, manifestByDigest, http.StatusOK},
			{http.MethodGet, "/v2/root/reg/app/blobs/" + rchDigest, http.StatusOK},
		}
		for _, tc := range cases {
			d := rchNewDaos()
			rchServe(middleware.TrackDownloadStat(rchOCIStatsHandler(d)), tc.method, tc.target, tc.status)
			require.Empty(t, d.downloads, tc.method+" "+tc.target)
		}
	})
	t.Run("a tag reference has no digest to record", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackDownloadStat(rchOCIStatsHandler(d)),
			http.MethodGet, "/v2/root/reg/app/manifests/latest", http.StatusOK)
		require.Empty(t, d.downloads)
	})
	t.Run("unknown registry is tolerated", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackDownloadStat(rchOCIStatsHandler(d)),
			http.MethodGet, "/v2/unknown/reg/app/manifests/"+rchDigest, http.StatusOK)
		require.Empty(t, d.downloads)
	})
}

func TestRchTrackBandwidthStatOCI(t *testing.T) {
	blobPath := "/v2/root/reg/app/blobs/" + rchDigest
	uploadPath := "/v2/root/reg/app/blobs/uploads/session-id?digest=" + rchDigest

	t.Run("records the downloaded bytes", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackBandwidthStat(rchOCIStatsHandler(d)), http.MethodGet, blobPath, http.StatusOK)
		require.Len(t, d.bandwidth, 1)
		require.Equal(t, registrytypes.BandwidthTypeDOWNLOAD, d.bandwidth[0].Type)
		require.Equal(t, int64(42), d.bandwidth[0].Bytes)
		require.Equal(t, int64(3), d.bandwidth[0].ImageID)
	})
	t.Run("records the uploaded bytes", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackBandwidthStat(rchOCIStatsHandler(d)), http.MethodPut, uploadPath, http.StatusCreated)
		require.Len(t, d.bandwidth, 1)
		require.Equal(t, registrytypes.BandwidthTypeUPLOAD, d.bandwidth[0].Type)
	})
	t.Run("unsuccessful requests are not recorded", func(t *testing.T) {
		cases := []struct {
			method string
			target string
			status int
		}{
			{http.MethodGet, blobPath, http.StatusNotFound},
			{http.MethodPut, uploadPath, http.StatusAccepted},
			{http.MethodPost, uploadPath, http.StatusCreated},
			{http.MethodGet, "/v2/root/reg/app/manifests/latest", http.StatusOK},
		}
		for _, tc := range cases {
			d := rchNewDaos()
			rchServe(middleware.TrackBandwidthStat(rchOCIStatsHandler(d)), tc.method, tc.target, tc.status)
			require.Empty(t, d.bandwidth, tc.method+" "+tc.target)
		}
	})
	t.Run("lookup failures are tolerated", func(t *testing.T) {
		failures := map[string]func(d *rchStatDaos){
			"blob":      func(d *rchStatDaos) { d.blobErr = errors.New("db") },
			"image":     func(d *rchStatDaos) { d.imageErr = errors.New("db") },
			"unknown":   func(d *rchStatDaos) { d.imageErr = basestore.ErrResourceNotFound },
			"bandwidth": func(d *rchStatDaos) { d.bandwidthErr = errors.New("db") },
		}
		for name, fail := range failures {
			d := rchNewDaos()
			fail(d)
			rec := rchServe(middleware.TrackBandwidthStat(rchOCIStatsHandler(d)), http.MethodGet, blobPath, http.StatusOK)
			require.Equal(t, http.StatusOK, rec.Code, name)
		}
	})
	t.Run("unknown registry is tolerated", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackBandwidthStat(rchOCIStatsHandler(d)),
			http.MethodGet, "/v2/unknown/reg/app/blobs/"+rchDigest, http.StatusOK)
		require.Empty(t, d.bandwidth)
	})
}

type rchAuditSvc struct {
	audit.Service
}

func (rchAuditSvc) Log(context.Context, types.Principal, audit.Resource, audit.Action, string, ...audit.Option) error {
	return nil
}

type rchAllowAll struct {
	authz.Authorizer
}

func (rchAllowAll) Check(context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission) (bool, error) {
	return true, nil
}

func (rchAllowAll) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return true, nil
}

func rchGenericStatsHandler(d *rchStatDaos) *generichandler.Handler {
	d.registry = &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5,
		PackageType: artifact.PackageTypeGENERIC, Type: artifact.RegistryTypeVIRTUAL,
	}
	dbStore := genericctrl.NewDBStore(
		rchRegistryAdapter{d: d}, rchImageAdapter{d: d}, rchArtifactAdapter{d: d},
		rchBandwidthDao{a: rchBandwidthAdapter{d: d}}, rchDownloadAdapter{d: d},
	)
	ctrl := genericctrl.NewController(
		nil, rchAllowAll{}, nil, dbStore, nil, rchSpaceFinder(), nil, nil, nil, nil, nil, nil,
	)
	return generichandler.NewGenericArtifactHandler(
		nil, ctrl, nil, nil, nil, nil, rchAllowAll{}, nil, rchSpaceFinder(), nil, rchAuditSvc{},
	)
}

func TestRchGenericArtifactStats(t *testing.T) {
	const path = "/generic/root/reg/tool/1.0.0"

	t.Run("download is recorded for ok and redirect", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusTemporaryRedirect} {
			d := rchNewDaos()
			rchServe(middleware.TrackDownloadStatForGenericArtifact(rchGenericStatsHandler(d)),
				http.MethodGet, path, status)
			require.Len(t, d.downloads, 1, "status %d", status)
		}
	})
	t.Run("download is not recorded for other requests", func(t *testing.T) {
		cases := []struct {
			method string
			status int
		}{{http.MethodGet, http.StatusNotFound}, {http.MethodPut, http.StatusOK}}
		for _, tc := range cases {
			d := rchNewDaos()
			rchServe(middleware.TrackDownloadStatForGenericArtifact(rchGenericStatsHandler(d)), tc.method, path, tc.status)
			require.Empty(t, d.downloads)
		}
	})
	t.Run("download failures are tolerated", func(t *testing.T) {
		failures := map[string]func(d *rchStatDaos){
			"registry": func(d *rchStatDaos) { d.registryErr = errors.New("db") },
			"image":    func(d *rchStatDaos) { d.imageErr = errors.New("db") },
			"artifact": func(d *rchStatDaos) { d.artifactErr = errors.New("db") },
			"create":   func(d *rchStatDaos) { d.downloadErr = errors.New("db") },
		}
		for name, fail := range failures {
			d := rchNewDaos()
			h := rchGenericStatsHandler(d)
			fail(d)
			rec := rchServe(middleware.TrackDownloadStatForGenericArtifact(h), http.MethodGet, path, http.StatusOK)
			require.Equal(t, http.StatusOK, rec.Code, name)
		}
	})
	t.Run("invalid path is tolerated", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackDownloadStatForGenericArtifact(rchGenericStatsHandler(d)),
			http.MethodGet, "/nope", http.StatusOK)
		require.Empty(t, d.downloads)
	})

	t.Run("bandwidth sums the file sizes", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackBandwidthStatForGenericArtifacts(rchGenericStatsHandler(d)),
			http.MethodGet, path, http.StatusOK)
		require.Len(t, d.bandwidth, 1)
		require.Equal(t, int64(15), d.bandwidth[0].Bytes)
		require.Equal(t, registrytypes.BandwidthTypeDOWNLOAD, d.bandwidth[0].Type)
	})
	t.Run("bandwidth is recorded for uploads", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackBandwidthStatForGenericArtifacts(rchGenericStatsHandler(d)),
			http.MethodPut, path, http.StatusCreated)
		require.Len(t, d.bandwidth, 1)
		require.Equal(t, registrytypes.BandwidthTypeUPLOAD, d.bandwidth[0].Type)
	})
	t.Run("bandwidth is not recorded for other requests", func(t *testing.T) {
		cases := []struct {
			method string
			status int
		}{
			{http.MethodGet, http.StatusNotFound},
			{http.MethodPut, http.StatusOK},
			{http.MethodDelete, http.StatusOK},
		}
		for _, tc := range cases {
			d := rchNewDaos()
			rchServe(middleware.TrackBandwidthStatForGenericArtifacts(rchGenericStatsHandler(d)), tc.method, path, tc.status)
			require.Empty(t, d.bandwidth, tc.method)
		}
	})
	t.Run("bandwidth failures are tolerated", func(t *testing.T) {
		failures := map[string]func(d *rchStatDaos){
			"registry":  func(d *rchStatDaos) { d.registryErr = errors.New("db") },
			"image":     func(d *rchStatDaos) { d.imageErr = errors.New("db") },
			"artifact":  func(d *rchStatDaos) { d.artifactErr = errors.New("db") },
			"metadata":  func(d *rchStatDaos) { d.artifact = &registrytypes.Artifact{Metadata: []byte("{bad")} },
			"bandwidth": func(d *rchStatDaos) { d.bandwidthErr = errors.New("db") },
		}
		for name, fail := range failures {
			d := rchNewDaos()
			h := rchGenericStatsHandler(d)
			fail(d)
			rec := rchServe(middleware.TrackBandwidthStatForGenericArtifacts(h), http.MethodGet, path, http.StatusOK)
			require.Equal(t, http.StatusOK, rec.Code, name)
		}
	})
	t.Run("bandwidth with an invalid path is tolerated", func(t *testing.T) {
		d := rchNewDaos()
		rchServe(middleware.TrackBandwidthStatForGenericArtifacts(rchGenericStatsHandler(d)),
			http.MethodGet, "/nope", http.StatusOK)
		require.Empty(t, d.bandwidth)
	})
}

func rchMavenStatsHandler(d *rchStatDaos, reg *registrytypes.Registry) *mavenhandler.Handler {
	dbStore := maven.NewDBStore(
		rchRegistryAdapter{d: d}, rchImageAdapter{d: d}, rchArtifactAdapter{d: d}, nil,
		rchBandwidthDao{a: rchBandwidthAdapter{d: d}}, rchDownloadAdapter{d: d}, nil, nil,
	)
	ctrl := maven.NewController(nil, nil, rchAllowAll{}, dbStore, rchSpaceFinder(), nil, nil)
	return mavenhandler.NewHandler(
		ctrl, nil, nil, nil, nil, rchAllowAll{}, rchSpaceFinder(), rchRegFinder{reg: reg}, nil, rchAuditSvc{},
	)
}

func rchMavenRegistry() *registrytypes.Registry {
	return &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5,
		PackageType: artifact.PackageTypeMAVEN, Type: artifact.RegistryTypeVIRTUAL,
	}
}

func TestRchMavenArtifactStats(t *testing.T) {
	const jar = "/maven/root/reg/io/example/app/1.0/app-1.0.jar"
	const pom = "/maven/root/reg/io/example/app/1.0/app-1.0.pom"

	newDaos := func() *rchStatDaos {
		d := rchNewDaos()
		d.registry = rchMavenRegistry()
		return d
	}

	t.Run("main artifact downloads are recorded", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusTemporaryRedirect} {
			d := newDaos()
			rchServe(middleware.TrackDownloadStatForMavenArtifact(rchMavenStatsHandler(d, d.registry)),
				http.MethodGet, jar, status)
			require.Len(t, d.downloads, 1, "status %d", status)
		}
	})
	t.Run("other files and requests are not recorded", func(t *testing.T) {
		cases := []struct {
			method string
			path   string
			status int
		}{
			{http.MethodGet, pom, http.StatusOK},
			{http.MethodGet, jar, http.StatusNotFound},
			{http.MethodPut, jar, http.StatusOK},
			{http.MethodGet, "/maven/root", http.StatusOK},
		}
		for _, tc := range cases {
			d := newDaos()
			rchServe(middleware.TrackDownloadStatForMavenArtifact(rchMavenStatsHandler(d, d.registry)),
				tc.method, tc.path, tc.status)
			require.Empty(t, d.downloads, tc.method+" "+tc.path)
		}
	})
	t.Run("download failures are tolerated", func(t *testing.T) {
		failures := map[string]func(d *rchStatDaos){
			"registry": func(d *rchStatDaos) { d.registryErr = errors.New("db") },
			"image":    func(d *rchStatDaos) { d.imageErr = errors.New("db") },
			"unknown":  func(d *rchStatDaos) { d.imageErr = basestore.ErrResourceNotFound },
			"artifact": func(d *rchStatDaos) { d.artifactErr = errors.New("db") },
			"create":   func(d *rchStatDaos) { d.downloadErr = errors.New("db") },
		}
		for name, fail := range failures {
			d := newDaos()
			h := rchMavenStatsHandler(d, d.registry)
			fail(d)
			rec := rchServe(middleware.TrackDownloadStatForMavenArtifact(h), http.MethodGet, jar, http.StatusOK)
			require.Equal(t, http.StatusOK, rec.Code, name)
		}
	})

	t.Run("bandwidth is recorded for downloads and uploads", func(t *testing.T) {
		d := newDaos()
		d.artifact = &registrytypes.Artifact{
			ID: 4, Metadata: []byte(`{"files":[{"file_name":"other.jar","size":3},{"file_name":"app-1.0.jar","size":7}]}`),
		}
		rchServe(middleware.TrackBandwidthStatForMavenArtifacts(rchMavenStatsHandler(d, d.registry)),
			http.MethodGet, jar, http.StatusOK)
		rchServe(middleware.TrackBandwidthStatForMavenArtifacts(rchMavenStatsHandler(d, d.registry)),
			http.MethodPut, jar, http.StatusCreated)
		require.Len(t, d.bandwidth, 2)
		require.Equal(t, registrytypes.BandwidthTypeDOWNLOAD, d.bandwidth[0].Type)
		require.Equal(t, registrytypes.BandwidthTypeUPLOAD, d.bandwidth[1].Type)
		require.Equal(t, int64(7), d.bandwidth[0].Bytes)
	})
	t.Run("bandwidth is not recorded for other requests", func(t *testing.T) {
		cases := []struct {
			method string
			status int
		}{
			{http.MethodGet, http.StatusNotFound},
			{http.MethodPut, http.StatusOK},
			{http.MethodDelete, http.StatusOK},
		}
		for _, tc := range cases {
			d := newDaos()
			rchServe(middleware.TrackBandwidthStatForMavenArtifacts(rchMavenStatsHandler(d, d.registry)),
				tc.method, jar, tc.status)
			require.Empty(t, d.bandwidth, tc.method)
		}
	})
	t.Run("bandwidth failures are tolerated", func(t *testing.T) {
		failures := map[string]func(d *rchStatDaos){
			"registry":  func(d *rchStatDaos) { d.registryErr = errors.New("db") },
			"image":     func(d *rchStatDaos) { d.imageErr = errors.New("db") },
			"unknown":   func(d *rchStatDaos) { d.imageErr = basestore.ErrResourceNotFound },
			"artifact":  func(d *rchStatDaos) { d.artifactErr = errors.New("db") },
			"metadata":  func(d *rchStatDaos) { d.artifact = &registrytypes.Artifact{Metadata: []byte("{bad")} },
			"bandwidth": func(d *rchStatDaos) { d.bandwidthErr = errors.New("db") },
		}
		for name, fail := range failures {
			d := newDaos()
			h := rchMavenStatsHandler(d, d.registry)
			fail(d)
			rec := rchServe(middleware.TrackBandwidthStatForMavenArtifacts(h), http.MethodGet, jar, http.StatusOK)
			require.Equal(t, http.StatusOK, rec.Code, name)
		}
	})
	t.Run("bandwidth with an invalid path is tolerated", func(t *testing.T) {
		d := newDaos()
		rchServe(middleware.TrackBandwidthStatForMavenArtifacts(rchMavenStatsHandler(d, d.registry)),
			http.MethodGet, "/maven/root", http.StatusOK)
		require.Empty(t, d.bandwidth)
	})
}

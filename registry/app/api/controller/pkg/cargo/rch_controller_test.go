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

package cargo_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	urlprovider "github.com/harness/gitness/app/url"
	cargoctrl "github.com/harness/gitness/registry/app/api/controller/pkg/cargo"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	cargometa "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	cargopkg "github.com/harness/gitness/registry/app/pkg/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

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

type rchRegistryDao struct {
	store.RegistryRepository
	reg     *registrytypes.Registry
	err     error
	keys    []string
	keysErr error
}

func (d *rchRegistryDao) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.reg, d.err
}

func (d *rchRegistryDao) FetchUpstreamProxyKeys(
	context.Context, []int64, ...registrytypes.QueryOption,
) ([]string, error) {
	return d.keys, d.keysErr
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

type rchArtifactDao struct {
	store.ArtifactRepository
	artifacts *[]registrytypes.ArtifactMetadata
	getErr    error
	count     int64
	countErr  error
}

func (d *rchArtifactDao) GetAllArtifactsByParentID(
	context.Context, int64, *[]string, string, string, int, int, string, bool, []string,
) (*[]registrytypes.ArtifactMetadata, error) {
	return d.artifacts, d.getErr
}

func (d *rchArtifactDao) CountAllArtifactsByParentID(
	context.Context, int64, *[]string, string, bool, []string,
) (int64, error) {
	return d.count, d.countErr
}

type rchPublicAccess struct {
	publicaccess.Service
	public bool
	err    error
}

func (p rchPublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return p.public, p.err
}

type rchURLProvider struct {
	urlprovider.Provider
}

func (rchURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, _ ...string) string {
	return "https://pkg.example.com/pkg/" + regRef + "/" + pkgType + "/"
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

type rchCargoRegistry struct {
	cargopkg.Registry
	err error
}

func (r *rchCargoRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *rchCargoRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeCARGO}
}

func (r *rchCargoRegistry) UploadPackage(
	context.Context, cargotype.ArtifactInfo, *cargometa.VersionMetadata, io.ReadCloser,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, r.err
}

func (r *rchCargoRegistry) DownloadPackageIndex(
	context.Context, cargotype.ArtifactInfo, string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, io.NopCloser(strings.NewReader("idx")), "", r.err
}

func (r *rchCargoRegistry) RegeneratePackageIndex(
	context.Context, cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, r.err
}

func (r *rchCargoRegistry) DownloadPackage(
	context.Context, cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, io.NopCloser(strings.NewReader("crate")), "", r.err
}

func (r *rchCargoRegistry) UpdateYank(
	context.Context, cargotype.ArtifactInfo, bool,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, r.err
}

type rchOpts struct {
	regErr     error
	daoErr     error
	finderReg  *registrytypes.Registry
	finderErr  error
	artifacts  *rchArtifactDao
	keysErr    error
	publicErr  error
	public     bool
	quarantine error
	upstreams  []int64
}

func rchSetup(t *testing.T, o rchOpts) cargoctrl.Controller {
	t.Helper()
	base.Register(&rchCargoRegistry{err: o.regErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "CARGO:VIRTUAL") })

	reg := &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5, UpstreamProxies: o.upstreams,
		PackageType: artifact.PackageTypeCARGO, Type: artifact.RegistryTypeVIRTUAL,
	}
	dao := &rchRegistryDao{reg: reg, err: o.daoErr, keys: []string{"upstream"}, keysErr: o.keysErr}
	finderReg := o.finderReg
	if finderReg == nil {
		finderReg = reg
	}
	if o.artifacts == nil {
		o.artifacts = &rchArtifactDao{}
	}
	return cargoctrl.NewController(
		nil, dao, rchRegFinder{reg: finderReg, err: o.finderErr}, nil, o.artifacts, nil, nil, rchURLProvider{},
		nil, nil, rchPublicAccess{public: o.public, err: o.publicErr}, rchSpaceFinder(),
		rchQuarantine{err: o.quarantine}, nil,
	)
}

func rchInfo() *cargotype.ArtifactInfo {
	return &cargotype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5, RootIdentifier: "root"},
			RegIdentifier: "reg",
			Image:         "serde",
		},
		Version:  "1.0.0",
		FileName: "serde-1.0.0.crate",
	}
}

type rchOp struct {
	name string
	call func(c cargoctrl.Controller) error
}

func rchOps() []rchOp {
	ctx := context.Background()
	return []rchOp{
		{"upload", func(c cargoctrl.Controller) error {
			_, err := c.UploadPackage(ctx, rchInfo(), &cargometa.VersionMetadata{}, io.NopCloser(strings.NewReader("x")))
			return err
		}},
		{"regenerate index", func(c cargoctrl.Controller) error {
			_, err := c.RegeneratePackageIndex(ctx, rchInfo())
			return err
		}},
		{"yank", func(c cargoctrl.Controller) error {
			_, err := c.UpdateYank(ctx, rchInfo(), true)
			return err
		}},
		{"unyank", func(c cargoctrl.Controller) error {
			_, err := c.UpdateYank(ctx, rchInfo(), false)
			return err
		}},
		{"download", func(c cargoctrl.Controller) error { return c.DownloadPackage(ctx, rchInfo()).GetError() }},
		{"download index", func(c cargoctrl.Controller) error {
			return c.DownloadPackageIndex(ctx, rchInfo(), "se/rd/serde").GetError()
		}},
	}
}

func TestRchCargoControllerSuccess(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.NoError(t, op.call(rchSetup(t, rchOpts{})))
		})
	}
}

func TestRchCargoControllerRegistryError(t *testing.T) {
	regErr := errors.New("backend failed")
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.ErrorIs(t, op.call(rchSetup(t, rchOpts{regErr: regErr})), regErr)
		})
	}
}

func TestRchCargoControllerRegistryNotFound(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.Error(t, op.call(rchSetup(t, rchOpts{daoErr: errors.New("missing")})))
		})
	}
}

func TestRchCargoControllerQuarantinedDownload(t *testing.T) {
	c := rchSetup(t, rchOpts{quarantine: usererror.ErrQuarantinedArtifact})
	err := c.DownloadPackage(context.Background(), rchInfo()).GetError()
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
}

func TestRchCargoControllerResponses(t *testing.T) {
	ctx := context.Background()
	c := rchSetup(t, rchOpts{})

	yank, err := c.UpdateYank(ctx, rchInfo(), true)
	require.NoError(t, err)
	require.True(t, yank.Ok)

	regen, err := c.RegeneratePackageIndex(ctx, rchInfo())
	require.NoError(t, err)
	require.True(t, regen.Ok)

	dl := c.DownloadPackage(ctx, rchInfo())
	require.NotNil(t, dl.ReadCloser)
	require.NoError(t, dl.ReadCloser.Close())

	idx := c.DownloadPackageIndex(ctx, rchInfo(), "se/rd/serde")
	require.NotNil(t, idx.ReadCloser)
	require.NoError(t, idx.ReadCloser.Close())
}

func TestRchCargoGetRegistryConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("private registry requires auth", func(t *testing.T) {
		resp, err := rchSetup(t, rchOpts{}).GetRegistryConfig(ctx, rchInfo())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.ResponseHeaders.Code)
		require.True(t, resp.Config.AuthRequired)
		require.Equal(t, "https://pkg.example.com/pkg/root/reg/cargo/", resp.Config.APIURL)
		require.Equal(t, "https://pkg.example.com/pkg/root/reg/cargo/api/v1/crates", resp.Config.DownloadURL)
	})
	t.Run("public registry does not require auth", func(t *testing.T) {
		resp, err := rchSetup(t, rchOpts{public: true}).GetRegistryConfig(ctx, rchInfo())
		require.NoError(t, err)
		require.False(t, resp.Config.AuthRequired)
	})
	t.Run("unknown space", func(t *testing.T) {
		info := rchInfo()
		info.ParentID = 99
		resp, err := rchSetup(t, rchOpts{}).GetRegistryConfig(ctx, info)
		require.Error(t, err)
		require.Equal(t, http.StatusInternalServerError, resp.ResponseHeaders.Code)
	})
	t.Run("public access lookup fails", func(t *testing.T) {
		resp, err := rchSetup(t, rchOpts{publicErr: errors.New("db")}).GetRegistryConfig(ctx, rchInfo())
		require.Error(t, err)
		require.Equal(t, http.StatusInternalServerError, resp.ResponseHeaders.Code)
	})
}

func TestRchCargoSearchPackage(t *testing.T) {
	ctx := context.Background()
	term := "serde"
	size := int32(500)
	found := []registrytypes.ArtifactMetadata{
		{Name: "serde", Version: "1.0.0", Metadata: []byte(`{"description":"serialization"}`)},
		{Name: "broken", Version: "0.1.0", Metadata: []byte("not-json")},
	}

	t.Run("maps artifacts to crates", func(t *testing.T) {
		c := rchSetup(t, rchOpts{artifacts: &rchArtifactDao{artifacts: &found, count: 2}})
		resp, err := c.SearchPackage(ctx, rchInfo(), &cargotype.SearchPackageRequestParams{SearchTerm: &term})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.ResponseHeaders.Code)
		require.Equal(t, int64(2), resp.Metadata.Total)
		require.Len(t, resp.Crates, 2)
		require.Equal(t, cargoctrl.SearchPackageResponseCrate{
			Name: "serde", MaxVersion: "1.0.0", Description: "serialization",
		}, resp.Crates[0])
		require.Empty(t, resp.Crates[1].Description)
	})
	t.Run("nil params and oversized page are accepted", func(t *testing.T) {
		c := rchSetup(t, rchOpts{artifacts: &rchArtifactDao{}})
		resp, err := c.SearchPackage(ctx, rchInfo(), nil)
		require.NoError(t, err)
		require.Empty(t, resp.Crates)
		resp, err = c.SearchPackage(ctx, rchInfo(), &cargotype.SearchPackageRequestParams{Size: &size})
		require.NoError(t, err)
		require.Empty(t, resp.Crates)
	})
	t.Run("includes upstream proxies", func(t *testing.T) {
		c := rchSetup(t, rchOpts{upstreams: []int64{3}, artifacts: &rchArtifactDao{}})
		_, err := c.SearchPackage(ctx, rchInfo(), nil)
		require.NoError(t, err)
	})
	t.Run("upstream key lookup fails", func(t *testing.T) {
		c := rchSetup(t, rchOpts{upstreams: []int64{3}, keysErr: errors.New("db")})
		_, err := c.SearchPackage(ctx, rchInfo(), nil)
		require.Error(t, err)
	})
	t.Run("registry lookup fails", func(t *testing.T) {
		c := rchSetup(t, rchOpts{finderErr: errors.New("missing")})
		_, err := c.SearchPackage(ctx, rchInfo(), nil)
		require.Error(t, err)
	})
	t.Run("artifact query fails", func(t *testing.T) {
		c := rchSetup(t, rchOpts{artifacts: &rchArtifactDao{getErr: errors.New("db")}})
		_, err := c.SearchPackage(ctx, rchInfo(), nil)
		require.Error(t, err)
	})
	t.Run("artifact count fails", func(t *testing.T) {
		c := rchSetup(t, rchOpts{artifacts: &rchArtifactDao{countErr: errors.New("db")}})
		_, err := c.SearchPackage(ctx, rchInfo(), nil)
		require.Error(t, err)
	})
}

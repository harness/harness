//  Copyright 2023 Harness, Inc.
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

package pkg

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	store2 "github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type rpfFakeSpaceCache struct {
	spaces map[int64]*types.SpaceCore
}

func (c *rpfFakeSpaceCache) Stats() (int64, int64) { return 0, 0 }

func (c *rpfFakeSpaceCache) Get(_ context.Context, key int64) (*types.SpaceCore, error) {
	s, ok := c.spaces[key]
	if !ok {
		return nil, errors.New("space not found")
	}
	return s, nil
}

func (c *rpfFakeSpaceCache) Evict(context.Context, int64) {}

type rpfFakeAuthorizer struct {
	authz.Authorizer
	allowed bool
	err     error
	checks  []types.PermissionCheck
}

func (a *rpfFakeAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, checks ...types.PermissionCheck,
) (bool, error) {
	a.checks = checks
	return a.allowed, a.err
}

func rpfSpaceFinder() refcache.SpaceFinder {
	return refcache.NewSpaceFinder(&rpfFakeSpaceCache{spaces: map[int64]*types.SpaceCore{
		7: {ID: 7, Path: "acme/team"},
	}}, nil, nil, cache.Evictor[*types.SpaceCore]{})
}

func TestRpfGetRegistryCheckAccess(t *testing.T) {
	art := ArtifactInfo{Registry: registrytypes.Registry{Name: "my-reg"}}
	session := &auth.Session{Principal: types.Principal{ID: 5, Type: enum.PrincipalTypeUser}}
	ctx := request.WithAuthSession(context.Background(), session)

	t.Run("allowed builds one check per permission", func(t *testing.T) {
		authorizer := &rpfFakeAuthorizer{allowed: true}
		err := GetRegistryCheckAccess(ctx, authorizer, rpfSpaceFinder(), 7, art,
			enum.PermissionArtifactsDownload, enum.PermissionArtifactsUpload)
		require.NoError(t, err)
		require.Len(t, authorizer.checks, 2)
		require.Equal(t, enum.PermissionArtifactsDownload, authorizer.checks[0].Permission)
		require.Equal(t, "acme/team", authorizer.checks[0].Scope.SpacePath)
		require.Equal(t, "my-reg", authorizer.checks[0].Resource.Identifier)
		require.Equal(t, enum.ResourceTypeRegistry, authorizer.checks[0].Resource.Type)
	})

	t.Run("denied", func(t *testing.T) {
		authorizer := &rpfFakeAuthorizer{allowed: false}
		err := GetRegistryCheckAccess(ctx, authorizer, rpfSpaceFinder(), 7, art, enum.PermissionArtifactsDownload)
		require.ErrorContains(t, err, "registry access check failed")
	})

	t.Run("authorizer error", func(t *testing.T) {
		authorizer := &rpfFakeAuthorizer{err: errors.New("boom")}
		err := GetRegistryCheckAccess(ctx, authorizer, rpfSpaceFinder(), 7, art, enum.PermissionArtifactsDownload)
		require.ErrorContains(t, err, "boom")
	})

	t.Run("unknown parent space", func(t *testing.T) {
		authorizer := &rpfFakeAuthorizer{allowed: true}
		err := GetRegistryCheckAccess(ctx, authorizer, rpfSpaceFinder(), 99, art, enum.PermissionArtifactsDownload)
		require.ErrorContains(t, err, "failed to find parent by ref")
		require.Empty(t, authorizer.checks)
	})
}

func TestRpfArtifactInfoMutators(t *testing.T) {
	a := &ArtifactInfo{BaseInfo: &BaseInfo{}, Image: "demo", RegistryID: 3}
	require.Equal(t, "demo", a.GetImage())
	require.Equal(t, int64(3), a.GetRegistryID())

	a.UpdateRegistryInfo(registrytypes.Registry{ID: 11, Name: "reg", ParentID: 22})
	require.Equal(t, int64(11), a.RegistryID)
	require.Equal(t, "reg", a.RegIdentifier)
	require.Equal(t, int64(22), a.ParentID)
	require.Equal(t, "reg", a.Registry.Name)

	a.SetRepoKey("other")
	require.Equal(t, "other", a.RegIdentifier)

	r := &RegistryInfo{}
	r.SetReference("v1")
	require.Equal(t, "v1", r.Reference)
}

func TestRpfMavenArtifactInfo(t *testing.T) {
	base := &ArtifactInfo{Image: "org.acme:demo"}
	m := MavenArtifactInfo{
		ArtifactInfo: base, GroupID: "org.acme", ArtifactID: "demo", Version: "1.0", FileName: "demo-1.0.jar",
	}
	require.Equal(t, "org.acme:demo", m.BaseArtifactInfo().Image)
	require.Equal(t, "1.0", m.GetVersion())
	require.Equal(t, "demo-1.0.jar", m.GetFileName())
	exists, iv := m.GetImageVersion()
	require.True(t, exists)
	require.Equal(t, "org.acme:demo:1.0", iv)

	m.Version = ""
	exists, iv = m.GetImageVersion()
	require.False(t, exists)
	require.Empty(t, iv)

	m.SetMavenRepoKey("maven-key")
	require.Equal(t, "maven-key", base.RegIdentifier)
}

func TestRpfGenericArtifactInfo(t *testing.T) {
	base := &ArtifactInfo{Image: "file"}
	g := GenericArtifactInfo{ArtifactInfo: base, FileName: "f.bin", Version: "2"}
	require.Equal(t, "file", g.BaseArtifactInfo().Image)
	require.Equal(t, "2", g.GetVersion())
	require.Equal(t, "f.bin", g.GetFileName())
	exists, iv := g.GetImageVersion()
	require.True(t, exists)
	require.Equal(t, "file:2", iv)

	g.Image = ""
	exists, iv = g.GetImageVersion()
	require.False(t, exists)
	require.Empty(t, iv)
}

func TestRpfPathFor(t *testing.T) {
	got, err := PathFor(huggingFaceTypeSpec{imageType: "models", imageName: "org/m", path: "blobs/abc"})
	require.NoError(t, err)
	require.Equal(t, "/models/org/m/blobs/abc", got)

	_, err = PathFor(rpfUnknownSpec{})
	require.ErrorContains(t, err, "unknown path spec")
}

type rpfUnknownSpec struct{}

func (rpfUnknownSpec) pathSpec() {}

type rpfFakeRegistryRepo struct {
	store2.RegistryRepository
	registries []registrytypes.Registry
	err        error
	gotIDs     []int64
}

func (r *rpfFakeRegistryRepo) GetByIDIn(
	_ context.Context, ids []int64, _ ...registrytypes.QueryOption,
) (*[]registrytypes.Registry, error) {
	r.gotIDs = ids
	if r.err != nil {
		return nil, r.err
	}
	return &r.registries, nil
}

type rpfArtifactStub struct{ Artifact }

func TestRpfCoreControllerGetArtifact(t *testing.T) {
	ctx := context.Background()
	local, remote := &rpfArtifactStub{}, &rpfArtifactStub{}
	TypeRegistry[LocalRegistry] = local
	TypeRegistry[RemoteRegistry] = remote
	t.Cleanup(func() {
		delete(TypeRegistry, LocalRegistry)
		delete(TypeRegistry, RemoteRegistry)
	})
	c := NewCoreController(&rpfFakeRegistryRepo{}, nil)

	require.Same(t, local, c.GetArtifact(ctx, registrytypes.Registry{Type: artifact.RegistryTypeVIRTUAL}))
	require.Same(t, remote, c.GetArtifact(ctx, registrytypes.Registry{Type: artifact.RegistryTypeUPSTREAM}))
	require.Nil(t, c.factory(ctx, ArtifactType(1<<7)))
}

func TestRpfCoreControllerGetOrderedRepos(t *testing.T) {
	ctx := context.Background()

	t.Run("no proxies", func(t *testing.T) {
		repo := &rpfFakeRegistryRepo{}
		c := NewCoreController(repo, nil)
		info := RegistryInfo{ArtifactInfo: &ArtifactInfo{Registry: registrytypes.Registry{ID: 1, Name: "virtual"}}}
		got, err := c.GetOrderedRepos(ctx, info)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Nil(t, repo.gotIDs)
	})

	t.Run("proxies keep the configured order and skip unknown ones", func(t *testing.T) {
		repo := &rpfFakeRegistryRepo{registries: []registrytypes.Registry{
			{ID: 20, Name: "second"}, {ID: 10, Name: "first"},
		}}
		c := NewCoreController(repo, nil)
		info := RegistryInfo{ArtifactInfo: &ArtifactInfo{Registry: registrytypes.Registry{
			ID: 1, Name: "virtual", UpstreamProxies: []int64{10, 99, 20},
		}}}
		got, err := c.GetOrderedRepos(ctx, info)
		require.NoError(t, err)
		require.Equal(t, []int64{10, 99, 20}, repo.gotIDs)
		names := make([]string, 0, len(got))
		for _, r := range got {
			names = append(names, r.Name)
		}
		require.Equal(t, []string{"virtual", "first", "second"}, names)
	})

	t.Run("lookup error returns the virtual registry and the error", func(t *testing.T) {
		repo := &rpfFakeRegistryRepo{err: errors.New("db down")}
		c := NewCoreController(repo, nil)
		info := RegistryInfo{ArtifactInfo: &ArtifactInfo{Registry: registrytypes.Registry{
			Name: "virtual", UpstreamProxies: []int64{1},
		}}}
		got, err := c.GetOrderedRepos(ctx, info)
		require.ErrorContains(t, err, "db down")
		require.Len(t, got, 1)
	})
}

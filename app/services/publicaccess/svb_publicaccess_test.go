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

package publicaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	registryrefcache "github.com/harness/gitness/registry/app/services/refcache"
	registrytypes "github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbAccessStore struct {
	store.PublicAccessStore

	public    bool
	findErr   error
	createErr error
	deleteErr error
	lastType  enum.PublicResourceType
	lastID    int64
	created   bool
	deleted   bool
}

func (f *svbAccessStore) Find(_ context.Context, typ enum.PublicResourceType, id int64) (bool, error) {
	f.lastType, f.lastID = typ, id
	return f.public, f.findErr
}

func (f *svbAccessStore) Create(_ context.Context, typ enum.PublicResourceType, id int64) error {
	f.lastType, f.lastID, f.created = typ, id, true
	return f.createErr
}

func (f *svbAccessStore) Delete(_ context.Context, typ enum.PublicResourceType, id int64) error {
	f.lastType, f.lastID, f.deleted = typ, id, true
	return f.deleteErr
}

type svbRepoIDCache struct {
	store.RepoIDCache

	err error
}

func (f *svbRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.RepositoryCore{ID: id}, nil
}

type svbSpaceIDCache struct {
	store.SpaceIDCache

	err error
}

func (f *svbSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.SpaceCore{ID: id}, nil
}

type svbRegistryFinder struct {
	registryrefcache.RegistryFinder

	err     error
	gotRoot string
	gotName string
}

func (f *svbRegistryFinder) FindByRootRef(
	_ context.Context, root string, name string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	f.gotRoot, f.gotName = root, name
	if f.err != nil {
		return nil, f.err
	}
	return &registrytypes.Registry{ID: 33}, nil
}

type svbEnv struct {
	svc      Service
	accesses *svbAccessStore
	repos    *svbRepoIDCache
	spaces   *svbSpaceIDCache
	registry *svbRegistryFinder
}

func svbNewEnv(enabled bool) *svbEnv {
	env := &svbEnv{
		accesses: &svbAccessStore{},
		repos:    &svbRepoIDCache{},
		spaces:   &svbSpaceIDCache{},
		registry: &svbRegistryFinder{},
	}
	env.svc = NewService(
		enabled,
		env.accesses,
		refcache.NewSpaceFinder(env.spaces, nil, nil, cache.Evictor[*types.SpaceCore]{}),
		refcache.NewRepoFinder(nil, nil, env.repos, nil, cache.Evictor[*types.RepositoryCore]{}),
		env.registry,
	)
	return env
}

func TestSvbGet(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		typ    enum.PublicResourceType
		path   string
		wantID int64
	}{
		{"repo", enum.PublicResourceTypeRepo, "12", 12},
		{"space", enum.PublicResourceTypeSpace, "7", 7},
		{"registry", enum.PublicResourceTypeRegistry, "root/reg", 33},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := svbNewEnv(true)
			env.accesses.public = true
			got, err := env.svc.Get(ctx, tc.typ, tc.path)
			require.NoError(t, err)
			require.True(t, got)
			require.Equal(t, tc.wantID, env.accesses.lastID)
			require.Equal(t, tc.typ, env.accesses.lastType)
		})
	}

	t.Run("registry path is split into root and leaf", func(t *testing.T) {
		env := svbNewEnv(true)
		_, err := env.svc.Get(ctx, enum.PublicResourceTypeRegistry, "acct/org/reg")
		require.NoError(t, err)
		require.Equal(t, "acct", env.registry.gotRoot)
		require.Equal(t, "reg", env.registry.gotName)
	})

	t.Run("errors", func(t *testing.T) {
		env := svbNewEnv(true)
		_, err := env.svc.Get(ctx, enum.PublicResourceType("bogus"), "x")
		require.ErrorContains(t, err, "invalid public resource type")

		env = svbNewEnv(true)
		env.repos.err = errSvbBoom
		_, err = env.svc.Get(ctx, enum.PublicResourceTypeRepo, "1")
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(true)
		env.spaces.err = errSvbBoom
		_, err = env.svc.Get(ctx, enum.PublicResourceTypeSpace, "1")
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(true)
		env.registry.err = errSvbBoom
		_, err = env.svc.Get(ctx, enum.PublicResourceTypeRegistry, "a/b")
		require.ErrorIs(t, err, errSvbBoom)

		env = svbNewEnv(true)
		env.accesses.findErr = errSvbBoom
		_, err = env.svc.Get(ctx, enum.PublicResourceTypeRepo, "1")
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbSet(t *testing.T) {
	ctx := context.Background()

	t.Run("enable", func(t *testing.T) {
		env := svbNewEnv(true)
		require.NoError(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", true))
		require.True(t, env.accesses.created)
		require.Equal(t, int64(4), env.accesses.lastID)
	})

	t.Run("enable not allowed", func(t *testing.T) {
		env := svbNewEnv(false)
		require.ErrorIs(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", true), ErrPublicAccessNotAllowed)
		require.False(t, env.accesses.created)
	})

	t.Run("disable is allowed even when creation is disabled", func(t *testing.T) {
		env := svbNewEnv(false)
		require.NoError(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", false))
		require.True(t, env.accesses.deleted)
	})

	t.Run("duplicate create is ignored", func(t *testing.T) {
		env := svbNewEnv(true)
		env.accesses.createErr = gitnessstore.ErrDuplicate
		require.NoError(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", true))
	})

	t.Run("store failures", func(t *testing.T) {
		env := svbNewEnv(true)
		env.accesses.createErr = errSvbBoom
		require.ErrorIs(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", true), errSvbBoom)

		env = svbNewEnv(true)
		env.accesses.deleteErr = errSvbBoom
		require.ErrorIs(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", false), errSvbBoom)

		env = svbNewEnv(true)
		env.repos.err = errSvbBoom
		require.ErrorIs(t, env.svc.Set(ctx, enum.PublicResourceTypeRepo, "4", true), errSvbBoom)
	})
}

func TestSvbDelete(t *testing.T) {
	ctx := context.Background()

	env := svbNewEnv(true)
	require.NoError(t, env.svc.Delete(ctx, enum.PublicResourceTypeSpace, "9"))
	require.True(t, env.accesses.deleted)

	env = svbNewEnv(true)
	env.accesses.deleteErr = gitnessstore.ErrResourceNotFound
	require.NoError(t, env.svc.Delete(ctx, enum.PublicResourceTypeSpace, "9"))

	env = svbNewEnv(true)
	env.accesses.deleteErr = errSvbBoom
	require.ErrorIs(t, env.svc.Delete(ctx, enum.PublicResourceTypeSpace, "9"), errSvbBoom)
}

func TestSvbIsPublicAccessSupported(t *testing.T) {
	ctx := context.Background()
	for _, enabled := range []bool{true, false} {
		got, err := svbNewEnv(enabled).svc.IsPublicAccessSupported(ctx, enum.PublicResourceTypeRepo, "a/b")
		require.NoError(t, err)
		require.Equal(t, enabled, got)
	}
}

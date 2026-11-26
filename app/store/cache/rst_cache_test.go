// Copyright 2023 Harness, Inc.
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

package cache_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errRstNotFound = errors.New("rst: not found")

// rstNewBus returns an in-memory bus with an explicit send timeout: the default of the in-memory bus is 60 (that
// is 60 nanoseconds as a time.Duration), which makes delivery of published messages non-deterministic.
func rstNewBus() *pubsub.InMemory {
	return pubsub.NewInMemory(pubsub.WithSendTimeout(5 * time.Second))
}

// rstSpaceStore overrides only the SpaceStore methods that the caches use.
type rstSpaceStore struct {
	store.SpaceStore

	findCalls     atomic.Int32
	insensitiveFn func(ref string) (int64, error)
}

func (s *rstSpaceStore) Find(_ context.Context, id int64) (*types.Space, error) {
	s.findCalls.Add(1)
	if id != 7 {
		return nil, errRstNotFound
	}

	return &types.Space{ID: 7, ParentID: 3, Identifier: "child", Path: "parent/child"}, nil
}

func (s *rstSpaceStore) FindByRefCaseInsensitive(_ context.Context, ref string) (int64, error) {
	return s.insensitiveFn(ref)
}

// rstRepoStore overrides only the RepoStore methods that the caches use.
type rstRepoStore struct {
	store.RepoStore

	findCalls   atomic.Int32
	activeCalls atomic.Int32
}

func (s *rstRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	s.findCalls.Add(1)
	if id != 11 {
		return nil, errRstNotFound
	}

	return &types.Repository{ID: 11, ParentID: 7, Identifier: "repo", Path: "parent/child/repo"}, nil
}

func (s *rstRepoStore) FindActiveByUID(_ context.Context, parentID int64, uid string) (*types.Repository, error) {
	s.activeCalls.Add(1)
	if parentID != 7 || uid != "repo" {
		return nil, errRstNotFound
	}

	return &types.Repository{ID: 11, ParentID: 7, Identifier: "repo"}, nil
}

// rstPathStore overrides only the SpacePathStore method that the cache uses.
type rstPathStore struct {
	store.SpacePathStore

	keys []string
}

func (s *rstPathStore) FindByPath(_ context.Context, path string) (*types.SpacePath, error) {
	s.keys = append(s.keys, path)
	if path == "missing" {
		return nil, errRstNotFound
	}

	return &types.SpacePath{Value: path, IsPrimary: true, SpaceID: 5}, nil
}

func TestRstEvictor(t *testing.T) {
	ctx := context.Background()

	t.Run("nil bus is a no-op", func(t *testing.T) {
		evictor := cache.NewEvictor[*types.SpaceCore]("ns", "topic", nil)

		called := false
		evictor.Subscribe(ctx, func(*types.SpaceCore) error {
			called = true
			return nil
		})
		evictor.Evict(ctx, &types.SpaceCore{ID: 1})

		require.False(t, called)
	})

	t.Run("evict reaches subscribers", func(t *testing.T) {
		evictor := cache.NewEvictor[*types.SpaceCore]("ns", "topic", rstNewBus())

		received := make(chan *types.SpaceCore, 1)
		evictor.Subscribe(ctx, func(core *types.SpaceCore) error {
			received <- core
			return nil
		})
		evictor.Evict(ctx, &types.SpaceCore{ID: 4, Identifier: "abc", Path: "x/abc"})

		select {
		case core := <-received:
			require.Equal(t, int64(4), core.ID)
			require.Equal(t, "abc", core.Identifier)
		case <-time.After(5 * time.Second):
			t.Fatal("subscriber did not receive the eviction event")
		}
	})
}

func TestRstSpaceIDCache(t *testing.T) {
	ctx := context.Background()
	spaces := &rstSpaceStore{}
	evictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", rstNewBus())
	c := cache.NewSpaceIDCache(ctx, spaces, evictor, time.Minute)

	core, err := c.Get(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, &types.SpaceCore{ID: 7, ParentID: 3, Identifier: "child", Path: "parent/child"}, core)

	_, err = c.Get(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, int32(1), spaces.findCalls.Load(), "second lookup must be served from the cache")

	_, err = c.Get(ctx, 8)
	require.ErrorIs(t, err, errRstNotFound)

	// a space update evicts every cached space.
	evictor.Evict(ctx, &types.SpaceCore{ID: 7})
	require.Eventually(t, func() bool {
		_, getErr := c.Get(ctx, 7)
		return getErr == nil && spaces.findCalls.Load() >= 3
	}, 5*time.Second, 10*time.Millisecond)

	c.Evict(ctx, 7)
	hits, misses := c.Stats()
	require.Positive(t, hits+misses)
}

func TestRstRepoCaches(t *testing.T) {
	ctx := context.Background()
	repos := &rstRepoStore{}
	spaceEvictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", rstNewBus())
	repoEvictor := cache.NewEvictor[*types.RepositoryCore]("ns", "repo", rstNewBus())

	t.Run("by id", func(t *testing.T) {
		c := cache.NewRepoIDCache(ctx, repos, spaceEvictor, repoEvictor, time.Minute)

		core, err := c.Get(ctx, 11)
		require.NoError(t, err)
		require.Equal(t, int64(11), core.ID)
		require.Equal(t, "repo", core.Identifier)

		_, err = c.Get(ctx, 11)
		require.NoError(t, err)
		require.Equal(t, int32(1), repos.findCalls.Load())

		_, err = c.Get(ctx, 12)
		require.ErrorIs(t, err, errRstNotFound)

		repoEvictor.Evict(ctx, &types.RepositoryCore{ID: 11})
		require.Eventually(t, func() bool {
			_, getErr := c.Get(ctx, 11)
			return getErr == nil && repos.findCalls.Load() >= 3
		}, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("by ref", func(t *testing.T) {
		c := cache.NewRepoRefCache(ctx, repos, spaceEvictor, repoEvictor, time.Minute)
		key := types.RepoCacheKey{SpaceID: 7, RepoIdentifier: "repo"}

		id, err := c.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(11), id)

		_, err = c.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int32(1), repos.activeCalls.Load())

		_, err = c.Get(ctx, types.RepoCacheKey{SpaceID: 7, RepoIdentifier: "other"})
		require.ErrorIs(t, err, errRstNotFound)

		repoEvictor.Evict(ctx, &types.RepositoryCore{ID: 11, ParentID: 7, Identifier: "repo"})
		require.Eventually(t, func() bool {
			_, getErr := c.Get(ctx, key)
			return getErr == nil && repos.activeCalls.Load() >= 3
		}, 5*time.Second, 10*time.Millisecond)

		spaceEvictor.Evict(ctx, &types.SpaceCore{ID: 7})
		before := repos.activeCalls.Load()
		require.Eventually(t, func() bool {
			_, getErr := c.Get(ctx, key)
			return getErr == nil && repos.activeCalls.Load() > before
		}, 5*time.Second, 10*time.Millisecond)
	})
}

func TestRstSpaceCaseInsensitiveCache(t *testing.T) {
	ctx := context.Background()

	var seen []string
	spaces := &rstSpaceStore{insensitiveFn: func(ref string) (int64, error) {
		seen = append(seen, ref)
		if ref == "missing" {
			return 0, errRstNotFound
		}

		return 42, nil
	}}
	evictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", rstNewBus())
	c := cache.NewSpaceCaseInsensitiveCache(ctx, spaces, evictor, time.Minute)

	id, err := c.Get(ctx, "Parent/Child")
	require.NoError(t, err)
	require.Equal(t, int64(42), id)

	// differently cased keys share one cache entry which holds the lowercase key.
	id, err = c.Get(ctx, "PARENT/child")
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
	require.Equal(t, []string{"parent/child"}, seen)

	_, err = c.Get(ctx, "missing")
	require.ErrorIs(t, err, errRstNotFound)

	c.Evict(ctx, "PARENT/CHILD")
	_, err = c.Get(ctx, "parent/child")
	require.NoError(t, err)
	require.Equal(t, []string{"parent/child", "missing", "parent/child"}, seen)

	hits, misses := c.Stats()
	require.Positive(t, hits+misses)

	evictor.Evict(ctx, &types.SpaceCore{ID: 1})
	require.Eventually(t, func() bool {
		before := len(seen)
		_, getErr := c.Get(ctx, "parent/child")
		return getErr == nil && len(seen) > before
	}, 5*time.Second, 10*time.Millisecond)
}

func TestRstSpacePathCache(t *testing.T) {
	ctx := context.Background()
	paths := &rstPathStore{}
	evictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", rstNewBus())
	c := cache.New(ctx, paths, store.ToLowerSpacePathTransformation, evictor, time.Minute)

	path, err := c.Get(ctx, "Parent/Child")
	require.NoError(t, err)
	require.Equal(t, int64(5), path.SpaceID)
	// the key is transformed before the store is consulted.
	require.Equal(t, []string{"parent/child"}, paths.keys)

	_, err = c.Get(ctx, "PARENT/CHILD")
	require.NoError(t, err)
	require.Len(t, paths.keys, 1, "differently cased keys must share the cache entry")

	_, err = c.Get(ctx, "missing")
	require.ErrorIs(t, err, errRstNotFound)

	c.Evict(ctx, "parent/child")
	_, err = c.Get(ctx, "parent/child")
	require.NoError(t, err)

	hits, misses := c.Stats()
	require.Positive(t, hits+misses)
}

type rstPrincipalView struct{ store.PrincipalInfoView }

type rstResourceView struct {
	store.InfraProviderResourceView
}

func TestRstWireProviders(t *testing.T) {
	ctx := context.Background()
	bus := rstNewBus()

	spaceEvictor := cache.ProvideEvictorSpaceCore(bus)
	repoEvictor := cache.ProvideEvictorRepositoryCore(bus)
	spaces := &rstSpaceStore{insensitiveFn: func(string) (int64, error) { return 42, nil }}
	repos := &rstRepoStore{}

	require.NotNil(t, cache.ProvidePrincipalInfoCache(rstPrincipalView{}))
	require.NotNil(t, cache.ProvideInfraProviderResourceCache(rstResourceView{}))

	spaceIDCache := cache.ProvideSpaceIDCache(ctx, spaces, spaceEvictor)
	core, err := spaceIDCache.Get(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, "child", core.Identifier)

	pathCache := cache.ProvideSpacePathCache(ctx, &rstPathStore{}, spaceEvictor, store.ToLowerSpacePathTransformation)
	path, err := pathCache.Get(ctx, "A/B")
	require.NoError(t, err)
	require.Equal(t, "a/b", path.Value)

	insensitive := cache.ProvideSpaceCaseInsensitiveCache(ctx, spaces, spaceEvictor)
	id, err := insensitive.Get(ctx, "Some/Space")
	require.NoError(t, err)
	require.Equal(t, int64(42), id)

	repoIDCache := cache.ProvideRepoIDCache(ctx, repos, spaceEvictor, repoEvictor)
	repoCore, err := repoIDCache.Get(ctx, 11)
	require.NoError(t, err)
	require.Equal(t, "repo", repoCore.Identifier)

	repoRefCache := cache.ProvideRepoRefCache(ctx, repos, spaceEvictor, repoEvictor)
	repoID, err := repoRefCache.Get(ctx, types.RepoCacheKey{SpaceID: 7, RepoIdentifier: "repo"})
	require.NoError(t, err)
	require.Equal(t, int64(11), repoID)
}

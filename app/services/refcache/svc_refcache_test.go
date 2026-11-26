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

package refcache

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errSvcNotFound = errors.New("not found")

type svcCache[K comparable, V any] struct {
	values map[K]V
	gets   int
}

func (c *svcCache[K, V]) Stats() (int64, int64)    { return 0, 0 }
func (c *svcCache[K, V]) Evict(context.Context, K) {}
func (c *svcCache[K, V]) Get(_ context.Context, key K) (V, error) {
	c.gets++
	v, ok := c.values[key]
	if !ok {
		var zero V
		return zero, errSvcNotFound
	}
	return v, nil
}

type svcRepoStore struct {
	store.RepoStore
	deleted    *types.Repository
	err        error
	gotID      int64
	gotSpaceID int64
	gotUID     string
	gotDeleted int64
}

func (s *svcRepoStore) FindDeleted(_ context.Context, id int64, deleted *int64) (*types.Repository, error) {
	s.gotID, s.gotDeleted = id, *deleted
	return s.deleted, s.err
}

func (s *svcRepoStore) FindDeletedByUID(
	_ context.Context, spaceID int64, uid string, deleted int64,
) (*types.Repository, error) {
	s.gotSpaceID, s.gotUID, s.gotDeleted = spaceID, uid, deleted
	return s.deleted, s.err
}

func svcNewRepoFinder() (RepoFinder, *svcRepoStore) {
	repoStore := &svcRepoStore{deleted: &types.Repository{ID: 5}}
	finder := NewRepoFinder(
		repoStore,
		&svcCache[string, *types.SpacePath]{values: map[string]*types.SpacePath{
			"root/space": {Value: "root/space", SpaceID: 10},
		}},
		&svcCache[int64, *types.RepositoryCore]{values: map[int64]*types.RepositoryCore{
			1: {ID: 1, Identifier: "repo"},
		}},
		&svcCache[types.RepoCacheKey, int64]{values: map[types.RepoCacheKey]int64{
			{SpaceID: 10, RepoIdentifier: "repo"}: 1,
		}},
		cache.Evictor[*types.RepositoryCore]{},
	)
	return finder, repoStore
}

func TestSvcRepoFinderFindByID(t *testing.T) {
	finder, _ := svcNewRepoFinder()
	repo, err := finder.FindByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "repo", repo.Identifier)

	_, err = finder.FindByID(context.Background(), 2)
	require.Error(t, err)
}

func TestSvcRepoFinderFindByRef(t *testing.T) {
	finder, _ := svcNewRepoFinder()
	ctx := context.Background()

	tests := []struct {
		name    string
		ref     string
		wantID  int64
		wantErr bool
	}{
		{"numeric id", "1", 1, false},
		{"path", "root/space/repo", 1, false},
		{"unknown numeric id", "99", 0, true},
		{"unknown space", "root/other/repo", 0, true},
		{"unknown repo in space", "root/space/other", 0, true},
		{"path without separator", "repo", 0, true},
		{"zero is treated as a path", "0", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := finder.FindByRef(ctx, tt.ref)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, repo.ID)
		})
	}
}

func TestSvcRepoFinderFindDeletedByRef(t *testing.T) {
	ctx := context.Background()

	t.Run("by id", func(t *testing.T) {
		finder, st := svcNewRepoFinder()
		repo, err := finder.FindDeletedByRef(ctx, "42", 1234)
		require.NoError(t, err)
		assert.Equal(t, int64(5), repo.ID)
		assert.Equal(t, int64(42), st.gotID)
		assert.Equal(t, int64(1234), st.gotDeleted)
	})
	t.Run("by path", func(t *testing.T) {
		finder, st := svcNewRepoFinder()
		_, err := finder.FindDeletedByRef(ctx, "root/space/repo", 1234)
		require.NoError(t, err)
		assert.Equal(t, int64(10), st.gotSpaceID)
		assert.Equal(t, "repo", st.gotUID)
		assert.Equal(t, int64(1234), st.gotDeleted)
	})
	t.Run("store failures", func(t *testing.T) {
		finder, st := svcNewRepoFinder()
		st.err = errors.New("db down")
		_, err := finder.FindDeletedByRef(ctx, "42", 1)
		require.Error(t, err)
		_, err = finder.FindDeletedByRef(ctx, "root/space/repo", 1)
		require.Error(t, err)
	})
	t.Run("invalid path and unknown space", func(t *testing.T) {
		finder, _ := svcNewRepoFinder()
		_, err := finder.FindDeletedByRef(ctx, "repo", 1)
		require.Error(t, err)
		_, err = finder.FindDeletedByRef(ctx, "root/other/repo", 1)
		require.Error(t, err)
	})
}

func svcNewSpaceFinder() SpaceFinder {
	return NewSpaceFinder(
		&svcCache[int64, *types.SpaceCore]{values: map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "root/space", Identifier: "space"},
		}},
		&svcCache[string, *types.SpacePath]{values: map[string]*types.SpacePath{
			"root/space": {Value: "root/space", SpaceID: 10},
		}},
		&svcCache[string, int64]{values: map[string]int64{"ROOT/Space": 10}},
		cache.Evictor[*types.SpaceCore]{},
	)
}

func TestSvcSpaceFinder(t *testing.T) {
	ctx := context.Background()
	finder := svcNewSpaceFinder()

	t.Run("find by id", func(t *testing.T) {
		space, err := finder.FindByID(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, "space", space.Identifier)
		_, err = finder.FindByID(ctx, 11)
		require.Error(t, err)
	})
	t.Run("find by ref", func(t *testing.T) {
		for _, ref := range []string{"10", "root/space"} {
			space, err := finder.FindByRef(ctx, ref)
			require.NoError(t, err)
			assert.Equal(t, int64(10), space.ID)
		}
		_, err := finder.FindByRef(ctx, "11")
		require.Error(t, err)
		_, err = finder.FindByRef(ctx, "root/unknown")
		require.Error(t, err)
	})
	t.Run("find by ref case insensitive", func(t *testing.T) {
		for _, ref := range []string{"10", "ROOT/Space"} {
			space, err := finder.FindByRefCaseInsensitive(ctx, ref)
			require.NoError(t, err)
			assert.Equal(t, int64(10), space.ID)
		}
		_, err := finder.FindByRefCaseInsensitive(ctx, "11")
		require.Error(t, err)
		_, err = finder.FindByRefCaseInsensitive(ctx, "unknown")
		require.Error(t, err)
	})
}

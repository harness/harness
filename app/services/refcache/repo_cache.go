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

package refcache

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

type (
	// RepoIDCache holds Repository objects fetched by their ID.
	RepoIDCache cache.Cache[int64, *types.RepositoryCore]

	// RepoRefCache holds repository IDs fetched by spaceID and repository identifier.
	RepoRefCache cache.Cache[RepoCacheKey, int64]
)

type RepoCacheKey struct {
	spaceID        int64
	repoIdentifier string
}

func NewRepoIDCache(
	repoStore store.RepoStore,
) RepoIDCache {
	return cache.New[int64, *types.RepositoryCore](
		repoIDCacheGetter{repoStore: repoStore},
		cacheDuration)
}

type repoIDCacheGetter struct {
	repoStore store.RepoStore
}

func (c repoIDCacheGetter) Find(ctx context.Context, repoID int64) (*types.RepositoryCore, error) {
	repo, err := c.repoStore.Find(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ID: %w", err)
	}

	return repo.Core(), nil
}

func NewRepoRefCache(
	repoStore store.RepoStore,
) RepoRefCache {
	return cache.New[RepoCacheKey, int64](
		repoCacheGetter{repoStore: repoStore},
		cacheDuration)
}

type repoCacheGetter struct {
	repoStore store.RepoStore
}

func (c repoCacheGetter) Find(ctx context.Context, repoKey RepoCacheKey) (int64, error) {
	repo, err := c.repoStore.FindActiveByUID(ctx, repoKey.spaceID, repoKey.repoIdentifier)
	if err != nil {
		return 0, fmt.Errorf("failed to find repo by space ID and repo uid: %w", err)
	}

	return repo.ID, nil
}

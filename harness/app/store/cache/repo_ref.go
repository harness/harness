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

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

func NewRepoRefCache(
	appCtx context.Context,
	repoStore store.RepoStore,
	evictorSpace Evictor[*types.SpaceCore],
	evictorRepo Evictor[*types.RepositoryCore],
	dur time.Duration,
) store.RepoRefCache {
	c := cache.New[types.RepoCacheKey, int64](repoCacheGetter{repoStore: repoStore}, dur)

	// In case when a space is updated, it's possible that a repo in the cache belongs the space or one of its parents.
	// Rather than to dig through the cache to find if this is actually the case, it's simpler to clear the cache.
	// Update of a space core (space identifier or space path) is a rare operation, so clearing cache is justified.
	evictorSpace.Subscribe(appCtx, func(*types.SpaceCore) error {
		c.EvictAll(appCtx)
		return nil
	})

	evictorRepo.Subscribe(appCtx, func(repoCore *types.RepositoryCore) error {
		c.Evict(appCtx, types.RepoCacheKey{
			SpaceID:        repoCore.ParentID,
			RepoIdentifier: repoCore.Identifier,
		})
		return nil
	})

	return c
}

type repoCacheGetter struct {
	repoStore store.RepoStore
}

func (c repoCacheGetter) Find(ctx context.Context, repoKey types.RepoCacheKey) (int64, error) {
	repo, err := c.repoStore.FindActiveByUID(ctx, repoKey.SpaceID, repoKey.RepoIdentifier)
	if err != nil {
		return 0, fmt.Errorf("failed to find repo by space ID and repo uid: %w", err)
	}

	return repo.ID, nil
}

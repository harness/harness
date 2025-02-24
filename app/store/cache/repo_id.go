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

func NewRepoIDCache(
	appCtx context.Context,
	repoStore store.RepoStore,
	evictorSpace Evictor[*types.SpaceCore],
	evictorRepo Evictor[*types.RepositoryCore],
	dur time.Duration,
) store.RepoIDCache {
	c := cache.New[int64, *types.RepositoryCore](repoIDCacheGetter{repoStore: repoStore}, dur)

	// In case when a space is updated, it's possible that a repo in the cache belongs the space or one of its parents.
	// Rather than to dig through the cache to find if this is actually the case, it's simpler to clear the cache.
	// Update of a space core (space identifier or space path) is a rare operation, so clearing cache is justified.
	evictorSpace.Subscribe(appCtx, func(*types.SpaceCore) error {
		c.EvictAll(appCtx)
		return nil
	})

	evictorRepo.Subscribe(appCtx, func(repoCore *types.RepositoryCore) error {
		c.Evict(appCtx, repoCore.ID)
		return nil
	})

	return c
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

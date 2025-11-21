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
	"strconv"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"
)

type RepoFinder struct {
	repoStore      store.RepoStore
	spacePathCache store.SpacePathCache
	repoIDCache    store.RepoIDCache
	repoRefCache   store.RepoRefCache
	evictor        cache.Evictor[*types.RepositoryCore]
}

func NewRepoFinder(
	repoStore store.RepoStore,
	spacePathCache store.SpacePathCache,
	repoIDCache store.RepoIDCache,
	repoRefCache store.RepoRefCache,
	evictor cache.Evictor[*types.RepositoryCore],
) RepoFinder {
	return RepoFinder{
		repoStore:      repoStore,
		spacePathCache: spacePathCache,
		repoIDCache:    repoIDCache,
		repoRefCache:   repoRefCache,
		evictor:        evictor,
	}
}

func (r RepoFinder) MarkChanged(ctx context.Context, repoCore *types.RepositoryCore) {
	r.evictor.Evict(ctx, repoCore)
}

func (r RepoFinder) Flush(ctx context.Context) {
	r.repoIDCache.EvictAll(ctx)
	r.repoRefCache.EvictAll(ctx)
	r.spacePathCache.EvictAll(ctx)
}

func (r RepoFinder) FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error) {
	return r.repoIDCache.Get(ctx, repoID)
}

func (r RepoFinder) FindByRef(ctx context.Context, repoRef string) (*types.RepositoryCore, error) {
	repoID, err := strconv.ParseInt(repoRef, 10, 64)
	if err != nil || repoID <= 0 {
		spaceRef, repoIdentifier, err := paths.DisectLeaf(repoRef)
		if err != nil {
			return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
		}

		spacePath, err := r.spacePathCache.Get(ctx, spaceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get space from cache: %w", err)
		}

		key := types.RepoCacheKey{SpaceID: spacePath.SpaceID, RepoIdentifier: repoIdentifier}

		repoID, err = r.repoRefCache.Get(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository ID by space ID and repo identifier: %w", err)
		}
	}

	repoCore, err := r.repoIDCache.Get(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by ID: %w", err)
	}

	return repoCore, nil
}

func (r RepoFinder) FindDeletedByRef(ctx context.Context, repoRef string, deleted int64) (*types.Repository, error) {
	repoID, err := strconv.ParseInt(repoRef, 10, 64)
	if err == nil && repoID >= 0 {
		repo, err := r.repoStore.FindDeleted(ctx, repoID, &deleted)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository by ID: %w", err)
		}

		return repo, nil
	}

	spaceRef, repoIdentifier, err := paths.DisectLeaf(repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
	}

	spacePath, err := r.spacePathCache.Get(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get space ID by space ref from cache: %w", err)
	}

	repo, err := r.repoStore.FindDeletedByUID(ctx, spacePath.SpaceID, repoIdentifier, deleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted repository ID by space ID and repo identifier: %w", err)
	}

	return repo, nil
}

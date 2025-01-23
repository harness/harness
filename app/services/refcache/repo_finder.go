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
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
)

type RepoFinder struct {
	repoStore  store.RepoStore
	spaceCache SpaceCache
	repoCache  repoCache
}

func NewRepoFinder(
	repoStore store.RepoStore,
	spaceCache SpaceCache,
) RepoFinder {
	return RepoFinder{
		repoStore:  repoStore,
		spaceCache: spaceCache,
		repoCache:  newRepoCache(repoStore),
	}
}

func (r RepoFinder) FindByRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	if id, err := strconv.ParseInt(repoRef, 10, 64); err == nil && id > 0 {
		repo, err := r.repoStore.Find(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository by ID: %w", err)
		}

		return repo, nil
	}

	spacePath, repoIdentifier, err := paths.DisectLeaf(repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
	}

	space, err := r.spaceCache.Get(ctx, spacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get space from cache: %w", err)
	}

	if id, err := strconv.ParseInt(repoIdentifier, 10, 64); err == nil && id > 0 {
		repo, err := r.repoStore.Find(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository by space path and ID: %w", err)
		}

		if repo.ParentID != space.ID {
			return nil, errors.NotFound("Repository not found")
		}

		return repo, nil
	}

	repo, err := r.repoCache.Get(ctx, repoCacheKey{spaceID: space.ID, repoIdentifier: repoIdentifier})
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by parent space ID and UID: %w", err)
	}

	repo.Version = -1 // destroy the repo version so that it can't be used for update

	return repo, nil
}

func (r RepoFinder) FindDeletedByRef(ctx context.Context, repoRef string, deleted int64) (*types.Repository, error) {
	if id, err := strconv.ParseInt(repoRef, 10, 64); err == nil && id > 0 {
		repo, err := r.repoStore.FindDeleted(ctx, id, &deleted)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository by ID: %w", err)
		}

		return repo, nil
	}

	spacePath, repoIdentifier, err := paths.DisectLeaf(repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
	}

	space, err := r.spaceCache.Get(ctx, spacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get space from cache: %w", err)
	}

	repo, err := r.repoStore.FindDeletedByUID(ctx, space.ID, repoIdentifier, deleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted repository by parent space ID and UID: %w", err)
	}

	repo.Version = -1 // destroy the repo version so that it can't be used for update

	return repo, nil
}

// repoCache holds Repository objects fetched by spaceID and repository identifier.
type repoCache cache.Cache[repoCacheKey, *types.Repository]

type repoCacheKey struct {
	spaceID        int64
	repoIdentifier string
}

func newRepoCache(
	repoStore store.RepoStore,
) repoCache {
	return cache.New[repoCacheKey, *types.Repository](
		repoCacheGetter{repoStore: repoStore},
		1*time.Minute)
}

type repoCacheGetter struct {
	repoStore store.RepoStore
}

func (c repoCacheGetter) Find(ctx context.Context, key repoCacheKey) (*types.Repository, error) {
	return c.repoStore.FindActiveByUID(ctx, key.spaceID, key.repoIdentifier)
}

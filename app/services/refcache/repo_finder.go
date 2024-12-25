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

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

type RepoFinder struct {
	repoStore  store.RepoStore
	spaceCache SpaceCache
}

func NewRepoFinder(
	repoStore store.RepoStore,
	spaceCache SpaceCache,
) RepoFinder {
	return RepoFinder{
		repoStore:  repoStore,
		spaceCache: spaceCache,
	}
}

func (r RepoFinder) FindByRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	spacePath, repoIdentifier, err := paths.DisectLeaf(repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
	}

	space, err := r.spaceCache.Get(ctx, spacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get space from cache: %w", err)
	}

	repo, err := r.repoStore.FindActiveByUID(ctx, space.ID, repoIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by parent space ID and UID: %w", err)
	}

	return repo, nil
}

func (r RepoFinder) FindDeletedByRef(ctx context.Context, repoRef string, deleted int64) (*types.Repository, error) {
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

	return repo, nil
}

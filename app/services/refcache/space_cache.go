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
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

type (
	// SpaceIDCache holds the immutable part of Space objects fetched by space ID.
	SpaceIDCache cache.Cache[int64, *types.SpaceCore]

	// SpaceRefCache holds the space ID fetched by space reference.
	SpaceRefCache cache.Cache[string, int64]
)

func NewSpaceIDCache(
	spaceStore store.SpaceStore,
) SpaceIDCache {
	return cache.New[int64, *types.SpaceCore](spaceIDCacheGetter{spaceStore: spaceStore}, cacheDuration)
}

type spaceIDCacheGetter struct {
	spaceStore store.SpaceStore
}

func (g spaceIDCacheGetter) Find(ctx context.Context, spaceID int64) (*types.SpaceCore, error) {
	space, err := g.spaceStore.Find(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by id: %w", err)
	}

	return space.Core(), nil
}

// spaceCache is a decorator of a Cache required to handle path transformations.
type spaceRefCache struct {
	inner                   SpaceRefCache
	spacePathTransformation store.SpacePathTransformation
}

var _ cache.Cache[string, int64] = spaceRefCache{}

func (c spaceRefCache) Get(ctx context.Context, spaceRef string) (int64, error) {
	segments := paths.Segments(spaceRef)
	uniqueKey := ""
	for i, segment := range segments {
		uniqueKey = paths.Concatenate(uniqueKey, c.spacePathTransformation(segment, i == 0))
	}

	return c.inner.Get(ctx, uniqueKey)
}

func (c spaceRefCache) Stats() (int64, int64) {
	return c.inner.Stats()
}

func (c spaceRefCache) Evict(ctx context.Context, spaceRef string) {
	c.inner.Evict(ctx, spaceRef)
}

func NewSpaceRefCache(
	spacePathStore store.SpacePathStore,
	spacePathTransformation store.SpacePathTransformation,
) SpaceRefCache {
	return &spaceRefCache{
		inner: cache.New[string, int64](
			pathToSpaceCacheGetter{
				spacePathStore: spacePathStore,
			},
			cacheDuration),
		spacePathTransformation: spacePathTransformation,
	}
}

type pathToSpaceCacheGetter struct {
	spacePathStore store.SpacePathStore
}

func (g pathToSpaceCacheGetter) Find(ctx context.Context, spaceRef string) (int64, error) {
	path, err := g.spacePathStore.FindByPath(ctx, spaceRef)
	if err != nil {
		return 0, fmt.Errorf("failed to get space path by space ref: %w", err)
	}

	return path.SpaceID, nil
}

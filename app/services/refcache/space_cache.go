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
	"github.com/harness/gitness/types"
)

// SpaceCache holds Space objects fetched by space reference.
// Whenever a space object needs be fetched by the full path and used only for read operations,
// it should be fetched using this cache.
type SpaceCache cache.Cache[string, *types.Space]

// spaceCache is a decorator of a Cache required to handle path transformations.
type spaceCache struct {
	inner                   SpaceCache
	spacePathTransformation store.SpacePathTransformation
}

func (c spaceCache) Get(ctx context.Context, key string) (*types.Space, error) {
	segments := paths.Segments(key)
	uniqueKey := ""
	for i, segment := range segments {
		uniqueKey = paths.Concatenate(uniqueKey, c.spacePathTransformation(segment, i == 0))
	}

	return c.inner.Get(ctx, uniqueKey)
}

func (c spaceCache) Stats() (int64, int64) {
	return c.inner.Stats()
}

func NewSpaceCache(
	spacePathStore store.SpacePathStore,
	spaceStore store.SpaceStore,
	spacePathTransformation store.SpacePathTransformation,
) SpaceCache {
	return &spaceCache{
		inner: cache.New[string, *types.Space](
			pathToSpaceCacheGetter{
				spacePathStore: spacePathStore,
				spaceStore:     spaceStore,
			},
			1*time.Minute),
		spacePathTransformation: spacePathTransformation,
	}
}

type pathToSpaceCacheGetter struct {
	spacePathStore store.SpacePathStore
	spaceStore     store.SpaceStore
}

func (g pathToSpaceCacheGetter) Find(ctx context.Context, spaceRef string) (*types.Space, error) {
	// ASSUMPTION: digits only is not a valid space path
	id, err := strconv.ParseInt(spaceRef, 10, 64)
	if err != nil {
		var path *types.SpacePath
		path, err = g.spacePathStore.FindByPath(ctx, spaceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get path: %w", err)
		}

		id = path.SpaceID
	}

	space, err := g.spaceStore.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by id: %w", err)
	}

	return space, nil
}

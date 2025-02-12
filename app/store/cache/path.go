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
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

// pathCacheGetter is used to hook a spacePathStore as source of a PathCache.
// IMPORTANT: It assumes that the pathCache already transformed the key.
type pathCacheGetter struct {
	spacePathStore store.SpacePathStore
}

func New(
	pathStore store.SpacePathStore,
	spacePathTransformation store.SpacePathTransformation,
) store.SpacePathCache {
	return &pathCache{
		inner: cache.New[string, *types.SpacePath](
			&pathCacheGetter{
				spacePathStore: pathStore,
			},
			1*time.Minute),
		spacePathTransformation: spacePathTransformation,
	}
}

func (g *pathCacheGetter) Find(ctx context.Context, key string) (*types.SpacePath, error) {
	path, err := g.spacePathStore.FindByPath(ctx, key)
	if err != nil {
		return nil, err
	}

	return path, nil
}

// pathCache is a decorator of a Cache required to handle path transformations.
type pathCache struct {
	inner                   cache.Cache[string, *types.SpacePath]
	spacePathTransformation store.SpacePathTransformation
}

func (c *pathCache) Get(ctx context.Context, key string) (*types.SpacePath, error) {
	// build unique key from provided value
	segments := paths.Segments(key)
	uniqueKey := ""
	for i, segment := range segments {
		uniqueKey = paths.Concatenate(uniqueKey, c.spacePathTransformation(segment, i == 0))
	}

	return c.inner.Get(ctx, uniqueKey)
}

func (c *pathCache) Stats() (int64, int64) {
	return c.inner.Stats()
}

func (c *pathCache) Evict(ctx context.Context, key string) {
	c.inner.Evict(ctx, key)
}

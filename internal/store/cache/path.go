// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"context"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

// pathCacheGetter is used to hook a SpacePathStore as source of a PathCache.
// IMPORTANT: It assumes that the pathCache already transformed the key.
type pathCacheGetter struct {
	spacePathStore store.SpacePathStore
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
		uniqueKey = paths.Concatinate(uniqueKey, c.spacePathTransformation(segment, i == 0))
	}

	return c.inner.Get(ctx, uniqueKey)
}

func (c *pathCache) Stats() (int64, int64) {
	return c.inner.Stats()
}

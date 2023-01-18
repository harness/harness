// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"context"
	"fmt"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

// pathCacheEntry is used to return the proper transformed value as identifier when storing a path in cache.
type pathCacheEntry struct {
	inner       *types.Path
	valueUnique string
}

func (e *pathCacheEntry) Identifier() string {
	return e.valueUnique
}

// pathCacheGetter is used to hook a PathStore as source of a pathCache.
// IMPORTANT: It assumes that the pathCache already transformed the key.
type pathCacheGetter struct {
	pathStore store.PathStore
}

func (g *pathCacheGetter) Find(ctx context.Context, key string) (*pathCacheEntry, error) {
	path, err := g.pathStore.FindValue(ctx, key)
	if err != nil {
		return nil, err
	}

	return &pathCacheEntry{
		inner: path,
		// key is already transformed - pathCache transforms the path before calling the inner cache.
		valueUnique: key,
	}, nil
}

// pathCache is a decorator of a Cache required to handle path transformations.
type pathCache struct {
	inner              cache.Cache[string, *pathCacheEntry]
	pathTransformation store.PathTransformation
}

func (c *pathCache) Get(ctx context.Context, key string) (*types.Path, error) {
	uniqueKey, err := c.pathTransformation(key)
	if err != nil {
		return nil, fmt.Errorf("failed to transform path: %w", err)
	}

	cacheEntry, err := c.inner.Get(ctx, uniqueKey)
	if err != nil {
		return nil, err
	}

	return cacheEntry.inner, nil
}

func (c *pathCache) Stats() (int64, int64) {
	return c.inner.Stats()
}

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
	"strings"
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

// spaceCaseInsensitiveCacheGetter is used to hook a spaceStore as source for case-insensitive lookups.
// IMPORTANT: It assumes that the spaceCaseInsensitiveCache already normalized the key.
type spaceCaseInsensitiveCacheGetter struct {
	spaceStore store.SpaceStore
}

func NewSpaceCaseInsensitiveCache(
	appCtx context.Context,
	spaceStore store.SpaceStore,
	evictor Evictor[*types.SpaceCore],
	dur time.Duration,
) store.SpaceCaseInsensitiveCache {
	innerCache := cache.New[string, int64](&spaceCaseInsensitiveCacheGetter{spaceStore: spaceStore}, dur)

	c := spaceCaseInsensitiveCache{
		inner: innerCache,
	}

	// When a space core is updated (identifier change), clear the cache
	// since case-insensitive lookups depend on space identifiers
	evictor.Subscribe(appCtx, func(*types.SpaceCore) error {
		innerCache.EvictAll(appCtx)
		return nil
	})

	return c
}

func (g *spaceCaseInsensitiveCacheGetter) Find(ctx context.Context, key string) (int64, error) {
	spaceID, err := g.spaceStore.FindByRefCaseInsensitive(ctx, key)
	if err != nil {
		return 0, err
	}

	return spaceID, nil
}

// spaceCaseInsensitiveCache is a wrapper that normalizes keys to lowercase.
type spaceCaseInsensitiveCache struct {
	inner cache.Cache[string, int64]
}

// normalizeKey converts a space reference to a lowercase cache key.
func (c spaceCaseInsensitiveCache) normalizeKey(key string) string {
	segments := paths.Segments(key)
	uniqueKey := ""
	for _, segment := range segments {
		uniqueKey = paths.Concatenate(uniqueKey, strings.ToLower(segment))
	}
	return uniqueKey
}

func (c spaceCaseInsensitiveCache) Get(ctx context.Context, key string) (int64, error) {
	return c.inner.Get(ctx, c.normalizeKey(key))
}

func (c spaceCaseInsensitiveCache) Stats() (int64, int64) {
	return c.inner.Stats()
}

func (c spaceCaseInsensitiveCache) Evict(ctx context.Context, key string) {
	c.inner.Evict(ctx, c.normalizeKey(key))
}

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
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// LRUCache is a bounded, TTL-based LRU cache. This is using hashicorp/golang-lru and
// we might move to custom implementation later.
// It is designed to work with the existing Evictor + PubSub distributed eviction model.
type LRUCache[K comparable, V any] struct {
	inner     *expirable.LRU[K, V]
	getter    Getter[K, V]
	countHit  atomic.Int64
	countMiss atomic.Int64
}

func NewLRU[K comparable, V any](getter Getter[K, V], maxSize int, maxAge time.Duration) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		inner:  expirable.NewLRU[K, V](maxSize, nil, maxAge),
		getter: getter,
	}
}

func (c *LRUCache[K, V]) Stats() (int64, int64) {
	return c.countHit.Load(), c.countMiss.Load()
}

func (c *LRUCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	if item, ok := c.inner.Get(key); ok {
		c.countHit.Add(1)
		return item, nil
	}

	c.countMiss.Add(1)

	item, err := c.getter.Find(ctx, key)
	if err != nil {
		var nothing V
		return nothing, fmt.Errorf("cache: failed to find one: %w", err)
	}

	c.inner.Add(key, item)

	return item, nil
}

func (c *LRUCache[K, V]) Evict(_ context.Context, key K) {
	c.inner.Remove(key)
}

func (c *LRUCache[K, V]) EvictAll(_ context.Context) {
	c.inner.Purge()
}

func (c *LRUCache[K, V]) Len() int {
	return c.inner.Len()
}

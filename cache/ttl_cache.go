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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/constraints"
)

// TTLCache is a generic TTL based cache that stores objects for the specified period.
// The TTLCache has no maximum capacity, so the idea is to store objects for short period.
// The goal of the TTLCache is to reduce database load.
// Every instance of TTLCache has a background routine that purges stale items.
type TTLCache[K comparable, V any] struct {
	mx        sync.RWMutex
	cache     map[K]cacheEntry[V]
	purgeStop chan struct{}
	getter    Getter[K, V]
	maxAge    time.Duration
	countHit  atomic.Int64
	countMiss atomic.Int64
}

// ExtendedTTLCache is an extended version of the TTLCache.
type ExtendedTTLCache[K constraints.Ordered, V Identifiable[K]] struct {
	TTLCache[K, V]
	getter ExtendedGetter[K, V]
}

type cacheEntry[V any] struct {
	added time.Time
	data  V
}

// New creates a new TTLCache instance and a background routine
// that periodically purges stale items.
func New[K comparable, V any](getter Getter[K, V], maxAge time.Duration) *TTLCache[K, V] {
	c := &TTLCache[K, V]{
		cache:     make(map[K]cacheEntry[V]),
		purgeStop: make(chan struct{}),
		getter:    getter,
		maxAge:    maxAge,
	}

	go c.purger()

	return c
}

// NewExtended creates a new TTLCacheExtended instance and a background routine
// that periodically purges stale items.
func NewExtended[K constraints.Ordered, V Identifiable[K]](
	getter ExtendedGetter[K, V],
	maxAge time.Duration,
) *ExtendedTTLCache[K, V] {
	c := &ExtendedTTLCache[K, V]{
		TTLCache: TTLCache[K, V]{
			cache:     make(map[K]cacheEntry[V]),
			purgeStop: make(chan struct{}),
			getter:    getter,
			maxAge:    maxAge,
		},
		getter: getter,
	}

	go c.purger()

	return c
}

// purger periodically evicts stale items from the Cache.
func (c *TTLCache[K, V]) purger() {
	purgeTick := time.NewTicker(time.Minute)
	defer purgeTick.Stop()

	for {
		select {
		case <-c.purgeStop:
			return
		case now := <-purgeTick.C:
			c.mx.Lock()
			for id, v := range c.cache {
				if now.Sub(v.added) >= c.maxAge {
					delete(c.cache, id)
				}
			}
			c.mx.Unlock()
		}
	}
}

// Stop stops the internal purger of stale elements.
func (c *TTLCache[K, V]) Stop() {
	close(c.purgeStop)
}

// Stats returns number of cache hits and misses and can be used to monitor the cache efficiency.
func (c *TTLCache[K, V]) Stats() (int64, int64) {
	return c.countHit.Load(), c.countMiss.Load()
}

func (c *TTLCache[K, V]) fetch(key K, now time.Time) (V, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()

	item, ok := c.cache[key]
	if !ok || now.Sub(item.added) > c.maxAge {
		c.countMiss.Add(1)
		var nothing V
		return nothing, false
	}

	c.countHit.Add(1)

	// we deliberately don't update the `item.added` timestamp for `now` because
	// we want to cache the items only for a short period.

	return item.data, true
}

// Map returns map with all objects requested through the slice of IDs.
func (c *ExtendedTTLCache[K, V]) Map(ctx context.Context, keys []K) (map[K]V, error) {
	m := make(map[K]V)
	now := time.Now()

	keys = Deduplicate(keys)

	// Check what's already available in the cache.

	var idx int
	for idx < len(keys) {
		key := keys[idx]

		item, ok := c.fetch(key, now)
		if !ok {
			idx++
			continue
		}

		// found in cache: Add to the result map and remove the ID from the list.
		m[key] = item
		keys[idx] = keys[len(keys)-1]
		keys = keys[:len(keys)-1]
	}

	if len(keys) == 0 {
		return m, nil
	}

	// Pull entries from the getter that are not in the cache.

	items, err := c.getter.FindMany(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to find many: %w", err)
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	for _, item := range items {
		id := item.Identifier()
		m[id] = item
		c.cache[id] = cacheEntry[V]{
			added: now,
			data:  item,
		}
	}

	return m, nil
}

// Get returns one object by its ID.
func (c *TTLCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	now := time.Now()
	var nothing V

	item, ok := c.fetch(key, now)
	if ok {
		return item, nil
	}

	item, err := c.getter.Find(ctx, key)
	if err != nil {
		return nothing, fmt.Errorf("cache: failed to find one: %w", err)
	}

	c.mx.Lock()
	c.cache[key] = cacheEntry[V]{
		added: now,
		data:  item,
	}
	c.mx.Unlock()

	return item, nil
}

// Deduplicate is a utility function that removes duplicates from slice.
func Deduplicate[V constraints.Ordered](slice []V) []V {
	if len(slice) <= 1 {
		return slice
	}

	sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })

	pointer := 0
	for i := 1; i < len(slice); i++ {
		if slice[pointer] != slice[i] {
			pointer++
			slice[pointer] = slice[i]
		}
	}

	return slice[:pointer+1]
}

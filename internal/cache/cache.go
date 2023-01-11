// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/exp/constraints"
)

// Cache is a generic cache that stores objects for the specified period.
// The Cache has no maximum capacity, so the idea is to store objects for short period.
// The goal of the Cache is to reduce database load.
// Every instance of Cache has a background routine that purges stale items.
type Cache[K constraints.Ordered, V Identifiable[K]] struct {
	mx        sync.RWMutex
	cache     map[K]cacheEntry[K, V]
	purgeStop chan struct{}
	getter    Getter[K, V]
	maxAge    time.Duration
	countHit  int64
	countMiss int64
}

type Identifiable[K constraints.Ordered] interface {
	Identifier() K
}

type Getter[K constraints.Ordered, V Identifiable[K]] interface {
	Find(ctx context.Context, id K) (V, error)
	FindMany(ctx context.Context, ids []K) ([]V, error)
}

type cacheEntry[K constraints.Ordered, V Identifiable[K]] struct {
	added time.Time
	data  V
}

// New creates a new Cache instance and a background routine
// that periodically purges stale items.
func New[K constraints.Ordered, V Identifiable[K]](getter Getter[K, V], maxAge time.Duration) *Cache[K, V] {
	c := &Cache[K, V]{
		cache:     make(map[K]cacheEntry[K, V]),
		purgeStop: make(chan struct{}),
		getter:    getter,
		maxAge:    maxAge,
	}

	go c.purger()

	return c
}

// purger periodically evicts stale items from the Cache.
func (c *Cache[K, V]) purger() {
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
func (c *Cache[K, V]) Stop() {
	close(c.purgeStop)
}

// Stats returns number of cache hits and misses and can be used to monitor the cache efficiency.
func (c *Cache[K, V]) Stats() (int64, int64) {
	return c.countHit, c.countMiss
}

func (c *Cache[K, V]) fetch(id K, now time.Time) (V, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()

	item, ok := c.cache[id]
	if !ok || now.Sub(item.added) > c.maxAge {
		c.countMiss++
		var nothing V
		return nothing, false
	}

	c.countHit++

	// we deliberately don'V update the `item.added` timestamp for `now` because
	// we want to cache the items only for a short period.

	return item.data, true
}

// Map returns map with all objects requested through the slice of IDs.
func (c *Cache[K, V]) Map(ctx context.Context, ids []K) (map[K]V, error) {
	m := make(map[K]V)
	now := time.Now()

	ids = deduplicate(ids)

	// Check what's already available in the cache.

	var idx int
	for idx < len(ids) {
		id := ids[idx]

		item, ok := c.fetch(id, now)
		if !ok {
			idx++
			continue
		}

		// found in cache: Add to the result map and remove the ID from the list.
		m[id] = item
		ids[idx] = ids[len(ids)-1]
		ids = ids[:len(ids)-1]
	}

	if len(ids) == 0 {
		return m, nil
	}

	// Pull entries from the getter that are not in the cache.

	items, err := c.getter.FindMany(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to find many: %w", err)
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	for _, item := range items {
		id := item.Identifier()
		m[id] = item
		c.cache[id] = cacheEntry[K, V]{
			added: now,
			data:  item,
		}
	}

	return m, nil
}

// Get returns one object by its ID.
func (c *Cache[K, V]) Get(ctx context.Context, id K) (V, error) {
	now := time.Now()
	var nothing V

	item, ok := c.fetch(id, now)
	if ok {
		return item, nil
	}

	item, err := c.getter.Find(ctx, id)
	if err != nil {
		return nothing, fmt.Errorf("cache: failed to find one: %w", err)
	}

	c.mx.Lock()
	c.cache[id] = cacheEntry[K, V]{
		added: now,
		data:  item,
	}
	c.mx.Unlock()

	return item, nil
}

// deduplicate is a utility function that removes duplicates from slice.
func deduplicate[V constraints.Ordered](slice []V) []V {
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

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

package sourcemap

import (
	"container/list"
	"sync"
)

// Cache is an in-memory LRU cache of parsed maps bounded by their estimated
// size in bytes. It is safe for concurrent use.
type Cache struct {
	mu       sync.Mutex
	maxBytes int
	bytes    int
	ll       *list.List // front = most recently used
	items    map[string]*list.Element
}

type cacheEntry struct {
	key  string
	m    *Map
	size int
}

// NewCache creates a cache holding at most maxBytes of parsed maps.
func NewCache(maxBytes int) *Cache {
	return &Cache{maxBytes: maxBytes, ll: list.New(), items: map[string]*list.Element{}}
}

// Get returns a cached map and marks it recently used.
func (c *Cache) Get(key string) (*Map, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(e)
	return e.Value.(*cacheEntry).m, true
}

// Put stores a map, evicting least recently used entries until the cache fits.
// A map larger than the whole budget is not stored.
func (c *Cache) Put(key string, m *Map) {
	size := m.Size()
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[key]; ok {
		c.remove(e)
	}
	if size > c.maxBytes {
		return
	}
	c.items[key] = c.ll.PushFront(&cacheEntry{key: key, m: m, size: size})
	c.bytes += size
	for c.bytes > c.maxBytes {
		c.remove(c.ll.Back())
	}
}

func (c *Cache) remove(e *list.Element) {
	ent := e.Value.(*cacheEntry)
	c.ll.Remove(e)
	delete(c.items, ent.key)
	c.bytes -= ent.size
}

// Len returns the number of cached maps.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// Bytes returns the estimated bytes currently held.
func (c *Cache) Bytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bytes
}

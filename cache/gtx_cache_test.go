// Copyright 2026 Harness, Inc.
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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gtxItem struct {
	ID    int
	Value string
}

func (i gtxItem) Identifier() int { return i.ID }

// gtxGetter is a getter that records the lookups.
type gtxGetter struct {
	mu        sync.Mutex
	items     map[int]gtxItem
	err       error
	findCalls []int
	manyCalls [][]int
}

func newGtxGetter(items ...gtxItem) *gtxGetter {
	g := &gtxGetter{items: map[int]gtxItem{}}
	for _, item := range items {
		g.items[item.ID] = item
	}

	return g
}

func (g *gtxGetter) Find(_ context.Context, key int) (gtxItem, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.findCalls = append(g.findCalls, key)
	if g.err != nil {
		return gtxItem{}, g.err
	}
	item, ok := g.items[key]
	if !ok {
		return gtxItem{}, errors.New("gtx not found")
	}

	return item, nil
}

func (g *gtxGetter) FindMany(_ context.Context, keys []int) ([]gtxItem, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.manyCalls = append(g.manyCalls, append([]int(nil), keys...))
	if g.err != nil {
		return nil, g.err
	}

	var out []gtxItem
	for _, key := range keys {
		if item, ok := g.items[key]; ok {
			out = append(out, item)
		}
	}

	return out, nil
}

func (g *gtxGetter) setItem(item gtxItem) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.items[item.ID] = item
}

func TestGtxTTLCache_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("hits and misses", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1, Value: "one"})
		c := New[int, gtxItem](getter, time.Minute)
		t.Cleanup(c.Stop)

		for range 3 {
			got, err := c.Get(ctx, 1)
			require.NoError(t, err)
			assert.Equal(t, "one", got.Value)
		}

		hits, misses := c.Stats()
		assert.Equal(t, int64(2), hits)
		assert.Equal(t, int64(1), misses)
		assert.Equal(t, []int{1}, getter.findCalls)
	})

	t.Run("getter errors are wrapped and not cached", func(t *testing.T) {
		getter := newGtxGetter()
		c := New[int, gtxItem](getter, time.Minute)
		t.Cleanup(c.Stop)

		_, err := c.Get(ctx, 7)
		require.ErrorContains(t, err, "cache: failed to find one: gtx not found")

		getter.setItem(gtxItem{ID: 7, Value: "seven"})
		got, err := c.Get(ctx, 7)
		require.NoError(t, err)
		assert.Equal(t, "seven", got.Value)
		assert.Len(t, getter.findCalls, 2)
	})

	t.Run("entries expire", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1, Value: "old"})
		c := New[int, gtxItem](getter, 30*time.Millisecond)
		t.Cleanup(c.Stop)

		_, err := c.Get(ctx, 1)
		require.NoError(t, err)

		getter.setItem(gtxItem{ID: 1, Value: "new"})
		time.Sleep(80 * time.Millisecond)

		got, err := c.Get(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, "new", got.Value)
		assert.Len(t, getter.findCalls, 2)
	})

	t.Run("evict", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1, Value: "one"}, gtxItem{ID: 2, Value: "two"})
		c := New[int, gtxItem](getter, time.Minute)
		t.Cleanup(c.Stop)

		for _, key := range []int{1, 2} {
			_, err := c.Get(ctx, key)
			require.NoError(t, err)
		}

		c.Evict(ctx, 1)
		_, err := c.Get(ctx, 1)
		require.NoError(t, err)
		_, err = c.Get(ctx, 2)
		require.NoError(t, err)

		assert.Equal(t, []int{1, 2, 1}, getter.findCalls)
	})

	t.Run("evict all", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1, Value: "one"}, gtxItem{ID: 2, Value: "two"})
		c := New[int, gtxItem](getter, time.Minute)
		t.Cleanup(c.Stop)

		for _, key := range []int{1, 2} {
			_, err := c.Get(ctx, key)
			require.NoError(t, err)
		}

		c.EvictAll(ctx)
		for _, key := range []int{1, 2} {
			_, err := c.Get(ctx, key)
			require.NoError(t, err)
		}

		assert.Len(t, getter.findCalls, 4)
	})

	t.Run("stop can be called once the purger is running", func(t *testing.T) {
		c := New[int, gtxItem](newGtxGetter(), time.Minute)
		assert.NotPanics(t, c.Stop)
	})
}

func TestGtxExtendedTTLCache_Map(t *testing.T) {
	ctx := context.Background()

	newCache := func(t *testing.T, getter *gtxGetter) *ExtendedTTLCache[int, gtxItem] {
		t.Helper()

		c := NewExtended[int, gtxItem](getter, time.Minute)
		t.Cleanup(c.Stop)

		return c
	}

	getter := func() *gtxGetter {
		return newGtxGetter(gtxItem{ID: 1, Value: "one"}, gtxItem{ID: 2, Value: "two"}, gtxItem{ID: 3, Value: "three"})
	}

	t.Run("empty key list", func(t *testing.T) {
		g := getter()
		got, err := newCache(t, g).Map(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.Empty(t, g.manyCalls)
	})

	t.Run("everything is loaded and deduplicated", func(t *testing.T) {
		g := getter()
		got, err := newCache(t, g).Map(ctx, []int{3, 1, 3, 2, 1})
		require.NoError(t, err)

		assert.Equal(t, map[int]gtxItem{
			1: {ID: 1, Value: "one"}, 2: {ID: 2, Value: "two"}, 3: {ID: 3, Value: "three"},
		}, got)
		require.Len(t, g.manyCalls, 1)
		assert.Equal(t, []int{1, 2, 3}, g.manyCalls[0])
	})

	t.Run("cached entries are not requested again", func(t *testing.T) {
		g := getter()
		c := newCache(t, g)

		_, err := c.Map(ctx, []int{1})
		require.NoError(t, err)

		got, err := c.Map(ctx, []int{1, 2})
		require.NoError(t, err)
		assert.Len(t, got, 2)

		require.Len(t, g.manyCalls, 2)
		assert.Equal(t, []int{2}, g.manyCalls[1])

		// everything cached: the getter isn't called at all.
		got, err = c.Map(ctx, []int{2, 1})
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Len(t, g.manyCalls, 2)

		hits, misses := c.Stats()
		assert.Equal(t, int64(3), hits)
		assert.Equal(t, int64(2), misses)
	})

	t.Run("unknown keys are omitted", func(t *testing.T) {
		got, err := newCache(t, getter()).Map(ctx, []int{1, 99})
		require.NoError(t, err)
		assert.Equal(t, map[int]gtxItem{1: {ID: 1, Value: "one"}}, got)
	})

	t.Run("map results are served by get", func(t *testing.T) {
		g := getter()
		c := newCache(t, g)

		_, err := c.Map(ctx, []int{1, 2})
		require.NoError(t, err)

		item, err := c.Get(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, "two", item.Value)
		assert.Empty(t, g.findCalls)
	})

	t.Run("getter errors", func(t *testing.T) {
		g := getter()
		g.err = errors.New("gtx getter failure")

		_, err := newCache(t, g).Map(ctx, []int{1})
		require.ErrorContains(t, err, "cache: failed to find many: gtx getter failure")
	})
}

func TestGtxLRUCache(t *testing.T) {
	ctx := context.Background()

	t.Run("hits and misses", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1, Value: "one"})
		c := NewLRU[int, gtxItem](getter, 10, time.Minute)

		for range 3 {
			got, err := c.Get(ctx, 1)
			require.NoError(t, err)
			assert.Equal(t, "one", got.Value)
		}

		hits, misses := c.Stats()
		assert.Equal(t, int64(2), hits)
		assert.Equal(t, int64(1), misses)
		assert.Equal(t, 1, c.Len())
		assert.Equal(t, []int{1}, getter.findCalls)
	})

	t.Run("getter errors are wrapped and not cached", func(t *testing.T) {
		getter := newGtxGetter()
		c := NewLRU[int, gtxItem](getter, 10, time.Minute)

		_, err := c.Get(ctx, 5)
		require.ErrorContains(t, err, "cache: failed to find one: gtx not found")
		assert.Zero(t, c.Len())
	})

	t.Run("least recently used entry is evicted when full", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1}, gtxItem{ID: 2}, gtxItem{ID: 3})
		c := NewLRU[int, gtxItem](getter, 2, time.Minute)

		for _, key := range []int{1, 2} {
			_, err := c.Get(ctx, key)
			require.NoError(t, err)
		}

		// touch 1, so that 2 becomes the least recently used entry.
		_, err := c.Get(ctx, 1)
		require.NoError(t, err)

		_, err = c.Get(ctx, 3)
		require.NoError(t, err)
		assert.Equal(t, 2, c.Len())

		before := len(getter.findCalls)
		_, err = c.Get(ctx, 1)
		require.NoError(t, err)
		assert.Len(t, getter.findCalls, before, "1 must still be cached")

		_, err = c.Get(ctx, 2)
		require.NoError(t, err)
		assert.Len(t, getter.findCalls, before+1, "2 must have been evicted")
	})

	t.Run("evict and evict all", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1}, gtxItem{ID: 2})
		c := NewLRU[int, gtxItem](getter, 10, time.Minute)

		for _, key := range []int{1, 2} {
			_, err := c.Get(ctx, key)
			require.NoError(t, err)
		}

		c.Evict(ctx, 1)
		assert.Equal(t, 1, c.Len())

		c.EvictAll(ctx)
		assert.Zero(t, c.Len())
	})

	t.Run("entries expire", func(t *testing.T) {
		getter := newGtxGetter(gtxItem{ID: 1, Value: "old"})
		c := NewLRU[int, gtxItem](getter, 10, 30*time.Millisecond)

		_, err := c.Get(ctx, 1)
		require.NoError(t, err)

		getter.setItem(gtxItem{ID: 1, Value: "new"})
		time.Sleep(80 * time.Millisecond)

		got, err := c.Get(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, "new", got.Value)
	})
}

func TestGtxCacheInterfaces(t *testing.T) {
	var (
		_ Cache[int, gtxItem]         = (*TTLCache[int, gtxItem])(nil)
		_ Cache[int, gtxItem]         = (*LRUCache[int, gtxItem])(nil)
		_ ExtendedCache[int, gtxItem] = (*ExtendedTTLCache[int, gtxItem])(nil)
	)
}

func TestGtxDeduplicate(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil", in: nil, want: nil},
		{name: "single", in: []string{"a"}, want: []string{"a"}},
		{name: "no duplicates are sorted", in: []string{"c", "a", "b"}, want: []string{"a", "b", "c"}},
		{name: "all equal", in: []string{"a", "a", "a"}, want: []string{"a"}},
		{name: "mixed", in: []string{"b", "a", "b", "c", "a"}, want: []string{"a", "b", "c"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, Deduplicate(test.in))
		})
	}
}

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
	"reflect"
	"testing"
	"time"
)

func TestDeduplicate(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "empty",
			input:    nil,
			expected: nil,
		},
		{
			name:     "one-element",
			input:    []int{1},
			expected: []int{1},
		},
		{
			name:     "one-element-duplicated",
			input:    []int{1, 1},
			expected: []int{1},
		},
		{
			name:     "two-elements",
			input:    []int{2, 1},
			expected: []int{1, 2},
		},
		{
			name:     "three-elements",
			input:    []int{2, 2, 3, 3, 1, 1},
			expected: []int{1, 2, 3},
		},
		{
			name:     "many-elements",
			input:    []int{2, 5, 1, 2, 3, 3, 4, 5, 1, 1},
			expected: []int{1, 2, 3, 4, 5},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input = Deduplicate(test.input)
			if want, got := test.expected, test.input; !reflect.DeepEqual(want, got) {
				t.Errorf("failed - want=%v, got=%v", want, got)
				return
			}
		})
	}
}

type testGetter struct {
	callCount int
}

func (g *testGetter) Find(_ context.Context, key string) (string, error) {
	g.callCount++
	return "value-" + key, nil
}

func TestTTLCache_EvictAll(t *testing.T) {
	t.Run("evictAll clears all entries", func(t *testing.T) {
		getter := &testGetter{}
		cache := New[string, string](getter, 1*time.Hour)
		defer cache.Stop()

		ctx := context.Background()

		// Add some entries to the cache
		_, _ = cache.Get(ctx, "key1")
		_, _ = cache.Get(ctx, "key2")
		_, _ = cache.Get(ctx, "key3")

		// Verify entries are in cache (should have 3 hits on second access)
		initialCallCount := getter.callCount
		_, _ = cache.Get(ctx, "key1")
		_, _ = cache.Get(ctx, "key2")
		_, _ = cache.Get(ctx, "key3")

		if getter.callCount != initialCallCount {
			t.Errorf("expected cache hits, but getter was called %d times", getter.callCount-initialCallCount)
		}

		// EvictAll the cache
		cache.EvictAll(ctx)

		// Verify cache is empty (should call getter again)
		beforeEvictAllCount := getter.callCount
		_, _ = cache.Get(ctx, "key1")
		_, _ = cache.Get(ctx, "key2")
		_, _ = cache.Get(ctx, "key3")

		if getter.callCount != beforeEvictAllCount+3 {
			t.Errorf("expected getter to be called 3 times after evictAll, called %d times",
				getter.callCount-beforeEvictAllCount)
		}
	})

	t.Run("evictAll on empty cache", func(t *testing.T) {
		getter := &testGetter{}
		cache := New[string, string](getter, 1*time.Hour)
		defer cache.Stop()

		ctx := context.Background()

		// EvictAll empty cache should not panic
		cache.EvictAll(ctx)

		// Verify cache still works
		val, err := cache.Get(ctx, "key1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "value-key1" {
			t.Errorf("expected 'value-key1', got '%s'", val)
		}
	})

	t.Run("evictAll multiple times", func(t *testing.T) {
		getter := &testGetter{}
		cache := New[string, string](getter, 1*time.Hour)
		defer cache.Stop()

		ctx := context.Background()

		// Add entries
		_, _ = cache.Get(ctx, "key1")
		_, _ = cache.Get(ctx, "key2")

		initialCallCount := getter.callCount

		// Use EvictAll method
		cache.EvictAll(ctx)

		// Verify cache was cleared
		beforeCount := getter.callCount
		_, _ = cache.Get(ctx, "key1")
		_, _ = cache.Get(ctx, "key2")

		if getter.callCount != beforeCount+2 {
			t.Errorf("expected getter to be called 2 times after EvictAll, called %d times", getter.callCount-beforeCount)
		}

		// Verify it had entries before EvictAll
		if initialCallCount < 2 {
			t.Error("expected cache to have entries before EvictAll")
		}
	})
}

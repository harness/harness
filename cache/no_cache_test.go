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
	"errors"
	"testing"
)

const testValue = "value"

type contextKey string

type mockGetter struct {
	findFunc func(ctx context.Context, key string) (string, error)
}

func (m *mockGetter) Find(ctx context.Context, key string) (string, error) {
	return m.findFunc(ctx, key)
}

func TestNewNoCache(t *testing.T) {
	getter := &mockGetter{
		findFunc: func(ctx context.Context, key string) (string, error) {
			return testValue, nil
		},
	}

	cache := NewNoCache[string, string](getter)
	if cache.getter == nil {
		t.Error("expected getter to be set")
	}
}

func TestNoCache_Stats(t *testing.T) {
	getter := &mockGetter{
		findFunc: func(ctx context.Context, key string) (string, error) {
			return testValue, nil
		},
	}

	cache := NewNoCache[string, string](getter)
	hits, misses := cache.Stats()

	if hits != 0 {
		t.Errorf("expected hits to be 0, got %d", hits)
	}
	if misses != 0 {
		t.Errorf("expected misses to be 0, got %d", misses)
	}
}

func TestNoCache_Get(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		getter := &mockGetter{
			findFunc: func(ctx context.Context, key string) (string, error) {
				if key == "test-key" {
					return "test-value", nil
				}
				return "", errors.New("not found")
			},
		}

		cache := NewNoCache[string, string](getter)
		value, err := cache.Get(context.Background(), "test-key")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != "test-value" {
			t.Errorf("expected value to be 'test-value', got '%s'", value)
		}
	})

	t.Run("get with error", func(t *testing.T) {
		expectedErr := errors.New("getter error")
		getter := &mockGetter{
			findFunc: func(ctx context.Context, key string) (string, error) {
				return "", expectedErr
			},
		}

		cache := NewNoCache[string, string](getter)
		_, err := cache.Get(context.Background(), "test-key")

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error to be %v, got %v", expectedErr, err)
		}
	})

	t.Run("multiple gets call getter each time", func(t *testing.T) {
		callCount := 0
		getter := &mockGetter{
			findFunc: func(ctx context.Context, key string) (string, error) {
				callCount++
				return testValue, nil
			},
		}

		cache := NewNoCache[string, string](getter)

		// Call Get multiple times with the same key
		_, _ = cache.Get(context.Background(), "key")
		_, _ = cache.Get(context.Background(), "key")
		_, _ = cache.Get(context.Background(), "key")

		if callCount != 3 {
			t.Errorf("expected getter to be called 3 times, got %d", callCount)
		}
	})

	t.Run("get with context", func(t *testing.T) {
		var receivedCtx context.Context
		getter := &mockGetter{
			findFunc: func(ctx context.Context, key string) (string, error) {
				receivedCtx = ctx
				return testValue, nil
			},
		}

		cache := NewNoCache[string, string](getter)
		ctx := context.WithValue(context.Background(), contextKey("test-key"), "test-value")
		_, _ = cache.Get(ctx, "key")

		if receivedCtx != ctx {
			t.Error("expected context to be passed to getter")
		}
	})
}

func TestNoCache_Evict(t *testing.T) {
	t.Run("evict does nothing", func(t *testing.T) {
		getter := &mockGetter{
			findFunc: func(ctx context.Context, key string) (string, error) {
				return testValue, nil
			},
		}

		cache := NewNoCache[string, string](getter)

		// Evict should not panic or cause any issues
		cache.Evict(context.Background(), "key")

		// Verify we can still get values after evict
		value, err := cache.Get(context.Background(), "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != "value" {
			t.Errorf("expected value to be 'value', got '%s'", value)
		}
	})
}

func TestNoCache_WithIntegerTypes(t *testing.T) {
	getter := &mockGetter{}

	type intGetter struct{}

	intGetterImpl := intGetter{}

	cache := NewNoCache[int, int](struct {
		Getter[int, int]
	}{
		Getter: struct{ Getter[int, int] }{
			Getter: nil,
		}.Getter,
	})

	// Just verify the cache can be created with different types
	_ = cache
	_ = getter
	_ = intGetterImpl
}

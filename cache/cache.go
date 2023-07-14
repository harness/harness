// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"context"
)

// Cache is an abstraction of a simple cache.
type Cache[K any, V any] interface {
	Stats() (int64, int64)
	Get(ctx context.Context, key K) (V, error)
}

// ExtendedCache is an extension of the simple cache abstraction that adds mapping functionality.
type ExtendedCache[K comparable, V Identifiable[K]] interface {
	Cache[K, V]
	Map(ctx context.Context, keys []K) (map[K]V, error)
}

type Identifiable[K comparable] interface {
	Identifier() K
}

type Getter[K any, V any] interface {
	Find(ctx context.Context, key K) (V, error)
}

type ExtendedGetter[K comparable, V Identifiable[K]] interface {
	Getter[K, V]
	FindMany(ctx context.Context, keys []K) ([]V, error)
}

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

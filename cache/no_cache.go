// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"context"
)

type NoCache[K any, V any] struct {
	getter Getter[K, V]
}

func NewNoCache[K any, V any](getter Getter[K, V]) NoCache[K, V] {
	return NoCache[K, V]{
		getter: getter,
	}
}

func (c NoCache[K, V]) Stats() (int64, int64) {
	return 0, 0
}

func (c NoCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	return c.getter.Find(ctx, key)
}

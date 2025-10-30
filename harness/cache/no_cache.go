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

func (c NoCache[K, V]) Evict(context.Context, K) {}

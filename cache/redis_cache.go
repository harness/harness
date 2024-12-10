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
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

type LogErrFn func(context.Context, error)

type Redis[K any, V any] struct {
	client     redis.UniversalClient
	duration   time.Duration
	getter     Getter[K, V]
	keyEncoder func(K) string
	codec      Codec[V]
	countHit   atomic.Int64
	countMiss  atomic.Int64
	logErrFn   LogErrFn
}

type Encoder[V any] interface {
	Encode(value V) string
}

type Decoder[V any] interface {
	Decode(encoded string) (V, error)
}

type Codec[V any] interface {
	Encoder[V]
	Decoder[V]
}

func NewRedis[K any, V any](
	client redis.UniversalClient,
	getter Getter[K, V],
	keyEncoder func(K) string,
	codec Codec[V],
	duration time.Duration,
	logErrFn LogErrFn,
) *Redis[K, V] {
	return &Redis[K, V]{
		client:     client,
		duration:   duration,
		getter:     getter,
		keyEncoder: keyEncoder,
		codec:      codec,
		logErrFn:   logErrFn,
	}
}

// Stats returns number of cache hits and misses and can be used to monitor the cache efficiency.
func (c *Redis[K, V]) Stats() (int64, int64) {
	return c.countHit.Load(), c.countMiss.Load()
}

// Get implements the cache.Cache interface.
func (c *Redis[K, V]) Get(ctx context.Context, key K) (V, error) {
	var nothing V

	strKey := c.keyEncoder(key)

	raw, err := c.client.Get(ctx, strKey).Result()
	if err == nil {
		value, decErr := c.codec.Decode(raw)
		if decErr == nil {
			c.countHit.Add(1)
			return value, nil
		}
	} else if !errors.Is(err, redis.Nil) && c.logErrFn != nil {
		c.logErrFn(ctx, err)
	}

	c.countMiss.Add(1)

	item, err := c.getter.Find(ctx, key)
	if err != nil {
		return nothing, fmt.Errorf("cache: failed to find one: %w", err)
	}

	err = c.client.Set(ctx, strKey, c.codec.Encode(item), c.duration).Err()
	if err != nil && c.logErrFn != nil {
		c.logErrFn(ctx, err)
	}

	return item, nil
}

// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis[K any, V any] struct {
	client     redis.UniversalClient
	duration   time.Duration
	getter     Getter[K, V]
	keyEncoder func(K) string
	codec      Codec[V]
	countHit   int64
	countMiss  int64
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
) *Redis[K, V] {
	return &Redis[K, V]{
		client:     client,
		duration:   duration,
		getter:     getter,
		keyEncoder: keyEncoder,
		codec:      codec,
		countHit:   0,
		countMiss:  0,
	}
}

// Stats returns number of cache hits and misses and can be used to monitor the cache efficiency.
func (c *Redis[K, V]) Stats() (int64, int64) {
	return c.countHit, c.countMiss
}

// Get implements the cache.Cache interface.
func (c *Redis[K, V]) Get(ctx context.Context, key K) (V, error) {
	var nothing V

	strKey := c.keyEncoder(key)

	raw, err := c.client.Get(ctx, strKey).Result()
	if err == nil {
		c.countHit++
		return c.codec.Decode(raw)
	}
	if err != redis.Nil {
		return nothing, err
	}

	c.countMiss++

	item, err := c.getter.Find(ctx, key)
	if err != nil {
		return nothing, fmt.Errorf("cache: failed to find one: %w", err)
	}

	err = c.client.Set(ctx, strKey, c.codec.Encode(item), c.duration).Err()
	if err != nil {
		return nothing, err
	}

	return item, nil
}

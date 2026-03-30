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
	"golang.org/x/exp/constraints"
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
	switch {
	case err == nil:
		value, decErr := c.codec.Decode(raw)
		if decErr == nil {
			c.countHit.Add(1)
			return value, nil
		}
		c.logIfConfigured(ctx, fmt.Errorf("cache: failed to decode key %q: %w", strKey, decErr))
		c.evictEncodedKey(ctx, strKey)
	case errors.Is(err, redis.Nil):
		// cache miss
	default:
		c.logIfConfigured(ctx, fmt.Errorf("cache: redis GET failed for key %q: %w", strKey, err))
	}

	c.countMiss.Add(1)

	item, err := c.getter.Find(ctx, key)
	if err != nil {
		return nothing, fmt.Errorf("cache: failed to find one: %w", err)
	}

	if err := c.client.Set(ctx, strKey, c.codec.Encode(item), c.duration).Err(); err != nil {
		c.logIfConfigured(ctx, fmt.Errorf("cache: redis SET failed for key %q: %w", strKey, err))
	}

	return item, nil
}

func (c *Redis[K, V]) Evict(ctx context.Context, key K) {
	c.evictEncodedKey(ctx, c.keyEncoder(key))
}

func (c *Redis[K, V]) evictEncodedKey(ctx context.Context, strKey string) {
	if err := c.client.Del(ctx, strKey).Err(); err != nil {
		c.logIfConfigured(ctx, fmt.Errorf("cache: redis DEL failed for key %q: %w", strKey, err))
	}
}

func (c *Redis[K, V]) logIfConfigured(ctx context.Context, err error) {
	if err != nil && c.logErrFn != nil {
		c.logErrFn(ctx, err)
	}
}

// ExtendedRedis is a Redis-backed cache that supports batch Map() operations.
type ExtendedRedis[K constraints.Ordered, V Identifiable[K]] struct {
	Redis[K, V]
	// getter is stored separately because the embedded Redis.getter field
	// only knows the Getter[K,V] interface. We need the ExtendedGetter[K,V]
	// interface here to access FindMany() for batch operations in Map().
	getter ExtendedGetter[K, V]
}

func NewExtendedRedis[K constraints.Ordered, V Identifiable[K]](
	client redis.UniversalClient,
	getter ExtendedGetter[K, V],
	keyEncoder func(K) string,
	codec Codec[V],
	duration time.Duration,
	logErrFn LogErrFn,
) *ExtendedRedis[K, V] {
	return &ExtendedRedis[K, V]{
		Redis: Redis[K, V]{
			client:     client,
			duration:   duration,
			getter:     getter,
			keyEncoder: keyEncoder,
			codec:      codec,
			logErrFn:   logErrFn,
		},
		getter: getter,
	}
}

// Map returns the found values for the requested keys.
// Missing keys are omitted from the returned map.
func (c *ExtendedRedis[K, V]) Map(ctx context.Context, keys []K) (map[K]V, error) {
	result := make(map[K]V)

	keys = Deduplicate(keys)
	if len(keys) == 0 {
		return result, nil
	}

	encodedKeys := make([]string, len(keys))
	for i, key := range keys {
		encodedKeys[i] = c.keyEncoder(key)
	}

	// Use pipelined GET instead of MGET so arbitrary keys do not fail with
	// CROSSSLOT when the UniversalClient is backed by Redis Cluster.
	pipe := c.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, strKey := range encodedKeys {
		cmds[i] = pipe.Get(ctx, strKey)
	}
	_, execErr := pipe.Exec(ctx)
	if execErr != nil && !errors.Is(execErr, redis.Nil) {
		c.logIfConfigured(ctx, fmt.Errorf("cache: redis pipeline GET failed: %w", execErr))
	}

	missedKeys := make([]K, 0, len(keys))
	missedSet := make(map[K]struct{}, len(keys))

	for i, key := range keys {
		raw, err := cmds[i].Result()
		switch {
		case err == nil:
			value, decErr := c.codec.Decode(raw)
			if decErr == nil {
				c.countHit.Add(1)
				result[key] = value
				continue
			}
			c.logIfConfigured(ctx, fmt.Errorf("cache: failed to decode key %q: %w", encodedKeys[i], decErr))
			c.evictEncodedKey(ctx, encodedKeys[i])
		case errors.Is(err, redis.Nil):
			// cache miss
		default:
			c.logIfConfigured(ctx, fmt.Errorf("cache: redis GET failed for key %q: %w", encodedKeys[i], err))
		}
		c.countMiss.Add(1)
		missedKeys = append(missedKeys, key)
		missedSet[key] = struct{}{}
	}

	if len(missedKeys) == 0 {
		return result, nil
	}

	// Batch fetch missed keys from source.
	items, err := c.getter.FindMany(ctx, missedKeys)
	if err != nil {
		return nil, fmt.Errorf("cache: failed to find many: %w", err)
	}

	// Store fetched items in Redis using pipeline.
	writePipe := c.client.Pipeline()
	for _, item := range items {
		id := item.Identifier()
		if _, ok := missedSet[id]; !ok {
			c.logIfConfigured(ctx, fmt.Errorf("cache: FindMany returned unexpected key %v", id))
			continue
		}
		result[id] = item
		writePipe.Set(ctx, c.keyEncoder(id), c.codec.Encode(item), c.duration)
	}
	if _, err := writePipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		c.logIfConfigured(ctx, fmt.Errorf("cache: redis pipeline SET failed: %w", err))
	}

	return result, nil
}

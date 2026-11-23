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

// Package idempotency provides stores that remember which operations were already performed.
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStore is a Store shared between instances.
type RedisStore struct {
	client redis.UniversalClient
}

// Values of a key, telling a running operation from a finished one.
const (
	redisValueInFlight = "0"
	redisValueDone     = "1"
)

func NewRedis(client redis.UniversalClient) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) Claim(ctx context.Context, key string, ttl time.Duration) (State, error) {
	if ttl <= 0 {
		return StateClaimed, fmt.Errorf("failed to claim key %q: %w", key, ErrInvalidTTL)
	}

	claimed, err := s.client.SetNX(ctx, key, redisValueInFlight, ttl).Result()
	if err != nil {
		return StateClaimed, fmt.Errorf("failed to claim key %q: %w", key, err)
	}
	if claimed {
		return StateClaimed, nil
	}

	value, err := s.client.Get(ctx, key).Result()
	switch {
	// The key expired between the two calls, so the caller retries.
	case errors.Is(err, redis.Nil):
		return StateInFlight, nil
	case err != nil:
		return StateClaimed, fmt.Errorf("failed to read state of key %q: %w", key, err)
	case value == redisValueDone:
		return StateDone, nil
	default:
		return StateInFlight, nil
	}
}

func (s *RedisStore) Commit(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("failed to commit key %q: %w", key, ErrInvalidTTL)
	}

	// Set instead of Expire, which is a no-op once the claim expired.
	err := s.client.Set(ctx, key, redisValueDone, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to commit key %q: %w", key, err)
	}

	return nil
}

func (s *RedisStore) Release(ctx context.Context, key string) error {
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to release key %q: %w", key, err)
	}

	return nil
}

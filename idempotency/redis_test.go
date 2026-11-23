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
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// Runs against a real redis, set GITNESS_TEST_REDIS_ADDR (e.g. localhost:6379).
func TestRedisStore(t *testing.T) {
	addr := os.Getenv("GITNESS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("GITNESS_TEST_REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	store := NewRedis(client)
	key := "notification:sent:test:" + t.Name() + time.Now().String()
	t.Cleanup(func() { _ = client.Del(ctx, key).Err() })

	state, err := store.Claim(ctx, key, inFlight)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)

	state, err = store.Claim(ctx, key, inFlight)
	require.NoError(t, err)
	require.Equal(t, StateInFlight, state)

	require.NoError(t, store.Commit(ctx, key, time.Hour))
	ttl, err := client.TTL(ctx, key).Result()
	require.NoError(t, err)
	require.Greater(t, ttl, inFlight)

	state, err = store.Claim(ctx, key, inFlight)
	require.NoError(t, err)
	require.Equal(t, StateDone, state)

	require.NoError(t, store.Release(ctx, key))
	state, err = store.Claim(ctx, key, inFlight)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)

	// Commit after the claim expired still records it as done.
	require.NoError(t, client.Del(ctx, key).Err())
	require.NoError(t, store.Commit(ctx, key, time.Hour))
	state, err = store.Claim(ctx, key, inFlight)
	require.NoError(t, err)
	require.Equal(t, StateDone, state)
}

func TestRedisStore_NonPositiveTTLIsRejected(t *testing.T) {
	addr := os.Getenv("GITNESS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("GITNESS_TEST_REDIS_ADDR is not set")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	store := NewRedis(client)
	key := "notification:sent:test:" + t.Name() + time.Now().String()
	t.Cleanup(func() { _ = client.Del(ctx, key).Err() })

	_, err := store.Claim(ctx, key, 0)
	require.ErrorIs(t, err, ErrInvalidTTL)
	require.ErrorIs(t, store.Commit(ctx, key, 0), ErrInvalidTTL)

	exists, err := client.Exists(ctx, key).Result()
	require.NoError(t, err)
	require.Zero(t, exists)
}

// fakeRedis implements the commands RedisStore uses, so its logic is covered without a redis server.
type fakeRedis struct {
	redis.UniversalClient

	values map[string]string
	ttls   map[string]time.Duration
	err    error

	// expireOnGet drops the key right before Get, like a key expiring between SetNX and Get.
	expireOnGet bool
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{values: map[string]string{}, ttls: map[string]time.Duration{}}
}

func (f *fakeRedis) SetNX(
	ctx context.Context,
	key string,
	value any,
	expiration time.Duration,
) *redis.BoolCmd {
	if f.err != nil {
		return redis.NewBoolResult(false, f.err)
	}
	if _, ok := f.values[key]; ok {
		return redis.NewBoolResult(false, nil)
	}
	f.values[key] = value.(string) //nolint:errcheck
	f.ttls[key] = expiration

	return redis.NewBoolResult(true, nil)
}

func (f *fakeRedis) Get(_ context.Context, key string) *redis.StringCmd {
	if f.expireOnGet {
		delete(f.values, key)
	}
	if f.err != nil {
		return redis.NewStringResult("", f.err)
	}
	value, ok := f.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}

	return redis.NewStringResult(value, nil)
}

func (f *fakeRedis) Set(_ context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	if f.err != nil {
		return redis.NewStatusResult("", f.err)
	}
	f.values[key] = value.(string) //nolint:errcheck
	f.ttls[key] = expiration

	return redis.NewStatusResult("OK", nil)
}

func (f *fakeRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	if f.err != nil {
		return redis.NewIntResult(0, f.err)
	}
	for _, key := range keys {
		delete(f.values, key)
		delete(f.ttls, key)
	}

	return redis.NewIntResult(int64(len(keys)), nil)
}

func TestRedisStore_Lifecycle(t *testing.T) {
	t.Parallel()

	client := newFakeRedis()
	store := NewRedis(client)
	ctx := context.Background()

	state, err := store.Claim(ctx, "k", inFlight)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
	require.Equal(t, inFlight, client.ttls["k"])

	state, err = store.Claim(ctx, "k", inFlight)
	require.NoError(t, err)
	require.Equal(t, StateInFlight, state)

	require.NoError(t, store.Commit(ctx, "k", time.Hour))
	require.Equal(t, time.Hour, client.ttls["k"])

	state, err = store.Claim(ctx, "k", inFlight)
	require.NoError(t, err)
	require.Equal(t, StateDone, state)

	require.NoError(t, store.Release(ctx, "k"))
	state, err = store.Claim(ctx, "k", inFlight)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
}

func TestRedisStore_CommitWorksWithoutAClaim(t *testing.T) {
	t.Parallel()

	store := NewRedis(newFakeRedis())
	ctx := context.Background()

	require.NoError(t, store.Commit(ctx, "k", time.Hour))

	state, err := store.Claim(ctx, "k", inFlight)
	require.NoError(t, err)
	require.Equal(t, StateDone, state)
}

// A key that expires between SetNX and Get must be retried, not reported as done.
func TestRedisStore_ClaimExpiringBetweenCallsIsInFlight(t *testing.T) {
	t.Parallel()

	client := newFakeRedis()
	client.values["k"] = redisValueInFlight
	client.expireOnGet = true

	state, err := NewRedis(client).Claim(context.Background(), "k", inFlight)
	require.NoError(t, err)
	require.Equal(t, StateInFlight, state)
}

func TestRedisStore_NonPositiveTTLIsRejectedWithoutTouchingRedis(t *testing.T) {
	t.Parallel()

	client := newFakeRedis()
	store := NewRedis(client)
	ctx := context.Background()

	_, err := store.Claim(ctx, "k", 0)
	require.ErrorIs(t, err, ErrInvalidTTL)
	require.ErrorIs(t, store.Commit(ctx, "k", -time.Second), ErrInvalidTTL)
	require.Empty(t, client.values)
}

func TestRedisStore_ErrorsArePropagated(t *testing.T) {
	t.Parallel()

	errRedis := errors.New("redis down")
	client := newFakeRedis()
	client.err = errRedis
	store := NewRedis(client)
	ctx := context.Background()

	_, err := store.Claim(ctx, "k", inFlight)
	require.ErrorIs(t, err, errRedis)
	require.ErrorIs(t, store.Commit(ctx, "k", time.Hour), errRedis)
	require.ErrorIs(t, store.Release(ctx, "k"), errRedis)
}

func TestRedisStore_ClaimFailsIfStateCannotBeRead(t *testing.T) {
	t.Parallel()

	errRedis := errors.New("read failed")
	client := newFakeRedis()
	client.values["k"] = redisValueInFlight

	store := NewRedis(&getFailingRedis{fakeRedis: client, err: errRedis})

	_, err := store.Claim(context.Background(), "k", inFlight)
	require.ErrorIs(t, err, errRedis)
}

// getFailingRedis fails only Get, so the claim itself is rejected as taken and the state read is what fails.
type getFailingRedis struct {
	*fakeRedis
	err error
}

func (g *getFailingRedis) Get(context.Context, string) *redis.StringCmd {
	return redis.NewStringResult("", g.err)
}

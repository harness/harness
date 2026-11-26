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

package lock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGtxOptions(t *testing.T) {
	genErr := errors.New("gtx gen failure")

	tests := []struct {
		name string
		opt  Option
		want func(t *testing.T, c Config)
	}{
		{
			name: "namespace", opt: WithNamespace("ns"),
			want: func(t *testing.T, c Config) { assert.Equal(t, "ns", c.Namespace) },
		},
		{
			name: "expiry", opt: WithExpiry(time.Minute),
			want: func(t *testing.T, c Config) { assert.Equal(t, time.Minute, c.Expiry) },
		},
		{name: "tries", opt: WithTries(7), want: func(t *testing.T, c Config) { assert.Equal(t, 7, c.Tries) }},
		{
			name: "retry delay", opt: WithRetryDelay(3 * time.Second),
			want: func(t *testing.T, c Config) {
				require.NotNil(t, c.DelayFunc)
				assert.Equal(t, 3*time.Second, c.DelayFunc(1))
				assert.Equal(t, 3*time.Second, c.DelayFunc(100))
			},
		},
		{
			name: "retry delay func",
			opt:  WithRetryDelayFunc(func(tries int) time.Duration { return time.Duration(tries) * time.Second }),
			want: func(t *testing.T, c Config) { assert.Equal(t, 4*time.Second, c.DelayFunc(4)) },
		},
		{
			name: "drift factor", opt: WithDriftFactor(0.5),
			want: func(t *testing.T, c Config) { assert.InEpsilon(t, 0.5, c.DriftFactor, 0) },
		},
		{
			name: "timeout factor", opt: WithTimeoutFactor(2),
			want: func(t *testing.T, c Config) { assert.InEpsilon(t, 2.0, c.TimeoutFactor, 0) },
		},
		{
			name: "gen value func", opt: WithGenValueFunc(func() (string, error) { return "", genErr }),
			want: func(t *testing.T, c Config) {
				_, err := c.GenValueFunc()
				require.ErrorIs(t, err, genErr)
			},
		},
		{name: "value", opt: WithValue("token"), want: func(t *testing.T, c Config) { assert.Equal(t, "token", c.Value) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var c Config
			test.opt.Apply(&c)
			test.want(t, c)
		})
	}
}

func TestGtxError(t *testing.T) {
	assert.Equal(t, "lock already held on key a:b:c",
		NewError(ErrorKindLockHeld, "a:b:c", nil).Error())
	assert.Equal(t, "lock provider error on key k with err: boom",
		NewError(ErrorKindProviderError, "k", errors.New("boom")).Error())
}

func gtxNewManager() *InMemory {
	return NewInMemory(Config{
		App:        "gtx",
		Namespace:  "default-ns",
		Expiry:     5 * time.Second,
		Tries:      3,
		RetryDelay: 10 * time.Millisecond,
	})
}

func TestGtxInMemory_NewMutex(t *testing.T) {
	t.Run("key is built from app, namespace and key", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k1")
		require.NoError(t, err)
		assert.Equal(t, "gtx:default-ns:k1", mx.Key())

		ns, key := SplitKey(mx.Key())
		assert.Equal(t, "default-ns", ns)
		assert.Equal(t, "k1", key)
	})

	t.Run("namespace option overrides the default", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k1", WithNamespace("custom"))
		require.NoError(t, err)
		assert.Equal(t, "gtx:custom:k1", mx.Key())
	})

	t.Run("generated token failure", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k1", WithGenValueFunc(func() (string, error) {
			return "", errors.New("gtx gen failure")
		}))
		require.Nil(t, mx)

		var lockErr *Error
		require.ErrorAs(t, err, &lockErr)
		assert.Equal(t, ErrorKindGenerateTokenFailed, lockErr.Kind)
		assert.Equal(t, "gtx:default-ns:k1", lockErr.Key)
	})

	t.Run("custom token", func(t *testing.T) {
		manager := gtxNewManager()
		ctx := context.Background()

		mx, err := manager.NewMutex("k1", WithGenValueFunc(func() (string, error) { return "generated", nil }))
		require.NoError(t, err)
		require.NoError(t, mx.Lock(ctx))
		assert.Equal(t, "generated", manager.keys["gtx:default-ns:k1"].token)
		require.NoError(t, mx.Unlock(ctx))

		// an explicit value takes precedence over the generator.
		mx, err = manager.NewMutex("k1",
			WithValue("explicit"), WithGenValueFunc(func() (string, error) { return "generated", nil }))
		require.NoError(t, err)
		require.NoError(t, mx.Lock(ctx))
		assert.Equal(t, "explicit", manager.keys["gtx:default-ns:k1"].token)
	})

	t.Run("random tokens are unique", func(t *testing.T) {
		manager := gtxNewManager()
		ctx := context.Background()

		tokens := map[string]struct{}{}
		for i := range 5 {
			mx, err := manager.NewMutex("k" + string(rune('a'+i)))
			require.NoError(t, err)
			require.NoError(t, mx.Lock(ctx))
		}
		for _, entry := range manager.keys {
			tokens[entry.token] = struct{}{}
			assert.NotEmpty(t, entry.token)
		}
		assert.Len(t, tokens, 5)
	})

	t.Run("default delay func uses the configured retry delay", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k1")
		require.NoError(t, err)

		m, ok := mx.(*inMemMutex)
		require.True(t, ok)
		assert.Equal(t, 10*time.Millisecond, m.delayFunc(1))
		assert.Equal(t, 5*time.Second, m.waitTime)
		assert.Equal(t, 3, m.tries)
	})

	t.Run("timeout factor scales the wait time", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k1", WithTimeoutFactor(0.5), WithExpiry(10*time.Second))
		require.NoError(t, err)

		m, ok := mx.(*inMemMutex)
		require.True(t, ok)
		assert.Equal(t, 5*time.Second, m.waitTime)
	})
}

func TestGtxInMemoryMutex_LockAndUnlock(t *testing.T) {
	ctx := context.Background()

	lockErrKind := func(t *testing.T, err error) ErrorKind {
		t.Helper()

		var lockErr *Error
		require.ErrorAs(t, err, &lockErr)

		return lockErr.Kind
	}

	t.Run("lock twice with the same mutex", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k")
		require.NoError(t, err)

		require.NoError(t, mx.Lock(ctx))
		assert.Equal(t, ErrorKindLockHeld, lockErrKind(t, mx.Lock(ctx)))
	})

	t.Run("unlock without lock", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k")
		require.NoError(t, err)

		assert.Equal(t, ErrorKindLockNotHeld, lockErrKind(t, mx.Unlock(ctx)))
	})

	t.Run("unlock twice", func(t *testing.T) {
		mx, err := gtxNewManager().NewMutex("k")
		require.NoError(t, err)

		require.NoError(t, mx.Lock(ctx))
		require.NoError(t, mx.Unlock(ctx))
		assert.Equal(t, ErrorKindLockNotHeld, lockErrKind(t, mx.Unlock(ctx)))
	})

	t.Run("lock can be taken again after unlock", func(t *testing.T) {
		manager := gtxNewManager()

		first, err := manager.NewMutex("k")
		require.NoError(t, err)
		second, err := manager.NewMutex("k", WithTries(1))
		require.NoError(t, err)

		require.NoError(t, first.Lock(ctx))
		assert.Equal(t, ErrorKindMaxRetriesExceeded, lockErrKind(t, second.Lock(ctx)))

		require.NoError(t, first.Unlock(ctx))
		require.NoError(t, second.Lock(ctx))
	})

	t.Run("other mutex can't release the lock", func(t *testing.T) {
		manager := gtxNewManager()

		holder, err := manager.NewMutex("k")
		require.NoError(t, err)
		require.NoError(t, holder.Lock(ctx))

		// the other mutex has the same key, but it never acquired the lock.
		other, err := manager.NewMutex("k")
		require.NoError(t, err)
		assert.Equal(t, ErrorKindLockNotHeld, lockErrKind(t, other.Unlock(ctx)))

		require.NoError(t, holder.Unlock(ctx))
	})

	t.Run("expired locks can be taken over", func(t *testing.T) {
		manager := gtxNewManager()

		first, err := manager.NewMutex("k", WithExpiry(20*time.Millisecond))
		require.NoError(t, err)
		require.NoError(t, first.Lock(ctx))

		time.Sleep(50 * time.Millisecond)

		second, err := manager.NewMutex("k", WithTries(1))
		require.NoError(t, err)
		require.NoError(t, second.Lock(ctx))
	})

	t.Run("canceled context while waiting", func(t *testing.T) {
		manager := gtxNewManager()

		holder, err := manager.NewMutex("k")
		require.NoError(t, err)
		require.NoError(t, holder.Lock(ctx))

		waiter, err := manager.NewMutex("k", WithTries(5), WithRetryDelay(time.Hour))
		require.NoError(t, err)

		cctx, cancel := context.WithCancel(ctx)
		cancel()

		assert.Equal(t, ErrorKindContext, lockErrKind(t, waiter.Lock(cctx)))
	})

	t.Run("timeout while waiting", func(t *testing.T) {
		manager := gtxNewManager()

		holder, err := manager.NewMutex("k")
		require.NoError(t, err)
		require.NoError(t, holder.Lock(ctx))

		waiter, err := manager.NewMutex("k",
			WithTries(5), WithRetryDelay(time.Hour), WithExpiry(20*time.Millisecond))
		require.NoError(t, err)

		assert.Equal(t, ErrorKindCannotLock, lockErrKind(t, waiter.Lock(ctx)))
	})

	t.Run("lock is acquired while waiting", func(t *testing.T) {
		manager := gtxNewManager()

		holder, err := manager.NewMutex("k", WithExpiry(60*time.Millisecond))
		require.NoError(t, err)
		require.NoError(t, holder.Lock(ctx))

		waiter, err := manager.NewMutex("k", WithTries(20), WithRetryDelay(10*time.Millisecond), WithExpiry(5*time.Second))
		require.NoError(t, err)
		require.NoError(t, waiter.Lock(ctx))
	})
}

func TestGtxProvideMutexManager(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() })

	assert.IsType(t, &InMemory{}, ProvideMutexManager(Config{Provider: MemoryProvider}, nil))
	assert.IsType(t, &Redis{}, ProvideMutexManager(Config{Provider: RedisProvider}, client))
	assert.Nil(t, ProvideMutexManager(Config{Provider: "gtx-unknown"}, nil))
	assert.Nil(t, ProvideMutexManager(Config{}, nil))
}

func TestGtxRedis_NewMutex(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })

	manager := NewRedis(Config{
		App:        "gtx",
		Namespace:  "default-ns",
		Expiry:     time.Second,
		Tries:      1,
		RetryDelay: time.Millisecond,
	}, client)

	t.Run("key", func(t *testing.T) {
		mx, err := manager.NewMutex("k1")
		require.NoError(t, err)
		assert.Equal(t, "gtx:default-ns:k1", mx.Key())

		mx, err = manager.NewMutex("k1", WithNamespace("custom"))
		require.NoError(t, err)
		assert.Equal(t, "gtx:custom:k1", mx.Key())
	})

	t.Run("optional settings are accepted", func(t *testing.T) {
		mx, err := manager.NewMutex("k1",
			WithRetryDelayFunc(func(int) time.Duration { return time.Millisecond }),
			WithGenValueFunc(func() (string, error) { return "token", nil }),
			WithTimeoutFactor(0.5), WithDriftFactor(0.1), WithTries(2), WithExpiry(time.Second))
		require.NoError(t, err)
		assert.Equal(t, "gtx:default-ns:k1", mx.Key())
	})

	t.Run("unreachable server", func(t *testing.T) {
		mx, err := manager.NewMutex("k1")
		require.NoError(t, err)

		err = mx.Lock(context.Background())
		var lockErr *Error
		require.ErrorAs(t, err, &lockErr)
		assert.Equal(t, "gtx:default-ns:k1", lockErr.Key)
		assert.NotEmpty(t, lockErr.Kind)

		err = mx.Unlock(context.Background())
		require.ErrorAs(t, err, &lockErr)
	})
}

func TestGtxTranslateRedisErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorKind
	}{
		{name: "failed to lock", err: redsync.ErrFailed, want: ErrorKindCannotLock},
		{name: "wrapped failure", err: errors.Join(errors.New("ctx"), redsync.ErrFailed), want: ErrorKindCannotLock},
		{name: "extend failed", err: redsync.ErrExtendFailed, want: ErrorKindProviderError},
		{name: "redis error", err: &redsync.RedisError{Node: 1, Err: errors.New("boom")}, want: ErrorKindProviderError},
		{name: "taken", err: &redsync.ErrTaken{}, want: ErrorKindLockHeld},
		{name: "node taken", err: &redsync.ErrNodeTaken{}, want: ErrorKindLockHeld},
		{name: "unknown", err: errors.New("other"), want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := translateRedisErr(test.err, "key")

			var lockErr *Error
			require.ErrorAs(t, err, &lockErr)
			assert.Equal(t, test.want, lockErr.Kind)
			assert.Equal(t, "key", lockErr.Key)
			require.ErrorIs(t, lockErr.Err, test.err)
		})
	}
}

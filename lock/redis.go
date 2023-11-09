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

package lock

import (
	"context"
	"errors"

	redislib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

// Redis wrapper for redsync.
type Redis struct {
	config Config
	rs     *redsync.Redsync
}

// NewRedis create an instance of redisync to be used to obtain a mutual exclusion
// lock.
func NewRedis(config Config, client redislib.UniversalClient) *Redis {
	pool := goredis.NewPool(client)
	return &Redis{
		config: config,
		rs:     redsync.New(pool),
	}
}

// Acquire new lock.
func (r *Redis) NewMutex(key string, options ...Option) (Mutex, error) {
	// copy default values
	config := r.config
	// customize config
	for _, opt := range options {
		opt.Apply(&config)
	}

	// convert to redis helper functions
	args := make([]redsync.Option, 0, 8)
	args = append(args,
		redsync.WithExpiry(config.Expiry),
		redsync.WithTimeoutFactor(config.TimeoutFactor),
		redsync.WithTries(config.Tries),
		redsync.WithRetryDelay(config.RetryDelay),
		redsync.WithDriftFactor(config.DriftFactor),
	)

	if config.DelayFunc != nil {
		args = append(args, redsync.WithRetryDelayFunc(redsync.DelayFunc(config.DelayFunc)))
	}

	if config.GenValueFunc != nil {
		args = append(args, redsync.WithGenValueFunc(config.GenValueFunc))
	}

	uniqKey := formatKey(config.App, config.Namespace, key)
	mutex := r.rs.NewMutex(uniqKey, args...)

	return &RedisMutex{
		mutex: mutex,
	}, nil
}

type RedisMutex struct {
	mutex *redsync.Mutex
}

// Key returns the key to be locked.
func (l *RedisMutex) Key() string {
	return l.mutex.Name()
}

// Lock acquires the lock. It fails with error if the lock is already held.
func (l *RedisMutex) Lock(ctx context.Context) error {
	err := l.mutex.LockContext(ctx)
	if err != nil {
		return translateRedisErr(err, l.Key())
	}
	return nil
}

// Unlock releases the lock. It fails with error if the lock is not currently held.
func (l *RedisMutex) Unlock(ctx context.Context) error {
	_, err := l.mutex.UnlockContext(ctx)
	if err != nil {
		return translateRedisErr(err, l.Key())
	}
	return nil
}

func translateRedisErr(err error, key string) error {
	var kind ErrorKind
	switch {
	case errors.Is(err, redsync.ErrFailed):
		kind = ErrorKindCannotLock
	case errors.Is(err, redsync.ErrExtendFailed), errors.Is(err, &redsync.RedisError{}):
		kind = ErrorKindProviderError
	case errors.Is(err, &redsync.ErrTaken{}), errors.Is(err, &redsync.ErrNodeTaken{}):
		kind = ErrorKindLockHeld
	}
	return NewError(kind, key, err)
}

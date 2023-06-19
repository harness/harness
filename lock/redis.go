// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	var kind KindError
	switch {
	case errors.Is(err, redsync.ErrFailed):
		kind = CannotLock
	case errors.Is(err, redsync.ErrExtendFailed), errors.Is(err, &redsync.RedisError{}):
		kind = ProviderError
	case errors.Is(err, &redsync.ErrTaken{}), errors.Is(err, &redsync.ErrNodeTaken{}):
		kind = LockHeld
	}
	return NewError(kind, key, err)
}

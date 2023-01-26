// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideConfig,
	ProvideMutexManager,
)

func ProvideConfig(config *types.Config) Config {
	return Config{
		app:           config.Lock.AppNamespace,
		namespace:     config.Lock.DefaultNamespace,
		provider:      Provider(config.Lock.Provider),
		expiry:        config.Lock.Expiry,
		tries:         config.Lock.Tries,
		retryDelay:    config.Lock.RetryDelay,
		driftFactor:   config.Lock.DriftFactor,
		timeoutFactor: config.Lock.TimeoutFactor,
	}
}

func ProvideMutexManager(config Config, client redis.UniversalClient) MutexManager {
	switch config.provider {
	case MemoryProvider:
		return NewInMemory(config)
	case RedisProvider:
		return NewRedis(config, client)
	}
	return nil
}

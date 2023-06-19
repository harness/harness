// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package lock

import (
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideMutexManager,
)

func ProvideMutexManager(config Config, client redis.UniversalClient) MutexManager {
	switch config.Provider {
	case MemoryProvider:
		return NewInMemory(config)
	case RedisProvider:
		return NewRedis(config, client)
	}
	return nil
}

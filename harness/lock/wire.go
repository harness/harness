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

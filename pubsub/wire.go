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

package pubsub

import (
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvidePubSub,
)

func ProvidePubSub(config Config, client redis.UniversalClient) PubSub {
	switch config.Provider {
	case ProviderRedis:
		return NewRedis(client,
			WithApp(config.App),
			WithNamespace(config.Namespace),
			WithHealthCheckInterval(config.HealthInterval),
			WithSendTimeout(config.SendTimeout),
			WithSize(config.ChannelSize),
		)
	case ProviderMemory:
		fallthrough
	default:
		return NewInMemory(
			WithApp(config.App),
			WithNamespace(config.Namespace),
			WithHealthCheckInterval(config.HealthInterval),
			WithSendTimeout(config.SendTimeout),
			WithSize(config.ChannelSize),
		)
	}
}

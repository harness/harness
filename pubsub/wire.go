// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubsub

import (
	"github.com/harness/gitness/types"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideConfig,
	ProvidePubSub,
)

func ProvideConfig(config *types.Config) Config {
	return Config{
		app:            config.PubSub.AppNamespace,
		namespace:      config.PubSub.DefaultNamespace,
		provider:       Provider(config.PubSub.Provider),
		healthInterval: config.PubSub.HealthInterval,
		sendTimeout:    config.PubSub.SendTimeout,
		channelSize:    config.PubSub.ChannelSize,
	}
}

func ProvidePubSub(config Config, client redis.UniversalClient) PubSub {
	switch config.provider {
	case ProviderRedis:
		return NewRedis(client,
			WithApp(config.app),
			WithNamespace(config.namespace),
			WithHealthCheckInterval(config.healthInterval),
			WithSendTimeout(config.sendTimeout),
			WithSize(config.channelSize),
		)
	case ProviderMemory:
		fallthrough
	default:
		return NewInMemory(
			WithApp(config.app),
			WithNamespace(config.namespace),
			WithHealthCheckInterval(config.healthInterval),
			WithSendTimeout(config.sendTimeout),
			WithSize(config.channelSize),
		)
	}
}

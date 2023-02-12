// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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

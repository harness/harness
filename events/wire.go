// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"errors"
	"fmt"

	"github.com/harness/gitness/stream"

	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideSystem,
)

func ProvideSystem(config Config, redisClient redis.Cmdable) (*System, error) {
	var system *System
	var err error
	switch config.Mode {
	case ModeInMemory:
		system, err = provideSystemInMemory(config)
	case ModeRedis:
		system, err = provideSystemRedis(config, redisClient)
	default:
		return nil, fmt.Errorf("events system mode '%s' is not supported", config.Mode)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to setup event system for mode '%s': %w", config.Mode, err)
	}

	return system, nil
}

func provideSystemInMemory(config Config) (*System, error) {
	broker := stream.NewMemoryBroker(config.MaxStreamLength)
	return NewSystem(
		newMemoryStreamConsumerFactoryMethod(broker, config.Namespace),
		newMemoryStreamProducer(broker, config.Namespace),
	)
}

func provideSystemRedis(config Config, redisClient redis.Cmdable) (*System, error) {
	if redisClient == nil {
		return nil, errors.New("redis client required")
	}

	return NewSystem(
		newRedisStreamConsumerFactoryMethod(redisClient, config.Namespace),
		newRedisStreamProducer(redisClient, config.Namespace,
			config.MaxStreamLength, config.ApproxMaxStreamLength),
	)
}

func newMemoryStreamConsumerFactoryMethod(broker *stream.MemoryBroker, namespace string) StreamConsumerFactoryFunc {
	return func(groupName string, consumerName string) (StreamConsumer, error) {
		return stream.NewMemoryConsumer(broker, namespace, groupName), nil
	}
}

func newMemoryStreamProducer(broker *stream.MemoryBroker, namespace string) StreamProducer {
	return stream.NewMemoryProducer(broker, namespace)
}

func newRedisStreamConsumerFactoryMethod(redisClient redis.Cmdable, namespace string) StreamConsumerFactoryFunc {
	return func(groupName string, consumerName string) (StreamConsumer, error) {
		return stream.NewRedisConsumer(redisClient, namespace, groupName, consumerName)
	}
}

func newRedisStreamProducer(redisClient redis.Cmdable, namespace string,
	maxStreamLength int64, approxMaxStreamLength bool) StreamProducer {
	return stream.NewRedisProducer(redisClient, namespace, maxStreamLength, approxMaxStreamLength)
}

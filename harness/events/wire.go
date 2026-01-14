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

func ProvideSystem(config Config, redisClient redis.UniversalClient) (*System, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("provided config is invalid: %w", err)
	}

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
	broker, err := stream.NewMemoryBroker(config.MaxStreamLength)
	if err != nil {
		return nil, err
	}

	return NewSystem(
		newMemoryStreamConsumerFactoryMethod(broker, config.Namespace),
		newMemoryStreamProducer(broker, config.Namespace),
	)
}

func provideSystemRedis(config Config, redisClient redis.UniversalClient) (*System, error) {
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
	return func(groupName string, _ string) (StreamConsumer, error) {
		return stream.NewMemoryConsumer(broker, namespace, groupName)
	}
}

func newMemoryStreamProducer(broker *stream.MemoryBroker, namespace string) StreamProducer {
	return stream.NewMemoryProducer(broker, namespace)
}

func newRedisStreamConsumerFactoryMethod(
	redisClient redis.UniversalClient,
	namespace string,
) StreamConsumerFactoryFunc {
	return func(groupName string, consumerName string) (StreamConsumer, error) {
		return stream.NewRedisConsumer(redisClient, namespace, groupName, consumerName)
	}
}

func newRedisStreamProducer(redisClient redis.UniversalClient, namespace string,
	maxStreamLength int64, approxMaxStreamLength bool) StreamProducer {
	return stream.NewRedisProducer(redisClient, namespace, maxStreamLength, approxMaxStreamLength)
}

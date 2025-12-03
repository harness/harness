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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

type Redis struct {
	config   Config
	client   redis.UniversalClient
	mutex    sync.RWMutex
	registry []Consumer
}

// NewRedis create an instance of redis PubSub implementation.
func NewRedis(client redis.UniversalClient, options ...Option) *Redis {
	config := Config{
		App:            "app",
		Namespace:      "default",
		HealthInterval: 3 * time.Second,
		SendTimeout:    60,
		ChannelSize:    100,
	}

	for _, f := range options {
		f.Apply(&config)
	}
	return &Redis{
		config:   config,
		client:   client,
		registry: make([]Consumer, 0, 16),
	}
}

// Subscribe consumer to process the event with payload.
func (r *Redis) Subscribe(
	ctx context.Context,
	topic string,
	handler func(payload []byte) error,
	options ...SubscribeOption,
) Consumer {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	config := SubscribeConfig{
		topics:         make([]string, 0, 8),
		app:            r.config.App,
		namespace:      r.config.Namespace,
		healthInterval: r.config.HealthInterval,
		sendTimeout:    r.config.SendTimeout,
		channelSize:    r.config.ChannelSize,
	}

	for _, f := range options {
		f.Apply(&config)
	}

	// create subscriber and map it to the registry
	subscriber := &redisSubscriber{
		config:  &config,
		handler: handler,
	}

	config.topics = append(config.topics, topic)

	topics := subscriber.formatTopics(config.topics...)
	subscriber.rdb = r.client.Subscribe(ctx, topics...)

	// start subscriber
	go subscriber.start(ctx)

	// register subscriber
	r.registry = append(r.registry, subscriber)

	return subscriber
}

// Publish event topic to message broker with payload.
func (r *Redis) Publish(ctx context.Context, topic string, payload []byte, opts ...PublishOption) error {
	pubConfig := PublishConfig{
		app:       r.config.App,
		namespace: r.config.Namespace,
	}
	for _, f := range opts {
		f.Apply(&pubConfig)
	}

	topic = formatTopic(pubConfig.app, pubConfig.namespace, topic)

	err := r.client.Publish(ctx, topic, payload).Err()
	if err != nil {
		return fmt.Errorf("failed to write to pubsub topic '%s'. Error: %w",
			topic, err)
	}
	return nil
}

func (r *Redis) Close(_ context.Context) error {
	for _, subscriber := range r.registry {
		err := subscriber.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

type redisSubscriber struct {
	config  *SubscribeConfig
	rdb     *redis.PubSub
	handler func([]byte) error
}

func (s *redisSubscriber) start(ctx context.Context) {
	// Go channel which receives messages.
	ch := s.rdb.Channel(
		redis.WithChannelHealthCheckInterval(s.config.healthInterval),
		redis.WithChannelSendTimeout(s.config.sendTimeout),
		redis.WithChannelSize(s.config.channelSize),
	)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				log.Ctx(ctx).Debug().Msg("redis channel was closed")
				return
			}
			if err := s.handler([]byte(msg.Payload)); err != nil {
				log.Ctx(ctx).Err(err).Msg("received an error from handler function")
			}
		}
	}
}

func (s *redisSubscriber) Subscribe(ctx context.Context, topics ...string) error {
	err := s.rdb.Subscribe(ctx, s.formatTopics(topics...)...)
	if err != nil {
		return fmt.Errorf("subscribe failed for chanels %v with error: %w",
			strings.Join(topics, ","), err)
	}
	return nil
}

func (s *redisSubscriber) Unsubscribe(ctx context.Context, topics ...string) error {
	err := s.rdb.Unsubscribe(ctx, s.formatTopics(topics...)...)
	if err != nil {
		return fmt.Errorf("unsubscribe failed for chanels %v with error: %w",
			strings.Join(topics, ","), err)
	}
	return nil
}

func (s *redisSubscriber) Close() error {
	err := s.rdb.Close()
	if err != nil {
		return fmt.Errorf("failed while closing subscriber with error: %w", err)
	}
	return nil
}

func (s *redisSubscriber) formatTopics(topics ...string) []string {
	result := make([]string, len(topics))
	for i, topic := range topics {
		result[i] = formatTopic(s.config.app, s.config.namespace, topic)
	}
	return result
}

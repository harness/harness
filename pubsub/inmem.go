// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pubsub

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

var (
	ErrClosed = errors.New("pubsub: subscriber is closed")
)

type InMemory struct {
	config   Config
	mutex    sync.Mutex
	registry []*inMemorySubscriber
}

// NewInMemory create an instance of memory pubsub implementation.
func NewInMemory(options ...Option) *InMemory {
	config := Config{
		app:            "app",
		namespace:      "default",
		healthInterval: 3 * time.Second,
		sendTimeout:    60,
		channelSize:    100,
	}

	for _, f := range options {
		f.Apply(&config)
	}
	return &InMemory{
		config:   config,
		registry: make([]*inMemorySubscriber, 0, 16),
	}
}

// Subscribe consumer to process the event with payload.
func (r *InMemory) Subscribe(
	ctx context.Context,
	topic string,
	handler func(payload []byte) error,
	options ...SubscribeOption,
) Consumer {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	config := SubscribeConfig{
		topics:      make([]string, 0, 8),
		app:         r.config.app,
		namespace:   r.config.namespace,
		sendTimeout: r.config.sendTimeout,
		channelSize: r.config.channelSize,
	}

	for _, f := range options {
		f.Apply(&config)
	}

	// create subscriber and map it to the registry
	subscriber := &inMemorySubscriber{
		config:  &config,
		handler: handler,
	}

	config.topics = append(config.topics, topic)
	subscriber.topics = subscriber.formatTopics(config.topics...)

	// start subscriber
	go subscriber.start(ctx)

	// register subscriber
	r.registry = append(r.registry, subscriber)

	return subscriber
}

// Publish event to message broker with payload.
func (r *InMemory) Publish(ctx context.Context, topic string, payload []byte, opts ...PublishOption) error {
	if len(r.registry) == 0 {
		log.Ctx(ctx).Warn().Msg("in pubsub Publish: no subscribers registered")
		return nil
	}
	pubConfig := PublishConfig{
		app:       r.config.app,
		namespace: r.config.namespace,
	}
	for _, f := range opts {
		f.Apply(&pubConfig)
	}

	topic = formatTopic(pubConfig.app, pubConfig.namespace, topic)
	for _, sub := range r.registry {
		if slices.Contains(sub.topics, topic) && !sub.isClosed() {
			go func(subscriber *inMemorySubscriber) {
				// timer is based on subscriber data
				t := time.NewTimer(subscriber.config.sendTimeout)
				defer t.Stop()
				select {
				case <-ctx.Done():
					return
				case subscriber.channel <- payload:
					log.Ctx(ctx).Trace().Msgf("in pubsub Publish: message %v sent to topic %s", string(payload), topic)
				case <-t.C:
					// channel is full for topic (message is dropped)
					log.Ctx(ctx).Warn().Msgf("in pubsub Publish: %s topic is full for %s (message is dropped)",
						topic, subscriber.config.sendTimeout)
				}
			}(sub)
		}
	}

	return nil
}

func (r *InMemory) Close(ctx context.Context) error {
	for _, subscriber := range r.registry {
		if err := subscriber.Close(); err != nil {
			return err
		}
	}
	return nil
}

type inMemorySubscriber struct {
	config  *SubscribeConfig
	handler func([]byte) error
	channel chan []byte
	once    sync.Once
	mutex   sync.RWMutex
	topics  []string
	closed  bool
}

func (s *inMemorySubscriber) start(ctx context.Context) {
	s.channel = make(chan []byte, s.config.channelSize)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-s.channel:
			if !ok {
				return
			}
			if err := s.handler(msg); err != nil {
				// TODO: bump err to caller
				log.Ctx(ctx).Err(err).Msgf("in pubsub start: error while running handler for topic")
			}
		}
	}
}

func (s *inMemorySubscriber) Subscribe(ctx context.Context, topics ...string) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	topics = s.formatTopics(topics...)
	for _, ch := range topics {
		if slices.Contains(s.topics, ch) {
			continue
		}
		s.topics = append(s.topics, ch)
	}
	return nil
}

func (s *inMemorySubscriber) Unsubscribe(ctx context.Context, topics ...string) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	topics = s.formatTopics(topics...)
	for i, ch := range topics {
		if slices.Contains(s.topics, ch) {
			s.topics[i] = s.topics[len(s.topics)-1]
			s.topics = s.topics[:len(s.topics)-1]
		}
	}
	return nil
}

func (s *inMemorySubscriber) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return ErrClosed
	}
	s.closed = true
	s.once.Do(func() {
		close(s.channel)
	})
	return nil
}

func (s *inMemorySubscriber) isClosed() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.closed
}

func (s *inMemorySubscriber) formatTopics(topics ...string) []string {
	result := make([]string, len(topics))
	for i, topic := range topics {
		result[i] = formatTopic(s.config.app, s.config.namespace, topic)
	}
	return result
}

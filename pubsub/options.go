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
	"time"
)

// An Option configures a pubsub instance.
type Option interface {
	Apply(*Config)
}

// OptionFunc is a function that configures a pubsub config.
type OptionFunc func(*Config)

// Apply calls f(config).
func (f OptionFunc) Apply(config *Config) {
	f(config)
}

// WithApp returns an option that set config app name.
func WithApp(value string) Option {
	return OptionFunc(func(m *Config) {
		m.App = value
	})
}

// WithNamespace returns an option that set config namespace.
func WithNamespace(value string) Option {
	return OptionFunc(func(m *Config) {
		m.Namespace = value
	})
}

// WithHealthCheckInterval specifies the config health check interval.
// PubSub will ping Server if it does not receive any messages
// within the interval (redis, ...).
// To disable health check, use zero interval.
func WithHealthCheckInterval(value time.Duration) Option {
	return OptionFunc(func(m *Config) {
		m.HealthInterval = value
	})
}

// WithSendTimeout specifies the pubsub send timeout after which
// the message is dropped.
func WithSendTimeout(value time.Duration) Option {
	return OptionFunc(func(m *Config) {
		m.SendTimeout = value
	})
}

// WithSize specifies the Go chan size in config that is used to buffer
// incoming messages.
func WithSize(value int) Option {
	return OptionFunc(func(m *Config) {
		m.ChannelSize = value
	})
}

type SubscribeConfig struct {
	topics         []string
	app            string
	namespace      string
	healthInterval time.Duration
	sendTimeout    time.Duration
	channelSize    int
}

// SubscribeOption configures a subscription config.
type SubscribeOption interface {
	Apply(*SubscribeConfig)
}

// SubscribeOptionFunc is a function that configures a subscription config.
type SubscribeOptionFunc func(*SubscribeConfig)

// Apply calls f(subscribeConfig).
func (f SubscribeOptionFunc) Apply(config *SubscribeConfig) {
	f(config)
}

// WithTopics specifies the topics to subsribe.
func WithTopics(topics ...string) SubscribeOption {
	return SubscribeOptionFunc(func(c *SubscribeConfig) {
		c.topics = topics
	})
}

// WithNamespace returns an channel option that configures namespace.
func WithChannelNamespace(value string) SubscribeOption {
	return SubscribeOptionFunc(func(c *SubscribeConfig) {
		c.namespace = value
	})
}

// WithChannelHealthCheckInterval specifies the channel health check interval.
// PubSub will ping Server if it does not receive any messages
// within the interval. To disable health check, use zero interval.
func WithChannelHealthCheckInterval(value time.Duration) SubscribeOption {
	return SubscribeOptionFunc(func(c *SubscribeConfig) {
		c.healthInterval = value
	})
}

// WithChannelSendTimeout specifies the channel send timeout after which
// the message is dropped.
func WithChannelSendTimeout(value time.Duration) SubscribeOption {
	return SubscribeOptionFunc(func(c *SubscribeConfig) {
		c.sendTimeout = value
	})
}

// WithChannelSize specifies the Go chan size that is used to buffer
// incoming messages for subscriber.
func WithChannelSize(value int) SubscribeOption {
	return SubscribeOptionFunc(func(c *SubscribeConfig) {
		c.channelSize = value
	})
}

type PublishConfig struct {
	app       string
	namespace string
}

type PublishOption interface {
	Apply(*PublishConfig)
}

// PublishOptionFunc is a function that configures a publish config.
type PublishOptionFunc func(*PublishConfig)

// Apply calls f(publishConfig).
func (f PublishOptionFunc) Apply(config *PublishConfig) {
	f(config)
}

// WithPublishApp modifies publish config app identifier.
func WithPublishApp(value string) PublishOption {
	return PublishOptionFunc(func(c *PublishConfig) {
		c.app = value
	})
}

// WithPublishNamespace modifies publish config namespace.
func WithPublishNamespace(value string) PublishOption {
	return PublishOptionFunc(func(c *PublishConfig) {
		c.namespace = value
	})
}

func formatTopic(app, ns, topic string) string {
	return app + ":" + ns + ":" + topic
}

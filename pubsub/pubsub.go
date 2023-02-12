// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pubsub

import "context"

type Publisher interface {
	// Publish topic to message broker with payload.
	Publish(ctx context.Context, topic string, payload []byte,
		options ...PublishOption) error
}

type PubSub interface {
	Publisher
	// Subscribe consumer to process the topic with payload, this should be
	// blocking operation.
	Subscribe(ctx context.Context, topic string,
		handler func(payload []byte) error, options ...SubscribeOption) Consumer
}

type Consumer interface {
	Subscribe(ctx context.Context, topics ...string) error
	Unsubscribe(ctx context.Context, topics ...string) error
	Close() error
}

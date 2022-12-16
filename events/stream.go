// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"
	"time"

	"github.com/harness/gitness/stream"
)

// StreamProducer is an abstraction of a producer from the streams package.
type StreamProducer interface {
	Send(ctx context.Context, streamID string, payload map[string]interface{}) (string, error)
}

// StreamConsumer is an abstraction of a consumer from the streams package.
type StreamConsumer interface {
	Register(streamID string, handler stream.HandlerFunc) error
	SetConcurrency(int) error
	SetProcessingTimeout(timeout time.Duration) error
	Start(ctx context.Context) error
	Errors() <-chan error
	Infos() <-chan string
}

// StreamConsumerFactoryFunc is an abstraction of a factory method for stream consumers.
type StreamConsumerFactoryFunc func(groupName string, consumerName string) (StreamConsumer, error)

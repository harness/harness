// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package stream

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrAlreadyStarted = errors.New("consumer already started")

	defaultConfig = ConsumerConfig{
		Concurrency: 2,
		DefaultHandlerConfig: HandlerConfig{
			idleTimeout: 1 * time.Minute,
			maxRetries:  2,
		},
	}
)

// ConsumerConfig defines the configuration of a consumer containing externally exposed values
// that can be configured using the available ConsumerOptions.
type ConsumerConfig struct {
	// Concurrency specifies the number of worker go routines executing stream handlers.
	Concurrency int

	// DefaultHandlerConfig is the default config used for stream handlers.
	DefaultHandlerConfig HandlerConfig
}

// HandlerConfig defines the configuration for a single stream handler containing externally exposed values
// that can be configured using the available HandlerOptions.
type HandlerConfig struct {
	// idleTimeout specifies the maximum duration a message stays read but unacknowleged
	// before it can be claimed by others.
	idleTimeout time.Duration

	// maxRetries specifies the max number a stream message is retried.
	maxRetries int
}

// HandlerFunc defines the signature of a function handling stream messages.
type HandlerFunc func(ctx context.Context, messageID string, payload map[string]interface{}) error

// handler defines a handler of a single stream.
type handler struct {
	handle HandlerFunc
	config HandlerConfig
}

// message is used internally for passing stream messages via channels.
type message struct {
	streamID string
	id       string
	values   map[string]interface{}
}

// transposeStreamID transposes the provided streamID based on the namespace.
func transposeStreamID(namespace string, streamID string) string {
	return fmt.Sprintf("%s:%s", namespace, streamID)
}

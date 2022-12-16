// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package stream

import (
	"context"
	"fmt"
	"time"
)

const (
	// MaxConcurrency is the max number of concurrent go routines (for message handling) for a single stream consumer.
	MaxConcurrency = 64

	// MinProcessingTimeout is the minumum time that can be configured as processing timeout for a stream consumer.
	MinProcessingTimeout = 1 * time.Minute
)

// HandlerFunc defines the signature of a function handling stream messages.
type HandlerFunc func(ctx context.Context, messageID string, payload map[string]interface{}) error

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

// consumerState specifies the different states of a consumer.
type consumerState string

const (
	// consumerStateSetup defines the state in which the consumer is being setup (Register, SetConcurrency, ...).
	// In other words, it's the state before the consumer was started.
	consumerStateSetup consumerState = "setup"

	// consumerStateStarted defines the state after the consumer was started (context not yet canceled).
	consumerStateStarted consumerState = "started"

	// consumerStateFinished defines the state after the consumer has been stopped (context canceled).
	consumerStateFinished consumerState = "finished"
)

// checkConsumerStateTransition returns an error in case the state transition is not allowed, nil otherwise.
// It is used to avoid that invalid operations are being executed in a given state (e.g. Register(...) when started).
func checkConsumerStateTransition(current, updated consumerState) error {
	switch {
	case current == consumerStateSetup && updated == consumerStateSetup:
		return nil
	case current == consumerStateSetup && updated == consumerStateStarted:
		return nil
	case current == consumerStateStarted && updated == consumerStateFinished:
		return nil
	default:
		return fmt.Errorf("consumer state transition from '%s' to '%s' is not possible", current, updated)
	}
}

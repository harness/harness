// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package stream

import (
	"context"
	"fmt"
)

// MemoryProducer sends messages to streams of a MemoryBroker.
type MemoryProducer struct {
	broker *MemoryBroker
	// namespace specifies the namespace of the keys - any stream key will be prefixed with it
	namespace string
}

func NewMemoryProducer(broker *MemoryBroker, namespace string) *MemoryProducer {
	return &MemoryProducer{
		broker:    broker,
		namespace: namespace,
	}
}

// Send sends information to the Broker.
// Returns the message ID in case of success.
func (p *MemoryProducer) Send(ctx context.Context, streamID string, payload map[string]interface{}) (string, error) {
	// ensure we transpose streamID using the key namespace
	transposedStreamID := transposeStreamID(p.namespace, streamID)

	msgID, err := p.broker.enqueue(
		transposedStreamID,
		message{
			streamID: transposedStreamID,
			values:   payload,
		})
	if err != nil {
		return "", fmt.Errorf("failed to write to stream '%s' (full stream '%s'). Error: %w",
			streamID, transposedStreamID, err)
	}

	return msgID, nil
}

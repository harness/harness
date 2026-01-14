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
func (p *MemoryProducer) Send(_ context.Context, streamID string, payload map[string]interface{}) (string, error) {
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

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
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"
)

// GenericReporter represents an event reporter that supports sending typesafe messages
// for an arbitrary set of custom events within an event category using the ReporterSendEvent method.
// NOTE: Optimally this should be an interface with SendEvent[T] method, but that's not possible in go.
type GenericReporter struct {
	producer StreamProducer
	category string
}

// ReportEvent reports an event using the provided GenericReporter.
// Returns the reported event's ID in case of success.
// NOTE: This call is blocking until the event was send (not until it was processed).
//
//nolint:revive // emphasize that this is meant to be an operation on *GenericReporter
func ReporterSendEvent[T interface{}](reporter *GenericReporter, ctx context.Context,
	eventType EventType, payload T) (string, error) {
	streamID := getStreamID(reporter.category, eventType)
	event := Event[T]{
		ID:        "", // will be set by GenericReader
		Timestamp: time.Now(),
		Payload:   payload,
	}

	buff := &bytes.Buffer{}
	encoder := gob.NewEncoder(buff)

	if err := encoder.Encode(&event); err != nil {
		return "", fmt.Errorf("failed to encode payload: %w", err)
	}

	streamPayload := map[string]interface{}{
		streamPayloadKey: buff.Bytes(),
	}

	// We are using the message ID as event ID.
	return reporter.producer.Send(ctx, streamID, streamPayload)
}

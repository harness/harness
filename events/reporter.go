// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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

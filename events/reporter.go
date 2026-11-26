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

	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/tracing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var tracer = otel.Tracer("github.com/harness/gitness/events")

// GenericReporter represents an event reporter that supports sending typesafe messages
// for an arbitrary set of custom events within an event category using the ReporterSendEvent method.
// NOTE: Optimally this should be an interface with SendEvent[T] method, but that's not possible in go.
type GenericReporter struct {
	producer StreamProducer
	category string
	traced   bool
}

// ReporterSendEvent reports an event using the provided GenericReporter.
// Returns the reported event's ID in case of success.
// NOTE: This call is blocking until the event was send (not until it was processed).
//
//nolint:revive // emphasize that this is meant to be an operation on *GenericReporter
func ReporterSendEvent[T any](reporter *GenericReporter, ctx context.Context,
	eventType EventType, payload T) (_ string, err error) {
	streamID := getStreamID(reporter.category, eventType)

	span := trace.Span(noop.Span{})
	if reporter.traced {
		ctx, span = tracer.Start(ctx, "send "+streamID,
			trace.WithSpanKind(trace.SpanKindProducer),
			trace.WithAttributes(
				semconv.MessagingOperationTypeSend,
				semconv.MessagingDestinationName(streamID),
			))
	}
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to send event")
		}
		span.End()
	}()

	eventID, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate event ID: %w", err)
	}
	span.SetAttributes(attribute.String("events.id", eventID.String()))

	event := Event[T]{
		ID:        eventID.String(),
		Timestamp: time.Now(),
		Payload:   payload,
	}

	buff := &bytes.Buffer{}
	encoder := gob.NewEncoder(buff)
	gob.Register((*response.StartResponse)(nil))
	gob.Register((*response.StopResponse)(nil))
	gob.Register((*response.DeleteResponse)(nil))

	if err := encoder.Encode(&event); err != nil {
		return "", fmt.Errorf("failed to encode payload: %w", err)
	}

	streamPayload := map[string]any{
		streamPayloadKey: buff.Bytes(),
	}
	// The trace context is stored next to the event so the reader can continue the trace of the sender.
	if reporter.traced {
		tracing.InjectMap(ctx, streamPayload)
	}

	messageID, err := reporter.producer.Send(ctx, streamID, streamPayload)
	if err != nil {
		return "", fmt.Errorf("failed to send event: %w", err)
	}
	span.SetAttributes(semconv.MessagingMessageID(messageID))

	return event.ID, nil
}

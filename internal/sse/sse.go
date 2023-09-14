// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types/enum"
)

// Event is a server sent event.
type Event struct {
	Type enum.SSEType    `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Streamer interface {
	// Publish publishes an event to a given space ID.
	Publish(ctx context.Context, spaceID int64, eventType enum.SSEType, data any) error

	// Streams streams the events on a space ID.
	Stream(ctx context.Context, spaceID int64) (<-chan *Event, <-chan error, func(context.Context) error)
}

type pubsubStreamer struct {
	pubsub    pubsub.PubSub
	namespace string
}

func NewStreamer(pubsub pubsub.PubSub, namespace string) Streamer {
	return &pubsubStreamer{
		pubsub:    pubsub,
		namespace: namespace,
	}
}

func (e *pubsubStreamer) Publish(ctx context.Context, spaceID int64, eventType enum.SSEType, data any) error {
	dataSerialized, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to serialize data: %w", err)
	}
	event := Event{
		Type: eventType,
		Data: dataSerialized,
	}
	serializedEvent, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}
	namespaceOption := pubsub.WithPublishNamespace(e.namespace)
	topic := getSpaceTopic(spaceID)
	err = e.pubsub.Publish(ctx, topic, serializedEvent, namespaceOption)
	if err != nil {
		return fmt.Errorf("failed to publish event on pubsub: %w", err)
	}

	return nil
}

func (e *pubsubStreamer) Stream(ctx context.Context, spaceID int64) (<-chan *Event, <-chan error, func(context.Context) error) {
	chEvent := make(chan *Event, 100) // TODO: check best size here
	chErr := make(chan error)
	g := func(payload []byte) error {
		event := &Event{}
		err := json.Unmarshal(payload, event)
		if err != nil {
			// This should never happen
			return err
		}
		select {
		case chEvent <- event:
		default:
		}

		return nil
	}
	namespaceOption := pubsub.WithChannelNamespace(e.namespace)
	topic := getSpaceTopic(spaceID)
	consumer := e.pubsub.Subscribe(ctx, topic, g, namespaceOption)
	unsubscribeFN := func(ctx context.Context) error {
		return consumer.Unsubscribe(ctx, topic)
	}

	return chEvent, chErr, unsubscribeFN
}

// getSpaceTopic creates the namespace name which will be `spaces:<id>`
func getSpaceTopic(spaceID int64) string {
	return "spaces:" + strconv.Itoa(int(spaceID))
}

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

package sse

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Event is a server sent event.
type Event struct {
	Type enum.SSEType    `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Streamer interface {
	// Publish publishes an event to a given space ID.
	Publish(ctx context.Context, spaceID int64, eventType enum.SSEType, data any)

	// Stream streams the events on a space ID.
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

func (e *pubsubStreamer) Publish(
	ctx context.Context,
	spaceID int64,
	eventType enum.SSEType,
	data any,
) {
	dataSerialized, err := json.Marshal(data)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to serialize data: %v", err.Error())
	}
	event := Event{
		Type: eventType,
		Data: dataSerialized,
	}
	serializedEvent, err := json.Marshal(event)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to serialize event: %v", err.Error())
	}
	namespaceOption := pubsub.WithPublishNamespace(e.namespace)
	topic := getSpaceTopic(spaceID)
	err = e.pubsub.Publish(ctx, topic, serializedEvent, namespaceOption)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to publish %s event", eventType)
	}
}

func (e *pubsubStreamer) Stream(
	ctx context.Context,
	spaceID int64,
) (<-chan *Event, <-chan error, func(context.Context) error) {
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
	cleanupFN := func(_ context.Context) error {
		return consumer.Close()
	}

	return chEvent, chErr, cleanupFN
}

// getSpaceTopic creates the namespace name which will be `spaces:<id>`.
func getSpaceTopic(spaceID int64) string {
	return "spaces:" + strconv.Itoa(int(spaceID))
}

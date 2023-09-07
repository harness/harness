// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types/enum"
)

// Event is an event which is sent to the UI via server-sent events.
type Event struct {
	Type enum.EventType  `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Events interface {
	// Publish publishes an event to a given space ID.
	Publish(ctx context.Context, spaceID int64, event *Event) error

	// Subscribe listens to events on a space ID.
	Subscribe(ctx context.Context, spaceID int64) (<-chan *Event, <-chan error)
}

type event struct {
	pubsub pubsub.PubSub
	topic  string
}

func New(pubsub pubsub.PubSub, topic string) Events {
	return &event{
		pubsub: pubsub,
		topic:  topic,
	}
}

func (e *event) Publish(ctx context.Context, spaceID int64, event *Event) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	option := pubsub.WithPublishNamespace(format(spaceID))
	return e.pubsub.Publish(ctx, e.topic, bytes, option)
}

// format creates the namespace name which will be spaces-<id>
func format(id int64) string {
	return "spaces-" + strconv.Itoa(int(id))
}

func (e *event) Subscribe(ctx context.Context, spaceID int64) (<-chan *Event, <-chan error) {
	chEvent := make(chan *Event, 100) // TODO: check best size here
	chErr := make(chan error)
	g := func(payload []byte) error {
		event := &Event{}
		err := json.Unmarshal(payload, event)
		if err != nil {
			// This should never happen
			return err
		}
		chEvent <- event
		return nil
	}
	option := pubsub.WithChannelNamespace(format(spaceID))
	e.pubsub.Subscribe(ctx, e.topic, g, option)
	return chEvent, chErr
}

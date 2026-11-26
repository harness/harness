// Copyright 2026 Harness, Inc.
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
	"testing"
	"time"

	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func infReceive(t *testing.T, ch <-chan *Event) *Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
		return nil
	}
}

func TestInfGetSpaceTopic(t *testing.T) {
	require.Equal(t, "spaces:42", getSpaceTopic(42))
}

func TestInfStreamer_PublishAndStream(t *testing.T) {
	ctx := context.Background()
	ps := pubsub.NewInMemory(pubsub.WithSendTimeout(time.Minute))
	streamer := NewStreamer(ps, "inf-ns")

	events, errs, cleanup := streamer.Stream(ctx, 7)
	require.NotNil(t, errs)
	other, _, otherCleanup := streamer.Stream(ctx, 8)

	streamer.Publish(ctx, 7, enum.SSETypeExecutionCanceled, map[string]string{"k": "v"})

	ev := infReceive(t, events)
	require.Equal(t, enum.SSETypeExecutionCanceled, ev.Type)
	var data map[string]string
	require.NoError(t, json.Unmarshal(ev.Data, &data))
	require.Equal(t, map[string]string{"k": "v"}, data)

	select {
	case unexpected := <-other:
		t.Fatalf("event for another space received: %+v", unexpected)
	case <-time.After(50 * time.Millisecond):
	}

	require.NoError(t, cleanup(ctx))
	require.NoError(t, otherCleanup(ctx))
}

func TestInfStreamer_PublishWithUnserialisableDataStillPublishesEvent(t *testing.T) {
	ctx := context.Background()
	streamer := NewStreamer(pubsub.NewInMemory(pubsub.WithSendTimeout(time.Minute)), "inf-ns")
	events, _, cleanup := streamer.Stream(ctx, 1)
	defer func() { require.NoError(t, cleanup(ctx)) }()

	streamer.Publish(ctx, 1, enum.SSETypeExecutionRunning, make(chan int))

	// the failure to serialise the data is only logged; the event type is still delivered.
	ev := infReceive(t, events)
	require.Equal(t, enum.SSETypeExecutionRunning, ev.Type)
}

func TestInfStreamer_InvalidPayloadIsDropped(t *testing.T) {
	ctx := context.Background()
	ps := pubsub.NewInMemory(pubsub.WithSendTimeout(time.Minute))
	streamer := NewStreamer(ps, "inf-ns")
	events, _, cleanup := streamer.Stream(ctx, 3)
	defer func() { require.NoError(t, cleanup(ctx)) }()

	require.NoError(t, ps.Publish(ctx, getSpaceTopic(3), []byte("not json"),
		pubsub.WithPublishNamespace("inf-ns")))
	streamer.Publish(ctx, 3, enum.SSETypeExecutionCompleted, 1)

	ev := infReceive(t, events)
	require.Equal(t, enum.SSETypeExecutionCompleted, ev.Type)
}

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

package stream

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gtxTimeout = 5 * time.Second

type gtxReceived struct {
	id      string
	payload map[string]any
}

func gtxWaitFor[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(gtxTimeout):
		t.Fatal("timed out waiting for a value")
		var zero T
		return zero
	}
}

func gtxExpectNothing[T any](t *testing.T, ch <-chan T) {
	t.Helper()

	select {
	case v := <-ch:
		t.Fatalf("unexpected value: %v", v)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestGtxMemoryBroker(t *testing.T) {
	t.Run("enqueue without groups discards the message", func(t *testing.T) {
		b, err := NewMemoryBroker(2)
		require.NoError(t, err)

		id, err := b.enqueue("s", message{streamID: "s"})
		require.NoError(t, err)
		assert.Empty(t, id)
	})

	t.Run("queues are created once per stream and group", func(t *testing.T) {
		b, err := NewMemoryBroker(2)
		require.NoError(t, err)

		first := b.messages("s", "g1")
		assert.Equal(t, first, b.messages("s", "g1"))
		assert.NotEqual(t, first, b.messages("s", "g2"))
		assert.NotEqual(t, first, b.messages("other", "g1"))
	})

	t.Run("ids", func(t *testing.T) {
		b, err := NewMemoryBroker(10)
		require.NoError(t, err)
		queue := b.messages("s", "g")

		first, err := b.enqueue("s", message{streamID: "s"})
		require.NoError(t, err)
		second, err := b.enqueue("s", message{streamID: "s", id: "*"})
		require.NoError(t, err)
		explicit, err := b.enqueue("s", message{streamID: "s", id: "custom-1"})
		require.NoError(t, err)

		assert.True(t, strings.HasSuffix(first, "-1"), first)
		assert.True(t, strings.HasSuffix(second, "-2"), second)
		assert.Equal(t, "custom-1", explicit)
		assert.Equal(t, strings.TrimSuffix(first, "-1"), strings.TrimSuffix(second, "-2"))
		assert.Len(t, strings.TrimSuffix(first, "-1"), idPrefixUIDLength)

		assert.Equal(t, first, (<-queue).id)
		assert.Equal(t, second, (<-queue).id)
		assert.Equal(t, "custom-1", (<-queue).id)
	})

	t.Run("full queue of one group doesn't block the others", func(t *testing.T) {
		b, err := NewMemoryBroker(1)
		require.NoError(t, err)

		full := b.messages("s", "full")
		other := b.messages("s", "other")

		_, err = b.enqueue("s", message{streamID: "s", id: "1"})
		require.NoError(t, err)
		assert.Equal(t, "1", (<-other).id) // drain only the other group.

		id, err := b.enqueue("s", message{streamID: "s", id: "2"})
		require.ErrorContains(t, err, "queue for group 'full' is full")
		assert.Equal(t, "2", id)
		assert.Equal(t, "2", (<-other).id)
		assert.Equal(t, "1", (<-full).id)
	})
}

func TestGtxMemoryProducerAndConsumer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	broker, err := NewMemoryBroker(10)
	require.NoError(t, err)

	producer := NewMemoryProducer(broker, "ns")

	t.Run("messages reach the handler of the group", func(t *testing.T) {
		consumer, err := NewMemoryConsumer(broker, "ns", "group")
		require.NoError(t, err)

		received := make(chan gtxReceived, 4)
		require.NoError(t, consumer.Register("orders", func(_ context.Context, id string, payload map[string]any) error {
			received <- gtxReceived{id: id, payload: payload}
			return nil
		}))

		// create the queue up front: messages sent to streams without a group are discarded.
		broker.messages(transposeStreamID("ns", "orders"), "group")
		require.NoError(t, consumer.Start(ctx))

		id, err := producer.Send(ctx, "orders", map[string]any{"k": "v", "n": 1})
		require.NoError(t, err)
		assert.NotEmpty(t, id)

		got := gtxWaitFor(t, received)
		assert.Equal(t, id, got.id)
		assert.Equal(t, map[string]any{"k": "v", "n": 1}, got.payload)
	})

	t.Run("other streams and namespaces are not delivered", func(t *testing.T) {
		consumer, err := NewMemoryConsumer(broker, "ns", "group-isolated")
		require.NoError(t, err)

		received := make(chan gtxReceived, 4)
		require.NoError(t, consumer.Register("isolated", func(_ context.Context, id string, p map[string]any) error {
			received <- gtxReceived{id: id, payload: p}
			return nil
		}))
		broker.messages(transposeStreamID("ns", "isolated"), "group-isolated")
		require.NoError(t, consumer.Start(ctx))

		_, err = producer.Send(ctx, "different-stream", map[string]any{"a": 1})
		require.NoError(t, err)
		_, err = NewMemoryProducer(broker, "other-ns").Send(ctx, "isolated", map[string]any{"a": 1})
		require.NoError(t, err)

		gtxExpectNothing(t, received)
	})

	t.Run("every group gets every message", func(t *testing.T) {
		received := make(chan string, 4)

		for _, group := range []string{"fan-a", "fan-b"} {
			consumer, err := NewMemoryConsumer(broker, "ns", group)
			require.NoError(t, err)

			require.NoError(t, consumer.Register("fanout", func(_ context.Context, _ string, _ map[string]any) error {
				received <- group
				return nil
			}))
			broker.messages(transposeStreamID("ns", "fanout"), group)
			require.NoError(t, consumer.Start(ctx))
		}

		_, err := producer.Send(ctx, "fanout", map[string]any{})
		require.NoError(t, err)

		got := []string{gtxWaitFor(t, received), gtxWaitFor(t, received)}
		assert.ElementsMatch(t, []string{"fan-a", "fan-b"}, got)
	})

	t.Run("send fails if a queue is full", func(t *testing.T) {
		small, err := NewMemoryBroker(1)
		require.NoError(t, err)
		small.messages(transposeStreamID("ns", "full"), "group")

		p := NewMemoryProducer(small, "ns")
		_, err = p.Send(ctx, "full", map[string]any{})
		require.NoError(t, err)

		_, err = p.Send(ctx, "full", map[string]any{})
		require.ErrorContains(t, err, "failed to write to stream 'full' (full stream 'ns:full')")
	})
}

func TestGtxMemoryConsumer_Validation(t *testing.T) {
	broker, err := NewMemoryBroker(1)
	require.NoError(t, err)

	noop := func(context.Context, string, map[string]any) error { return nil }

	t.Run("group name is required", func(t *testing.T) {
		_, err := NewMemoryConsumer(broker, "ns", "")
		require.Error(t, err)
	})

	t.Run("register validation", func(t *testing.T) {
		c, err := NewMemoryConsumer(broker, "ns", "g")
		require.NoError(t, err)

		require.Error(t, c.Register("", noop))
		require.Error(t, c.Register("s", nil))
		require.NoError(t, c.Register("s", noop))
		require.ErrorContains(t, c.Register("s", noop), "already registered")
	})

	t.Run("handler options override the defaults", func(t *testing.T) {
		c, err := NewMemoryConsumer(broker, "ns", "g")
		require.NoError(t, err)

		c.Configure(WithHandlerOptions(WithMaxRetries(7), WithIdleTimeout(10*time.Second)))
		require.NoError(t, c.Register("defaults", noop))
		require.NoError(t, c.Register("custom", noop, WithMaxRetries(1), WithIdleTimeout(time.Hour)))

		assert.Equal(t, 7, c.streams["ns:defaults"].config.maxRetries)
		assert.Equal(t, 10*time.Second, c.streams["ns:defaults"].config.idleTimeout)
		assert.Equal(t, 1, c.streams["ns:custom"].config.maxRetries)
		assert.Equal(t, time.Hour, c.streams["ns:custom"].config.idleTimeout)
	})

	t.Run("start requires a stream", func(t *testing.T) {
		c, err := NewMemoryConsumer(broker, "ns", "g")
		require.NoError(t, err)

		require.ErrorContains(t, c.Start(context.Background()), "no streams registered")
	})

	t.Run("lifecycle", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		c, err := NewMemoryConsumer(broker, "ns", "g")
		require.NoError(t, err)
		c.Configure(WithConcurrency(3))
		assert.Equal(t, 3, c.Config.Concurrency)
		require.NoError(t, c.Register("s", noop))

		require.NoError(t, c.Start(ctx))
		require.ErrorIs(t, c.Start(ctx), ErrAlreadyStarted)
		require.ErrorIs(t, c.Register("another", noop), ErrAlreadyStarted)

		c.Configure(WithConcurrency(9))
		assert.Equal(t, 3, c.Config.Concurrency, "configure after start has no effect")

		// canceling the context stops the routines and closes the channels.
		cancel()
		select {
		case _, ok := <-c.Errors():
			assert.False(t, ok)
		case <-time.After(gtxTimeout):
			t.Fatal("error channel not closed")
		}
		select {
		case _, ok := <-c.Infos():
			assert.False(t, ok)
		case <-time.After(gtxTimeout):
			t.Fatal("info channel not closed")
		}
	})
}

func TestGtxMemoryConsumer_HandlerFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	broker, err := NewMemoryBroker(10)
	require.NoError(t, err)

	consumer, err := NewMemoryConsumer(broker, "ns", "g")
	require.NoError(t, err)

	require.NoError(t, consumer.Register("failing", func(context.Context, string, map[string]any) error {
		return errors.New("gtx handler failure")
	}, WithIdleTimeout(time.Hour)))
	require.NoError(t, consumer.Register("panicking", func(context.Context, string, map[string]any) error {
		panic("gtx handler panic")
	}, WithIdleTimeout(time.Hour)))

	for _, s := range []string{"failing", "panicking"} {
		broker.messages(transposeStreamID("ns", s), "g")
	}
	require.NoError(t, consumer.Start(ctx))

	producer := NewMemoryProducer(broker, "ns")

	failingID, err := producer.Send(ctx, "failing", map[string]any{})
	require.NoError(t, err)

	got := gtxWaitFor(t, consumer.Errors())
	assert.ErrorContains(t, got, "failed to process message with id '"+failingID+"' in stream 'ns:failing'")
	assert.ErrorContains(t, got, "gtx handler failure")

	panicID, err := producer.Send(ctx, "panicking", map[string]any{})
	require.NoError(t, err)

	got = gtxWaitFor(t, consumer.Errors())
	assert.ErrorContains(t, got, "PANIC when processing message '"+panicID+"' in stream 'ns:panicking'")
}

func TestGtxMemoryConsumer_UnknownStreamIsDiscarded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	broker, err := NewMemoryBroker(1)
	require.NoError(t, err)

	consumer, err := NewMemoryConsumer(broker, "ns", "g")
	require.NoError(t, err)
	require.NoError(t, consumer.Register("s", func(context.Context, string, map[string]any) error { return nil }))
	require.NoError(t, consumer.Start(ctx))

	consumer.messageQueue <- memoryMessage{message: message{streamID: "foreign", id: "m1"}}

	got := gtxWaitFor(t, consumer.Errors())
	assert.ErrorContains(t, got, "discard message with id 'm1' from stream 'foreign' - doesn't belong to us")
}

func TestGtxMemoryConsumer_RetryMessage(t *testing.T) {
	broker, err := NewMemoryBroker(1)
	require.NoError(t, err)

	consumer, err := NewMemoryConsumer(broker, "ns", "g")
	require.NoError(t, err)

	t.Run("requeues until the retry limit", func(t *testing.T) {
		consumer.retryMessage(memoryMessage{message: message{streamID: "s", id: "m1"}, retries: 1}, 2)

		got := gtxWaitFor(t, consumer.messageQueue)
		assert.Equal(t, "m1", got.id)
		assert.Equal(t, int64(2), got.retries)
		gtxExpectNothing(t, consumer.Errors())
	})

	t.Run("discards after the retry limit", func(t *testing.T) {
		consumer.retryMessage(memoryMessage{message: message{streamID: "s", id: "m2"}, retries: 2}, 2)

		got := gtxWaitFor(t, consumer.Errors())
		assert.ErrorContains(t, got, "discard message with id 'm2' from stream 's' - failed 2 retries")
		gtxExpectNothing(t, consumer.messageQueue)
	})

	t.Run("push error drops errors if nobody reads them", func(t *testing.T) {
		for range 3 * cap(consumer.errorCh) {
			consumer.pushError(errors.New("gtx overflow"))
		}
		assert.Len(t, consumer.errorCh, cap(consumer.errorCh))
	})
}

func TestGtxMemoryConsumer_RetryPostTimeout(t *testing.T) {
	broker, err := NewMemoryBroker(1)
	require.NoError(t, err)

	t.Run("retries once the idle timeout expired", func(t *testing.T) {
		consumer, err := NewMemoryConsumer(broker, "ns", "g")
		require.NoError(t, err)

		h := handler{config: HandlerConfig{idleTimeout: 10 * time.Millisecond, maxRetries: 3}}
		go consumer.retryPostTimeout(context.Background(), h, memoryMessage{message: message{id: "m1"}})

		got := gtxWaitFor(t, consumer.messageQueue)
		assert.Equal(t, int64(1), got.retries)
	})

	t.Run("canceled retry doesn't requeue", func(t *testing.T) {
		consumer, err := NewMemoryConsumer(broker, "ns", "g")
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		h := handler{config: HandlerConfig{idleTimeout: time.Hour, maxRetries: 3}}
		done := make(chan struct{})
		go func() {
			consumer.retryPostTimeout(ctx, h, memoryMessage{message: message{id: "m1"}})
			close(done)
		}()

		gtxWaitFor(t, done)
		gtxExpectNothing(t, consumer.messageQueue)
	})
}

func TestGtxDiscardedMessageError(t *testing.T) {
	ackErr := errors.New("gtx ack failure")

	withAckErr := &DiscardedMessageError{MessageID: "m1", StreamID: "s", RetryCount: 3, AckErr: ackErr}
	assert.Equal(t,
		"failed to force acknowledge (discard) message 'm1' (Retries: 3) in stream 's': gtx ack failure",
		withAckErr.Error())
	assert.ErrorIs(t, withAckErr, ackErr)

	withoutAckErr := &DiscardedMessageError{MessageID: "m2", StreamID: "t", RetryCount: 1}
	assert.Equal(t, "force acknowledged (discarded) message 'm2' (Retries: 1) in stream 't'", withoutAckErr.Error())
	assert.NoError(t, withoutAckErr.Unwrap())
}

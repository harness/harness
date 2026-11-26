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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func infNewBroker(t *testing.T, size int64) *MemoryBroker {
	t.Helper()
	b, err := NewMemoryBroker(size)
	require.NoError(t, err)
	return b
}

func TestInfMemoryBroker_EnqueueWithoutGroupsDiscards(t *testing.T) {
	b := infNewBroker(t, 4)
	id, err := b.enqueue("ns:stream", message{streamID: "ns:stream"})
	require.NoError(t, err)
	require.Empty(t, id)
}

func TestInfMemoryBroker_EnqueueGeneratesSequentialIDs(t *testing.T) {
	b := infNewBroker(t, 4)
	q := b.messages("ns:s", "g1")

	id1, err := b.enqueue("ns:s", message{streamID: "ns:s"})
	require.NoError(t, err)
	id2, err := b.enqueue("ns:s", message{streamID: "ns:s", id: "*"})
	require.NoError(t, err)
	require.NotEqual(t, id1, id2)
	require.Contains(t, id1, b.idPrefix+"-")

	explicit, err := b.enqueue("ns:s", message{streamID: "ns:s", id: "custom-id"})
	require.NoError(t, err)
	require.Equal(t, "custom-id", explicit)

	require.Equal(t, id1, (<-q).id)
	require.Equal(t, id2, (<-q).id)
	require.Equal(t, "custom-id", (<-q).id)
}

func TestInfMemoryBroker_FanOutToAllGroups(t *testing.T) {
	b := infNewBroker(t, 4)
	q1 := b.messages("ns:s", "g1")
	q2 := b.messages("ns:s", "g2")

	// asking for the same group returns the same queue
	again := b.messages("ns:s", "g1")
	_, err := b.enqueue("ns:s", message{streamID: "ns:s", values: map[string]any{"k": "v"}})
	require.NoError(t, err)
	require.Equal(t, "v", (<-again).values["k"])
	require.Equal(t, "v", (<-q2).values["k"])
	require.Empty(t, q1)
}

func TestInfMemoryBroker_FullQueueReturnsError(t *testing.T) {
	b := infNewBroker(t, 1)
	b.messages("ns:s", "g1")
	_, err := b.enqueue("ns:s", message{})
	require.NoError(t, err)
	_, err = b.enqueue("ns:s", message{})
	require.ErrorContains(t, err, "queue for group 'g1' is full")
}

func TestInfMemoryProducer_Send(t *testing.T) {
	b := infNewBroker(t, 4)
	p := NewMemoryProducer(b, "myns")
	ctx := context.Background()

	// no consumer group yet: message is discarded without error.
	id, err := p.Send(ctx, "events", map[string]any{"a": 1})
	require.NoError(t, err)
	require.Empty(t, id)

	q := b.messages("myns:events", "grp")
	id, err = p.Send(ctx, "events", map[string]any{"a": 1})
	require.NoError(t, err)
	require.NotEmpty(t, id)
	m := <-q
	require.Equal(t, id, m.id)
	require.Equal(t, "myns:events", m.streamID)
	require.Equal(t, 1, m.values["a"])

	// overflow produces a wrapped error
	small := infNewBroker(t, 1)
	sp := NewMemoryProducer(small, "n")
	small.messages("n:s", "g")
	_, err = sp.Send(ctx, "s", nil)
	require.NoError(t, err)
	_, err = sp.Send(ctx, "s", nil)
	require.ErrorContains(t, err, "failed to write to stream 's' (full stream 'n:s')")
}

func TestInfMemoryConsumer_Validation(t *testing.T) {
	b := infNewBroker(t, 4)
	_, err := NewMemoryConsumer(b, "ns", "")
	require.Error(t, err)

	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)
	noop := func(context.Context, string, map[string]any) error { return nil }

	require.Error(t, c.Register("", noop))
	require.Error(t, c.Register("s", nil))
	require.NoError(t, c.Register("s", noop))
	require.ErrorContains(t, c.Register("s", noop), "already registered")

	c.Configure(WithConcurrency(3))
	require.Equal(t, 3, c.Config.Concurrency)

	empty, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)
	require.EqualError(t, empty.Start(context.Background()), "no streams registered")
}

func TestInfMemoryConsumer_ConsumesMessages(t *testing.T) {
	b := infNewBroker(t, 16)
	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)
	c.Configure(WithConcurrency(2))

	var mx sync.Mutex
	got := map[string]any{}
	require.NoError(t, c.Register("s", func(_ context.Context, id string, payload map[string]any) error {
		mx.Lock()
		defer mx.Unlock()
		got[id] = payload["n"]
		return nil
	}))

	b.messages("ns:s", "grp")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, c.Start(ctx))
	require.ErrorIs(t, c.Start(ctx), ErrAlreadyStarted)
	require.ErrorIs(t, c.Register("other", func(context.Context, string, map[string]any) error { return nil }),
		ErrAlreadyStarted)
	c.Configure(WithConcurrency(9)) // ignored after start
	require.Equal(t, 2, c.Config.Concurrency)

	p := NewMemoryProducer(b, "ns")
	ids := map[string]int{}
	for i := 0; i < 5; i++ {
		id, sendErr := p.Send(ctx, "s", map[string]any{"n": i})
		require.NoError(t, sendErr)
		ids[id] = i
	}

	require.Eventually(t, func() bool {
		mx.Lock()
		defer mx.Unlock()
		return len(got) == len(ids)
	}, 5*time.Second, 5*time.Millisecond)
	mx.Lock()
	for id, n := range ids {
		require.Equal(t, n, got[id])
	}
	mx.Unlock()

	cancel()
	// channels are closed once all goroutines are done
	require.Eventually(t, func() bool {
		select {
		case _, ok := <-c.Errors():
			return !ok
		default:
			return false
		}
	}, 5*time.Second, 5*time.Millisecond)
	require.NotNil(t, c.Infos())
}

func TestInfMemoryConsumer_HandlerErrorAndPanicAreReported(t *testing.T) {
	b := infNewBroker(t, 16)
	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)
	c.Configure(WithConcurrency(1))

	require.NoError(t, c.Register("fail", func(context.Context, string, map[string]any) error {
		return errors.New("handler failed")
	}, WithMaxRetries(0), WithIdleTimeout(time.Hour)))
	require.NoError(t, c.Register("panic", func(context.Context, string, map[string]any) error {
		panic("oops")
	}, WithMaxRetries(0), WithIdleTimeout(time.Hour)))

	b.messages("ns:fail", "grp")
	b.messages("ns:panic", "grp")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, c.Start(ctx))

	p := NewMemoryProducer(b, "ns")
	_, err = p.Send(ctx, "fail", nil)
	require.NoError(t, err)
	_, err = p.Send(ctx, "panic", nil)
	require.NoError(t, err)

	var msgs []string
	require.Eventually(t, func() bool {
		select {
		case e := <-c.Errors():
			msgs = append(msgs, e.Error())
		default:
		}
		return len(msgs) >= 2
	}, 5*time.Second, 5*time.Millisecond)

	joined := msgs[0] + msgs[1]
	require.Contains(t, joined, "failed to process message")
	require.Contains(t, joined, "PANIC when processing message")
}

func TestInfMemoryConsumer_RetryPostTimeout(t *testing.T) {
	b := infNewBroker(t, 4)
	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)
	h := handler{config: HandlerConfig{idleTimeout: 20 * time.Millisecond, maxRetries: 2}}
	m := memoryMessage{message: message{id: "m1", streamID: "ns:s"}}

	// timer expiry requeues the message with an increased retry count.
	c.retryPostTimeout(context.Background(), h, m)
	select {
	case got := <-c.messageQueue:
		require.Equal(t, int64(1), got.retries)
	case <-time.After(5 * time.Second):
		t.Fatal("expected message to be requeued after the idle timeout")
	}

	// a canceled context (message processed) cancels the retry.
	h.config.idleTimeout = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.retryPostTimeout(ctx, h, m)
	require.Empty(t, c.messageQueue)
}

func TestInfMemoryConsumer_RetryMessageRespectsMaxRetries(t *testing.T) {
	b := infNewBroker(t, 4)
	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)

	c.retryMessage(memoryMessage{message: message{id: "m1", streamID: "s"}, retries: 3}, 3)
	select {
	case e := <-c.Errors():
		require.ErrorContains(t, e, "failed 3 retries")
	default:
		t.Fatal("expected a discard error")
	}

	c.retryMessage(memoryMessage{message: message{id: "m2", streamID: "s"}, retries: 0}, 3)
	select {
	case m := <-c.messageQueue:
		require.Equal(t, int64(1), m.retries)
		require.Equal(t, "m2", m.id)
	case <-time.After(5 * time.Second):
		t.Fatal("expected message to be requeued")
	}
}

func TestInfMemoryConsumer_DiscardsMessagesFromUnknownStreams(t *testing.T) {
	b := infNewBroker(t, 4)
	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		c.consume(ctx)
		close(done)
	}()

	c.messageQueue <- memoryMessage{message: message{id: "x", streamID: "ns:unknown"}}
	select {
	case e := <-c.Errors():
		require.ErrorContains(t, e, "doesn't belong to us")
	case <-time.After(5 * time.Second):
		t.Fatal("expected error for unknown stream")
	}
	cancel()
	<-done
}

func TestInfMemoryConsumer_PushErrorDoesNotBlockWhenFull(t *testing.T) {
	b := infNewBroker(t, 4)
	c, err := NewMemoryConsumer(b, "ns", "grp")
	require.NoError(t, err)
	for i := 0; i < cap(c.errorCh)+10; i++ {
		c.pushError(errors.New("x"))
	}
	require.Len(t, c.errorCh, cap(c.errorCh))
}

func TestInfDiscardedMessageError(t *testing.T) {
	plain := &DiscardedMessageError{MessageID: "1-0", StreamID: "s", RetryCount: 2}
	require.Contains(t, plain.Error(), "force acknowledged (discarded) message '1-0' (Retries: 2) in stream 's'")
	require.NoError(t, plain.Unwrap())

	ackErr := errors.New("ack failed")
	wrapped := &DiscardedMessageError{MessageID: "1-0", StreamID: "s", RetryCount: 2, AckErr: ackErr}
	require.Contains(t, wrapped.Error(), "failed to force acknowledge")
	require.Contains(t, wrapped.Error(), "ack failed")
	require.ErrorIs(t, wrapped, ackErr)
}

func TestInfTransposeStreamID(t *testing.T) {
	require.Equal(t, "ns:stream", transposeStreamID("ns", "stream"))
}

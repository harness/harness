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

package pubsub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestInMemory() *InMemory {
	return NewInMemory(WithSendTimeout(time.Second))
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func TestInMemory_PublishSubscribe(t *testing.T) {
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	ctx := context.Background()

	received := make(chan []byte, 1)
	sub := ps.Subscribe(ctx, "topic-a", func(payload []byte) error {
		received <- payload
		return nil
	})
	t.Cleanup(func() { _ = sub.Close() })

	if err := ps.Publish(ctx, "topic-a", []byte("hello")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case msg := <-received:
		if string(msg) != "hello" {
			t.Fatalf("got %q, want %q", msg, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive payload")
	}
}

func TestInMemory_FanOutToMultipleSubscribers(t *testing.T) {
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	ctx := context.Background()
	const numSubs = 5

	var received sync.WaitGroup
	received.Add(numSubs)
	var count atomic.Int32

	for range numSubs {
		sub := ps.Subscribe(ctx, "fanout", func(_ []byte) error {
			count.Add(1)
			received.Done()
			return nil
		})
		t.Cleanup(func() { _ = sub.Close() })
	}

	if err := ps.Publish(ctx, "fanout", []byte("x")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	done := make(chan struct{})
	go func() { received.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("only %d of %d subscribers received", count.Load(), numSubs)
	}
}

func TestInMemory_PublishToDifferentTopicIsNotDelivered(t *testing.T) {
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	ctx := context.Background()

	var got atomic.Int32
	sub := ps.Subscribe(ctx, "topic-a", func(_ []byte) error {
		got.Add(1)
		return nil
	})
	t.Cleanup(func() { _ = sub.Close() })

	if err := ps.Publish(ctx, "topic-b", []byte("x")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	if n := got.Load(); n != 0 {
		t.Fatalf("handler fired %d times, want 0", n)
	}
}

func TestInMemory_PublishWithNoSubscribers(t *testing.T) {
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	if err := ps.Publish(context.Background(), "nobody", []byte("x")); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func TestInMemory_ConsumerSubscribeAddsTopic(t *testing.T) {
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	ctx := context.Background()

	received := make(chan string, 2)
	sub := ps.Subscribe(ctx, "first", func(payload []byte) error {
		received <- string(payload)
		return nil
	})
	t.Cleanup(func() { _ = sub.Close() })

	if err := sub.Subscribe(ctx, "second"); err != nil {
		t.Fatalf("Consumer.Subscribe: %v", err)
	}

	if err := ps.Publish(ctx, "first", []byte("a")); err != nil {
		t.Fatalf("Publish first: %v", err)
	}
	if err := ps.Publish(ctx, "second", []byte("b")); err != nil {
		t.Fatalf("Publish second: %v", err)
	}

	got := map[string]bool{}
	for range 2 {
		select {
		case m := <-received:
			got[m] = true
		case <-time.After(time.Second):
			t.Fatalf("timed out, got %v", got)
		}
	}
	if !got["a"] || !got["b"] {
		t.Fatalf("missing payloads, got %v", got)
	}
}

func TestInMemory_SubscribeOnClosedSubscriberStillSucceeds(t *testing.T) {
	// The minimal implementation does not reject Subscribe/Unsubscribe after
	// Close — closed subscribers simply stop receiving (isClosed() filters them
	// in Publish). This pins that contract so it isn't regressed silently.
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	sub := ps.Subscribe(context.Background(), "topic", func(_ []byte) error { return nil })
	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Second Close reports ErrClosed.
	if err := sub.Close(); !errors.Is(err, ErrClosed) {
		t.Fatalf("second Close: got %v, want %v", err, ErrClosed)
	}
}

func TestInMemory_CloseStopsDelivery(t *testing.T) {
	ps := newTestInMemory()
	ctx := context.Background()

	var handled atomic.Int32
	sub := ps.Subscribe(ctx, "topic", func(_ []byte) error {
		handled.Add(1)
		return nil
	})

	if err := sub.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Publish after Close must not panic and must not deliver. Publish is
	// synchronous and isClosed() filters out closed subscribers, so no
	// goroutine is spawned and no message is queued.
	if err := ps.Publish(ctx, "topic", []byte("x")); err != nil {
		t.Fatalf("Publish after close: %v", err)
	}
	if n := handled.Load(); n != 0 {
		t.Fatalf("handler fired %d times after close, want 0", n)
	}
}

// TestInMemory_PublishCloseRace exercises the fix for the send-on-closed-channel
// race: repeatedly publish while a subscriber is concurrently closed. With the
// pre-fix code (Close did close(s.channel)) this could panic with "send on
// closed channel". It must now be safe.
func TestInMemory_PublishCloseRace(t *testing.T) {
	ps := newTestInMemory()
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	ctx := context.Background()

	const iterations = 200
	for range iterations {
		sub := ps.Subscribe(ctx, "race", func(_ []byte) error { return nil })

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 10 {
				_ = ps.Publish(ctx, "race", []byte("x"))
			}
		}()
		go func() {
			defer wg.Done()
			_ = sub.Close()
		}()
		wg.Wait()
	}
}

func TestInMemory_PublishRespectsContextCancellation(t *testing.T) {
	ps := NewInMemory(
		WithSendTimeout(time.Minute),
		WithSize(1),
	)
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	var startOnce sync.Once
	started := make(chan struct{})
	block := make(chan struct{})
	sub := ps.Subscribe(context.Background(), "slow", func(_ []byte) error {
		startOnce.Do(func() { close(started) })
		<-block
		return nil
	})
	t.Cleanup(func() { _ = sub.Close() })
	defer close(block)

	if err := ps.Publish(context.Background(), "slow", []byte("1")); err != nil {
		t.Fatalf("Publish 1: %v", err)
	}
	<-started
	if err := ps.Publish(context.Background(), "slow", []byte("2")); err != nil {
		t.Fatalf("Publish 2: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- ps.Publish(ctx, "slow", []byte("3")) }()

	select {
	case err := <-done:
		t.Fatalf("Publish returned before cancel — setup did not block as expected: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish did not return after context cancellation")
	}
}

func TestInMemory_NamespaceIsolation(t *testing.T) {
	ps := NewInMemory(WithSendTimeout(time.Second))
	t.Cleanup(func() { _ = ps.Close(context.Background()) })

	ctx := context.Background()

	var a, b atomic.Int32
	subA := ps.Subscribe(ctx, "topic", func(_ []byte) error {
		a.Add(1)
		return nil
	}, WithChannelNamespace("ns-a"))
	t.Cleanup(func() { _ = subA.Close() })

	subB := ps.Subscribe(ctx, "topic", func(_ []byte) error {
		b.Add(1)
		return nil
	}, WithChannelNamespace("ns-b"))
	t.Cleanup(func() { _ = subB.Close() })

	if err := ps.Publish(ctx, "topic", []byte("x"), WithPublishNamespace("ns-a")); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	waitFor(t, time.Second, func() bool { return a.Load() == 1 })
	if n := b.Load(); n != 0 {
		t.Fatalf("ns-b handler fired %d times, want 0", n)
	}
}

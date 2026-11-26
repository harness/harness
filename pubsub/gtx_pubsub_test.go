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

package pubsub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const gtxTimeout = 5 * time.Second

func gtxReceive(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(gtxTimeout):
		t.Fatal("timed out waiting for a message")
		return ""
	}
}

func gtxExpectNothing(t *testing.T, ch <-chan string) {
	t.Helper()

	select {
	case v := <-ch:
		t.Fatalf("unexpected message: %s", v)
	case <-time.After(100 * time.Millisecond):
	}
}

func gtxSubscribe(
	ctx context.Context, t *testing.T, ps *InMemory, topic string, opts ...SubscribeOption,
) <-chan string {
	t.Helper()

	received := make(chan string, 8)
	consumer := ps.Subscribe(ctx, topic, func(payload []byte) error {
		received <- string(payload)
		return nil
	}, opts...)
	t.Cleanup(func() { _ = consumer.Close() })

	return received
}

func TestGtxOptions(t *testing.T) {
	t.Run("pubsub options", func(t *testing.T) {
		var c Config
		for _, opt := range []Option{
			WithApp("app"), WithNamespace("ns"), WithHealthCheckInterval(time.Minute),
			WithSendTimeout(time.Hour), WithSize(7),
		} {
			opt.Apply(&c)
		}

		assert.Equal(t, Config{
			App: "app", Namespace: "ns", HealthInterval: time.Minute, SendTimeout: time.Hour, ChannelSize: 7,
		}, c)
	})

	t.Run("subscribe options", func(t *testing.T) {
		var c SubscribeConfig
		for _, opt := range []SubscribeOption{
			WithTopics("a", "b"), WithChannelNamespace("ns"), WithChannelHealthCheckInterval(time.Minute),
			WithChannelSendTimeout(time.Hour), WithChannelSize(9),
		} {
			opt.Apply(&c)
		}

		assert.Equal(t, SubscribeConfig{
			topics: []string{"a", "b"}, namespace: "ns", healthInterval: time.Minute, sendTimeout: time.Hour, channelSize: 9,
		}, c)
	})

	t.Run("publish options", func(t *testing.T) {
		var c PublishConfig
		WithPublishApp("app").Apply(&c)
		WithPublishNamespace("ns").Apply(&c)

		assert.Equal(t, PublishConfig{app: "app", namespace: "ns"}, c)
	})

	assert.Equal(t, "app:ns:topic", formatTopic("app", "ns", "topic"))
}

func TestGtxInMemory_Namespaces(t *testing.T) {
	ctx := context.Background()
	ps := NewInMemory(WithApp("app"), WithNamespace("ns"), WithSendTimeout(gtxTimeout))
	t.Cleanup(func() { _ = ps.Close(ctx) })

	defaultNS := gtxSubscribe(ctx, t, ps, "topic")
	otherNS := gtxSubscribe(ctx, t, ps, "topic", WithChannelNamespace("other"))

	require.NoError(t, ps.Publish(ctx, "topic", []byte("default")))
	assert.Equal(t, "default", gtxReceive(t, defaultNS))
	gtxExpectNothing(t, otherNS)

	require.NoError(t, ps.Publish(ctx, "topic", []byte("other"), WithPublishNamespace("other")))
	assert.Equal(t, "other", gtxReceive(t, otherNS))
	gtxExpectNothing(t, defaultNS)

	require.NoError(t, ps.Publish(ctx, "topic", []byte("foreign-app"), WithPublishApp("another-app")))
	gtxExpectNothing(t, defaultNS)
	gtxExpectNothing(t, otherNS)
}

func TestGtxInMemory_SubscribeAndUnsubscribe(t *testing.T) {
	ctx := context.Background()
	ps := NewInMemory(WithSendTimeout(gtxTimeout))
	t.Cleanup(func() { _ = ps.Close(ctx) })

	received := make(chan string, 8)
	consumer := ps.Subscribe(ctx, "first", func(payload []byte) error {
		received <- string(payload)
		return nil
	}, WithTopics("preset"), WithChannelSize(4), WithChannelSendTimeout(gtxTimeout))
	t.Cleanup(func() { _ = consumer.Close() })

	// topics provided via WithTopics are replaced by the explicit topic, because the latter is appended.
	require.NoError(t, ps.Publish(ctx, "first", []byte("1")))
	assert.Equal(t, "1", gtxReceive(t, received))

	require.NoError(t, ps.Publish(ctx, "second", []byte("nobody")))
	gtxExpectNothing(t, received)

	require.NoError(t, consumer.Subscribe(ctx, "second", "second", "first"))
	require.NoError(t, ps.Publish(ctx, "second", []byte("2")))
	assert.Equal(t, "2", gtxReceive(t, received))

	// a message is delivered once even if the topic was added several times.
	gtxExpectNothing(t, received)
}

func TestGtxInMemory_UnsubscribeFirstTopic(t *testing.T) {
	ctx := context.Background()
	ps := NewInMemory(WithSendTimeout(gtxTimeout))
	t.Cleanup(func() { _ = ps.Close(ctx) })

	received := make(chan string, 8)
	consumer := ps.Subscribe(ctx, "only", func(payload []byte) error {
		received <- string(payload)
		return nil
	})
	t.Cleanup(func() { _ = consumer.Close() })

	require.NoError(t, consumer.Unsubscribe(ctx, "only"))
	require.NoError(t, consumer.Unsubscribe(ctx, "never-subscribed"))

	require.NoError(t, ps.Publish(ctx, "only", []byte("dropped")))
	gtxExpectNothing(t, received)
}

func TestGtxInMemory_Close(t *testing.T) {
	ctx := context.Background()
	ps := NewInMemory(WithSendTimeout(gtxTimeout))

	received := gtxSubscribe(ctx, t, ps, "topic")
	consumer := ps.Subscribe(ctx, "other", func([]byte) error { return nil })

	require.NoError(t, consumer.Close())
	require.ErrorIs(t, consumer.Close(), ErrClosed)

	// closing the pubsub closes the remaining subscribers and ignores the already closed ones.
	require.NoError(t, ps.Close(ctx))
	require.NoError(t, ps.Close(ctx))

	require.NoError(t, ps.Publish(ctx, "topic", []byte("ignored")))
	gtxExpectNothing(t, received)
}

func TestGtxInMemory_PublishWithoutSubscribers(t *testing.T) {
	ps := NewInMemory()
	require.NoError(t, ps.Publish(context.Background(), "topic", []byte("nobody")))
}

func TestGtxInMemory_HandlerErrorDoesNotStopTheSubscriber(t *testing.T) {
	ctx := context.Background()
	ps := NewInMemory(WithSendTimeout(gtxTimeout))
	t.Cleanup(func() { _ = ps.Close(ctx) })

	var mu sync.Mutex
	calls := make(chan string, 4)
	consumer := ps.Subscribe(ctx, "topic", func(payload []byte) error {
		mu.Lock()
		defer mu.Unlock()
		calls <- string(payload)
		if string(payload) == "bad" {
			return assert.AnError
		}
		return nil
	})
	t.Cleanup(func() { _ = consumer.Close() })

	require.NoError(t, ps.Publish(ctx, "topic", []byte("bad")))
	assert.Equal(t, "bad", gtxReceive(t, calls))

	require.NoError(t, ps.Publish(ctx, "topic", []byte("good")))
	assert.Equal(t, "good", gtxReceive(t, calls))
}

func TestGtxInMemory_FullChannelDropsMessages(t *testing.T) {
	ctx := context.Background()
	ps := NewInMemory(WithSendTimeout(20*time.Millisecond), WithSize(1))
	t.Cleanup(func() { _ = ps.Close(ctx) })

	block := make(chan struct{})
	handled := make(chan string, 8)
	consumer := ps.Subscribe(ctx, "topic", func(payload []byte) error {
		handled <- string(payload)
		<-block
		return nil
	})
	t.Cleanup(func() { _ = consumer.Close() })

	// the handler blocks on the first message, the second fills the channel and the third is dropped.
	require.NoError(t, ps.Publish(ctx, "topic", []byte("1")))
	assert.Equal(t, "1", gtxReceive(t, handled))
	require.NoError(t, ps.Publish(ctx, "topic", []byte("2")))
	require.NoError(t, ps.Publish(ctx, "topic", []byte("3")))

	close(block)
	assert.Equal(t, "2", gtxReceive(t, handled))
	gtxExpectNothing(t, handled)
}

func TestGtxInMemory_PublishWithCanceledContext(t *testing.T) {
	ps := NewInMemory(WithSendTimeout(gtxTimeout), WithSize(1))

	block := make(chan struct{})
	consumer := ps.Subscribe(context.Background(), "topic", func([]byte) error {
		<-block
		return nil
	})
	t.Cleanup(func() {
		close(block)
		_ = consumer.Close()
	})

	// fill the handler and the channel.
	require.NoError(t, ps.Publish(context.Background(), "topic", []byte("1")))
	require.NoError(t, ps.Publish(context.Background(), "topic", []byte("2")))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	require.NoError(t, ps.Publish(ctx, "topic", []byte("3")))
	assert.Less(t, time.Since(start), gtxTimeout/2, "a canceled context must not wait for the send timeout")
}

func TestGtxProvidePubSub(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() })

	tests := []struct {
		name     string
		provider Provider
		want     any
	}{
		{name: "memory", provider: ProviderMemory, want: &InMemory{}},
		{name: "redis", provider: ProviderRedis, want: &Redis{}},
		{name: "empty falls back to memory", provider: "", want: &InMemory{}},
		{name: "unknown falls back to memory", provider: "gtx", want: &InMemory{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ps := ProvidePubSub(Config{
				App: "app", Namespace: "ns", Provider: test.provider,
				HealthInterval: time.Second, SendTimeout: time.Second, ChannelSize: 5,
			}, client)

			assert.IsType(t, test.want, ps)
		})
	}

	t.Run("configuration is passed on", func(t *testing.T) {
		ps, ok := ProvidePubSub(Config{
			App: "app", Namespace: "ns", Provider: ProviderMemory,
			HealthInterval: time.Second, SendTimeout: time.Minute, ChannelSize: 5,
		}, nil).(*InMemory)
		require.True(t, ok)

		assert.Equal(t, "app", ps.config.App)
		assert.Equal(t, "ns", ps.config.Namespace)
		assert.Equal(t, time.Minute, ps.config.SendTimeout)
		assert.Equal(t, 5, ps.config.ChannelSize)
	})
}

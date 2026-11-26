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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInfOptions(t *testing.T) {
	cfg := Config{}
	for _, o := range []Option{
		WithApp("app1"), WithNamespace("ns1"), WithHealthCheckInterval(time.Second),
		WithSendTimeout(2 * time.Second), WithSize(7),
	} {
		o.Apply(&cfg)
	}
	require.Equal(t, Config{
		App: "app1", Namespace: "ns1", HealthInterval: time.Second, SendTimeout: 2 * time.Second, ChannelSize: 7,
	}, cfg)

	sub := SubscribeConfig{}
	for _, o := range []SubscribeOption{
		WithTopics("a", "b"), WithChannelNamespace("cns"), WithChannelHealthCheckInterval(time.Minute),
		WithChannelSendTimeout(time.Hour), WithChannelSize(3),
	} {
		o.Apply(&sub)
	}
	require.Equal(t, SubscribeConfig{
		topics: []string{"a", "b"}, namespace: "cns", healthInterval: time.Minute, sendTimeout: time.Hour, channelSize: 3,
	}, sub)

	pub := PublishConfig{}
	for _, o := range []PublishOption{WithPublishApp("papp"), WithPublishNamespace("pns")} {
		o.Apply(&pub)
	}
	require.Equal(t, PublishConfig{app: "papp", namespace: "pns"}, pub)

	require.Equal(t, "a:n:t", formatTopic("a", "n", "t"))
}

func TestInfRedis_NewAppliesDefaultsAndOptions(t *testing.T) {
	fake := infNewRedisFake()
	r := NewRedis(fake.client())
	require.Equal(t, "app", r.config.App)
	require.Equal(t, "default", r.config.Namespace)
	require.Equal(t, 3*time.Second, r.config.HealthInterval)
	require.Equal(t, 100, r.config.ChannelSize)

	r = NewRedis(fake.client(), WithApp("x"), WithNamespace("y"), WithSize(5))
	require.Equal(t, "x", r.config.App)
	require.Equal(t, "y", r.config.Namespace)
	require.Equal(t, 5, r.config.ChannelSize)
}

func TestInfRedis_Publish(t *testing.T) {
	ctx := context.Background()
	fake := infNewRedisFake()
	r := NewRedis(fake.client(), WithApp("gitness"), WithNamespace("jobs"))

	require.NoError(t, r.Publish(ctx, "cancel", []byte("job-1")))
	require.NoError(t, r.Publish(ctx, "cancel", []byte("job-2"),
		WithPublishApp("other"), WithPublishNamespace("ns")))

	cmds := fake.commands()
	require.Equal(t, []string{"publish", "gitness:jobs:cancel", "job-1"}, cmds[0])
	require.Equal(t, []string{"publish", "other:ns:cancel", "job-2"}, cmds[1])

	fake.setPublishErr("ERR no can do")
	err := r.Publish(ctx, "cancel", []byte("x"))
	require.ErrorContains(t, err, "failed to write to pubsub topic 'gitness:jobs:cancel'")
}

func TestInfRedis_SubscribeReceivesMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fake := infNewRedisFake()
	r := NewRedis(fake.client(), WithApp("gitness"), WithNamespace("jobs"), WithSendTimeout(time.Minute))

	received := make(chan string, 4)
	failOnce := make(chan struct{}, 1)
	failOnce <- struct{}{}
	r.Subscribe(ctx, "events", func(payload []byte) error {
		received <- string(payload)
		select {
		case <-failOnce:
			return errors.New("handler failed") // only logged; the subscriber keeps running
		default:
			return nil
		}
	}, WithChannelNamespace("custom"), WithTopics("extra"))

	// both the explicit topics and the main topic are subscribed, using the channel namespace.
	subscribed := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case ch := <-fake.subscribed:
			subscribed[ch] = true
		case <-time.After(5 * time.Second):
			t.Fatal("subscription not registered")
		}
	}
	require.Equal(t, map[string]bool{"gitness:custom:extra": true, "gitness:custom:events": true}, subscribed)

	pubClient := NewRedis(fake.client(), WithApp("gitness"), WithNamespace("custom"))
	require.NoError(t, pubClient.Publish(ctx, "events", []byte("first")))
	require.NoError(t, pubClient.Publish(ctx, "extra", []byte("second")))
	require.Equal(t, "first", <-received)
	require.Equal(t, "second", <-received)

	require.NoError(t, r.Close(ctx))
}

func TestInfRedis_SubscriberTopicManagement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fake := infNewRedisFake()
	r := NewRedis(fake.client(), WithApp("a"), WithNamespace("n"), WithSendTimeout(time.Minute))

	consumer := r.Subscribe(ctx, "t1", func([]byte) error { return nil })
	<-fake.subscribed

	require.NoError(t, consumer.Subscribe(ctx, "t2", "t3"))
	require.NoError(t, consumer.Unsubscribe(ctx, "t2"))

	require.Eventually(t, func() bool {
		var seenSub, seenUnsub bool
		for _, c := range fake.commands() {
			if strings.EqualFold(c[0], "subscribe") && len(c) == 3 && c[1] == "a:n:t2" && c[2] == "a:n:t3" {
				seenSub = true
			}
			if strings.EqualFold(c[0], "unsubscribe") && len(c) == 2 && c[1] == "a:n:t2" {
				seenUnsub = true
			}
		}
		return seenSub && seenUnsub
	}, 5*time.Second, 5*time.Millisecond)

	require.NoError(t, consumer.Close())
	require.Error(t, consumer.Subscribe(ctx, "t4"), "closed subscribers reject operations")
	require.Error(t, consumer.Unsubscribe(ctx, "t4"))
}

func TestInfRedis_CloseWithoutSubscribers(t *testing.T) {
	r := NewRedis(infNewRedisFake().client())
	require.NoError(t, r.Close(context.Background()))
}

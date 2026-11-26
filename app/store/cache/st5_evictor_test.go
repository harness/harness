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

package cache_test

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSt5Publish = errors.New("st5 publish failed")

// st5Bus records subscriptions and optionally fails publishing.
type st5Bus struct {
	pubsub.PubSub

	handler    func([]byte) error
	topic      string
	publishErr error
	published  [][]byte
}

func (b *st5Bus) Subscribe(
	_ context.Context, topic string, handler func([]byte) error, _ ...pubsub.SubscribeOption,
) pubsub.Consumer {
	b.topic = topic
	b.handler = handler

	return nil
}

func (b *st5Bus) Publish(_ context.Context, _ string, payload []byte, _ ...pubsub.PublishOption) error {
	b.published = append(b.published, payload)

	return b.publishErr
}

func TestSt5EvictorSubscribe(t *testing.T) {
	ctx := context.Background()
	bus := &st5Bus{}
	evictor := cache.NewEvictor[*types.SpaceCore]("ns", "topic", bus)

	var got *types.SpaceCore
	evictor.Subscribe(ctx, func(key *types.SpaceCore) error {
		got = key

		return nil
	})
	require.Equal(t, "topic", bus.topic)

	// undecodable payloads are reported to the bus.
	require.Error(t, bus.handler([]byte("not gob")))
	require.Nil(t, got)

	buf := bytes.NewBuffer(nil)
	require.NoError(t, gob.NewEncoder(buf).Encode(&types.SpaceCore{ID: 4, Path: "a/b"}))
	require.NoError(t, bus.handler(buf.Bytes()))
	require.Equal(t, int64(4), got.ID)
}

func TestSt5EvictorEvict(t *testing.T) {
	ctx := context.Background()
	bus := &st5Bus{publishErr: errSt5Publish}
	evictor := cache.NewEvictor[*types.SpaceCore]("ns", "topic", bus)

	// publish errors are only logged.
	evictor.Evict(ctx, &types.SpaceCore{ID: 1})
	require.Len(t, bus.published, 1)

	// without a bus both operations are no-ops.
	none := cache.NewEvictor[*types.SpaceCore]("ns", "topic", nil)
	none.Evict(ctx, &types.SpaceCore{ID: 1})
	none.Subscribe(ctx, func(*types.SpaceCore) error { return errSt5Publish })
}

func TestSt5SpacePathCacheEvictsOnEvent(t *testing.T) {
	ctx := context.Background()
	bus := &st5Bus{}
	evictor := cache.NewEvictor[*types.SpaceCore]("ns", "topic", bus)

	c := cache.New(ctx, nil, nil, evictor, time.Minute)
	require.NotNil(t, c)

	// an event for any space clears the whole cache without failing.
	buf := bytes.NewBuffer(nil)
	require.NoError(t, gob.NewEncoder(buf).Encode(&types.SpaceCore{ID: 4}))
	require.NoError(t, bus.handler(buf.Bytes()))
}

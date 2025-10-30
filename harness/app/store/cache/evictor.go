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

package cache

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/harness/gitness/pubsub"

	"github.com/rs/zerolog/log"
)

type Evictor[T any] struct {
	nameSpace string
	topicName string
	bus       pubsub.PubSub
}

func NewEvictor[T any](
	nameSpace string,
	topicName string,
	bus pubsub.PubSub,
) Evictor[T] {
	return Evictor[T]{
		nameSpace: nameSpace,
		topicName: topicName,
		bus:       bus,
	}
}

func (e Evictor[T]) Subscribe(ctx context.Context, fn func(key T) error) {
	if e.bus == nil {
		return
	}

	_ = e.bus.Subscribe(ctx, e.topicName, func(payload []byte) error {
		var key T
		err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&key)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to process update event from type: %T", key)
			return err
		}

		return fn(key)
	}, pubsub.WithChannelNamespace(e.nameSpace))
}

func (e Evictor[T]) Evict(ctx context.Context, key T) {
	if e.bus == nil {
		return
	}

	buf := bytes.NewBuffer(nil)
	_ = gob.NewEncoder(buf).Encode(key)

	err := e.bus.Publish(
		ctx,
		e.topicName,
		buf.Bytes(),
		pubsub.WithPublishNamespace(e.nameSpace),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to publish update event for type %T", key)
	}
}

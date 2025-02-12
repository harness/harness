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

package refcache

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type SpaceFinder struct {
	spaceIDCache  SpaceIDCache
	spaceRefCache SpaceRefCache
	pubsub        pubsub.PubSub
}

func NewSpaceFinder(
	spaceIDCache SpaceIDCache,
	spaceRefCache SpaceRefCache,
	bus pubsub.PubSub,
) SpaceFinder {
	s := SpaceFinder{
		spaceIDCache:  spaceIDCache,
		spaceRefCache: spaceRefCache,
		pubsub:        bus,
	}

	ctx := context.Background()

	_ = bus.Subscribe(ctx, pubsubTopicSpaceUpdate, func(payload []byte) error {
		spaceID := int64(binary.LittleEndian.Uint64(payload))
		space, err := s.spaceIDCache.Get(ctx, spaceID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("spaceFinder: pubsub subscriber: failed to get space by ID from cache")
			return err
		}

		s.spaceRefCache.Evict(ctx, space.Path)
		s.spaceIDCache.Evict(ctx, spaceID)

		return nil
	}, pubsub.WithChannelNamespace(pubsubNamespace))

	return s
}

func (s SpaceFinder) MarkChanged(ctx context.Context, spaceID int64) {
	var buff [8]byte
	binary.LittleEndian.PutUint64(buff[:], uint64(spaceID))
	err := s.pubsub.Publish(ctx, pubsubTopicSpaceUpdate, buff[:], pubsub.WithPublishNamespace(pubsubNamespace))
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish space update event")
	}
}

func (s SpaceFinder) FindByID(ctx context.Context, spaceID int64) (*types.SpaceCore, error) {
	spaceCore, err := s.spaceIDCache.Get(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get space by ID from cache: %w", err)
	}

	return spaceCore, nil
}

func (s SpaceFinder) FindByRef(ctx context.Context, spacePath string) (*types.SpaceCore, error) {
	spaceID, err := strconv.ParseInt(spacePath, 10, 64)
	if err != nil || spaceID <= 0 {
		spaceID, err = s.spaceRefCache.Get(ctx, spacePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get space ID by space path from cache: %w", err)
		}
	}

	spaceCore, err := s.spaceIDCache.Get(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get space by ID from cache: %w", err)
	}

	return spaceCore, nil
}

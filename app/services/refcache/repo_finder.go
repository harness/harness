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

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type RepoFinder struct {
	repoStore     store.RepoStore
	spaceRefCache SpaceRefCache
	repoIDCache   RepoIDCache
	repoRefCache  RepoRefCache
	pubsub        pubsub.PubSub
}

func NewRepoFinder(
	repoStore store.RepoStore,
	spaceRefCache SpaceRefCache,
	repoIDCache RepoIDCache,
	repoRefCache RepoRefCache,
	bus pubsub.PubSub,
) RepoFinder {
	r := RepoFinder{
		repoStore:     repoStore,
		spaceRefCache: spaceRefCache,
		repoIDCache:   repoIDCache,
		repoRefCache:  repoRefCache,
		pubsub:        bus,
	}

	ctx := context.Background()

	_ = bus.Subscribe(ctx, pubsubTopicRepoUpdate, func(payload []byte) error {
		repoID := int64(binary.LittleEndian.Uint64(payload))
		repo, err := r.repoIDCache.Get(ctx, repoID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("repoFinder: pubsub subscriber: failed to get repo by ID from cache")
			return err
		}

		r.repoRefCache.Evict(ctx, RepoCacheKey{spaceID: repo.ParentID, repoIdentifier: repo.Identifier})
		r.repoIDCache.Evict(ctx, repoID)

		return nil
	}, pubsub.WithChannelNamespace(pubsubNamespace))

	return r
}

func (r RepoFinder) MarkChanged(ctx context.Context, repoID int64) {
	var buff [8]byte
	binary.LittleEndian.PutUint64(buff[:], uint64(repoID))
	err := r.pubsub.Publish(ctx, pubsubTopicRepoUpdate, buff[:], pubsub.WithPublishNamespace(pubsubNamespace))
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish repo update event")
	}
}

func (r RepoFinder) FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error) {
	return r.repoIDCache.Get(ctx, repoID)
}

func (r RepoFinder) FindByRef(ctx context.Context, repoRef string) (*types.RepositoryCore, error) {
	repoID, err := strconv.ParseInt(repoRef, 10, 64)
	if err != nil || repoID <= 0 {
		spacePath, repoIdentifier, err := paths.DisectLeaf(repoRef)
		if err != nil {
			return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
		}

		spaceID, err := r.spaceRefCache.Get(ctx, spacePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get space from cache: %w", err)
		}

		repoID, err = r.repoRefCache.Get(ctx, RepoCacheKey{spaceID: spaceID, repoIdentifier: repoIdentifier})
		if err != nil {
			return nil, fmt.Errorf("failed to get repository ID by space ID and repo identifier: %w", err)
		}
	}

	repoCore, err := r.repoIDCache.Get(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository by ID: %w", err)
	}

	return repoCore, nil
}

func (r RepoFinder) FindDeletedByRef(ctx context.Context, repoRef string, deleted int64) (*types.Repository, error) {
	repoID, err := strconv.ParseInt(repoRef, 10, 64)
	if err == nil && repoID >= 0 {
		repo, err := r.repoStore.FindDeleted(ctx, repoID, &deleted)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository by ID: %w", err)
		}

		return repo, nil
	}

	spaceRef, repoIdentifier, err := paths.DisectLeaf(repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to disect extract repo idenfifier from path: %w", err)
	}

	spaceID, err := r.spaceRefCache.Get(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get space ID by space ref from cache: %w", err)
	}

	repo, err := r.repoStore.FindDeletedByUID(ctx, spaceID, repoIdentifier, deleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted repository ID by space ID and repo identifier: %w", err)
	}

	return repo, nil
}

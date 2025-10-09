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
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

func NewSpaceIDCache(
	appCtx context.Context,
	spaceStore store.SpaceStore,
	evictor Evictor[*types.SpaceCore],
	dur time.Duration,
) store.SpaceIDCache {
	c := cache.New[int64, *types.SpaceCore](spaceIDCacheGetter{spaceStore: spaceStore}, dur)

	// In case when a space is updated, we should remove from the cache the space and all of its subspaces.
	// Rather than to dig through the cache to find all subspaces, it's simpler to clear the cache.
	// Update of a space core (space identifier or space path) is a rare operation, so clearing cache is justified.
	evictor.Subscribe(appCtx, func(*types.SpaceCore) error {
		c.EvictAll(appCtx)
		return nil
	})

	return c
}

type spaceIDCacheGetter struct {
	spaceStore store.SpaceStore
}

func (g spaceIDCacheGetter) Find(ctx context.Context, spaceID int64) (*types.SpaceCore, error) {
	space, err := g.spaceStore.Find(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by id: %w", err)
	}

	return space.Core(), nil
}

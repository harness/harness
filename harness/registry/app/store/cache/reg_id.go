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

	cache2 "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
)

func NewRegistryIDCache(
	appCtx context.Context,
	regSource store.RegistryRepository,
	evictorRepo cache2.Evictor[*types.Registry],
	dur time.Duration,
) store.RegistryIDCache {
	c := cache.New[int64, *types.Registry](registryIDCacheGetter{regSource: regSource}, dur)

	evictorRepo.Subscribe(appCtx, func(repoCore *types.Registry) error {
		c.Evict(appCtx, repoCore.ID)
		return nil
	})

	return c
}

type registryIDCacheGetter struct {
	regSource store.RegistryRepository
}

func (c registryIDCacheGetter) Find(ctx context.Context, repoID int64) (*types.Registry, error) {
	repo, err := c.regSource.Get(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ID: %w", err)
	}

	return repo, nil
}

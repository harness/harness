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

func NewRegistryUUIDCache(
	appCtx context.Context,
	regSource store.RegistryRepository,
	evictorRepo cache2.Evictor[*types.Registry],
	dur time.Duration,
) store.RegistryUUIDCache {
	c := cache.New[string, *types.Registry](registryUUIDCacheGetter{regSource: regSource}, dur)

	evictorRepo.Subscribe(appCtx, func(repoCore *types.Registry) error {
		c.Evict(appCtx, repoCore.UUID)
		return nil
	})

	return c
}

type registryUUIDCacheGetter struct {
	regSource store.RegistryRepository
}

func (c registryUUIDCacheGetter) Find(ctx context.Context, repoUUID string) (*types.Registry, error) {
	repo, err := c.regSource.GetByUUID(ctx, repoUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by UUID: %w", err)
	}

	return repo, nil
}

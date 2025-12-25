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

func NewRegistryRootRefCache(
	appCtx context.Context,
	regSource store.RegistryRepository,
	evictorReg cache2.Evictor[*types.Registry],
	dur time.Duration,
) store.RegistryRootRefCache {
	c := cache.New[types.RegistryRootRefCacheKey, int64](registryRootRefCacheGetter{
		regSource: regSource,
	}, dur)
	evictorReg.Subscribe(appCtx, func(key *types.Registry) error {
		c.Evict(appCtx, types.RegistryRootRefCacheKey{
			RootParentID:       key.RootParentID,
			RegistryIdentifier: key.Name,
		})
		return nil
	})

	return c
}

type registryRootRefCacheGetter struct {
	regSource store.RegistryRepository
}

func (c registryRootRefCacheGetter) Find(ctx context.Context, key types.RegistryRootRefCacheKey) (
	int64,
	error,
) {
	repo, err := c.regSource.GetByRootParentIDAndName(ctx, key.RootParentID, key.RegistryIdentifier)
	if err != nil {
		return -1, fmt.Errorf("failed to find repo by %d:%s %w", key.RootParentID, key.RegistryIdentifier, err)
	}

	return repo.ID, nil
}

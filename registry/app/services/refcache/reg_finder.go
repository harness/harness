//  Copyright 2023 Harness, Inc.
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
	"fmt"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
)

type RegistryFinder interface {
	MarkChanged(ctx context.Context, reg *types.Registry)
	FindByID(ctx context.Context, repoID int64) (*types.Registry, error)
	FindByRootRef(ctx context.Context, rootParentRef string, regIdentifier string) (
		*types.Registry,
		error,
	)
	FindByRootParentID(ctx context.Context, rootParentID int64, regIdentifier string) (
		*types.Registry,
		error,
	)
	Update(ctx context.Context, registry *types.Registry) (err error)
	Delete(ctx context.Context, parentID int64, name string) (err error)
}

type registryFinder struct {
	inner               store.RegistryRepository
	regIDCache          store.RegistryIDCache
	regRootRefCache     store.RegistryRootRefCache
	spaceFinder         refcache.SpaceFinder
	evictor             cache.Evictor[*types.Registry]
	upstreamProxyFinder UpstreamProxyFinder
}

func NewRegistryFinder(
	registryRepository store.RegistryRepository,
	regIDCache store.RegistryIDCache,
	regRootRefCache store.RegistryRootRefCache,
	evictor cache.Evictor[*types.Registry],
	spaceFinder refcache.SpaceFinder,
	upstreamProxyFinder UpstreamProxyFinder,
) RegistryFinder {
	return registryFinder{
		inner:               registryRepository,
		regIDCache:          regIDCache,
		regRootRefCache:     regRootRefCache,
		evictor:             evictor,
		spaceFinder:         spaceFinder,
		upstreamProxyFinder: upstreamProxyFinder,
	}
}

func (r registryFinder) MarkChanged(ctx context.Context, reg *types.Registry) {
	r.evictor.Evict(ctx, reg)
	r.evictUpstreamProxyCache(ctx, reg.ID)
}

func (r registryFinder) evictUpstreamProxyCache(ctx context.Context, registryID int64) {
	upstreamProxy, err := r.upstreamProxyFinder.Get(ctx, registryID)
	if err == nil && upstreamProxy != nil {
		r.upstreamProxyFinder.MarkChanged(ctx, upstreamProxy)
	}
}

func (r registryFinder) FindByID(ctx context.Context, repoID int64) (*types.Registry, error) {
	return r.regIDCache.Get(ctx, repoID)
}

func (r registryFinder) FindByRootRef(ctx context.Context, rootParentRef string, regIdentifier string) (
	*types.Registry,
	error,
) {
	space, err := r.spaceFinder.FindByRef(ctx, rootParentRef)
	if err != nil {
		return nil, fmt.Errorf("error finding space by root-ref: %w", err)
	}
	return r.FindByRootParentID(ctx, space.ID, regIdentifier)
}

func (r registryFinder) FindByRootParentID(ctx context.Context, rootParentID int64, regIdentifier string) (
	*types.Registry,
	error,
) {
	registryID, err := r.regRootRefCache.Get(ctx,
		types.RegistryRootRefCacheKey{RootParentID: rootParentID, RegistryIdentifier: regIdentifier})
	if err != nil {
		return nil, fmt.Errorf("error finding registry by root-ref: %w", err)
	}
	return r.regIDCache.Get(ctx, registryID)
}

func (r registryFinder) Update(ctx context.Context, registry *types.Registry) (err error) {
	err = r.inner.Update(ctx, registry)
	if err == nil {
		r.MarkChanged(ctx, registry)
	}
	return err
}

func (r registryFinder) Delete(ctx context.Context, parentID int64, name string) (err error) {
	registry, err := r.inner.GetByParentIDAndName(ctx, parentID, name)
	if err != nil {
		return fmt.Errorf("error finding registry by parent-ref: %w", err)
	}
	err = r.inner.Delete(ctx, parentID, name)
	if err == nil {
		r.MarkChanged(ctx, registry)
	}
	return err
}

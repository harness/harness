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

	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
)

type UpstreamProxyFinder interface {
	MarkChanged(ctx context.Context, upstreamProxyConfig *types.UpstreamProxy)
	Get(ctx context.Context, registryID int64) (*types.UpstreamProxy, error)
	Update(ctx context.Context, upstreamProxyConfig *types.UpstreamProxyConfig) error
}

type upstreamProxyFinder struct {
	inner                        store.UpstreamProxyConfigRepository
	upstreamProxyRegistryIDCache store.UpstreamProxyRegistryIDCache
	evictor                      cache.Evictor[*types.UpstreamProxy]
}

func NewUpstreamProxyFinder(
	upstreamProxyRepository store.UpstreamProxyConfigRepository,
	upstreamProxyRegistryIDCache store.UpstreamProxyRegistryIDCache,
	evictor cache.Evictor[*types.UpstreamProxy],
) UpstreamProxyFinder {
	return &upstreamProxyFinder{
		inner:                        upstreamProxyRepository,
		upstreamProxyRegistryIDCache: upstreamProxyRegistryIDCache,
		evictor:                      evictor,
	}
}

func (u *upstreamProxyFinder) MarkChanged(ctx context.Context, upstreamProxyConfig *types.UpstreamProxy) {
	u.evictor.Evict(ctx, upstreamProxyConfig)
}

func (u *upstreamProxyFinder) Get(
	ctx context.Context,
	registryID int64,
) (*types.UpstreamProxy, error) {
	return u.upstreamProxyRegistryIDCache.Get(ctx, registryID)
}

func (u *upstreamProxyFinder) Update(
	ctx context.Context,
	upstreamProxyConfig *types.UpstreamProxyConfig,
) error {
	err := u.inner.Update(ctx, upstreamProxyConfig)
	if err != nil {
		return err
	}

	upstreamProxy, err := u.upstreamProxyRegistryIDCache.Get(ctx, upstreamProxyConfig.RegistryID)

	if err == nil {
		u.MarkChanged(ctx, upstreamProxy)
	}
	return err
}

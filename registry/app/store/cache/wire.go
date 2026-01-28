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
	"time"

	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/google/wire"
)

const (
	registryCacheDuration      = 15 * time.Minute
	upstreamProxyCacheDuration = 15 * time.Minute
)

const (
	pubsubNamespace                = "cache-evictor"
	pubsubTopicRegCoreUpdate       = "reg-core-update"
	pubsubTopicUpstreamProxyUpdate = "upstream-proxy-update"
)

func ProvideEvictorRegistryCore(pubsub pubsub.PubSub) cache.Evictor[*types.Registry] {
	return cache.NewEvictor[*types.Registry](pubsubNamespace, pubsubTopicRegCoreUpdate, pubsub)
}

func ProvideEvictorUpstreamProxy(pubsub pubsub.PubSub) cache.Evictor[*types.UpstreamProxy] {
	return cache.NewEvictor[*types.UpstreamProxy](pubsubNamespace, pubsubTopicUpstreamProxyUpdate, pubsub)
}

func ProvideRegistryIDCache(
	appCtx context.Context,
	regSource store.RegistryRepository,
	evictorRepo cache.Evictor[*types.Registry],
) store.RegistryIDCache {
	return NewRegistryIDCache(appCtx, regSource, evictorRepo, registryCacheDuration)
}

func ProvideRegRootRefCache(
	appCtx context.Context,
	regSource store.RegistryRepository,
	evictorRepo cache.Evictor[*types.Registry],
) store.RegistryRootRefCache {
	return NewRegistryRootRefCache(appCtx, regSource, evictorRepo, registryCacheDuration)
}

func ProvideUpstreamProxyRegistryIDCache(
	appCtx context.Context,
	upstreamProxySource store.UpstreamProxyConfigRepository,
	evictor cache.Evictor[*types.UpstreamProxy],
) store.UpstreamProxyRegistryIDCache {
	return NewUpstreamProxyRegistryIDCache(appCtx, upstreamProxySource, evictor, upstreamProxyCacheDuration)
}

var WireSet = wire.NewSet(
	ProvideEvictorRegistryCore,
	ProvideEvictorUpstreamProxy,
	ProvideRegRootRefCache,
	ProvideRegistryIDCache,
	ProvideUpstreamProxyRegistryIDCache,
)

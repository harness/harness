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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSetSpace provides a wire set for this package.
var WireSetSpace = wire.NewSet(
	ProvidePrincipalInfoCache,
	ProvideEvictorSpaceCore,
	ProvideSpaceIDCache,
	ProvideSpacePathCache,
	ProvideSpaceCaseInsensitiveCache,
	ProvideInfraProviderResourceCache,
)

// WireSetRepo provides a repository related wire set for this package.
var WireSetRepo = wire.NewSet(
	ProvideEvictorRepositoryCore,
	ProvideRepoIDCache,
	ProvideRepoRefCache,
)

const (
	principalInfoCacheDuration = 30 * time.Second
	spaceCacheDuration         = 15 * time.Minute
	repositoryCacheDuration    = 15 * time.Minute
)

const (
	pubsubNamespace            = "cache-evictor"
	pubsubTopicSpaceCoreUpdate = "space-core-update"
	pubsubTopicRepoCoreUpdate  = "repo-core-update"
)

func ProvideEvictorSpaceCore(pubsub pubsub.PubSub) Evictor[*types.SpaceCore] {
	return NewEvictor[*types.SpaceCore](pubsubNamespace, pubsubTopicSpaceCoreUpdate, pubsub)
}

func ProvideEvictorRepositoryCore(pubsub pubsub.PubSub) Evictor[*types.RepositoryCore] {
	return NewEvictor[*types.RepositoryCore](pubsubNamespace, pubsubTopicRepoCoreUpdate, pubsub)
}

// ProvidePrincipalInfoCache provides a cache for storing types.PrincipalInfo objects.
func ProvidePrincipalInfoCache(getter store.PrincipalInfoView) store.PrincipalInfoCache {
	return cache.NewExtended[int64, *types.PrincipalInfo](getter, principalInfoCacheDuration)
}

func ProvideSpaceIDCache(
	appCtx context.Context,
	spaceStore store.SpaceStore,
	evictor Evictor[*types.SpaceCore],
) store.SpaceIDCache {
	return NewSpaceIDCache(appCtx, spaceStore, evictor, spaceCacheDuration)
}

// ProvideSpacePathCache provides a cache for storing routing paths and their types.SpacePath objects.
func ProvideSpacePathCache(
	appCtx context.Context,
	pathStore store.SpacePathStore,
	evictor Evictor[*types.SpaceCore],
	spacePathTransformation store.SpacePathTransformation,
) store.SpacePathCache {
	return New(appCtx, pathStore, spacePathTransformation, evictor, spaceCacheDuration)
}

// ProvideSpaceCaseInsensitiveCache provides a cache for case-insensitive space lookups.
func ProvideSpaceCaseInsensitiveCache(
	appCtx context.Context,
	spaceStore store.SpaceStore,
	evictor Evictor[*types.SpaceCore],
) store.SpaceCaseInsensitiveCache {
	return NewSpaceCaseInsensitiveCache(
		appCtx,
		spaceStore,
		evictor,
		spaceCacheDuration,
	)
}

func ProvideRepoIDCache(
	appCtx context.Context,
	repoStore store.RepoStore,
	evictorSpace Evictor[*types.SpaceCore],
	evictorRepo Evictor[*types.RepositoryCore],
) store.RepoIDCache {
	return NewRepoIDCache(appCtx, repoStore, evictorSpace, evictorRepo, repositoryCacheDuration)
}

func ProvideRepoRefCache(
	appCtx context.Context,
	repoStore store.RepoStore,
	evictorSpace Evictor[*types.SpaceCore],
	evictorRepo Evictor[*types.RepositoryCore],
) store.RepoRefCache {
	return NewRepoRefCache(appCtx, repoStore, evictorSpace, evictorRepo, repositoryCacheDuration)
}

// ProvideInfraProviderResourceCache provides a cache for storing types.InfraProviderResource objects.
func ProvideInfraProviderResourceCache(getter store.InfraProviderResourceView) store.InfraProviderResourceCache {
	return cache.NewExtended[int64, *types.InfraProviderResource](getter, 5*time.Minute)
}

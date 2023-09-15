// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package cache

import (
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvidePrincipalInfoCache,
	ProvidePathCache,
	ProvideRepoGitInfoCache,
)

// ProvidePrincipalInfoCache provides a cache for storing types.PrincipalInfo objects.
func ProvidePrincipalInfoCache(getter store.PrincipalInfoView) store.PrincipalInfoCache {
	return cache.NewExtended[int64, *types.PrincipalInfo](getter, 30*time.Second)
}

// ProvidePathCache provides a cache for storing routing paths and their types.SpacePath objects.
func ProvidePathCache(
	pathStore store.SpacePathStore,
	spacePathTransformation store.SpacePathTransformation,
) store.SpacePathCache {
	return &pathCache{
		inner: cache.New[string, *types.SpacePath](
			&pathCacheGetter{
				spacePathStore: pathStore,
			},
			1*time.Minute),
		spacePathTransformation: spacePathTransformation,
	}
}

// ProvideRepoGitInfoCache provides a cache for storing types.RepositoryGitInfo objects.
func ProvideRepoGitInfoCache(getter store.RepoGitInfoView) store.RepoGitInfoCache {
	return cache.New[int64, *types.RepositoryGitInfo](getter, 15*time.Minute)
}

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
)

// ProvidePrincipalInfoCache provides a cache for storing types.PrincipalInfo objects.
func ProvidePrincipalInfoCache(getter store.PrincipalInfoView) store.PrincipalInfoCache {
	return cache.NewExtended[int64, *types.PrincipalInfo](getter, 30*time.Second)
}

// ProvidePathCache provides a cache for storing routing paths and their types.Path objects.
func ProvidePathCache(pathStore store.PathStore, pathTransformation store.PathTransformation) store.PathCache {
	return &pathCache{
		inner: cache.New[string, *pathCacheEntry](
			&pathCacheGetter{
				pathStore: pathStore,
			},
			60*time.Second),
		pathTransformation: pathTransformation,
	}
}

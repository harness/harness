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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvidePrincipalInfoCache,
	ProvidePathCache,
	ProvideInfraProviderResourceCache,
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
	return New(pathStore, spacePathTransformation)
}

// ProvideInfraProviderResourceCache provides a cache for storing types.InfraProviderResource objects.
func ProvideInfraProviderResourceCache(getter store.InfraProviderResourceView) store.InfraProviderResourceCache {
	return cache.NewExtended[int64, *types.InfraProviderResource](getter, 5*time.Minute)
}

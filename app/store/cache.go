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

package store

import (
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

type (
	// PrincipalInfoCache caches principal IDs to principal info.
	PrincipalInfoCache cache.ExtendedCache[int64, *types.PrincipalInfo]

	// SpaceIDCache holds the immutable part of Space objects fetched by space ID.
	SpaceIDCache cache.Cache[int64, *types.SpaceCore]

	// SpacePathCache caches a raw path to a space path.
	SpacePathCache cache.Cache[string, *types.SpacePath]

	// SpaceCaseInsensitiveCache caches case-insensitive space references to space IDs.
	SpaceCaseInsensitiveCache cache.Cache[string, int64]

	// RepoIDCache holds Repository objects fetched by their ID.
	RepoIDCache cache.Cache[int64, *types.RepositoryCore]

	// RepoRefCache holds repository IDs fetched by spaceID and repository identifier.
	RepoRefCache cache.Cache[types.RepoCacheKey, int64]

	// InfraProviderResourceCache caches infraprovider resourceIDs to infraprovider resource.
	InfraProviderResourceCache cache.ExtendedCache[int64, *types.InfraProviderResource]
)

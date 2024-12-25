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

package refcache

import (
	"github.com/harness/gitness/app/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideSpaceCache,
	ProvideRepoFinder,
)

// ProvideSpaceCache provides a cache for Space objects fetched by space reference.
func ProvideSpaceCache(
	spacePathCache store.SpacePathStore,
	spaceStore store.SpaceStore,
	pathTransform store.SpacePathTransformation,
) SpaceCache {
	return NewSpaceCache(spacePathCache, spaceStore, pathTransform)
}

// ProvideRepoFinder provides a repository finder that finds repositories by their path.
func ProvideRepoFinder(
	repoStore store.RepoStore,
	spaceCache SpaceCache,
) RepoFinder {
	return NewRepoFinder(repoStore, spaceCache)
}

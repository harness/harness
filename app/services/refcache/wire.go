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
	"github.com/harness/gitness/pubsub"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideSpaceIDCache,
	ProvideSpaceRefCache,
	ProvideSpaceFinder,
	ProvideRepoIDCache,
	ProvideRepoRefCache,
	ProvideRepoFinder,
)

func ProvideSpaceIDCache(
	spaceStore store.SpaceStore,
) SpaceIDCache {
	return NewSpaceIDCache(spaceStore)
}

func ProvideSpaceRefCache(
	spacePathStore store.SpacePathStore,
	pathTransform store.SpacePathTransformation,
) SpaceRefCache {
	return NewSpaceRefCache(spacePathStore, pathTransform)
}

func ProvideSpaceFinder(
	spaceIDCache SpaceIDCache,
	spaceRefCache SpaceRefCache,
	pubsub pubsub.PubSub,
) SpaceFinder {
	return NewSpaceFinder(spaceIDCache, spaceRefCache, pubsub)
}

func ProvideRepoIDCache(
	repoStore store.RepoStore,
) RepoIDCache {
	return NewRepoIDCache(repoStore)
}

func ProvideRepoRefCache(
	repoStore store.RepoStore,
) RepoRefCache {
	return NewRepoRefCache(repoStore)
}

func ProvideRepoFinder(
	repoStore store.RepoStore,
	spaceRefCache SpaceRefCache,
	repoIDCache RepoIDCache,
	repoRefCache RepoRefCache,
	pubsub pubsub.PubSub,
) RepoFinder {
	return NewRepoFinder(repoStore, spaceRefCache, repoIDCache, repoRefCache, pubsub)
}

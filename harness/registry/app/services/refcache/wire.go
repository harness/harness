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
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/google/wire"
)

func ProvideRegistryFinder(
	registryRepository store.RegistryRepository,
	regIDCache store.RegistryIDCache,
	regRootRefCache store.RegistryRootRefCache,
	evictor cache.Evictor[*types.Registry],
	spaceFinder refcache.SpaceFinder,
) RegistryFinder {
	return NewRegistryFinder(registryRepository, regIDCache, regRootRefCache, evictor, spaceFinder)
}

var WireSet = wire.NewSet(
	ProvideRegistryFinder,
)

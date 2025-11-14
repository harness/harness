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

package quarantine

import (
	"context"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/registry/app/store"

	"github.com/google/wire"
)

const (
	quarantineCacheDuration = 5 * time.Minute
)

// WireSet provides the quarantine service and finder.
var WireSet = wire.NewSet(
	ProvideService,
	ProvideQuarantineCache,
	ProvideFinder,
)

// ProvideService provides the quarantine service (repository layer only).
func ProvideService(
	quarantineRepo store.QuarantineArtifactRepository,
	manifestRepo store.ManifestRepository,
) *Service {
	return NewService(quarantineRepo, manifestRepo)
}

// ProvideQuarantineCache provides the quarantine cache.
func ProvideQuarantineCache(
	_ context.Context,
	service *Service,
) cache.Cache[CacheKey, bool] {
	getter := quarantineCacheGetter{service: service}
	return cache.New[CacheKey, bool](getter, quarantineCacheDuration)
}

// ProvideFinder provides the quarantine finder (with caching).
func ProvideFinder(
	service *Service,
	quarantineCache cache.Cache[CacheKey, bool],
) Finder {
	return NewFinder(service, quarantineCache)
}

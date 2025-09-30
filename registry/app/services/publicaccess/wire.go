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

package publicaccess

import (
	"context"
	"time"

	"github.com/harness/gitness/app/services/publicaccess"
	cache2 "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"

	"github.com/google/wire"
)

const (
	registryPublicAccessCacheDuration = 10 * time.Minute
)

const (
	pubsubNamespace                  = "cache-evictor"
	pubsubTopicRegPublicAccessUpdate = "reg-public-access-update"
)

func ProvideEvictorPublicAccess(pubsub pubsub.PubSub) cache2.Evictor[*CacheKey] {
	return cache2.NewEvictor[*CacheKey](pubsubNamespace, pubsubTopicRegPublicAccessUpdate, pubsub)
}

func ProvidePublicAccessCache(
	appCtx context.Context,
	publicAccessService publicaccess.Service,
	evictor cache2.Evictor[*CacheKey],
) Cache {
	return NewPublicAccessCacheCache(appCtx, publicAccessService, evictor, registryPublicAccessCacheDuration)
}

func ProvideRegistryPublicAccess(
	publicAccessService publicaccess.Service,
	publicAccessCache Cache,
	evictor cache2.Evictor[*CacheKey],
) CacheService {
	return NewPublicAccessService(
		publicAccessService,
		publicAccessCache,
		evictor,
	)
}

var WireSet = wire.NewSet(
	ProvideEvictorPublicAccess,
	ProvidePublicAccessCache,
	ProvideRegistryPublicAccess,
)

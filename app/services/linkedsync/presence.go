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

package linkedsync

import (
	"context"
	"time"

	"github.com/harness/gitness/cache"
)

const (
	// presenceCacheSize bounds memory; root spaces are evicted LRU, and a miss
	// costs one existence query.
	presenceCacheSize = 4096

	// presenceCacheTTL is deliberately short. A newly linked repo only starts
	// mirroring once the cached "no linked repos" answer expires, and the
	// linked-repo reconciler (job_sync_linked_repositories, every 4h) is the
	// backstop for anything missed inside that window.
	presenceCacheTTL = 30 * time.Second
)

// LinkedRepoPresenceStore is the narrow slice of the linked repo store needed
// to answer "does this root space use linked repos at all?".
type LinkedRepoPresenceStore interface {
	ExistsByRootSpace(ctx context.Context, rootSpaceIdentifier string) (bool, error)
}

// PresenceCache answers whether a root space owns any linked repo, memoized per
// root space. Inbound webhook consumers read a platform-wide stream, so without
// this the per-event linked-repo lookup runs for every root space on the platform
// whether or not it uses linked repos at all.
type PresenceCache struct {
	cache *cache.LRUCache[string, bool]
}

func NewPresenceCache(store LinkedRepoPresenceStore) *PresenceCache {
	return &PresenceCache{
		cache: cache.NewLRU[string, bool](
			presenceGetter{store: store}, presenceCacheSize, presenceCacheTTL,
		),
	}
}

// HasAny reports whether the root space owns at least one linked repo. Both
// answers are cached; errors are not, so a transient database failure is
// retried on the next event rather than read as "no linked repos".
func (p *PresenceCache) HasAny(ctx context.Context, rootSpaceIdentifier string) (bool, error) {
	return p.cache.Get(ctx, rootSpaceIdentifier)
}

// presenceGetter adapts the store to cache.Getter.
type presenceGetter struct {
	store LinkedRepoPresenceStore
}

func (g presenceGetter) Find(ctx context.Context, rootSpaceIdentifier string) (bool, error) {
	return g.store.ExistsByRootSpace(ctx, rootSpaceIdentifier)
}

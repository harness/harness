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
	"context"
	"testing"

	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type ra5UpstreamRepo struct {
	store.UpstreamProxyConfigRepository
	err error
}

func (r ra5UpstreamRepo) Update(_ context.Context, _ *types.UpstreamProxyConfig) error { return r.err }

func TestRa5UpstreamProxyFinder(t *testing.T) {
	ctx := context.Background()
	bus := &ra5Bus{}
	proxies := map[int64]*types.UpstreamProxy{1: {ID: 5, RegistryID: 1}}
	newFinder := func(repoErr error) UpstreamProxyFinder {
		return NewUpstreamProxyFinder(
			ra5UpstreamRepo{err: repoErr},
			ra5Cache[int64, *types.UpstreamProxy]{items: proxies},
			appcache.NewEvictor[*types.UpstreamProxy]("ns", "topic", bus),
		)
	}

	f := newFinder(nil)
	p, err := f.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(5), p.ID)
	_, err = f.Get(ctx, 2)
	require.Error(t, err)

	f.MarkChanged(ctx, p)
	require.Len(t, bus.topics, 1)

	require.NoError(t, f.Update(ctx, &types.UpstreamProxyConfig{RegistryID: 1}))
	require.Len(t, bus.topics, 2)

	// Cache miss after a successful update surfaces the cache error and does not evict.
	require.Error(t, f.Update(ctx, &types.UpstreamProxyConfig{RegistryID: 2}))
	require.Len(t, bus.topics, 2)

	require.ErrorIs(t, newFinder(errRa5).Update(ctx, &types.UpstreamProxyConfig{RegistryID: 1}), errRa5)
	require.Len(t, bus.topics, 2)
}

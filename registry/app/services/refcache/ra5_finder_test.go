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
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gtypes "github.com/harness/gitness/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var errRa5 = errors.New("ra5 failure")

// ra5Cache is a map backed cache.Cache.
type ra5Cache[K comparable, V any] struct {
	items map[K]V
	err   error
}

func (c ra5Cache[K, V]) Stats() (int64, int64) { return 0, 0 }

func (c ra5Cache[K, V]) Get(_ context.Context, key K) (V, error) {
	var zero V
	if c.err != nil {
		return zero, c.err
	}
	v, ok := c.items[key]
	if !ok {
		return zero, errRa5
	}
	return v, nil
}

func (c ra5Cache[K, V]) Evict(_ context.Context, _ K) {}

type ra5Bus struct {
	pubsub.PubSub
	topics []string
	err    error
}

func (b *ra5Bus) Publish(_ context.Context, topic string, _ []byte, _ ...pubsub.PublishOption) error {
	b.topics = append(b.topics, topic)
	return b.err
}

type ra5RegRepo struct {
	store.RegistryRepository
	reg       *types.Registry
	getErr    error
	updateErr error
	deleteErr error
	deleted   bool
}

func (r *ra5RegRepo) Update(_ context.Context, _ *types.Registry) error { return r.updateErr }

func (r *ra5RegRepo) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Registry, error) {
	return r.reg, r.getErr
}

func (r *ra5RegRepo) Delete(_ context.Context, _ int64, _ string) error {
	r.deleted = true
	return r.deleteErr
}

type ra5UpstreamFinder struct {
	UpstreamProxyFinder
	proxies map[int64]*types.UpstreamProxy
	changed []*types.UpstreamProxy
}

func (u *ra5UpstreamFinder) Get(_ context.Context, id int64) (*types.UpstreamProxy, error) {
	p, ok := u.proxies[id]
	if !ok {
		return nil, errRa5
	}
	return p, nil
}

func (u *ra5UpstreamFinder) MarkChanged(_ context.Context, p *types.UpstreamProxy) {
	u.changed = append(u.changed, p)
}

type ra5Fixture struct {
	finder   RegistryFinder
	repo     *ra5RegRepo
	bus      *ra5Bus
	upstream *ra5UpstreamFinder
}

func ra5NewFinder(regs map[int64]*types.Registry, byUUID map[string]int64) ra5Fixture {
	bus := &ra5Bus{}
	repo := &ra5RegRepo{reg: &types.Registry{ID: 1}}
	upstream := &ra5UpstreamFinder{proxies: map[int64]*types.UpstreamProxy{1: {ID: 5}}}
	spaces := refcache.NewSpaceFinder(
		ra5Cache[int64, *gtypes.SpaceCore]{items: map[int64]*gtypes.SpaceCore{9: {ID: 9, Identifier: "root"}}},
		ra5Cache[string, *gtypes.SpacePath]{items: map[string]*gtypes.SpacePath{"root": {SpaceID: 9}}},
		nil,
		appcache.Evictor[*gtypes.SpaceCore]{},
	)
	finder := NewRegistryFinder(
		repo,
		ra5Cache[int64, *types.Registry]{items: regs},
		ra5Cache[string, int64]{items: byUUID},
		ra5Cache[types.RegistryRootRefCacheKey, int64]{items: map[types.RegistryRootRefCacheKey]int64{
			{RootParentID: 9, RegistryIdentifier: "live"}:    1,
			{RootParentID: 9, RegistryIdentifier: "deleted"}: 2,
		}},
		appcache.NewEvictor[*types.Registry]("ns", "topic", bus),
		spaces,
		upstream,
	)
	return ra5Fixture{finder: finder, repo: repo, bus: bus, upstream: upstream}
}

func TestRa5RegistryFinderLookups(t *testing.T) {
	ctx := context.Background()
	deletedAt := time.Now()
	id := uuid.New()
	fx := ra5NewFinder(
		map[int64]*types.Registry{1: {ID: 1, Name: "live"}, 2: {ID: 2, Name: "deleted", DeletedAt: &deletedAt}},
		map[string]int64{id.String(): 1},
	)

	reg, err := fx.finder.FindByID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "live", reg.Name)
	_, err = fx.finder.FindByID(ctx, 99)
	require.Error(t, err)

	reg, err = fx.finder.FindByUUID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, int64(1), reg.ID)
	_, err = fx.finder.FindByUUID(ctx, uuid.New())
	require.ErrorContains(t, err, "failed to find registry ID by UUID")

	reg, err = fx.finder.FindByRootRef(ctx, "root", "live")
	require.NoError(t, err)
	require.Equal(t, int64(1), reg.ID)
	_, err = fx.finder.FindByRootRef(ctx, "nospace", "live")
	require.ErrorContains(t, err, "error finding space by root-ref")

	_, err = fx.finder.FindByRootParentID(ctx, 9, "unknown")
	require.ErrorContains(t, err, "error finding registry by root-ref")

	// Cached registry ID that no longer resolves.
	broken := ra5NewFinder(map[int64]*types.Registry{}, nil)
	_, err = broken.finder.FindByRootParentID(ctx, 9, "live")
	require.Error(t, err)

	tests := []struct {
		name    string
		ident   string
		opts    []types.QueryOption
		wantErr string
	}{
		{"default excludes deleted - live", "live", nil, ""},
		{"default excludes deleted - deleted", "deleted", nil, "registry is deleted"},
		{"only deleted - deleted", "deleted", []types.QueryOption{types.WithOnlyDeleted()}, ""},
		{"only deleted - live", "live", []types.QueryOption{types.WithOnlyDeleted()}, "registry is not deleted"},
		{"all - deleted", "deleted", []types.QueryOption{types.WithAllDeleted()}, ""},
		{"all - live", "live", []types.QueryOption{types.WithAllDeleted()}, ""},
		{"exclude - deleted", "deleted", []types.QueryOption{types.WithExcludeDeleted()}, "registry is deleted"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fx.finder.FindByRootParentID(ctx, 9, tc.ident, tc.opts...)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestRa5RegistryFinderMutations(t *testing.T) {
	ctx := context.Background()
	fx := ra5NewFinder(nil, nil)

	require.NoError(t, fx.finder.Update(ctx, &types.Registry{ID: 1}))
	require.Equal(t, []string{"topic"}, fx.bus.topics)
	require.Len(t, fx.upstream.changed, 1)

	fx.repo.updateErr = errRa5
	require.ErrorIs(t, fx.finder.Update(ctx, &types.Registry{ID: 1}), errRa5)
	require.Len(t, fx.bus.topics, 1)

	// Upstream proxy lookup failure does not break MarkChanged.
	fx.finder.MarkChanged(ctx, &types.Registry{ID: 77})
	require.Len(t, fx.bus.topics, 2)
	require.Len(t, fx.upstream.changed, 1)

	require.NoError(t, fx.finder.Delete(ctx, 3, "name"))
	require.True(t, fx.repo.deleted)
	require.Len(t, fx.bus.topics, 3)

	fx.repo.deleteErr = errRa5
	require.ErrorIs(t, fx.finder.Delete(ctx, 3, "name"), errRa5)
	require.Len(t, fx.bus.topics, 3)

	fx.repo.getErr = errRa5
	require.ErrorContains(t, fx.finder.Delete(ctx, 3, "name"), "error finding registry by parent-ref")
}

func TestRa5GetUpstreamProxyByRegistryUUIDs(t *testing.T) {
	ctx := context.Background()
	fx := ra5NewFinder(nil, map[string]int64{"u1": 1, "u2": 2})

	got, err := fx.finder.GetUpstreamProxyByRegistryUUIDs(ctx, []string{"u1", "u2"})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, int64(5), got["u1"].ID)
	require.Nil(t, got["u2"])

	_, err = fx.finder.GetUpstreamProxyByRegistryUUIDs(ctx, []string{"missing"})
	require.ErrorContains(t, err, "failed to find registry ID for UUID missing")

	got, err = fx.finder.GetUpstreamProxyByRegistryUUIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

// Copyright 2026 Harness, Inc.
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
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// rdbFakeRegistryRepo serves registries from memory and counts lookups.
type rdbFakeRegistryRepo struct {
	store.RegistryRepository

	mu        sync.Mutex
	byID      map[int64]*types.Registry
	calls     int
	lastOpts  types.QueryOptions
	lookupErr error
}

func (f *rdbFakeRegistryRepo) record(opts []types.QueryOption) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastOpts = types.MakeQueryOptions(opts...)
}

func (f *rdbFakeRegistryRepo) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *rdbFakeRegistryRepo) Get(_ context.Context, id int64, opts ...types.QueryOption) (*types.Registry, error) {
	f.record(opts)
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if r, ok := f.byID[id]; ok {
		return r, nil
	}
	return nil, errors.New("not found")
}

func (f *rdbFakeRegistryRepo) GetByUUID(_ context.Context, uuid string) (*types.Registry, error) {
	f.record(nil)
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	for _, r := range f.byID {
		if r.UUID == uuid {
			return r, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *rdbFakeRegistryRepo) GetByRootParentIDAndName(
	_ context.Context, rootParentID int64, name string, opts ...types.QueryOption,
) (*types.Registry, error) {
	f.record(opts)
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	for _, r := range f.byID {
		if r.RootParentID == rootParentID && r.Name == name {
			return r, nil
		}
	}
	return nil, errors.New("not found")
}

// rdbFakeUpstreamRepo serves upstream proxies from memory.
type rdbFakeUpstreamRepo struct {
	store.UpstreamProxyConfigRepository

	mu       sync.Mutex
	byID     map[int64]*types.UpstreamProxy
	calls    int
	lastOpts types.QueryOptions
}

func (f *rdbFakeUpstreamRepo) Get(
	_ context.Context, id int64, opts ...types.QueryOption,
) (*types.UpstreamProxy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastOpts = types.MakeQueryOptions(opts...)
	if p, ok := f.byID[id]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func (f *rdbFakeUpstreamRepo) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// rdbSyncBus is a pubsub.PubSub that delivers published payloads to subscribers synchronously.
type rdbSyncBus struct {
	mu       sync.Mutex
	handlers map[string][]func([]byte) error
}

func rdbNewSyncBus() *rdbSyncBus {
	return &rdbSyncBus{handlers: map[string][]func([]byte) error{}}
}

func (b *rdbSyncBus) Publish(_ context.Context, topic string, payload []byte, _ ...pubsub.PublishOption) error {
	b.mu.Lock()
	handlers := append([]func([]byte) error(nil), b.handlers[topic]...)
	b.mu.Unlock()
	for _, h := range handlers {
		if err := h(payload); err != nil {
			return err
		}
	}
	return nil
}

func (b *rdbSyncBus) Subscribe(
	_ context.Context, topic string, handler func([]byte) error, _ ...pubsub.SubscribeOption,
) pubsub.Consumer {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
	return nil
}

func rdbNewRegistryRepo() *rdbFakeRegistryRepo {
	return &rdbFakeRegistryRepo{byID: map[int64]*types.Registry{
		1: {ID: 1, UUID: "uuid-one", Name: "one", RootParentID: 100},
		2: {ID: 2, UUID: "uuid-two", Name: "two", RootParentID: 100},
	}}
}

func TestRdbRegistryIDCache(t *testing.T) {
	ctx := context.Background()
	repo := rdbNewRegistryRepo()
	bus := rdbNewSyncBus()
	evictor := ProvideEvictorRegistryCore(bus)
	c := ProvideRegistryIDCache(ctx, repo, evictor)

	got, err := c.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "one", got.Name)
	require.Equal(t, types.DeleteFilterIncludeDeleted, repo.lastOpts.DeleteFilter)

	// A second lookup is served from the cache.
	_, err = c.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, repo.callCount())

	_, err = c.Get(ctx, 99)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to find repo by ID")

	// Evicting through the pubsub bus forces a reload.
	evictor.Evict(ctx, &types.Registry{ID: 1})
	_, err = c.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 3, repo.callCount())
}

func TestRdbRegistryRootRefCache(t *testing.T) {
	ctx := context.Background()
	repo := rdbNewRegistryRepo()
	bus := rdbNewSyncBus()
	evictor := ProvideEvictorRegistryCore(bus)
	c := ProvideRegRootRefCache(ctx, repo, evictor)

	key := types.RegistryRootRefCacheKey{RootParentID: 100, RegistryIdentifier: "two"}
	id, err := c.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, int64(2), id)
	require.Equal(t, types.DeleteFilterIncludeDeleted, repo.lastOpts.DeleteFilter)

	_, err = c.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, 1, repo.callCount())

	_, err = c.Get(ctx, types.RegistryRootRefCacheKey{RootParentID: 100, RegistryIdentifier: "missing"})
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to find repo by 100:missing")

	evictor.Evict(ctx, &types.Registry{RootParentID: 100, Name: "two"})
	_, err = c.Get(ctx, key)
	require.NoError(t, err)
	require.Equal(t, 3, repo.callCount())
}

func TestRdbRegistryUUIDToIDCache(t *testing.T) {
	ctx := context.Background()
	repo := rdbNewRegistryRepo()
	bus := rdbNewSyncBus()
	evictor := ProvideEvictorRegistryCore(bus)
	c := ProvideRegistryUUIDToIDCache(ctx, repo, evictor)

	id, err := c.Get(ctx, "uuid-two")
	require.NoError(t, err)
	require.Equal(t, int64(2), id)

	_, err = c.Get(ctx, "uuid-two")
	require.NoError(t, err)
	require.Equal(t, 1, repo.callCount())

	_, err = c.Get(ctx, "uuid-missing")
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to find repo by UUID")

	evictor.Evict(ctx, &types.Registry{UUID: "uuid-two"})
	_, err = c.Get(ctx, "uuid-two")
	require.NoError(t, err)
	require.Equal(t, 3, repo.callCount())
}

func TestRdbUpstreamProxyRegistryIDCache(t *testing.T) {
	ctx := context.Background()
	repo := &rdbFakeUpstreamRepo{byID: map[int64]*types.UpstreamProxy{
		5: {ID: 50, RegistryID: 5, RepoKey: "up"},
	}}
	bus := rdbNewSyncBus()
	evictor := ProvideEvictorUpstreamProxy(bus)
	c := ProvideUpstreamProxyRegistryIDCache(ctx, repo, evictor)

	got, err := c.Get(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, "up", got.RepoKey)
	require.Equal(t, types.DeleteFilterIncludeDeleted, repo.lastOpts.DeleteFilter)

	_, err = c.Get(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, 1, repo.callCount())

	_, err = c.Get(ctx, 6)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to find upstream proxy config by registry ID")

	evictor.Evict(ctx, &types.UpstreamProxy{RegistryID: 5})
	_, err = c.Get(ctx, 5)
	require.NoError(t, err)
	require.Equal(t, 3, repo.callCount())
}

func TestRdbEvictorsWithoutBus(t *testing.T) {
	ctx := context.Background()
	repo := rdbNewRegistryRepo()

	// Without a pubsub bus the evictors are no-ops and the caches still work.
	var noBus appcache.Evictor[*types.Registry]
	c := NewRegistryIDCache(ctx, repo, noBus, time.Minute)
	got, err := c.Get(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, "two", got.Name)

	rootRef := NewRegistryRootRefCache(ctx, repo, noBus, time.Minute)
	id, err := rootRef.Get(ctx, types.RegistryRootRefCacheKey{RootParentID: 100, RegistryIdentifier: "one"})
	require.NoError(t, err)
	require.Equal(t, int64(1), id)

	uuidCache := NewRegistryUUIDToIDCache(ctx, repo, noBus, time.Minute)
	id, err = uuidCache.Get(ctx, "uuid-one")
	require.NoError(t, err)
	require.Equal(t, int64(1), id)

	repo.lookupErr = errors.New("db down")
	_, err = c.Get(ctx, 1)
	require.ErrorContains(t, err, "db down")
}

func TestRdbDownloadCountCacheProviders(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = client.Close() })
	source := &mockDownloadStatSource{}

	require.NotNil(t, ProvideDownloadCountRegistryCache(client, source))
	require.NotNil(t, ProvideDownloadCountImageCache(client, source))
	require.NotNil(t, ProvideDownloadCountArtifactCache(client, source))
	require.NotNil(t, ProvideDownloadCountManifestCache(client, source))
}

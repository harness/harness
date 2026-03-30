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

	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockExtendedCache implements cache.ExtendedCache[K, V] for testing.
type mockExtendedCache[K comparable, V any] struct {
	mock.Mock
}

func (m *mockExtendedCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	args := m.Called(ctx, key)
	if err := args.Error(1); err != nil {
		var zero V
		return zero, err
	}
	val, ok := args.Get(0).(V)
	if !ok {
		var zero V
		return zero, nil
	}
	return val, nil
}

func (m *mockExtendedCache[K, V]) Map(ctx context.Context, keys []K) (map[K]V, error) {
	args := m.Called(ctx, keys)
	if err := args.Error(1); err != nil {
		return nil, err
	}
	val, ok := args.Get(0).(map[K]V)
	if !ok {
		return nil, nil //nolint:nilnil
	}
	return val, nil
}

func (m *mockExtendedCache[K, V]) Stats() (int64, int64) {
	args := m.Called()
	hits, _ := args.Get(0).(int64)
	misses, _ := args.Get(1).(int64)
	return hits, misses
}

func (m *mockExtendedCache[K, V]) Evict(ctx context.Context, key K) {
	m.Called(ctx, key)
}

func TestFindByRegistryIDs_Success(t *testing.T) {
	regCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	regCache.On("Map", mock.Anything, []int64{1, 2}).
		Return(map[int64]*types.DownloadCount{
			1: {EntityID: 1, Count: 42},
			2: {EntityID: 2, Count: 10},
		}, nil)

	finder := NewDownloadCountFinder(regCache, nil, nil, nil)
	counts, err := finder.FindByRegistryIDs(context.Background(), []int64{1, 2})

	assert.NoError(t, err)
	assert.Equal(t, int64(42), counts[1])
	assert.Equal(t, int64(10), counts[2])
}

func TestFindByImageID_Success(t *testing.T) {
	imgCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	imgCache.On("Get", mock.Anything, int64(5)).
		Return(&types.DownloadCount{EntityID: 5, Count: 100}, nil)

	finder := NewDownloadCountFinder(nil, imgCache, nil, nil)
	count, err := finder.FindByImageID(context.Background(), 5)

	assert.NoError(t, err)
	assert.Equal(t, int64(100), count)
}

func TestFindByImageID_Error_ReturnsZero(t *testing.T) {
	imgCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	imgCache.On("Get", mock.Anything, int64(5)).
		Return((*types.DownloadCount)(nil), errors.New("cache error"))

	finder := NewDownloadCountFinder(nil, imgCache, nil, nil)
	count, err := finder.FindByImageID(context.Background(), 5)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestFindByArtifactID_Success(t *testing.T) {
	artCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	artCache.On("Get", mock.Anything, int64(10)).
		Return(&types.DownloadCount{EntityID: 10, Count: 200}, nil)

	finder := NewDownloadCountFinder(nil, nil, artCache, nil)
	count, err := finder.FindByArtifactID(context.Background(), 10)

	assert.NoError(t, err)
	assert.Equal(t, int64(200), count)
}

func TestFindByArtifactID_Error_ReturnsZero(t *testing.T) {
	artCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	artCache.On("Get", mock.Anything, int64(10)).
		Return((*types.DownloadCount)(nil), errors.New("cache error"))

	finder := NewDownloadCountFinder(nil, nil, artCache, nil)
	count, err := finder.FindByArtifactID(context.Background(), 10)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestFindByManifests_Success(t *testing.T) {
	manCache := &mockExtendedCache[string, *types.ManifestDownloadCount]{}
	manCache.On("Map", mock.Anything, mock.Anything).
		Return(map[string]*types.ManifestDownloadCount{
			"7:sha256:abc": {Key: "7:sha256:abc", Count: 55},
			"7:sha256:def": {Key: "7:sha256:def", Count: 30},
		}, nil)

	finder := NewDownloadCountFinder(nil, nil, nil, manCache)
	result, err := finder.FindByManifests(
		context.Background(),
		[]string{"sha256:abc", "sha256:def"},
		7,
	)

	assert.NoError(t, err)
	assert.Equal(t, int64(55), result["sha256:abc"])
	assert.Equal(t, int64(30), result["sha256:def"])
}

func TestFindByManifests_Error_ReturnsEmpty(t *testing.T) {
	manCache := &mockExtendedCache[string, *types.ManifestDownloadCount]{}
	manCache.On("Map", mock.Anything, mock.Anything).
		Return((map[string]*types.ManifestDownloadCount)(nil), errors.New("cache error"))

	finder := NewDownloadCountFinder(nil, nil, nil, manCache)
	result, err := finder.FindByManifests(
		context.Background(),
		[]string{"sha256:abc"},
		7,
	)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestFindByManifests_EmptyDigests(t *testing.T) {
	manCache := &mockExtendedCache[string, *types.ManifestDownloadCount]{}
	manCache.On("Map", mock.Anything, mock.Anything).
		Return(map[string]*types.ManifestDownloadCount{}, nil)

	finder := NewDownloadCountFinder(nil, nil, nil, manCache)
	result, err := finder.FindByManifests(context.Background(), []string{}, 7)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestFindByRegistryIDs_Error_ReturnsEmpty(t *testing.T) {
	regCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	regCache.On("Map", mock.Anything, []int64{1}).
		Return((map[int64]*types.DownloadCount)(nil), errors.New("cache error"))

	finder := NewDownloadCountFinder(regCache, nil, nil, nil)
	counts, err := finder.FindByRegistryIDs(context.Background(), []int64{1})

	assert.NoError(t, err)
	assert.Empty(t, counts)
}

func TestFindByImageIDs_Success(t *testing.T) {
	imgCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	imgCache.On("Map", mock.Anything, []int64{3, 4}).
		Return(map[int64]*types.DownloadCount{
			3: {EntityID: 3, Count: 15},
			4: {EntityID: 4, Count: 25},
		}, nil)

	finder := NewDownloadCountFinder(nil, imgCache, nil, nil)
	counts, err := finder.FindByImageIDs(context.Background(), []int64{3, 4})

	assert.NoError(t, err)
	assert.Equal(t, int64(15), counts[3])
	assert.Equal(t, int64(25), counts[4])
}

func TestFindByImageIDs_Error_ReturnsEmpty(t *testing.T) {
	imgCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	imgCache.On("Map", mock.Anything, []int64{3}).
		Return((map[int64]*types.DownloadCount)(nil), errors.New("cache error"))

	finder := NewDownloadCountFinder(nil, imgCache, nil, nil)
	counts, err := finder.FindByImageIDs(context.Background(), []int64{3})

	assert.NoError(t, err)
	assert.Empty(t, counts)
}

func TestFindByArtifactIDs_Success(t *testing.T) {
	artCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	artCache.On("Map", mock.Anything, []int64{10, 20}).
		Return(map[int64]*types.DownloadCount{
			10: {EntityID: 10, Count: 50},
			20: {EntityID: 20, Count: 75},
		}, nil)

	finder := NewDownloadCountFinder(nil, nil, artCache, nil)
	counts, err := finder.FindByArtifactIDs(context.Background(), []int64{10, 20})

	assert.NoError(t, err)
	assert.Equal(t, int64(50), counts[10])
	assert.Equal(t, int64(75), counts[20])
}

func TestFindByArtifactIDs_Error_ReturnsEmpty(t *testing.T) {
	artCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	artCache.On("Map", mock.Anything, []int64{10}).
		Return((map[int64]*types.DownloadCount)(nil), errors.New("cache error"))

	finder := NewDownloadCountFinder(nil, nil, artCache, nil)
	counts, err := finder.FindByArtifactIDs(context.Background(), []int64{10})

	assert.NoError(t, err)
	assert.Empty(t, counts)
}

func TestNewDownloadCountFinder(t *testing.T) {
	regCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	imgCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	artCache := &mockExtendedCache[int64, *types.DownloadCount]{}
	manCache := &mockExtendedCache[string, *types.ManifestDownloadCount]{}

	finder := NewDownloadCountFinder(regCache, imgCache, artCache, manCache)
	assert.NotNil(t, finder)
}

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

package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
)

// mockDownloadStatSource implements store.DownloadStatRepository for testing.
type mockDownloadStatSource struct {
	registryCounts map[int64]int64
	registryErr    error
	imageCounts    map[int64]int64
	imageErr       error
	artifactCounts map[int64]int64
	artifactErr    error
	manifestCounts map[string]int64
	manifestErr    error
}

func (m *mockDownloadStatSource) Create(
	_ context.Context, _ *types.DownloadStat,
) error {
	return nil
}

func (m *mockDownloadStatSource) CreateByRegistryIDImageAndArtifactName(
	_ context.Context, _ int64, _ string, _ string, _ *artifact.ArtifactType,
) error {
	return nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForRegistryID(
	_ context.Context, registryID int64,
) (int64, error) {
	if m.registryErr != nil {
		return 0, m.registryErr
	}
	return m.registryCounts[registryID], nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForRegistryIDs(
	_ context.Context, registryIDs []int64,
) ([]*types.DownloadCount, error) {
	if m.registryErr != nil {
		return nil, m.registryErr
	}
	var results []*types.DownloadCount
	for _, id := range registryIDs {
		if count, ok := m.registryCounts[id]; ok {
			results = append(results, &types.DownloadCount{EntityID: id, Count: count})
		}
	}
	return results, nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForImage(
	_ context.Context, imageID int64,
) (int64, error) {
	if m.imageErr != nil {
		return 0, m.imageErr
	}
	return m.imageCounts[imageID], nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForImageIDs(
	_ context.Context, imageIDs []int64,
) ([]*types.DownloadCount, error) {
	if m.imageErr != nil {
		return nil, m.imageErr
	}
	var results []*types.DownloadCount
	for _, id := range imageIDs {
		if count, ok := m.imageCounts[id]; ok {
			results = append(results, &types.DownloadCount{EntityID: id, Count: count})
		}
	}
	return results, nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForArtifactID(
	_ context.Context, artifactID int64,
) (int64, error) {
	if m.artifactErr != nil {
		return 0, m.artifactErr
	}
	return m.artifactCounts[artifactID], nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForArtifactIDs(
	_ context.Context, artifactIDs []int64,
) ([]*types.DownloadCount, error) {
	if m.artifactErr != nil {
		return nil, m.artifactErr
	}
	var results []*types.DownloadCount
	for _, id := range artifactIDs {
		if count, ok := m.artifactCounts[id]; ok {
			results = append(results, &types.DownloadCount{EntityID: id, Count: count})
		}
	}
	return results, nil
}

func (m *mockDownloadStatSource) GetTotalDownloadsForManifests(
	_ context.Context, versions []string, _ int64,
) (map[string]int64, error) {
	if m.manifestErr != nil {
		return nil, m.manifestErr
	}
	result := make(map[string]int64)
	for _, v := range versions {
		if count, ok := m.manifestCounts[v]; ok {
			result[v] = count
		}
	}
	return result, nil
}

func TestRegistryGetter_Find_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		registryCounts: map[int64]int64{1: 42},
	}
	getter := downloadCountRegistryCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), dc.Count)
	assert.Equal(t, int64(1), dc.EntityID)
}

func TestRegistryGetter_Find_Error(t *testing.T) {
	src := &mockDownloadStatSource{registryErr: errors.New("db error")}
	getter := downloadCountRegistryCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download count for registry 1")
	assert.Nil(t, dc)
}

func TestDownloadCountCodec(t *testing.T) {
	codec := downloadCountCodec{}
	original := &types.DownloadCount{EntityID: 42, Count: 100}
	encoded := codec.Encode(original)
	decoded, err := codec.Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, original.EntityID, decoded.EntityID)
	assert.Equal(t, original.Count, decoded.Count)
}

func TestManifestDownloadCountCodec(t *testing.T) {
	codec := manifestDownloadCountCodec{}
	original := &types.ManifestDownloadCount{Key: "7:sha256:abc", Count: 55}
	encoded := codec.Encode(original)
	decoded, err := codec.Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Key, decoded.Key)
	assert.Equal(t, original.Count, decoded.Count)
}

func TestImageGetter_Find_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		imageCounts: map[int64]int64{5: 100},
	}
	getter := downloadCountImageCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), dc.Count)
	assert.Equal(t, int64(5), dc.EntityID)
}

func TestImageGetter_Find_Error(t *testing.T) {
	src := &mockDownloadStatSource{imageErr: errors.New("db error")}
	getter := downloadCountImageCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download count for image 5")
	assert.Nil(t, dc)
}

func TestArtifactGetter_Find_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		artifactCounts: map[int64]int64{10: 200},
	}
	getter := downloadCountArtifactCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(200), dc.Count)
	assert.Equal(t, int64(10), dc.EntityID)
}

func TestArtifactGetter_Find_Error(t *testing.T) {
	src := &mockDownloadStatSource{artifactErr: errors.New("db error")}
	getter := downloadCountArtifactCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download count for artifact 10")
	assert.Nil(t, dc)
}

func TestManifestGetter_Find_Success(t *testing.T) {
	digest := "sha256:abc123"
	src := &mockDownloadStatSource{
		manifestCounts: map[string]int64{digest: 55},
	}
	getter := downloadCountManifestCacheGetter{source: src}
	dc, err := getter.Find(context.Background(), ManifestCacheKey(7, digest))
	assert.NoError(t, err)
	assert.Equal(t, int64(55), dc.Count)
}

func TestManifestGetter_Find_DBError(t *testing.T) {
	src := &mockDownloadStatSource{manifestErr: errors.New("db error")}
	getter := downloadCountManifestCacheGetter{source: src}
	dc, err := getter.Find(
		context.Background(), ManifestCacheKey(7, "sha256:abc"),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download count for manifest")
	assert.Nil(t, dc)
}

func TestManifestGetter_Find_InvalidKey_NoColon(t *testing.T) {
	src := &mockDownloadStatSource{}
	getter := downloadCountManifestCacheGetter{source: src}
	_, err := getter.Find(context.Background(), "invalidkey")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest cache key")
}

func TestManifestGetter_Find_InvalidKey_NonNumericID(t *testing.T) {
	src := &mockDownloadStatSource{}
	getter := downloadCountManifestCacheGetter{source: src}
	_, err := getter.Find(context.Background(), "abc:sha256:def")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest cache key")
}

func TestManifestGetter_Find_InvalidKey_EmptyDigest(t *testing.T) {
	src := &mockDownloadStatSource{}
	getter := downloadCountManifestCacheGetter{source: src}
	_, err := getter.Find(context.Background(), "123:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest cache key")
}

func TestManifestCacheKey_Format(t *testing.T) {
	key := ManifestCacheKey(123, "sha256:abc")
	assert.Equal(t, "123:sha256:abc", key)
}

func TestParseManifestCacheKey_Valid(t *testing.T) {
	imageID, digest, err := ParseManifestCacheKey("123:sha256:abc")
	assert.NoError(t, err)
	assert.Equal(t, int64(123), imageID)
	assert.Equal(t, "sha256:abc", digest)
}

func TestParseManifestCacheKey_EmptyDigest(t *testing.T) {
	_, _, err := ParseManifestCacheKey("123:")
	assert.Error(t, err)
}

func TestParseManifestCacheKey_NoColon(t *testing.T) {
	_, _, err := ParseManifestCacheKey("nocolon")
	assert.Error(t, err)
}

func TestParseManifestCacheKey_NonNumericID(t *testing.T) {
	_, _, err := ParseManifestCacheKey("abc:sha256:def")
	assert.Error(t, err)
}

func TestRegistryGetter_FindMany_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		registryCounts: map[int64]int64{1: 10, 2: 20},
	}
	getter := downloadCountRegistryCacheGetter{source: src}
	results, err := getter.FindMany(context.Background(), []int64{1, 2, 3})
	assert.NoError(t, err)
	assert.Len(t, results, 3) // 2 found + 1 backfilled
}

func TestRegistryGetter_FindMany_Error(t *testing.T) {
	src := &mockDownloadStatSource{registryErr: errors.New("db error")}
	getter := downloadCountRegistryCacheGetter{source: src}
	_, err := getter.FindMany(context.Background(), []int64{1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download counts for registries")
}

func TestImageGetter_FindMany_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		imageCounts: map[int64]int64{5: 50},
	}
	getter := downloadCountImageCacheGetter{source: src}
	results, err := getter.FindMany(context.Background(), []int64{5, 6})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestImageGetter_FindMany_Error(t *testing.T) {
	src := &mockDownloadStatSource{imageErr: errors.New("db error")}
	getter := downloadCountImageCacheGetter{source: src}
	_, err := getter.FindMany(context.Background(), []int64{5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download counts for images")
}

func TestArtifactGetter_FindMany_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		artifactCounts: map[int64]int64{10: 100},
	}
	getter := downloadCountArtifactCacheGetter{source: src}
	results, err := getter.FindMany(context.Background(), []int64{10, 11})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestArtifactGetter_FindMany_Error(t *testing.T) {
	src := &mockDownloadStatSource{artifactErr: errors.New("db error")}
	getter := downloadCountArtifactCacheGetter{source: src}
	_, err := getter.FindMany(context.Background(), []int64{10})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download counts for artifacts")
}

func TestManifestGetter_FindMany_Success(t *testing.T) {
	src := &mockDownloadStatSource{
		manifestCounts: map[string]int64{"sha256:aaa": 5, "sha256:bbb": 10},
	}
	getter := downloadCountManifestCacheGetter{source: src}
	results, err := getter.FindMany(context.Background(), []string{
		ManifestCacheKey(1, "sha256:aaa"),
		ManifestCacheKey(1, "sha256:bbb"),
	})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestManifestGetter_FindMany_Error(t *testing.T) {
	src := &mockDownloadStatSource{manifestErr: errors.New("db error")}
	getter := downloadCountManifestCacheGetter{source: src}
	_, err := getter.FindMany(context.Background(), []string{
		ManifestCacheKey(1, "sha256:aaa"),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get download counts for manifests")
}

func TestManifestGetter_FindMany_InvalidKey(t *testing.T) {
	src := &mockDownloadStatSource{}
	getter := downloadCountManifestCacheGetter{source: src}
	_, err := getter.FindMany(context.Background(), []string{"badkey"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest cache key")
}

func TestManifestGetter_FindMany_MultipleImageIDs(t *testing.T) {
	src := &mockDownloadStatSource{
		manifestCounts: map[string]int64{"sha256:aaa": 5, "sha256:bbb": 10},
	}
	getter := downloadCountManifestCacheGetter{source: src}
	results, err := getter.FindMany(context.Background(), []string{
		ManifestCacheKey(1, "sha256:aaa"),
		ManifestCacheKey(2, "sha256:bbb"),
	})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestBackfillMissing(t *testing.T) {
	results := []*types.DownloadCount{
		{EntityID: 1, Count: 10},
	}
	backfilled := backfillMissing([]int64{1, 2, 3}, results)
	assert.Len(t, backfilled, 3)

	counts := make(map[int64]int64)
	for _, r := range backfilled {
		counts[r.EntityID] = r.Count
	}
	assert.Equal(t, int64(10), counts[1])
	assert.Equal(t, int64(0), counts[2])
	assert.Equal(t, int64(0), counts[3])
}

func TestBackfillMissing_AllPresent(t *testing.T) {
	results := []*types.DownloadCount{
		{EntityID: 1, Count: 10},
		{EntityID: 2, Count: 20},
	}
	backfilled := backfillMissing([]int64{1, 2}, results)
	assert.Len(t, backfilled, 2)
}

func TestDownloadCountKeyEncoder(t *testing.T) {
	encoder := downloadCountKeyEncoder("dl_count:registry:")
	assert.Equal(t, "dl_count:registry:42", encoder(42))
	assert.Equal(t, "dl_count:registry:0", encoder(0))
}

func TestManifestCountKeyEncoder(t *testing.T) {
	assert.Equal(t, "dl_count:manifest:7:sha256:abc", manifestCountKeyEncoder("7:sha256:abc"))
}

func TestDownloadCountCodec_Decode_InvalidFormat(t *testing.T) {
	codec := downloadCountCodec{}
	_, err := codec.Decode("nocolon")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid download count cache value")
}

func TestDownloadCountCodec_Decode_InvalidEntityID(t *testing.T) {
	codec := downloadCountCodec{}
	_, err := codec.Decode("abc:123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid entity ID")
}

func TestDownloadCountCodec_Decode_InvalidCount(t *testing.T) {
	codec := downloadCountCodec{}
	_, err := codec.Decode("123:abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid count")
}

func TestManifestDownloadCountCodec_Decode_InvalidFormat(t *testing.T) {
	codec := manifestDownloadCountCodec{}
	_, err := codec.Decode("nopipe")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest download count cache value")
}

func TestManifestDownloadCountCodec_Decode_InvalidCount(t *testing.T) {
	codec := manifestDownloadCountCodec{}
	_, err := codec.Decode("7:sha256:abc|notanumber")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid count in manifest cache value")
}

func TestLogRedisErr(t *testing.T) {
	// Just ensure it doesn't panic.
	logRedisErr(context.Background(), errors.New("test error"))
}

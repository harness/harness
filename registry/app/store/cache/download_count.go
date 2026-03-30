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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

// Registry-level download count cache getter.

type downloadCountRegistryCacheGetter struct {
	source store.DownloadStatRepository
}

func (g downloadCountRegistryCacheGetter) Find(ctx context.Context, registryID int64) (*types.DownloadCount, error) {
	count, err := g.source.GetTotalDownloadsForRegistryID(ctx, registryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get download count for registry %d: %w", registryID, err)
	}
	return &types.DownloadCount{EntityID: registryID, Count: count}, nil
}

//nolint:lll
func (g downloadCountRegistryCacheGetter) FindMany(ctx context.Context, registryIDs []int64) ([]*types.DownloadCount, error) {
	results, err := g.source.GetTotalDownloadsForRegistryIDs(ctx, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get download counts for registries: %w", err)
	}
	return backfillMissing(registryIDs, results), nil
}

func NewDownloadCountRegistryCache(
	client redis.UniversalClient,
	source store.DownloadStatRepository,
	dur time.Duration,
) store.DownloadCountRegistryCache {
	return cache.NewExtendedRedis[int64, *types.DownloadCount](
		client,
		downloadCountRegistryCacheGetter{source: source},
		downloadCountKeyEncoder("dl_count:registry:"),
		downloadCountCodec{},
		dur,
		logRedisErr,
	)
}

// Image-level download count cache getter.

type downloadCountImageCacheGetter struct {
	source store.DownloadStatRepository
}

func (g downloadCountImageCacheGetter) Find(ctx context.Context, imageID int64) (*types.DownloadCount, error) {
	count, err := g.source.GetTotalDownloadsForImage(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get download count for image %d: %w", imageID, err)
	}
	return &types.DownloadCount{EntityID: imageID, Count: count}, nil
}

func (g downloadCountImageCacheGetter) FindMany(ctx context.Context, imageIDs []int64) ([]*types.DownloadCount, error) {
	results, err := g.source.GetTotalDownloadsForImageIDs(ctx, imageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get download counts for images: %w", err)
	}
	return backfillMissing(imageIDs, results), nil
}

func NewDownloadCountImageCache(
	client redis.UniversalClient,
	source store.DownloadStatRepository,
	dur time.Duration,
) store.DownloadCountImageCache {
	return cache.NewExtendedRedis[int64, *types.DownloadCount](
		client,
		downloadCountImageCacheGetter{source: source},
		downloadCountKeyEncoder("dl_count:image:"),
		downloadCountCodec{},
		dur,
		logRedisErr,
	)
}

// Artifact-level download count cache getter.

type downloadCountArtifactCacheGetter struct {
	source store.DownloadStatRepository
}

func (g downloadCountArtifactCacheGetter) Find(ctx context.Context, artifactID int64) (*types.DownloadCount, error) {
	count, err := g.source.GetTotalDownloadsForArtifactID(ctx, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get download count for artifact %d: %w", artifactID, err)
	}
	return &types.DownloadCount{EntityID: artifactID, Count: count}, nil
}

//nolint:lll
func (g downloadCountArtifactCacheGetter) FindMany(ctx context.Context, artifactIDs []int64) ([]*types.DownloadCount, error) {
	results, err := g.source.GetTotalDownloadsForArtifactIDs(ctx, artifactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get download counts for artifacts: %w", err)
	}
	return backfillMissing(artifactIDs, results), nil
}

func NewDownloadCountArtifactCache(
	client redis.UniversalClient,
	source store.DownloadStatRepository,
	dur time.Duration,
) store.DownloadCountArtifactCache {
	return cache.NewExtendedRedis[int64, *types.DownloadCount](
		client,
		downloadCountArtifactCacheGetter{source: source},
		downloadCountKeyEncoder("dl_count:artifact:"),
		downloadCountCodec{},
		dur,
		logRedisErr,
	)
}

// Manifest-level download count cache getter.
// Key is "imageID:digest" composite string.

type downloadCountManifestCacheGetter struct {
	source store.DownloadStatRepository
}

func (g downloadCountManifestCacheGetter) Find(ctx context.Context, key string) (*types.ManifestDownloadCount, error) {
	imageID, digest, err := ParseManifestCacheKey(key)
	if err != nil {
		return nil, err
	}
	counts, err := g.source.GetTotalDownloadsForManifests(ctx, []string{digest}, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get download count for manifest %s: %w", digest, err)
	}
	return &types.ManifestDownloadCount{Key: key, Count: counts[digest]}, nil
}

func (g downloadCountManifestCacheGetter) FindMany(
	ctx context.Context, keys []string,
) ([]*types.ManifestDownloadCount, error) {
	// Group keys by imageID for batch queries.
	grouped := make(map[int64][]string)
	keyByDigest := make(map[int64]map[string]string)
	for _, key := range keys {
		imageID, digest, err := ParseManifestCacheKey(key)
		if err != nil {
			return nil, err
		}
		grouped[imageID] = append(grouped[imageID], digest)
		if keyByDigest[imageID] == nil {
			keyByDigest[imageID] = make(map[string]string)
		}
		keyByDigest[imageID][digest] = key
	}

	var results []*types.ManifestDownloadCount
	for imageID, digests := range grouped {
		counts, err := g.source.GetTotalDownloadsForManifests(ctx, digests, imageID)
		if err != nil {
			return nil, fmt.Errorf("failed to get download counts for manifests: %w", err)
		}
		for _, digest := range digests {
			results = append(results, &types.ManifestDownloadCount{
				Key:   keyByDigest[imageID][digest],
				Count: counts[digest],
			})
		}
	}
	return results, nil
}

func NewDownloadCountManifestCache(
	client redis.UniversalClient,
	source store.DownloadStatRepository,
	dur time.Duration,
) store.DownloadCountManifestCache {
	return cache.NewExtendedRedis[string, *types.ManifestDownloadCount](
		client,
		downloadCountManifestCacheGetter{source: source},
		manifestCountKeyEncoder,
		manifestDownloadCountCodec{},
		dur,
		logRedisErr,
	)
}

// ManifestCacheKey builds a composite cache key "imageID:digest".
func ManifestCacheKey(imageID int64, digest string) string {
	return fmt.Sprintf("%d:%s", imageID, digest)
}

// ParseManifestCacheKey parses a composite cache key into imageID and digest.
func ParseManifestCacheKey(key string) (int64, string, error) {
	idStr, digest, ok := strings.Cut(key, ":")
	if !ok || digest == "" {
		return 0, "", fmt.Errorf("invalid manifest cache key: %s", key)
	}
	imageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid manifest cache key: %s", key)
	}
	return imageID, digest, nil
}

// backfillMissing ensures all requested IDs have entries, defaulting to zero count.
func backfillMissing(requestedIDs []int64, results []*types.DownloadCount) []*types.DownloadCount {
	found := make(map[int64]bool, len(results))
	for _, r := range results {
		found[r.EntityID] = true
	}
	for _, id := range requestedIDs {
		if !found[id] {
			results = append(results, &types.DownloadCount{EntityID: id, Count: 0})
		}
	}
	return results
}

// Redis key encoders.

func downloadCountKeyEncoder(prefix string) func(int64) string {
	return func(id int64) string {
		return prefix + strconv.FormatInt(id, 10)
	}
}

func manifestCountKeyEncoder(key string) string {
	return "dl_count:manifest:" + key
}

// Redis codecs.

// downloadCountCodec encodes/decodes *types.DownloadCount as "entityID:count".
type downloadCountCodec struct{}

func (downloadCountCodec) Encode(v *types.DownloadCount) string {
	return strconv.FormatInt(v.EntityID, 10) + ":" + strconv.FormatInt(v.Count, 10)
}

func (downloadCountCodec) Decode(s string) (*types.DownloadCount, error) {
	idStr, countStr, ok := strings.Cut(s, ":")
	if !ok {
		return nil, fmt.Errorf("invalid download count cache value: %s", s)
	}
	entityID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid entity ID in cache value: %s", s)
	}
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid count in cache value: %s", s)
	}
	return &types.DownloadCount{EntityID: entityID, Count: count}, nil
}

// manifestDownloadCountCodec encodes/decodes *types.ManifestDownloadCount as "key:count".
type manifestDownloadCountCodec struct{}

func (manifestDownloadCountCodec) Encode(v *types.ManifestDownloadCount) string {
	return v.Key + "|" + strconv.FormatInt(v.Count, 10)
}

func (manifestDownloadCountCodec) Decode(s string) (*types.ManifestDownloadCount, error) {
	idx := strings.LastIndex(s, "|")
	if idx < 0 {
		return nil, fmt.Errorf("invalid manifest download count cache value: %s", s)
	}
	key := s[:idx]
	count, err := strconv.ParseInt(s[idx+1:], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid count in manifest cache value: %s", s)
	}
	return &types.ManifestDownloadCount{Key: key, Count: count}, nil
}

func logRedisErr(ctx context.Context, err error) {
	log.Ctx(ctx).Warn().Err(err).Msg("download count redis cache error")
}

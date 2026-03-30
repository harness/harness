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

	"github.com/harness/gitness/registry/app/store"
	storecache "github.com/harness/gitness/registry/app/store/cache"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type downloadCountFinder struct {
	registryCache store.DownloadCountRegistryCache
	imageCache    store.DownloadCountImageCache
	artifactCache store.DownloadCountArtifactCache
	manifestCache store.DownloadCountManifestCache
}

func NewDownloadCountFinder(
	registryCache store.DownloadCountRegistryCache,
	imageCache store.DownloadCountImageCache,
	artifactCache store.DownloadCountArtifactCache,
	manifestCache store.DownloadCountManifestCache,
) store.DownloadCountFinder {
	return &downloadCountFinder{
		registryCache: registryCache,
		imageCache:    imageCache,
		artifactCache: artifactCache,
		manifestCache: manifestCache,
	}
}

func (f *downloadCountFinder) FindByRegistryIDs(ctx context.Context, registryIDs []int64) (map[int64]int64, error) {
	dcMap, err := f.registryCache.Map(ctx, registryIDs)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to get download counts for registries from cache")
		return make(map[int64]int64), nil
	}
	return extractInt64Counts(dcMap), nil
}

func (f *downloadCountFinder) FindByImageID(ctx context.Context, imageID int64) (int64, error) {
	dc, err := f.imageCache.Get(ctx, imageID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to get download count for image %d from cache", imageID)
		return 0, nil
	}
	return dc.Count, nil
}

func (f *downloadCountFinder) FindByImageIDs(ctx context.Context, imageIDs []int64) (map[int64]int64, error) {
	dcMap, err := f.imageCache.Map(ctx, imageIDs)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to get download counts for images from cache")
		return make(map[int64]int64), nil
	}
	return extractInt64Counts(dcMap), nil
}

func (f *downloadCountFinder) FindByArtifactID(ctx context.Context, artifactID int64) (int64, error) {
	dc, err := f.artifactCache.Get(ctx, artifactID)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msgf("failed to get download count for artifact %d from cache", artifactID)
		return 0, nil
	}
	return dc.Count, nil
}

func (f *downloadCountFinder) FindByArtifactIDs(ctx context.Context, artifactIDs []int64) (map[int64]int64, error) {
	dcMap, err := f.artifactCache.Map(ctx, artifactIDs)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to get download counts for artifacts from cache")
		return make(map[int64]int64), nil
	}
	return extractInt64Counts(dcMap), nil
}

func (f *downloadCountFinder) FindByManifests(
	ctx context.Context, digests []string, imageID int64,
) (map[string]int64, error) {
	keys := make([]string, len(digests))
	for i, d := range digests {
		keys[i] = storecache.ManifestCacheKey(imageID, d)
	}
	dcMap, err := f.manifestCache.Map(ctx, keys)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to get download counts for manifests from cache")
		return make(map[string]int64), nil
	}
	result := make(map[string]int64, len(dcMap))
	for _, v := range dcMap {
		_, digest, err := storecache.ParseManifestCacheKey(v.Key)
		if err != nil {
			continue
		}
		result[digest] = v.Count
	}
	return result, nil
}

func extractInt64Counts(dcMap map[int64]*types.DownloadCount) map[int64]int64 {
	result := make(map[int64]int64, len(dcMap))
	for k, v := range dcMap {
		result[k] = v.Count
	}
	return result
}

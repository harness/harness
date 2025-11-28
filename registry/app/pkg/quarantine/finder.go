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
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	cache2 "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

// CacheKey represents the cache key for quarantine status.
type CacheKey struct {
	RegistryID   int64
	Image        string
	Version      string
	artifactType *artifact.ArtifactType
}

// Finder provides cached access to quarantine status checks.
type Finder interface {
	// CheckArtifactQuarantineStatus checks if an artifact is quarantined using cache.
	CheckArtifactQuarantineStatus(
		ctx context.Context,
		registryID int64,
		image string,
		version string,
		artifactType *artifact.ArtifactType,
	) error

	// CheckOCIManifestQuarantineStatus checks if an OCI manifest is quarantined using cache.
	CheckOCIManifestQuarantineStatus(
		ctx context.Context,
		registryID int64,
		image string,
		tag string,
		digestStr string,
	) error

	// EvictCache evicts the cache entry for a specific artifact.
	EvictCache(
		ctx context.Context,
		registryID int64,
		image string,
		version string,
		artifactType *artifact.ArtifactType,
	)
}

// finder implements the Finder interface with caching.
type finder struct {
	service         *Service
	quarantineCache cache.Cache[CacheKey, bool]
	evictor         cache2.Evictor[*CacheKey]
}

// NewFinder creates a new quarantine finder that handles caching.
func NewFinder(
	service *Service,
	quarantineCache cache.Cache[CacheKey, bool],
	evictor cache2.Evictor[*CacheKey],
) Finder {
	return &finder{
		service:         service,
		quarantineCache: quarantineCache,
		evictor:         evictor,
	}
}

// CheckArtifactQuarantineStatus checks if an artifact is quarantined.
// It returns nil if the artifact is not quarantined, or an error if it is quarantined.
// Uses cache to avoid repeated database queries.
func (f *finder) CheckArtifactQuarantineStatus(
	ctx context.Context,
	registryID int64,
	image string,
	version string,
	artifactType *artifact.ArtifactType,
) error {
	cacheKey := CacheKey{
		RegistryID:   registryID,
		Image:        image,
		Version:      version,
		artifactType: artifactType,
	}

	// Check cache first
	isQuarantined, err := f.quarantineCache.Get(ctx, cacheKey)
	if err != nil {
		return fmt.Errorf("failed to check quarantine status: %w", err)
	}

	if isQuarantined {
		return usererror.ErrQuarantinedArtifact
	}
	return nil
}

// CheckOCIManifestQuarantineStatus checks if an OCI manifest is quarantined.
// It handles digest resolution from tags and uses the cache for performance.
// Returns nil if not quarantined, or an error if quarantined.
func (f *finder) CheckOCIManifestQuarantineStatus(
	ctx context.Context,
	registryID int64,
	image string,
	tag string,
	digestStr string,
) error {
	// Resolve the digest using the service
	parsedDigest, err := f.service.ResolveDigest(ctx, registryID, image, tag, digestStr)
	if err != nil {
		return fmt.Errorf("error while checking the quarantine status, failed to parse digest: %w", err)
	}

	// If no digest could be resolved, nothing to check
	if parsedDigest == "" {
		return nil
	}

	typesDigest, err := types.NewDigest(parsedDigest)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to parse digest to check quarantine status")
		return fmt.Errorf("error while checking the quarantine status, failed to create digest: %w", err)
	}
	return f.CheckArtifactQuarantineStatus(ctx, registryID, image, typesDigest.String(), nil)
}

// EvictCache evicts the cache entry for a specific artifact.
// This should be called when quarantine status changes.
// It uses the evictor to both evict the cache and publish events.
func (f *finder) EvictCache(
	ctx context.Context,
	registryID int64,
	image string,
	version string,
	artifactType *artifact.ArtifactType,
) {
	cacheKey := CacheKey{
		RegistryID:   registryID,
		Image:        image,
		Version:      version,
		artifactType: artifactType,
	}
	// Use evictor to evict cache and publish event
	f.evictor.Evict(ctx, &cacheKey)
}

// quarantineCacheGetter implements the cache getter interface.
type quarantineCacheGetter struct {
	service *Service
}

func (g quarantineCacheGetter) Find(ctx context.Context, key CacheKey) (bool, error) {
	return g.service.CheckArtifactQuarantineStatus(ctx, key.RegistryID, key.Image, key.Version, key.artifactType)
}

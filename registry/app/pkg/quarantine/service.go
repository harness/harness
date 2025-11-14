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

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

// Service provides quarantine-related operations at the repository layer.
type Service struct {
	quarantineRepo store.QuarantineArtifactRepository
	manifestRepo   store.ManifestRepository
}

// NewService creates a new quarantine service.
func NewService(
	quarantineRepo store.QuarantineArtifactRepository,
	manifestRepo store.ManifestRepository,
) *Service {
	return &Service{
		quarantineRepo: quarantineRepo,
		manifestRepo:   manifestRepo,
	}
}

// CheckArtifactQuarantineStatus checks if an artifact is quarantined by querying the repository.
// Returns true if quarantined, false otherwise.
func (s *Service) CheckArtifactQuarantineStatus(
	ctx context.Context,
	registryID int64,
	image string,
	version string,
	artifactType *artifact.ArtifactType,
) (bool, error) {
	quarantineArtifacts, err := s.quarantineRepo.GetByFilePath(
		ctx, "", registryID, image, version, artifactType,
	)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to check quarantine status")
		return false, fmt.Errorf("failed to check quarantine status: %w", err)
	}
	return len(quarantineArtifacts) > 0, nil
}

// ResolveDigest resolves a digest from either a digest string or a tag name.
// Returns empty string if neither is provided.
func (s *Service) ResolveDigest(
	ctx context.Context,
	registryID int64,
	image string,
	tag string,
	digestStr string,
) (digest.Digest, error) {
	if digestStr != "" {
		parsedDigest, err := digest.Parse(digestStr)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to parse digest to check quarantine status")
			return "", fmt.Errorf("failed to parse digest to check quarantine status: %w", err)
		}
		return parsedDigest, nil
	}

	if tag != "" {
		dbManifest, err := s.manifestRepo.FindManifestDigestByTagName(ctx, registryID, image, tag)
		if err != nil {
			return "", fmt.Errorf("failed to find manifest digest: %w", err)
		}
		parsedDigest, err := dbManifest.Parse()
		if err != nil {
			return "", fmt.Errorf("failed to parse digest: %w", err)
		}
		return parsedDigest, nil
	}

	return "", nil
}

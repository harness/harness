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

package reindexing

import (
	"context"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/services/webhook"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

// Service provides centralized reindexing logic for all deletion and restore flows.
// This ensures consistent reindexing behavior across:
// - Hard delete (existing flow via metadata controller).
// - Hard delete (new flow via deletion service).
// - Soft delete (V2 API).
// - Restore (V2 API).
// - Cleanup jobs.
type Service struct {
	postProcessingReporter *registrypostprocessingevents.Reporter
	artifactEventReporter  registryevents.Reporter
}

// NewService creates a new reindexing service.
func NewService(
	postProcessingReporter *registrypostprocessingevents.Reporter,
	artifactEventReporter registryevents.Reporter,
) *Service {
	return &Service{
		postProcessingReporter: postProcessingReporter,
		artifactEventReporter:  artifactEventReporter,
	}
}

// TriggerArtifactVersionReindexing triggers re-indexing events after artifact version
// deletion/restore based on package type. This should be used consistently across all
// flows: hard delete, soft delete, and restore.
func (s *Service) TriggerArtifactVersionReindexing(
	ctx context.Context,
	packageType artifact.PackageType,
	registryID int64,
	imageName string,
	versionName string,
	principalID int64,
) {
	switch packageType {
	case artifact.PackageTypeRPM:
		// RPM requires registry-level reindexing
		s.postProcessingReporter.BuildRegistryIndex(ctx, registryID, make([]registrytypes.SourceRef, 0))
	case artifact.PackageTypeGO:
		// Send webhook event for Go package artifact deletion
		payload := webhook.GetArtifactDeletedPayloadForCommonArtifacts(
			principalID,
			registryID,
			packageType,
			imageName,
			versionName,
		)
		s.artifactEventReporter.ArtifactDeleted(ctx, &payload)
		// Trigger package index rebuild
		s.postProcessingReporter.BuildPackageIndex(ctx, registryID, imageName)
	case artifact.PackageTypeDOCKER, artifact.PackageTypeHELM, artifact.PackageTypeNPM,
		artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON, artifact.PackageTypeGENERIC,
		artifact.PackageTypeNUGET, artifact.PackageTypeCARGO, artifact.PackageTypeHUGGINGFACE:
		// No reindexing needed for these package types
	default:
		// Unknown package types: log warning
		log.Ctx(ctx).Warn().Msgf("unknown package type for reindexing: %s", packageType)
	}
}

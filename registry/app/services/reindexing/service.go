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
	"fmt"

	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/services/webhook"
	registrytypes "github.com/harness/gitness/registry/types"
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
	packageWrapper         interfaces.PackageWrapper
}

// NewService creates a new reindexing service.
func NewService(
	postProcessingReporter *registrypostprocessingevents.Reporter,
	artifactEventReporter registryevents.Reporter,
	packageWrapper interfaces.PackageWrapper,
) *Service {
	return &Service{
		postProcessingReporter: postProcessingReporter,
		artifactEventReporter:  artifactEventReporter,
		packageWrapper:         packageWrapper,
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
) error {
	switch packageType {
	case artifact.PackageTypeRPM:
		// RPM requires registry-level reindexing
		s.postProcessingReporter.BuildRegistryIndex(ctx, registryID, make([]registrytypes.SourceRef, 0))
		return nil
	case artifact.PackageTypeGO:
		// Send webhook event for Go package artifact deletion
		if principalID > 0 {
			payload := webhook.GetArtifactDeletedPayloadForCommonArtifacts(
				principalID,
				registryID,
				packageType,
				imageName,
				versionName,
			)
			s.artifactEventReporter.ArtifactDeleted(ctx, &payload)
		}
		// Trigger package index rebuild
		s.postProcessingReporter.BuildPackageIndex(ctx, registryID, imageName)
		return nil
	case artifact.PackageTypeDOCKER, artifact.PackageTypeHELM, artifact.PackageTypeNPM,
		artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON, artifact.PackageTypeGENERIC,
		artifact.PackageTypeNUGET:
		// No reindexing needed for these package types
		return nil
		// Cargo/Composer/Conda/Dart/Swift/HuggingFace: Use package wrapper for reindexing
	case artifact.PackageTypeHUGGINGFACE, artifact.PackageTypeCARGO:
		return s.packageWrapper.TriggerIndexEvents(ctx, registryID, imageName, versionName)
	default:
		return s.packageWrapper.TriggerIndexEvents(ctx, registryID, imageName, versionName)
	}
}

// TriggerImageReindexing triggers reindexing for image/package operations WITHOUT deleting entities.
// This is used by soft delete and restore flows to perform the same reindexing as hard delete.
// Only Cargo/Composer/Conda/Dart/Swift package types require reindexing for image-level operations.
func (s *Service) TriggerImageReindexing(
	ctx context.Context,
	regInfo *registrytypes.RegistryRequestBaseInfo,
) error {
	packageType := regInfo.PackageType

	//nolint:exhaustive
	switch packageType {
	case artifact.PackageTypeRPM:
		// RPM does not support image-level operations
		return fmt.Errorf("package-level operations not supported for RPM")
	case artifact.PackageTypeDOCKER, artifact.PackageTypeHELM,
		artifact.PackageTypeGENERIC, artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON,
		artifact.PackageTypeNPM, artifact.PackageTypeNUGET, artifact.PackageTypeGO:
		// Known types: no reindexing needed for image-level operations
		return nil
	case artifact.PackageTypeHUGGINGFACE:
		return fmt.Errorf("unsupported package type: %s", packageType)
	default:
		// Cargo/Composer/Conda/Dart/Swift: Build registry index
		err := s.packageWrapper.ReportBuildRegistryIndexEvent(
			ctx, regInfo.RegistryID, make([]registrytypes.SourceRef, 0),
		)
		if err != nil {
			return fmt.Errorf("failed to report build registry index event: %w", err)
		}
		return nil
	}
}

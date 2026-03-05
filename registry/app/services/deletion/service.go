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

package deletion

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/services/reindexing"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/services/webhook"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// PackageWrapper defines the interface for handling custom package types.
// This matches the interfaces.PackageWrapper interface from the API layer.
type PackageWrapper interface {
	DeleteArtifactVersion(
		ctx context.Context,
		regInfo *registrytypes.RegistryRequestBaseInfo,
		imageInfo *registrytypes.Image,
		artifactName string,
		versionName string,
	) error
	DeleteArtifact(ctx context.Context, regInfo *registrytypes.RegistryRequestBaseInfo, artifactName string) error
}

// Service provides package-type-specific deletion logic for registry entities.
// This service is used by both API controllers and cleanup jobs to ensure consistent deletion behavior.
type Service struct {
	artifactStore         store.ArtifactRepository
	imageStore            store.ImageRepository
	manifestStore         store.ManifestRepository
	tagStore              store.TagRepository
	registryBlobStore     store.RegistryBlobRepository
	fileManager           filemanager.FileManager
	tx                    dbtx.Transactor
	untaggedImagesEnabled func(ctx context.Context) bool
	packageWrapper        PackageWrapper
	reindexingService     *reindexing.Service
	artifactEventReporter *registryevents.Reporter
	urlProvider           url.Provider
}

// NewService creates a new deletion service.
func NewService(
	artifactStore store.ArtifactRepository,
	imageStore store.ImageRepository,
	manifestStore store.ManifestRepository,
	tagStore store.TagRepository,
	registryBlobStore store.RegistryBlobRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	untaggedImagesEnabled func(ctx context.Context) bool,
	packageWrapper PackageWrapper,
	reindexingService *reindexing.Service,
	artifactEventReporter *registryevents.Reporter,
	urlProvider url.Provider,
) *Service {
	return &Service{
		artifactStore:         artifactStore,
		imageStore:            imageStore,
		manifestStore:         manifestStore,
		tagStore:              tagStore,
		registryBlobStore:     registryBlobStore,
		fileManager:           fileManager,
		tx:                    tx,
		untaggedImagesEnabled: untaggedImagesEnabled,
		packageWrapper:        packageWrapper,
		reindexingService:     reindexingService,
		artifactEventReporter: artifactEventReporter,
		urlProvider:           urlProvider,
	}
}

// DeleteImageByPackageType deletes a package.
func (s *Service) DeleteImageByPackageType(
	ctx context.Context,
	regInfo *registrytypes.RegistryRequestBaseInfo,
	packageType artifact.PackageType,
	imageName string,
) error {
	registryID := regInfo.RegistryID

	//nolint:exhaustive
	switch packageType {
	case artifact.PackageTypeDOCKER, artifact.PackageTypeHELM:
		return s.DeleteOCIImage(ctx, registryID, imageName)
	case artifact.PackageTypeGENERIC, artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON,
		artifact.PackageTypeNPM, artifact.PackageTypeNUGET, artifact.PackageTypeGO:
		return s.DeleteGenericImage(ctx, registryID, packageType, imageName)
	case artifact.PackageTypeRPM:
		return fmt.Errorf("delete artifact not supported for rpm")
	case artifact.PackageTypeHUGGINGFACE:
		return fmt.Errorf("unsupported package type: %s", packageType)
	default:
		return s.packageWrapper.DeleteArtifact(ctx, regInfo, imageName)
	}
}

// DeleteArtifactVersionByPackageType deletes an artifact version and triggers reindexing.
// This is the unified method used by both controllers and cleanup jobs.
// For OCI types (Docker/Helm), fires webhook if principalID is non-nil.
// For non-OCI types, reindexing is handled internally.
// principalID is nullable - pass nil for cleanup jobs, non-nil for user-initiated deletions.
// registryName is only needed for webhooks (can be empty for cleanup jobs).
func (s *Service) DeleteArtifactVersionByPackageType(
	ctx context.Context,
	regInfo *registrytypes.RegistryRequestBaseInfo,
	imageName string,
	versionName string,
	principalID *int64,
	registryName string,
) error {
	registryID := regInfo.RegistryID
	packageType := regInfo.PackageType
	var err error

	// Perform deletion based on package type
	//nolint:exhaustive
	switch packageType {
	case artifact.PackageTypeDOCKER, artifact.PackageTypeHELM:
		// OCI types: deletion and webhook handled in DeleteOCIArtifactVersion
		err = s.DeleteOCIArtifactVersion(ctx, regInfo, imageName, versionName, principalID, registryName)
		if err != nil {
			return err
		}
	case artifact.PackageTypeNPM, artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON,
		artifact.PackageTypeGENERIC, artifact.PackageTypeNUGET, artifact.PackageTypeRPM,
		artifact.PackageTypeGO:
		// Non-OCI types: delete + trigger reindexing
		err = s.DeleteGenericArtifact(ctx, registryID, packageType, imageName, versionName)
		if err != nil {
			return err
		}
	default:
		// Unknown types: delegate to package wrapper
		imageInfo, err := s.imageStore.GetByName(ctx, registryID, imageName, registrytypes.WithAllDeleted())
		if err != nil {
			return err
		}
		if err := s.packageWrapper.DeleteArtifactVersion(ctx, regInfo, imageInfo, imageName, versionName); err != nil {
			return err
		}
	}

	// Trigger package-specific reindexing
	var principalIDValue int64
	if principalID != nil {
		principalIDValue = *principalID
	}
	s.reindexingService.TriggerArtifactVersionReindexing(
		ctx, packageType, registryID, imageName, versionName, principalIDValue,
	)

	return nil
}

// DeleteOCIArtifactVersion handles Docker/Helm artifact version deletion with webhook support.
// Similar to original deleteOciVersionWithAudit but moved to service layer.
func (s *Service) DeleteOCIArtifactVersion(
	ctx context.Context,
	regInfo *registrytypes.RegistryRequestBaseInfo,
	imageName string,
	versionName string,
	principalID *int64,
	registryName string,
) error {
	var existingDigest digest.Digest
	//nolint:nestif
	if s.untaggedImagesEnabled(ctx) {
		err := s.tx.WithTx(ctx, func(ctx context.Context) error {
			d := digest.Digest(versionName)
			dgst, _ := registrytypes.NewDigest(d)
			existingManifest, err := s.manifestStore.FindManifestByDigest(
				ctx, regInfo.RegistryID, imageName, dgst,
			)
			if err != nil {
				return fmt.Errorf("failed to find existing manifest for: %s, err: %w", versionName, err)
			}
			if existingManifest.MediaType != v1.MediaTypeImageIndex &&
				existingManifest.MediaType != manifestlist.MediaTypeManifestList {
				manifests, err := s.manifestStore.ReferencedBy(ctx, existingManifest)
				if err != nil {
					return fmt.Errorf("failed to find existing manifests referencing : %s, err: %w",
						versionName, err)
				}
				if len(manifests) > 0 {
					var parentsDigests []string
					for _, m := range manifests {
						parentsDigests = append(parentsDigests, m.Digest.String())
					}
					return fmt.Errorf("cannot delete manifest: %s, as it is referenced by: %s",
						versionName, parentsDigests)
				}
			}
			err = s.manifestStore.Delete(ctx, regInfo.RegistryID, existingManifest.ID)
			if err != nil {
				return err
			}
			existingDigest = d
			_, err = s.tagStore.DeleteTagByManifestID(ctx, regInfo.RegistryID, existingManifest.ID)
			if err != nil {
				return fmt.Errorf("failed to delete tags for: %s, err: %w", versionName, err)
			}
			err = s.artifactStore.DeleteByVersionAndImageName(ctx, imageName, dgst.String(), regInfo.RegistryID)
			if err != nil {
				return err
			}

			count, err := s.manifestStore.CountByImageName(ctx, regInfo.RegistryID, imageName)
			if err != nil {
				return err
			}
			if count < 1 {
				err = s.imageStore.DeleteByImageNameAndRegID(
					ctx, regInfo.RegistryID, imageName,
				)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to delete artifact version: %w", err)
		}
	} else {
		// Non-untagged mode: capture digest before deleting tag
		if tag, err := s.tagStore.FindTag(ctx, regInfo.RegistryID, imageName, versionName); err == nil && tag != nil {
			if manifest, err := s.manifestStore.Get(ctx, tag.ManifestID); err == nil && manifest != nil {
				existingDigest = manifest.Digest
			}
		}
		err := s.tagStore.DeleteTag(ctx, regInfo.RegistryID, imageName, versionName)
		if err != nil {
			return err
		}
	}

	// Fire webhook if principalID provided (user-initiated deletion)
	if principalID != nil && existingDigest != "" {
		payload := webhook.GetArtifactDeletedPayload(ctx, *principalID, regInfo.RegistryID,
			registryName, versionName, existingDigest.String(), regInfo.RootIdentifier,
			regInfo.PackageType, imageName, s.urlProvider, s.untaggedImagesEnabled(ctx))
		s.artifactEventReporter.ArtifactDeleted(ctx, &payload)
	}

	return nil
}

// DeleteGenericArtifact handles generic package deletion (NPM, Maven, Python, etc.).
func (s *Service) DeleteGenericArtifact(
	ctx context.Context,
	registryID int64,
	packageType artifact.PackageType,
	artifactName string,
	versionName string,
) error {
	// Get file path
	filePath, err := utils.GetFilePath(packageType, artifactName, versionName)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}

	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Delete files from storage
		err = s.fileManager.DeleteFile(ctx, registryID, filePath)
		if err != nil {
			return err
		}

		// Delete artifact from DB
		err = s.artifactStore.DeleteByVersionAndImageName(ctx, artifactName, versionName, registryID)
		if err != nil {
			return fmt.Errorf("failed to delete version: %w", err)
		}

		// Delete image if no other artifacts linked
		err = s.imageStore.DeleteByImageNameIfNoLinkedArtifacts(ctx, registryID, artifactName)
		if err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}

		return nil
	})
}

// DeleteOCIImage handles Docker/Helm image deletion (deletes all artifacts, manifests, blobs).
func (s *Service) DeleteOCIImage(
	ctx context.Context,
	registryID int64,
	imageName string,
) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Delete manifests linked to the image
		_, err := s.manifestStore.DeleteManifestByImageName(ctx, registryID, imageName)
		if err != nil {
			return fmt.Errorf("failed to delete manifests: %w", err)
		}

		// Delete registry blobs linked to the image
		_, err = s.registryBlobStore.UnlinkBlobByImageName(ctx, registryID, imageName)
		if err != nil {
			return fmt.Errorf("failed to delete registry blobs: %w", err)
		}

		// Delete all artifacts linked to image
		err = s.artifactStore.DeleteByImageNameAndRegistryID(ctx, registryID, imageName)
		if err != nil {
			return fmt.Errorf("failed to delete artifacts: %w", err)
		}

		// Delete image
		err = s.imageStore.DeleteByImageNameAndRegID(ctx, registryID, imageName)
		if err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}

		return nil
	})
}

// DeleteGenericImage handles generic package image deletion (deletes files and artifacts).
func (s *Service) DeleteGenericImage(
	ctx context.Context,
	registryID int64,
	packageType artifact.PackageType,
	imageName string,
) error {
	// Get file path
	filePath, err := utils.GetFilePath(packageType, imageName, "")
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}

	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Delete files from storage
		err = s.fileManager.DeleteFile(ctx, registryID, filePath)
		if err != nil {
			return fmt.Errorf("failed to delete files: %w", err)
		}

		// Delete all artifacts
		err = s.artifactStore.DeleteByImageNameAndRegistryID(ctx, registryID, imageName)
		if err != nil {
			return fmt.Errorf("failed to delete artifacts: %w", err)
		}

		// Delete image
		err = s.imageStore.DeleteByImageNameAndRegID(ctx, registryID, imageName)
		if err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}

		return nil
	})
}

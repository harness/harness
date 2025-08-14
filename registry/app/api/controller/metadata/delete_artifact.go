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

package metadata

import (
	"context"
	"fmt"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	registryTypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteArtifact(ctx context.Context, r artifact.DeleteArtifactRequestObject) (
	artifact.DeleteArtifactResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.DeleteArtifact400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.DeleteArtifact400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	if err = apiauth.CheckSpaceScope(
		ctx,
		c.Authorizer,
		session,
		space,
		enum.ResourceTypeRegistry,
		enum.PermissionArtifactsDelete,
	); err != nil {
		return artifact.DeleteArtifact403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	repoEntity, err := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.ParentID, regInfo.RegistryIdentifier)
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteArtifact404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
			),
		}, nil
	}

	artifactName := string(r.Artifact)
	_, err = c.ImageStore.GetByName(ctx, regInfo.RegistryID, artifactName)
	if err != nil {
		//nolint:nilerr
		return artifact.DeleteArtifact404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "artifact doesn't exist with this key"),
			),
		}, nil
	}

	switch regInfo.PackageType {
	case artifact.PackageTypeDOCKER:
		err = c.deleteOCIImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeHELM:
		err = c.deleteOCIImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeGENERIC:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeMAVEN:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypePYTHON:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeNPM:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeNUGET:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeRPM:
		err = fmt.Errorf("delete artifact not supported for rpm")
	case artifact.PackageTypeCARGO:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeGO:
		err = c.deleteGenericImage(ctx, regInfo, artifactName)
	case artifact.PackageTypeHUGGINGFACE:
		err = fmt.Errorf("unsupported package type: %s", regInfo.PackageType)
	default:
		err = fmt.Errorf("unsupported package type: %s", regInfo.PackageType)
	}

	if err != nil {
		return throwDeleteArtifact500Error(err), err
	}

	auditErr := c.AuditService.Log(
		ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRegistryArtifact, string(r.Artifact)),
		audit.ActionDeleted,
		regInfo.ParentRef,
		audit.WithData("registry name", repoEntity.Name),
		audit.WithData("artifact name", string(r.Artifact)),
	)
	if auditErr != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete tag operation: %s", auditErr)
	}

	return artifact.DeleteArtifact200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func (c *APIController) deleteOCIImage(
	ctx context.Context,
	regInfo *registryTypes.RegistryRequestBaseInfo,
	artifactName string,
) error {
	err := c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Delete manifests linked to the image
			_, err := c.ManifestStore.DeleteManifestByImageName(ctx, regInfo.RegistryID, artifactName)
			if err != nil {
				return fmt.Errorf("failed to delete manifests: %w", err)
			}

			// Delete registry blobs linked to the image
			_, err = c.RegistryBlobStore.UnlinkBlobByImageName(ctx, regInfo.RegistryID, artifactName)
			if err != nil {
				return fmt.Errorf("failed to delete registry blobs: %w", err)
			}

			// Delete Artifacts linked to image
			err = c.ArtifactStore.DeleteByImageNameAndRegistryID(ctx, regInfo.RegistryID, artifactName)
			if err != nil {
				return fmt.Errorf("failed to delete versions: %w", err)
			}

			// Delete image
			err = c.ImageStore.DeleteByImageNameAndRegID(
				ctx, regInfo.RegistryID, artifactName,
			)
			if err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
			}
			return nil
		},
	)
	return err
}

func (c *APIController) deleteGenericImage(
	ctx context.Context,
	regInfo *registryTypes.RegistryRequestBaseInfo,
	artifactName string,
) error {
	err := c.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Get File Path
			filePath, err := utils.GetFilePath(regInfo.PackageType, artifactName, "")
			if err != nil {
				return fmt.Errorf("failed to get file path: %w", err)
			}
			// Delete Artifact Files
			err = c.fileManager.DeleteNode(ctx, regInfo.RegistryID, filePath)
			if err != nil {
				return fmt.Errorf("failed to delete artifact files: %w", err)
			}
			// Delete Artifacts
			err = c.ArtifactStore.DeleteByImageNameAndRegistryID(ctx, regInfo.RegistryID, artifactName)
			if err != nil {
				return fmt.Errorf("failed to delete versions: %w", err)
			}
			// Delete image
			err = c.ImageStore.DeleteByImageNameAndRegID(
				ctx, regInfo.RegistryID, artifactName,
			)
			if err != nil {
				return fmt.Errorf("failed to delete artifact: %w", err)
			}
			return nil
		},
	)
	return err
}

func throwDeleteArtifact500Error(err error) artifact.DeleteArtifact500JSONResponse {
	return artifact.DeleteArtifact500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}
}

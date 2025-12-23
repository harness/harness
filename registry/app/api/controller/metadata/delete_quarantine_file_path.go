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

package metadata

import (
	"context"
	"net/http"
	"path"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteQuarantineFilePath(
	ctx context.Context,
	r artifact.DeleteQuarantineFilePathRequestObject,
) (artifact.DeleteQuarantineFilePathResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.DeleteQuarantineFilePath400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.DeleteQuarantineFilePath400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryEdit)

	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		statusCode, message := HandleAuthError(err)
		if statusCode == http.StatusUnauthorized {
			return artifact.DeleteQuarantineFilePath401JSONResponse{
				UnauthenticatedJSONResponse: artifact.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, message),
				),
			}, nil
		}
		return artifact.DeleteQuarantineFilePath403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, message),
			),
		}, nil
	}

	artifactName := r.Params.Artifact
	version := r.Params.Version
	filePath := r.Params.FilePath

	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		at := artifact.ArtifactType(*r.Params.ArtifactType)
		artifactType = &at
	}
	img, err := c.ImageStore.GetByNameAndType(ctx, regInfo.RegistryID, string(*artifactName), artifactType)

	if err != nil {
		return artifact.DeleteQuarantineFilePath500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var versionID *int64
	var rootPath string
	var art *types.Artifact
	if version != nil { //nolint:nestif
		var parsedVersion = string(*version)
		if regInfo.PackageType == artifact.PackageTypeDOCKER || regInfo.PackageType == artifact.PackageTypeHELM {
			parsedDigest, err := digest.Parse(parsedVersion)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to parse digest for create quarantine file path")
				return artifact.DeleteQuarantineFilePath500JSONResponse{
					InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
						*GetErrorResponse(http.StatusInternalServerError, err.Error()),
					),
				}, nil
			}
			typesDigest, err := types.NewDigest(parsedDigest)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to create types digest for create quarantine file path")
				return artifact.DeleteQuarantineFilePath500JSONResponse{
					InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
						*GetErrorResponse(http.StatusInternalServerError, err.Error()),
					),
				}, nil
			}
			digestVal := typesDigest.String()
			parsedVersion = digestVal
		}
		art, err = c.ArtifactStore.GetByName(ctx, img.ID, parsedVersion)
		if err != nil {
			return artifact.DeleteQuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		versionID = &art.ID
	}

	var nodeID *string

	if filePath != nil {
		rootPath, err = utils.GetFilePath(regInfo.PackageType, string(*artifactName), string(*version))
		if err != nil {
			return artifact.DeleteQuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		completePath := path.Join(rootPath, string(*filePath))
		node, err := c.fileManager.GetNode(ctx, regInfo.RegistryID, completePath)
		if err != nil {
			return artifact.DeleteQuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		nodeID = &node.ID
	}
	err = c.QuarantineArtifactRepository.DeleteByRegistryIDArtifactAndFilePath(ctx,
		regInfo.RegistryID, versionID, img.ID, nodeID)
	if err != nil {
		return artifact.DeleteQuarantineFilePath500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	// Evict cache after deleting quarantine entry
	if version != nil {
		c.QuarantineFinder.EvictCache(ctx, regInfo.RegistryID, string(*artifactName), art.Version, artifactType)
	}

	return artifact.DeleteQuarantineFilePath200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

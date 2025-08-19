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
	"github.com/harness/gitness/types/enum"
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
		regInfo.RegistryIdentifier, enum.PermissionArtifactsQuarantine)

	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.DeleteQuarantineFilePath403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	artifactName := r.Params.Artifact
	version := r.Params.Version
	filePath := r.Params.FilePath

	img, err := c.ImageStore.GetByName(ctx, regInfo.RegistryID, string(*artifactName))

	if err != nil {
		return artifact.DeleteQuarantineFilePath500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var versionID *int64
	var rootPath string
	if version != nil {
		art, err := c.ArtifactStore.GetByName(ctx, img.ID, string(*version))
		if err != nil {
			return artifact.DeleteQuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		versionID = &art.ID
		rootPath, err = utils.GetFilePath(regInfo.PackageType, string(*artifactName), string(*version))
		if err != nil {
			return artifact.DeleteQuarantineFilePath500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
	}

	var nodeID *string

	if filePath != nil {
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
	return artifact.DeleteQuarantineFilePath200JSONResponse{
		SuccessJSONResponse: artifact.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

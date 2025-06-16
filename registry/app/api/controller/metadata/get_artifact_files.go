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
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetArtifactFiles(
	ctx context.Context,
	r artifact.GetArtifactFilesRequestObject,
) (artifact.GetArtifactFilesResponseObject, error) {
	reqInfo, err := c.GetArtifactFilesRequestInfo(ctx, r)
	if err != nil {
		return artifact.GetArtifactFiles400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	space, err := c.SpaceFinder.FindByRef(ctx, reqInfo.ParentRef)
	if err != nil {
		return artifact.GetArtifactFiles400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		reqInfo.RegistryIdentifier, enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetArtifactFiles403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)
	version := string(r.Version)

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, reqInfo.ParentID, reqInfo.RegistryIdentifier)

	if err != nil {
		return artifact.GetArtifactFiles500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	img, err := c.ImageStore.GetByName(ctx, reqInfo.RegistryID, image)

	if err != nil {
		return artifact.GetArtifactFiles500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	art, err := c.ArtifactStore.GetByName(ctx, img.ID, version)

	if err != nil {
		return artifact.GetArtifactFiles500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	registryURL := c.URLProvider.RegistryURL(ctx,
		reqInfo.RootIdentifier, strings.ToLower(string(registry.PackageType)), reqInfo.RegistryIdentifier)

	if registry.PackageType == artifact.PackageTypeNPM {
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier, "npm")
	}

	filePathPrefix, err := utils.GetFilePath(registry.PackageType, img.Name, art.Version)
	if err != nil {
		return failedToFetchFilesResponse(err, art)
	}
	filePathPattern := filePathPrefix + "/%"
	fileMetadataList, err := c.fileManager.GetFilesMetadata(ctx, filePathPattern, img.RegistryID,
		reqInfo.sortByField, reqInfo.sortByOrder, reqInfo.limit, reqInfo.offset, reqInfo.searchTerm)
	if err != nil {
		return failedToFetchFilesResponse(err, art)
	}

	count, err := c.fileManager.CountFilesByPath(ctx, filePathPattern, img.RegistryID)

	if err != nil {
		log.Error().Msgf("Failed to count files for artifact, err: %v", err.Error())
		return artifact.GetArtifactFiles500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError,
					fmt.Sprintf("Failed to count files for artifact with name: [%s]", art.Version)),
			),
		}, nil
	}

	//nolint:exhaustive
	switch registry.PackageType {
	case artifact.PackageTypeGENERIC, artifact.PackageTypeMAVEN, artifact.PackageTypePYTHON,
		artifact.PackageTypeNPM, artifact.PackageTypeRPM, artifact.PackageTypeNUGET:
		return artifact.GetArtifactFiles200JSONResponse{
			FileDetailResponseJSONResponse: *GetAllArtifactFilesResponse(
				fileMetadataList, count, reqInfo.pageNumber, reqInfo.limit, registryURL, img.Name, art.Version,
				registry.PackageType, c.SetupDetailsAuthHeaderPrefix),
		}, nil
	default:
		return artifact.GetArtifactFiles400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, "Invalid package type"),
			),
		}, nil
	}
}

func failedToFetchFilesResponse(err error, art *types.Artifact) (artifact.GetArtifactFilesResponseObject, error) {
	log.Error().Msgf("Failed to fetch files for artifact, err: %v", err.Error())
	return artifact.GetArtifactFiles500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError,
				fmt.Sprintf("Failed to fetch files for artifact with name: [%s]", art.Version)),
		),
	}, nil
}

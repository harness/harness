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
	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return artifact.GetArtifactFiles400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
	}
	img, err := c.ImageStore.GetByNameAndType(ctx, reqInfo.RegistryID, image, artifactType)

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

	var registryURL string
	switch registry.PackageType { //nolint:exhaustive
	case artifact.PackageTypeGENERIC:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier, "files")
	case artifact.PackageTypeNPM:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier, "npm")
	case artifact.PackageTypeRPM:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier, "rpm")
	case artifact.PackageTypeCARGO:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier, "cargo")
	case artifact.PackageTypeGO:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier, "go")
	case artifact.PackageTypeHUGGINGFACE:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier,
			"huggingface")
	case artifact.PackageTypeMAVEN:
		registryURL = c.URLProvider.PackageURL(ctx, reqInfo.RootIdentifier+"/"+reqInfo.RegistryIdentifier,
			"maven")
	default:
		registryURL = c.URLProvider.RegistryURL(ctx,
			reqInfo.RootIdentifier, strings.ToLower(string(registry.PackageType)), reqInfo.RegistryIdentifier)
	}

	var filePathPrefix string
	var err2 error
	switch registry.PackageType { //nolint:exhaustive
	case artifact.PackageTypeHUGGINGFACE:
		filePathPrefix, err2 = utils.GetFilePathWithArtifactType(registry.PackageType,
			img.Name, art.Version, artifactType)
	default:
		filePathPrefix, err2 = utils.GetFilePath(registry.PackageType, img.Name, art.Version)
	}
	if err2 != nil {
		return failedToFetchFilesResponse(ctx, err2, art)
	}

	filePathPattern := filePathPrefix + "/%"
	fileMetadataList, err := c.fileManager.GetFilesMetadata(ctx, filePathPattern, img.RegistryID,
		reqInfo.sortByField, reqInfo.sortByOrder, reqInfo.limit, reqInfo.offset, reqInfo.searchTerm)
	if err != nil {
		return failedToFetchFilesResponse(ctx, err, art)
	}

	count, err := c.fileManager.CountFilesByPath(ctx, filePathPattern, img.RegistryID)

	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to count files for artifact, err: %v", err.Error())
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
		artifact.PackageTypeNPM, artifact.PackageTypeRPM, artifact.PackageTypeNUGET,
		artifact.PackageTypeCARGO, artifact.PackageTypeGO, artifact.PackageTypeHUGGINGFACE:
		return artifact.GetArtifactFiles200JSONResponse{
			FileDetailResponseJSONResponse: *GetAllArtifactFilesResponse(fileMetadataList, count, reqInfo.pageNumber,
				reqInfo.limit, registryURL, img.Name, art.Version, registry.PackageType,
				c.SetupDetailsAuthHeaderPrefix, artifactType),
		}, nil
	default:
		return artifact.GetArtifactFiles400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, "Invalid package type"),
			),
		}, nil
	}
}

func failedToFetchFilesResponse(
	ctx context.Context,
	err error,
	art *types.Artifact,
) (artifact.GetArtifactFilesResponseObject, error) {
	log.Ctx(ctx).Error().Msgf("Failed to fetch files for artifact, err: %v", err.Error())
	return artifact.GetArtifactFiles500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError,
				fmt.Sprintf("Failed to fetch files for artifact with name: [%s]", art.Version)),
		),
	}, nil
}

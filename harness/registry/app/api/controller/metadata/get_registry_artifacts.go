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
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetAllArtifactsByRegistry(
	ctx context.Context,
	r artifact.GetAllArtifactsByRegistryRequestObject,
) (artifact.GetAllArtifactsByRegistryResponseObject, error) {
	registryRequestParams := &RegistryRequestParams{
		packageTypesParam: nil,
		page:              r.Params.Page,
		size:              r.Params.Size,
		search:            r.Params.SearchTerm,
		Resource:          ArtifactResource,
		ParentRef:         "",
		RegRef:            string(r.RegistryRef),
		labelsParam:       nil,
		sortOrder:         r.Params.SortOrder,
		sortField:         r.Params.SortField,
		registryIDsParam:  nil,
	}
	regInfo, err := c.GetRegistryRequestInfo(ctx, *registryRequestParams)
	if err != nil {
		return c.getAllArtifactsByRegistry400JsonResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetAllArtifactsByRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetAllArtifactsByRegistry403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	registry, err := c.RegistryRepository.GetByParentIDAndName(ctx, space.ID, regInfo.RegistryIdentifier)
	if err != nil {
		return artifact.GetAllArtifactsByRegistry500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	var artifacts *[]types.ArtifactMetadata
	var count int64
	var artifactType *artifact.ArtifactType
	if r.Params.ArtifactType != nil {
		artifactType, err = ValidateAndGetArtifactType(registry.PackageType, string(*r.Params.ArtifactType))
		if err != nil {
			return artifact.GetAllArtifactsByRegistry400JSONResponse{
				BadRequestJSONResponse: artifact.BadRequestJSONResponse(
					*GetErrorResponse(http.StatusBadRequest, err.Error()),
				),
			}, nil
		}
	}
	//nolint:nestif
	if registry.PackageType == artifact.PackageTypeDOCKER || registry.PackageType == artifact.PackageTypeHELM {
		artifacts, err = c.TagStore.GetAllArtifactsByRepo(
			ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
			regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm, regInfo.labels,
		)
		count, _ = c.TagStore.CountAllArtifactsByRepo(
			ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
			regInfo.searchTerm, regInfo.labels,
		)
		if err != nil {
			return artifact.GetAllArtifactsByRegistry500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
	} else {
		artifacts, err = c.ArtifactStore.GetArtifactsByRepo(
			ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
			regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm, regInfo.labels,
			artifactType)
		count, _ = c.ArtifactStore.CountArtifactsByRepo(
			ctx, regInfo.ParentID, regInfo.RegistryIdentifier,
			regInfo.searchTerm, regInfo.labels, artifactType)
		if err != nil {
			return artifact.GetAllArtifactsByRegistry500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
	}
	artifacts, err = c.enrichArtifactWithQuarantineInfo(ctx, artifacts, registry.ID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to fetch the quarantine info for artifacts")
		return artifact.GetAllArtifactsByRegistry500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.GetAllArtifactsByRegistry200JSONResponse{
		ListRegistryArtifactResponseJSONResponse: *GetAllArtifactByRegistryResponse(
			artifacts, count, regInfo.pageNumber, regInfo.limit,
		),
	}, nil
}

func (c *APIController) enrichArtifactWithQuarantineInfo(
	ctx context.Context,
	artifacts *[]types.ArtifactMetadata,
	registryID int64,
) (*[]types.ArtifactMetadata, error) {
	if artifacts == nil || len(*artifacts) == 0 {
		return artifacts, nil
	}

	// Collect unique image names
	imageNameMap := make(map[string]bool)
	var imageNames []string

	for _, artifact := range *artifacts {
		if !imageNameMap[artifact.Name] {
			imageNameMap[artifact.Name] = true
			imageNames = append(imageNames, artifact.Name)
		}
	}

	// Get quarantine status for all images
	quarantineStatuses, err := c.TagStore.GetQuarantineStatusForImages(ctx, imageNames, registryID)
	if err != nil {
		return nil, err
	}

	// Create a map of image name to quarantine status
	quarantineStatusMap := make(map[string]bool)
	for i, imageName := range imageNames {
		if i < len(quarantineStatuses) {
			quarantineStatusMap[imageName] = quarantineStatuses[i]
		}
	}

	// Enrich artifacts with quarantine information
	for i := range *artifacts {
		artifact := &(*artifacts)[i]
		if isQuarantined, exists := quarantineStatusMap[artifact.Name]; exists {
			artifact.IsQuarantined = isQuarantined
		} else {
			artifact.IsQuarantined = false
		}
	}

	return artifacts, nil
}

func (c *APIController) getAllArtifactsByRegistry400JsonResponse(err error) (
	artifact.GetAllArtifactsByRegistryResponseObject, error,
) {
	return artifact.GetAllArtifactsByRegistry400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

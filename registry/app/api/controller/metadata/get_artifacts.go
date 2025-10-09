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
	"github.com/harness/gitness/registry/utils"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) GetAllArtifacts(
	ctx context.Context,
	r artifact.GetAllArtifactsRequestObject,
) (artifact.GetAllArtifactsResponseObject, error) {
	registryRequestParams := &RegistryRequestParams{
		packageTypesParam: r.Params.PackageType,
		page:              r.Params.Page,
		size:              r.Params.Size,
		search:            r.Params.SearchTerm,
		Resource:          ArtifactResource,
		ParentRef:         string(r.SpaceRef),
		RegRef:            "",
		labelsParam:       nil,
		sortOrder:         r.Params.SortOrder,
		sortField:         r.Params.SortField,
		registryIDsParam:  r.Params.RegIdentifier,
	}

	regInfo, err := c.GetRegistryRequestInfo(ctx, *registryRequestParams)
	if err != nil {
		return c.getAllArtifacts400JsonResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetAllArtifacts400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return artifact.GetAllArtifacts403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}
	latestVersion := false
	if r.Params.LatestVersion != nil {
		latestVersion = bool(*r.Params.LatestVersion)
	}
	var artifacts *[]types.ArtifactMetadata
	if c.UntaggedImagesEnabled(ctx) {
		artifacts, err = c.TagStore.GetAllArtifactsByParentIDUntagged(
			ctx, regInfo.ParentID, &regInfo.registryIDs,
			regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm,
			regInfo.packageTypes)
	} else {
		artifacts, err = c.TagStore.GetAllArtifactsByParentID(
			ctx, regInfo.ParentID, &regInfo.registryIDs,
			regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm,
			latestVersion, regInfo.packageTypes)
	}
	count, _ := c.TagStore.CountAllArtifactsByParentID(
		ctx, regInfo.ParentID, &regInfo.registryIDs,
		regInfo.searchTerm, latestVersion, regInfo.packageTypes, c.UntaggedImagesEnabled(ctx))
	if err != nil {
		return artifact.GetAllArtifacts500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	// Enrich artifacts with quarantine information
	if artifacts != nil && len(*artifacts) > 0 {
		enrichedArtifacts, err := c.enrichArtifactsWithQuarantineInfo(ctx, artifacts, regInfo.ParentID)
		if err != nil {
			return artifact.GetAllArtifacts500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
		artifacts = enrichedArtifacts
	}

	return artifact.GetAllArtifacts200JSONResponse{
		ListArtifactResponseJSONResponse: *GetAllArtifactResponse(ctx, artifacts, count, regInfo.pageNumber, regInfo.limit,
			regInfo.RootIdentifier, c.URLProvider, c.SetupDetailsAuthHeaderPrefix, c.UntaggedImagesEnabled(ctx),
			c.PackageWrapper),
	}, nil
}

// enrichArtifactsWithQuarantineInfo enriches artifacts with quarantine information.
func (c *APIController) enrichArtifactsWithQuarantineInfo(
	ctx context.Context,
	artifacts *[]types.ArtifactMetadata,
	parentID int64,
) (*[]types.ArtifactMetadata, error) {
	if artifacts == nil || len(*artifacts) == 0 {
		return artifacts, nil
	}

	// Collect all artifact identifiers and map them to their indices
	artifactIdentifiers := make([]types.ArtifactIdentifier, 0, len(*artifacts))
	artifactIndexMap := make(map[types.ArtifactIdentifier]int) // artifactIdentifier -> index in artifacts slice

	for i, art := range *artifacts {
		// Only process artifacts that have a version
		if art.Version != "" {
			var version = art.Version
			var err error

			// For Docker and Helm packages, convert version to parsed digest string
			if c.UntaggedImagesEnabled(ctx) &&
				(art.PackageType == artifact.PackageTypeDOCKER || art.PackageType == artifact.PackageTypeHELM) {
				version, err = utils.GetParsedDigest(version)
				if err != nil {
					log.Ctx(ctx).Warn().Err(err).
						Msgf("Failed to parse digest info while fetching global artifacts")
					return nil, err
				}
			}

			artifactID := types.ArtifactIdentifier{
				Name:         art.Name,
				Version:      version,
				RegistryName: art.RepoName,
			}
			artifactIdentifiers = append(artifactIdentifiers, artifactID)
			artifactIndexMap[artifactID] = i
		}
	}

	// Get quarantine information for all artifacts using regID
	quarantineMap, err := c.TagStore.GetQuarantineInfoForArtifacts(ctx, artifactIdentifiers, parentID)
	if err != nil {
		return nil, err
	}

	// Update artifacts with quarantine information
	for artifactID, quarantineInfo := range quarantineMap {
		if index, exists := artifactIndexMap[artifactID]; exists {
			(*artifacts)[index].IsQuarantined = true
			(*artifacts)[index].QuarantineReason = &quarantineInfo.Reason
		}
	}

	return artifacts, nil
}

func (c *APIController) getAllArtifacts400JsonResponse(err error) (artifact.GetAllArtifactsResponseObject, error) {
	return artifact.GetAllArtifacts400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

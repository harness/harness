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
	"path"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

func (c *APIController) GetAllRegistries(
	ctx context.Context,
	r artifact.GetAllRegistriesRequestObject,
) (artifact.GetAllRegistriesResponseObject, error) {
	registryRequestParams := &RegistryRequestParams{
		packageTypesParam: r.Params.PackageType,
		page:              r.Params.Page,
		size:              r.Params.Size,
		search:            r.Params.SearchTerm,
		Resource:          RepositoryResource,
		ParentRef:         string(r.SpaceRef),
		RegRef:            "",
		labelsParam:       nil,
		sortOrder:         r.Params.SortOrder,
		sortField:         r.Params.SortField,
		registryIDsParam:  nil,
		recursive:         r.Params.Recursive,
		scope:             r.Params.Scope,
	}
	regInfo, err := c.GetRegistryRequestInfo(ctx, *registryRequestParams)
	if err != nil {
		return artifact.GetAllRegistries400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetAllRegistries400JSONResponse{
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
		enum.PermissionRegistryView,
	); err != nil {
		return artifact.GetAllRegistries403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	var repos *[]store.RegistryMetadata
	repoType := ""
	if r.Params.Type != nil {
		repoType = string(*r.Params.Type)
	}
	ok := c.PackageWrapper.IsValidPackageTypes(regInfo.packageTypes)
	if !ok {
		return artifact.GetAllRegistries400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, "invalid package type"),
			),
		}, nil
	}
	e := ValidateScope(regInfo.scope)
	if e != nil {
		return artifact.GetAllRegistries400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, e.Error()),
			),
		}, nil
	}
	ok = c.PackageWrapper.IsValidRepoType(repoType)
	if !ok {
		return artifact.GetAllRegistries400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, "invalid repository type"),
			),
		}, nil
	}

	parentIDs := []int64{regInfo.ParentID}

	if regInfo.recursive || regInfo.scope == string(artifact.Ancestors) {
		parentIDs, err = c.SpaceStore.GetAncestorIDs(ctx, regInfo.ParentID)
		if err != nil {
			return artifact.GetAllRegistries500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
	}

	if regInfo.scope == string(artifact.Descendants) {
		parentIDs, err = c.SpaceStore.GetDescendantsIDs(ctx, regInfo.ParentID)
		if err != nil {
			return artifact.GetAllRegistries500JSONResponse{
				InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
					*GetErrorResponse(http.StatusInternalServerError, err.Error()),
				),
			}, nil
		}
	}

	var count int64
	repos, err = c.RegistryRepository.GetAll(
		ctx,
		parentIDs,
		regInfo.packageTypes,
		regInfo.sortByField,
		regInfo.sortByOrder,
		regInfo.limit,
		regInfo.offset,
		regInfo.searchTerm,
		repoType,
	)
	count, _ = c.RegistryRepository.CountAll(
		ctx,
		parentIDs,
		regInfo.packageTypes,
		regInfo.searchTerm,
		repoType,
	)
	if err != nil {
		return artifact.GetAllRegistries500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.GetAllRegistries200JSONResponse{
		ListRegistryResponseJSONResponse: *c.GetAllRegistryResponse(ctx,
			repos, count, regInfo.pageNumber,
			regInfo.limit, regInfo.RootIdentifier, c.URLProvider,
		),
	}, nil
}

func (c *APIController) GetAllRegistryResponse(
	ctx context.Context,
	repos *[]store.RegistryMetadata,
	count int64,
	pageNumber int64,
	pageSize int,
	rootIdentifier string,
	urlProvider url.Provider,
) *artifact.ListRegistryResponseJSONResponse {
	repoMetadataList := c.GetRegistryMetadata(ctx, repos, rootIdentifier, urlProvider)
	pageCount := GetPageCount(count, pageSize)
	listRepository := &artifact.ListRegistry{
		ItemCount:  &count,
		PageCount:  &pageCount,
		PageIndex:  &pageNumber,
		PageSize:   &pageSize,
		Registries: repoMetadataList,
	}
	response := &artifact.ListRegistryResponseJSONResponse{
		Data:   *listRepository,
		Status: artifact.StatusSUCCESS,
	}
	return response
}

func (c *APIController) GetRegistryMetadata(
	ctx context.Context,
	registryMetadatas *[]store.RegistryMetadata,
	rootIdentifier string,
	urlProvider url.Provider,
) []artifact.RegistryMetadata {
	repoMetadataList := []artifact.RegistryMetadata{}
	for _, reg := range *registryMetadatas {
		modifiedAt := GetTimeInMs(reg.LastModified)
		var labels *[]string
		if !commons.IsEmpty(reg.Labels) {
			temp := []string(reg.Labels)
			labels = &temp
		}
		var description string
		if !commons.IsEmpty(reg.Description) {
			description = reg.Description
		}
		var artifactCount *int64
		if reg.ArtifactCount != 0 {
			artifactCount = ptr.Int64(reg.ArtifactCount)
		}
		var downloadCount *int64
		if reg.DownloadCount != 0 {
			downloadCount = ptr.Int64(reg.DownloadCount)
		}

		regURL := urlProvider.RegistryURL(ctx, rootIdentifier, reg.RegIdentifier)
		if reg.PackageType == artifact.PackageTypeGENERIC {
			regURL = urlProvider.RegistryURL(ctx, rootIdentifier, "generic", reg.RegIdentifier)
		}

		path := c.GetRegistryPath(ctx, reg.ParentID, reg.RegIdentifier)
		// fix: refactor it
		size := GetSize(reg.Size)
		repoMetadata := artifact.RegistryMetadata{
			Identifier:     reg.RegIdentifier,
			Description:    &description,
			PackageType:    reg.PackageType,
			Type:           reg.Type,
			LastModified:   &modifiedAt,
			Url:            regURL,
			ArtifactsCount: artifactCount,
			DownloadsCount: downloadCount,
			RegistrySize:   &size,
			Labels:         labels,
			Path:           &path,
		}
		repoMetadataList = append(repoMetadataList, repoMetadata)
	}
	return repoMetadataList
}

func (c *APIController) GetRegistryPath(ctx context.Context, parentID int64, regIdentifier string) string {
	if parentID != 0 {
		space, err := c.SpaceFinder.FindByID(ctx, parentID)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("Failed to find space by id %d: %s", parentID, err.Error())
			return ""
		}
		if space != nil {
			return path.Join(space.Path, regIdentifier)
		}
	}
	return ""
}

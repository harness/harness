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
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

func (c *APIController) GetAllRegistries(
	ctx context.Context,
	r artifact.GetAllRegistriesRequestObject,
) (artifact.GetAllRegistriesResponseObject, error) {
	registryRequestParams := &RegistryRequestParams{
		packageTypesParam: nil,
		page:              r.Params.Page,
		size:              r.Params.Size,
		search:            r.Params.SearchTerm,
		resource:          RepositoryResource,
		parentRef:         string(r.SpaceRef),
		regRef:            "",
		labelsParam:       nil,
		sortOrder:         r.Params.SortOrder,
		sortField:         r.Params.SortField,
		registryIDsParam:  nil,
	}
	regInfo, _ := c.GetRegistryRequestInfo(ctx, *registryRequestParams)

	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
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
	e := ValidatePackageTypes(regInfo.packageTypes)
	if e != nil {
		return nil, e
	}
	e = ValidateRepoType(repoType)
	if e != nil {
		return nil, e
	}
	var count int64
	repos, err = c.RegistryRepository.GetAll(
		ctx,
		regInfo.parentID,
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
		regInfo.parentID,
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
		ListRegistryResponseJSONResponse: *GetAllRegistryResponse(
			repos, count, regInfo.pageNumber,
			regInfo.limit, regInfo.RootIdentifier, c.URLProvider.RegistryURL(),
		),
	}, nil
}

func GetAllRegistryResponse(
	repos *[]store.RegistryMetadata,
	count int64,
	pageNumber int64,
	pageSize int,
	rootIdentifier string,
	registryURL string,
) *artifact.ListRegistryResponseJSONResponse {
	repoMetadataList := GetRegistryMetadata(repos, rootIdentifier, registryURL)
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

func GetRegistryMetadata(
	registryMetadatas *[]store.RegistryMetadata,
	rootIdentifier string,
	registryURL string,
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
		// fix: refactor it
		size := GetSize(reg.Size)
		repoMetadata := artifact.RegistryMetadata{
			Identifier:     reg.RegIdentifier,
			Description:    &description,
			PackageType:    reg.PackageType,
			Type:           reg.Type,
			LastModified:   &modifiedAt,
			Url:            GetRepoURL(rootIdentifier, reg.RegIdentifier, registryURL),
			ArtifactsCount: artifactCount,
			DownloadsCount: downloadCount,
			RegistrySize:   &size,
			Labels:         labels,
		}
		repoMetadataList = append(repoMetadataList, repoMetadata)
	}
	return repoMetadataList
}

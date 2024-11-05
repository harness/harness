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
	"github.com/harness/gitness/types/enum"
)

func (c *APIController) GetAllArtifacts(
	ctx context.Context,
	r artifact.GetAllArtifactsRequestObject,
) (artifact.GetAllArtifactsResponseObject, error) {
	registryRequestParams := &RegistryRequestParams{
		packageTypesParam: nil,
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

	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetAllArtifacts400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := GetPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryView)
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
	artifacts, err := c.TagStore.GetAllArtifactsByParentID(
		ctx, regInfo.parentID, &regInfo.registryIDs,
		regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm, latestVersion)
	count, _ := c.TagStore.CountAllArtifactsByParentID(
		ctx, regInfo.parentID, &regInfo.registryIDs,
		regInfo.searchTerm, latestVersion)
	if err != nil {
		return artifact.GetAllArtifacts500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.GetAllArtifacts200JSONResponse{
		ListArtifactResponseJSONResponse: *GetAllArtifactResponse(artifacts, count, regInfo.pageNumber, regInfo.limit,
			c.URLProvider.RegistryRefURL(ctx, regInfo.RegistryRef)),
	}, nil
}

func (c *APIController) getAllArtifacts400JsonResponse(err error) (artifact.GetAllArtifactsResponseObject, error) {
	return artifact.GetAllArtifacts400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

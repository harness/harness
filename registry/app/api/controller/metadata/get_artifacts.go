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
)

func (c *APIController) GetAllArtifacts(
	ctx context.Context,
	r artifact.GetAllArtifactsRequestObject,
) (artifact.GetAllArtifactsResponseObject, error) {
	ref := ""
	if r.Params.RegIdentifier != nil {
		ref2, err2 := GetRegRef(string(r.SpaceRef), string(*r.Params.RegIdentifier))
		if err2 != nil {
			return c.getAllArtifacts400JsonResponse(err2)
		}
		ref = ref2
	}

	regInfo, err := c.GetRegistryRequestInfo(
		ctx, r.Params.PackageType, r.Params.Page, r.Params.Size,
		r.Params.SearchTerm, ArtifactResource, string(r.SpaceRef), ref, r.Params.Label,
		r.Params.SortOrder, r.Params.SortField,
	)
	if err != nil {
		return c.getAllArtifacts400JsonResponse(err)
	}

	space, err := c.spaceStore.FindByRef(ctx, regInfo.parentRef)
	if err != nil {
		return artifact.GetAllArtifacts400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := getPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryView)
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

	var artifacts *[]types.ArtifactMetadata
	var count int64
	if len(regInfo.RegistryIdentifier) == 0 {
		artifacts, err = c.TagStore.GetAllArtifactsByParentID(
			ctx, regInfo.parentID, &regInfo.packageTypes,
			regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm, regInfo.labels,
		)
		count, _ = c.TagStore.CountAllArtifactsByParentID(
			ctx, regInfo.parentID, &regInfo.packageTypes,
			regInfo.searchTerm, regInfo.labels,
		)
	} else {
		artifacts, err = c.TagStore.GetAllArtifactsByRepo(
			ctx, regInfo.parentID, regInfo.RegistryIdentifier,
			regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm, regInfo.labels,
		)
		count, _ = c.TagStore.CountAllArtifactsByRepo(
			ctx, regInfo.parentID, regInfo.RegistryIdentifier,
			regInfo.searchTerm, regInfo.labels,
		)
	}
	if err != nil {
		return artifact.GetAllArtifacts500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.GetAllArtifacts200JSONResponse{
		ListArtifactResponseJSONResponse: *GetAllArtifactResponse(artifacts, count, regInfo.pageNumber, regInfo.limit),
	}, nil
}

func (c *APIController) getAllArtifacts400JsonResponse(err error) (artifact.GetAllArtifactsResponseObject, error) {
	return artifact.GetAllArtifacts400JSONResponse{
		BadRequestJSONResponse: artifact.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

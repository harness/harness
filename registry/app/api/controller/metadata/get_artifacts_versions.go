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

func (c *APIController) GetAllArtifactVersions(
	ctx context.Context,
	r artifact.GetAllArtifactVersionsRequestObject,
) (artifact.GetAllArtifactVersionsResponseObject, error) {
	regInfo, _ := c.GetRegistryRequestInfo(
		ctx, nil, r.Params.Page, r.Params.Size,
		r.Params.SearchTerm, ArtifactVersionResource, "", string(r.RegistryRef),
		nil, r.Params.SortOrder, r.Params.SortField,
	)

	space, err := c.spaceStore.FindByRef(ctx, regInfo.parentRef)
	if err != nil {
		return artifact.GetAllArtifactVersions400JSONResponse{
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
		return artifact.GetAllArtifactVersions403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	image := string(r.Artifact)

	tags, err := c.TagStore.GetAllTagsByRepoAndImage(
		ctx, regInfo.parentID, regInfo.RegistryIdentifier,
		image, regInfo.sortByField, regInfo.sortByOrder, regInfo.limit, regInfo.offset, regInfo.searchTerm,
	)

	latestTag, _ := c.TagStore.GetLatestTagName(ctx, regInfo.parentID, regInfo.RegistryIdentifier, image)

	count, _ := c.TagStore.CountAllTagsByRepoAndImage(
		ctx, regInfo.parentID, regInfo.RegistryIdentifier,
		image, regInfo.searchTerm,
	)

	if err != nil {
		return artifact.GetAllArtifactVersions500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}

	return artifact.GetAllArtifactVersions200JSONResponse{
		ListArtifactVersionResponseJSONResponse: *GetAllArtifactVersionResponse(
			ctx, tags, latestTag, image, count,
			regInfo, regInfo.pageNumber, regInfo.limit, regInfo.rootIdentifier, c.URLProvider.RegistryURL(),
		),
	}, nil
}

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

func (c *APIController) ListArtifactLabels(
	ctx context.Context,
	r artifact.ListArtifactLabelsRequestObject,
) (artifact.ListArtifactLabelsResponseObject, error) {
	regInfo, _ := c.GetRegistryRequestInfo(
		ctx, nil, r.Params.Page, r.Params.Size,
		r.Params.SearchTerm, ArtifactResource, "", string(r.RegistryRef),
		nil, nil, nil,
	)

	space, err := c.spaceStore.FindByRef(ctx, regInfo.parentRef)
	if err != nil {
		return artifact.ListArtifactLabels400JSONResponse{
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
		return artifact.ListArtifactLabels403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	labels, err := c.ArtifactStore.GetLabelsByParentIDAndRepo(
		ctx, regInfo.parentID,
		regInfo.RegistryIdentifier, regInfo.limit, regInfo.offset, regInfo.searchTerm,
	)
	count, _ := c.ArtifactStore.CountLabelsByParentIDAndRepo(
		ctx, regInfo.parentID,
		regInfo.RegistryIdentifier, regInfo.searchTerm,
	)

	if err != nil {
		return artifact.ListArtifactLabels500JSONResponse{
			InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
				*GetErrorResponse(http.StatusInternalServerError, err.Error()),
			),
		}, nil
	}
	return artifact.ListArtifactLabels200JSONResponse{
		ListArtifactLabelResponseJSONResponse: *GetAllArtifactLabelsResponse(
			&labels, count,
			regInfo.pageNumber, regInfo.limit,
		),
	}, nil
}

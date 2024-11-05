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

func (c *APIController) GetRegistry(
	ctx context.Context,
	r artifact.GetRegistryRequestObject,
) (artifact.GetRegistryResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return artifact.GetRegistry400JSONResponse{
			BadRequestJSONResponse: artifact.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, nil
	}
	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return artifact.GetRegistry400JSONResponse{
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
		return artifact.GetRegistry403JSONResponse{
			UnauthorizedJSONResponse: artifact.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}
	repoEntity, _ := c.RegistryRepository.GetByParentIDAndName(ctx, regInfo.parentID, regInfo.RegistryIdentifier)
	if string(repoEntity.Type) == string(artifact.RegistryTypeVIRTUAL) {
		cleanupPolicies, err := c.CleanupPolicyStore.GetByRegistryID(ctx, repoEntity.ID)
		if err != nil {
			return throwGetRegistry500Error(err), nil
		}
		if len(repoEntity.Name) == 0 {
			return artifact.GetRegistry404JSONResponse{
				NotFoundJSONResponse: artifact.NotFoundJSONResponse(
					*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
				),
			}, nil
		}
		return artifact.GetRegistry200JSONResponse{
			RegistryResponseJSONResponse: *CreateVirtualRepositoryResponse(
				repoEntity, c.getUpstreamProxyKeys(
					ctx,
					repoEntity.UpstreamProxies,
				), cleanupPolicies, c.URLProvider.RegistryRefURL(ctx, regInfo.RegistryRef),
			),
		}, nil
	}
	upstreamproxyEntity, err := c.UpstreamProxyStore.GetByRegistryIdentifier(
		ctx,
		regInfo.parentID, regInfo.RegistryIdentifier,
	)
	if len(upstreamproxyEntity.RepoKey) == 0 {
		return artifact.GetRegistry404JSONResponse{
			NotFoundJSONResponse: artifact.NotFoundJSONResponse(
				*GetErrorResponse(http.StatusNotFound, "registry doesn't exist with this key"),
			),
		}, nil
	}
	if err != nil {
		return throwGetRegistry500Error(err), nil
	}
	return artifact.GetRegistry200JSONResponse{
		RegistryResponseJSONResponse: *CreateUpstreamProxyResponseJSONResponse(upstreamproxyEntity),
	}, nil
}

func throwGetRegistry500Error(err error) artifact.GetRegistry500JSONResponse {
	return artifact.GetRegistry500JSONResponse{
		InternalServerErrorJSONResponse: artifact.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}
}

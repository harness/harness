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
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/types/enum"
)

func (c *APIController) DeleteWebhook(
	ctx context.Context,
	r api.DeleteWebhookRequestObject,
) (api.DeleteWebhookResponseObject, error) {
	regInfo, err := c.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return deleteWebhookInternalErrorResponse(err)
	}

	space, err := c.SpaceStore.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return deleteWebhookInternalErrorResponse(err)
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := GetPermissionChecks(space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		return api.DeleteWebhook403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	webhookIdentifier := string(r.WebhookIdentifier)
	err = c.WebhooksRepository.DeleteByRegistryAndIdentifier(ctx, regInfo.RegistryID, webhookIdentifier)
	if err != nil {
		return deleteWebhookInternalErrorResponse(err)
	}
	return api.DeleteWebhook200JSONResponse{
		SuccessJSONResponse: api.SuccessJSONResponse(*GetSuccessResponse()),
	}, nil
}

func deleteWebhookInternalErrorResponse(err error) (api.DeleteWebhookResponseObject, error) {
	return api.DeleteWebhook500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, err
}

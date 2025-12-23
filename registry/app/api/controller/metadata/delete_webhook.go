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
	"fmt"
	"net/http"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *APIController) DeleteWebhook(
	ctx context.Context,
	r api.DeleteWebhookRequestObject,
) (api.DeleteWebhookResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return deleteWebhookInternalErrorResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return deleteWebhookInternalErrorResponse(err)
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space,
		regInfo.RegistryIdentifier, enum.PermissionRegistryEdit)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		statusCode, message := HandleAuthError(err)
		if statusCode == http.StatusUnauthorized {
			return api.DeleteWebhook401JSONResponse{
				UnauthenticatedJSONResponse: api.UnauthenticatedJSONResponse(
					*GetErrorResponse(http.StatusUnauthorized, message),
				),
			}, nil
		}
		return api.DeleteWebhook403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, message),
			),
		}, nil
	}

	webhookIdentifier := string(r.WebhookIdentifier)
	existingWebhook, err := c.WebhooksRepository.GetByRegistryAndIdentifier(ctx, regInfo.RegistryID, webhookIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get existing webhook: %s with error: %v",
			webhookIdentifier, err)
		return deleteWebhookInternalErrorResponse(fmt.Errorf("failed to get existing webhook"))
	}
	if existingWebhook.Type == enum.WebhookTypeInternal {
		return deleteWebhookBadRequestErrorResponse(fmt.Errorf("cannot delete internal webhook: %s", webhookIdentifier))
	}

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
	}, nil
}

func deleteWebhookBadRequestErrorResponse(err error) (api.DeleteWebhookResponseObject, error) {
	return api.DeleteWebhook400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

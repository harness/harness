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

func (c *APIController) UpdateWebhook(
	ctx context.Context,
	r api.UpdateWebhookRequestObject,
) (api.UpdateWebhookResponseObject, error) {
	webhookRequest := api.WebhookRequest(*r.Body)
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return updateWebhookInternalErrorResponse(err)
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return updateWebhookInternalErrorResponse(err)
	}
	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryEdit)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		log.Ctx(ctx).Error().Msgf("permission check failed while updating webhook for registry: %s, error: %v",
			regInfo.RegistryIdentifier, err)
		return api.UpdateWebhook403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}
	existingWebhook, err := c.WebhooksRepository.GetByRegistryAndIdentifier(ctx,
		regInfo.RegistryID, webhookRequest.Identifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get existing webhook: %s with error: %v",
			webhookRequest.Identifier, err)
		return updateWebhookInternalErrorResponse(fmt.Errorf("failed to get existing webhook"))
	}
	if existingWebhook.Type == enum.WebhookTypeInternal {
		return updateWebhookBadRequestErrorResponse(fmt.Errorf("cannot edit internal webhook: %s",
			webhookRequest.Identifier))
	}

	webhook, err := c.RegistryMetadataHelper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to update webhook: %s with error: %v", webhookRequest.Identifier, err)
		return updateWebhookBadRequestErrorResponse(fmt.Errorf("failed to update webhook"))
	}
	webhook.Identifier = string(r.WebhookIdentifier)

	err = c.WebhooksRepository.Update(ctx, webhook)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to update webhook: %s for registry: %s with error: %v",
			webhookRequest.Identifier, regInfo.RegistryRef, err)
		return updateWebhookBadRequestErrorResponse(fmt.Errorf("failed to update webhook"))
	}

	updatedWebhook, err := c.WebhooksRepository.GetByRegistryAndIdentifier(
		ctx, regInfo.RegistryID, webhookRequest.Identifier,
	)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get updated webhook: %s with error: %v",
			webhookRequest.Identifier, err)
		return updateWebhookInternalErrorResponse(fmt.Errorf("failed to get updated webhook"))
	}

	webhookResponseEntity, err := c.RegistryMetadataHelper.MapToWebhookResponseEntity(ctx, updatedWebhook)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get updated webhook: %s with error: %v",
			webhookRequest.Identifier, err)
		return updateWebhookInternalErrorResponse(fmt.Errorf("failed to get updated webhook"))
	}

	return api.UpdateWebhook201JSONResponse{
		WebhookResponseJSONResponse: api.WebhookResponseJSONResponse{
			Data:   *webhookResponseEntity,
			Status: api.StatusSUCCESS,
		},
	}, nil
}

func updateWebhookInternalErrorResponse(err error) (api.UpdateWebhookResponseObject, error) {
	return api.UpdateWebhook500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

func updateWebhookBadRequestErrorResponse(err error) (api.UpdateWebhookResponseObject, error) {
	return api.UpdateWebhook400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

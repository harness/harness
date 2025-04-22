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

const webhookScopeRegistry = int64(0)

func (c *APIController) CreateWebhook(
	ctx context.Context,
	r api.CreateWebhookRequestObject,
) (api.CreateWebhookResponseObject, error) {
	webhookRequest := api.WebhookRequest(*r.Body)
	if webhookRequest.Identifier == internalWebhookIdentifier {
		return createWebhookBadRequestErrorResponse(
			fmt.Errorf("webhook identifier %s is reserved", internalWebhookIdentifier),
		)
	}
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return createWebhookBadRequestErrorResponse(err)
	}
	if regInfo.RegistryType != api.RegistryTypeVIRTUAL {
		log.Ctx(ctx).Error().Msgf("failed to store webhook: %s with error: %v", webhookRequest.Identifier, err)
		return createWebhookBadRequestErrorResponse(
			fmt.Errorf("not allowed to create webhook for %s registry", regInfo.RegistryType),
		)
	}
	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return createWebhookBadRequestErrorResponse(err)
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
		log.Ctx(ctx).Error().Msgf("permission check failed while creating webhook for registry: %s, error: %v",
			regInfo.RegistryIdentifier, err)
		return api.CreateWebhook403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	webhook, err := c.RegistryMetadataHelper.MapToWebhookCore(ctx, webhookRequest, regInfo)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to store webhook: %s with error: %v", webhookRequest.Identifier, err)
		return createWebhookBadRequestErrorResponse(fmt.Errorf("failed to store webhook %w", err))
	}

	webhook.Type = enum.WebhookTypeExternal
	webhook.CreatedBy = session.Principal.ID

	err = c.WebhooksRepository.Create(ctx, webhook)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to store webhook: %s with error: %v", webhookRequest.Identifier, err)
		if isDuplicateKeyError(err) {
			return createWebhookBadRequestErrorResponse(fmt.Errorf(
				"failed to store webhook, Webhook with identifier %s already exists", webhookRequest.Identifier,
			))
		}
		return createWebhookBadRequestErrorResponse(fmt.Errorf("failed to store webhook: %w", err))
	}

	createdWebhook, err := c.WebhooksRepository.GetByRegistryAndIdentifier(
		ctx, regInfo.RegistryID, webhookRequest.Identifier,
	)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to stored webhook: %s with error: %v",
			webhookRequest.Identifier, err)
		return createWebhookInternalErrorResponse(fmt.Errorf("failed to stored webhook: %w", err))
	}

	webhookResponseEntity, err := c.RegistryMetadataHelper.MapToWebhookResponseEntity(ctx, createdWebhook)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to stored webhook: %s with error: %v",
			webhookRequest.Identifier, err)
		return createWebhookInternalErrorResponse(fmt.Errorf("failed to stored webhook: %w", err))
	}
	return api.CreateWebhook201JSONResponse{
		WebhookResponseJSONResponse: api.WebhookResponseJSONResponse{
			Data:   *webhookResponseEntity,
			Status: api.StatusSUCCESS,
		},
	}, nil
}

func createWebhookBadRequestErrorResponse(err error) (api.CreateWebhookResponseObject, error) {
	return api.CreateWebhook400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse(
			*GetErrorResponse(http.StatusBadRequest, err.Error()),
		),
	}, nil
}

func createWebhookInternalErrorResponse(err error) (api.CreateWebhookResponseObject, error) {
	return api.CreateWebhook500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

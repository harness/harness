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

func (c *APIController) GetWebhook(
	ctx context.Context,
	r api.GetWebhookRequestObject,
) (api.GetWebhookResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get registry details: %v", err)
		return getWebhookInternalErrorResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to find space: %v", err)
		return getWebhookInternalErrorResponse(err)
	}

	session, _ := request.AuthSessionFrom(ctx)
	permissionChecks := c.RegistryMetadataHelper.GetPermissionChecks(space, regInfo.RegistryIdentifier,
		enum.PermissionRegistryView)
	if err = apiauth.CheckRegistry(
		ctx,
		c.Authorizer,
		session,
		permissionChecks...,
	); err != nil {
		log.Ctx(ctx).Error().Msgf("permission check failed while getting webhook for registry: %s, error: %v",
			regInfo.RegistryIdentifier, err)
		return api.GetWebhook403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	webhookIdentifier := string(r.WebhookIdentifier)
	webhook, err := c.WebhooksRepository.GetByRegistryAndIdentifier(ctx, regInfo.RegistryID, webhookIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get webhook: %s with error: %v", webhookIdentifier, err)
		return getWebhookInternalErrorResponse(fmt.Errorf("failed to get webhook"))
	}

	webhookResponseEntity, err := c.RegistryMetadataHelper.MapToWebhookResponseEntity(ctx, webhook)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to get webhook: %s with error: %v", webhookIdentifier, err)
		return getWebhookInternalErrorResponse(fmt.Errorf("failed to get webhook"))
	}
	return api.GetWebhook200JSONResponse{
		WebhookResponseJSONResponse: api.WebhookResponseJSONResponse{
			Data:   *webhookResponseEntity,
			Status: api.StatusSUCCESS,
		},
	}, nil
}

func getWebhookInternalErrorResponse(err error) (api.GetWebhookResponseObject, error) {
	return api.GetWebhook500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, err
}

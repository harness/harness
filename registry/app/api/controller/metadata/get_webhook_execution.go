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
	"strconv"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const getWebhookErrMsg = "failed to get webhook execution for registry: %s, webhook: %s with error: %v"

func (c *APIController) GetWebhookExecution(
	ctx context.Context,
	r api.GetWebhookExecutionRequestObject,
) (api.GetWebhookExecutionResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return getWebhooksExecutionsInternalErrorResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return getWebhooksExecutionsInternalErrorResponse(err)
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
		log.Ctx(ctx).Error().Msgf("permission check failed while get webhook execution for registry: %s, error: %v",
			regInfo.RegistryIdentifier, err)
		return api.GetWebhookExecution403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, err
	}

	webhookExecutionID, err := strconv.ParseInt(string(r.WebhookExecutionId), 10, 64)
	if err != nil || webhookExecutionID <= 0 {
		log.Ctx(ctx).Error().Msgf("invalid webhook execution identifier: %s, err: %v", string(r.WebhookExecutionId), err)
		return api.GetWebhookExecution400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse(
				*GetErrorResponse(http.StatusBadRequest, err.Error()),
			),
		}, err
	}

	w, err := c.WebhooksExecutionRepository.Find(ctx, webhookExecutionID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(getWebhookErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return getWebhooksExecutionsInternalErrorResponse(err)
	}
	webhookExecution, err := MapToWebhookExecutionResponseEntity(*w)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(getWebhookErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return getWebhooksExecutionsInternalErrorResponse(err)
	}
	return api.GetWebhookExecution200JSONResponse{
		WebhookExecutionResponseJSONResponse: api.WebhookExecutionResponseJSONResponse{
			Data:   *webhookExecution,
			Status: api.StatusSUCCESS,
		},
	}, nil
}

func getWebhooksExecutionsInternalErrorResponse(err error) (api.GetWebhookExecution500JSONResponse, error) {
	return api.GetWebhookExecution500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, err
}

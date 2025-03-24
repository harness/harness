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
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const listWebhooksErrMsg = "failed to list webhooks executions for registry: %s, webhook: %s with error: %v"

func (c *APIController) ListWebhookExecutions(
	ctx context.Context,
	r api.ListWebhookExecutionsRequestObject,
) (api.ListWebhookExecutionsResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return listWebhooksExecutionsInternalErrorResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return listWebhooksExecutionsInternalErrorResponse(err)
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
		log.Ctx(ctx).Error().Msgf("permission check failed while listing webhooks for registry: %s, error: %v",
			regInfo.RegistryIdentifier, err)
		return api.ListWebhookExecutions403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	size := GetOffset(r.Params.Size, r.Params.Page)
	limit := GetPageLimit(r.Params.Size)
	pageNumber := GetPageNumber(r.Params.Page)
	reg, err := c.RegistryRepository.GetByParentIDAndName(ctx, space.ID, regInfo.RegistryIdentifier)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(listWebhooksErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return listWebhooksExecutionsInternalErrorResponse(fmt.Errorf("failed to find registry: %w", err))
	}
	webhook, err := c.WebhooksRepository.GetByRegistryAndIdentifier(ctx, reg.ID, string(r.WebhookIdentifier))
	if err != nil {
		log.Ctx(ctx).Error().Msgf(listWebhooksErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return listWebhooksExecutionsInternalErrorResponse(
			fmt.Errorf("failed to find webhook [%s] : %w", r.WebhookIdentifier, err),
		)
	}
	we, err := c.WebhooksExecutionRepository.ListForWebhook(ctx, webhook.ID, limit, int(pageNumber), size)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(listWebhooksErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return listWebhooksExecutionsInternalErrorResponse(fmt.Errorf("failed to list webhook executions: %w", err))
	}
	webhookExecutions, err := mapToAPIListWebhooksExecutions(we)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(listWebhooksErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return listWebhooksExecutionsInternalErrorResponse(err)
	}
	count, err := c.WebhooksExecutionRepository.CountForWebhook(ctx, webhook.ID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf(listWebhooksErrMsg, regInfo.RegistryRef, r.WebhookIdentifier, err)
		return listWebhooksExecutionsInternalErrorResponse(fmt.Errorf("failed to get webhook executions count: %w", err))
	}
	pageCount := GetPageCount(count, limit)
	currentPageSize := len(webhookExecutions)
	return api.ListWebhookExecutions200JSONResponse{
		ListWebhooksExecutionResponseJSONResponse: api.ListWebhooksExecutionResponseJSONResponse{
			Data: api.ListWebhooksExecutions{
				Executions: webhookExecutions,
				ItemCount:  &count,
				PageCount:  &pageCount,
				PageIndex:  &pageNumber,
				PageSize:   &currentPageSize,
			},
			Status: api.StatusSUCCESS,
		},
	}, nil
}

func listWebhooksExecutionsInternalErrorResponse(err error) (api.ListWebhookExecutionsResponseObject, error) {
	return api.ListWebhookExecutions500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

func mapToAPIListWebhooksExecutions(executions []*gitnesstypes.WebhookExecutionCore) ([]api.WebhookExecution, error) {
	webhooksExecutionEntities := make([]api.WebhookExecution, 0, len(executions))
	for _, e := range executions {
		webhookExecution, err := MapToWebhookExecutionResponseEntity(*e)
		if err != nil {
			return nil, err
		}
		webhooksExecutionEntities = append(webhooksExecutionEntities, *webhookExecution)
	}
	return webhooksExecutionEntities, nil
}

func MapToWebhookExecutionResponseEntity(
	execution gitnesstypes.WebhookExecutionCore,
) (*api.WebhookExecution, error) {
	webhookResponseEntity := api.WebhookExecution{
		Created:  &execution.Created,
		Duration: &execution.Duration,
		Id:       &execution.ID,
		Error:    &execution.Error,
		Request: &api.WebhookExecRequest{
			Body:    &execution.Request.Body,
			Headers: &execution.Request.Headers,
			Url:     &execution.Request.URL,
		},
		Response: &api.WebhookExecResponse{
			Body:       &execution.Response.Body,
			Headers:    &execution.Response.Headers,
			Status:     &execution.Response.Status,
			StatusCode: &execution.Response.StatusCode,
		},
		RetriggerOf:   execution.RetriggerOf,
		Retriggerable: &execution.Retriggerable,
		WebhookId:     &execution.WebhookID,
	}
	webhookExecResult := mapTpAPIExecutionResult(execution.Result)
	if webhookExecResult != "" {
		webhookResponseEntity.Result = &webhookExecResult
	}
	triggerType := mapTpAPITriggerType(execution.TriggerType)
	if triggerType != "" {
		webhookResponseEntity.TriggerType = &triggerType
	}
	return &webhookResponseEntity, nil
}

func mapTpAPIExecutionResult(result enum.WebhookExecutionResult) api.WebhookExecResult {
	switch result {
	case enum.WebhookExecutionResultSuccess:
		return api.WebhookExecResultSUCCESS
	case enum.WebhookExecutionResultFatalError:
		return api.WebhookExecResultFATALERROR
	case enum.WebhookExecutionResultRetriableError:
		return api.WebhookExecResultRETRIABLEERROR
	}
	return ""
}

//nolint:exhaustive
func mapTpAPITriggerType(trigger enum.WebhookTrigger) api.Trigger {
	//nolint:exhaustive
	switch trigger {
	case enum.WebhookTriggerArtifactCreated:
		return api.TriggerARTIFACTCREATION
	case enum.WebhookTriggerArtifactDeleted:
		return api.TriggerARTIFACTDELETION
	}
	return ""
}

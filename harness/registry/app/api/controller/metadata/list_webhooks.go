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

func (c *APIController) ListWebhooks(
	ctx context.Context,
	r api.ListWebhooksRequestObject,
) (api.ListWebhooksResponseObject, error) {
	regInfo, err := c.RegistryMetadataHelper.GetRegistryRequestBaseInfo(ctx, "", string(r.RegistryRef))
	if err != nil {
		return listWebhookInternalErrorResponse(err)
	}

	space, err := c.SpaceFinder.FindByRef(ctx, regInfo.ParentRef)
	if err != nil {
		return listWebhookInternalErrorResponse(err)
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
		log.Ctx(ctx).Error().Msgf("permission check failed while listing webhook for registry: %s, error: %v",
			regInfo.RegistryIdentifier, err)
		return api.ListWebhooks403JSONResponse{
			UnauthorizedJSONResponse: api.UnauthorizedJSONResponse(
				*GetErrorResponse(http.StatusForbidden, err.Error()),
			),
		}, nil
	}

	offset := GetOffset(r.Params.Size, r.Params.Page)
	limit := GetPageLimit(r.Params.Size)
	pageNumber := GetPageNumber(r.Params.Page)

	searchTerm := ""
	if r.Params.SearchTerm != nil {
		searchTerm = string(*r.Params.SearchTerm)
	}
	sortByField := ""
	sortByOrder := ""
	if r.Params.SortOrder != nil {
		sortByOrder = string(*r.Params.SortOrder)
	}
	if r.Params.SortField != nil {
		sortByField = string(*r.Params.SortField)
	}

	webhooks, err := c.WebhooksRepository.ListByRegistry(
		ctx,
		sortByField,
		sortByOrder,
		limit,
		offset,
		searchTerm,
		regInfo.RegistryID,
	)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to list webhooks for registry: %s with error: %v", regInfo.RegistryRef, err)
		return listWebhookInternalErrorResponse(fmt.Errorf("failed list to webhooks"))
	}

	count, err := c.WebhooksRepository.CountAllByRegistry(ctx, regInfo.RegistryID, searchTerm)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to list webhooks for registry: %s with error: %v", regInfo.RegistryRef, err)
		return listWebhookInternalErrorResponse(fmt.Errorf("failed list to webhooks"))
	}
	webhooksResponse, err := c.mapToListWebhookResponseEntity(ctx, webhooks)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to list webhooks for registry: %s with error: %v", regInfo.RegistryRef, err)
		return listWebhookInternalErrorResponse(fmt.Errorf("failed to  list webhooks"))
	}
	pageCount := GetPageCount(count, limit)

	return api.ListWebhooks200JSONResponse{
		ListWebhooksResponseJSONResponse: api.ListWebhooksResponseJSONResponse{
			Data: api.ListWebhooks{
				PageIndex: &pageNumber,
				PageCount: &pageCount,
				PageSize:  &limit,
				ItemCount: &count,
				Webhooks:  webhooksResponse,
			},
			Status: api.StatusSUCCESS,
		},
	}, nil
}

func listWebhookInternalErrorResponse(err error) (api.ListWebhooksResponseObject, error) {
	return api.ListWebhooks500JSONResponse{
		InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse(
			*GetErrorResponse(http.StatusInternalServerError, err.Error()),
		),
	}, nil
}

func (c *APIController) mapToListWebhookResponseEntity(
	ctx context.Context,
	webhooks []*gitnesstypes.WebhookCore,
) ([]api.Webhook, error) {
	webhooksEntities := make([]api.Webhook, 0, len(webhooks))
	for _, d := range webhooks {
		webhook, err := c.RegistryMetadataHelper.MapToWebhookResponseEntity(ctx, d)
		if err != nil {
			return nil, err
		}
		webhooksEntities = append(webhooksEntities, *webhook)
	}
	return webhooksEntities, nil
}

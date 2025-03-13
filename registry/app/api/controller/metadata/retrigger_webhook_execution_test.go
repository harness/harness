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
	"testing"
	"time"

	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//nolint:lll
func TestReTriggerWebhookExecution_Success(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	mockMockWebhookService := new(MockWebhookService)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
		WebhookService:              mockMockWebhookService,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	latestExecutionResult := enum.WebhookExecutionResultSuccess
	mockMockWebhookService.On("ReTriggerWebhookExecution", ctx, int64(1)).Return(&gitnesswebhook.TriggerResult{
		Execution: &gitnesstypes.WebhookExecutionCore{
			ID:            1,
			Created:       time.Now().Unix(),
			Duration:      100,
			Error:         "none",
			Request:       gitnesstypes.WebhookExecutionRequest{Body: "{}", Headers: "headers", URL: "http://example.com"},
			Response:      gitnesstypes.WebhookExecutionResponse{Body: "{}", Headers: "headers", Status: "200 OK", StatusCode: 200},
			RetriggerOf:   nil,
			Retriggerable: true,
			WebhookID:     4,
			Result:        enum.WebhookExecutionResultSuccess,
			TriggerType:   enum.WebhookTriggerArtifactCreated},
		Webhook: &gitnesstypes.WebhookCore{
			Identifier:            "webhook",
			DisplayName:           "webhook",
			URL:                   "http://example.com",
			Enabled:               true,
			Insecure:              false,
			Triggers:              []enum.WebhookTrigger{enum.WebhookTriggerArtifactCreated},
			Created:               time.Now().Unix(),
			Updated:               time.Now().Unix(),
			Description:           "Test webhook",
			SecretSpaceID:         1,
			ExtraHeaders:          []gitnesstypes.ExtraHeader{{Key: "key", Value: "value"}},
			LatestExecutionResult: &latestExecutionResult,
		},
		TriggerType: enum.WebhookTriggerArtifactCreated,
	}, nil)

	r := api.ReTriggerWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "1",
	}

	response, err := controller.ReTriggerWebhookExecution(ctx, r)
	reTriggerWebhookExecution200JSONResponse, ok := response.(api.ReTriggerWebhookExecution200JSONResponse)
	if !ok {
		t.Fatalf("expected api.ReTriggerWebhookExecution200JSONResponse, got %T", response)
	}

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, api.StatusSUCCESS, reTriggerWebhookExecution200JSONResponse.Status)
	assert.Equal(t, int64(1), *reTriggerWebhookExecution200JSONResponse.Data.Id)
	assert.Equal(t, "none", *reTriggerWebhookExecution200JSONResponse.Data.Error)
	assert.Equal(t, "{}", *reTriggerWebhookExecution200JSONResponse.Data.Request.Body)
	assert.Equal(t, "headers", *reTriggerWebhookExecution200JSONResponse.Data.Request.Headers)
	assert.Equal(t, "http://example.com", *reTriggerWebhookExecution200JSONResponse.Data.Request.Url)
	assert.Equal(t, "{}", *reTriggerWebhookExecution200JSONResponse.Data.Response.Body)
	assert.Equal(t, "headers", *reTriggerWebhookExecution200JSONResponse.Data.Response.Headers)
	assert.Equal(t, "200 OK", *reTriggerWebhookExecution200JSONResponse.Data.Response.Status)
	assert.Equal(t, 200, *reTriggerWebhookExecution200JSONResponse.Data.Response.StatusCode)
	assert.Equal(t, api.WebhookExecResultSUCCESS, *reTriggerWebhookExecution200JSONResponse.Data.Result)
	assert.Equal(t, api.TriggerARTIFACTCREATION, *reTriggerWebhookExecution200JSONResponse.Data.TriggerType)
	assert.Equal(t, api.StatusSUCCESS, reTriggerWebhookExecution200JSONResponse.Status)

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockWebhooksExecutionRepository.AssertExpectations(t)
	mockMockWebhookService.AssertExpectations(t)
}

//nolint:lll
func TestReTriggerWebhookExecution_PermissionCheckFails(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	mockMockWebhookService := new(MockWebhookService)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
		WebhookService:              mockMockWebhookService,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(false, nil)

	r := api.ReTriggerWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "1",
	}

	response, err := controller.ReTriggerWebhookExecution(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ReTriggerWebhookExecution403JSONResponse{}, response)
	assert.Contains(t, err.Error(), "not authorized")
	assert.Equal(t, "not authorized", response.(api.ReTriggerWebhookExecution403JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}

//nolint:lll
func TestReTriggerWebhookExecution_InvalidExecutionIdentifier(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	mockMockWebhookService := new(MockWebhookService)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
		WebhookService:              mockMockWebhookService,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)

	r := api.ReTriggerWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "invalid",
	}

	response, err := controller.ReTriggerWebhookExecution(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ReTriggerWebhookExecution400JSONResponse{}, response)
	assert.Contains(t, err.Error(), "strconv.ParseInt: parsing \"invalid\": invalid syntax")
	assert.Equal(t, "invalid webhook execution identifier: invalid, err: strconv.ParseInt: parsing \"invalid\": invalid syntax",
		response.(api.ReTriggerWebhookExecution400JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}

//nolint:lll
func TestReTriggerWebhookExecution_ReTriggerExecutionError(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	mockMockWebhookService := new(MockWebhookService)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
		WebhookService:              mockMockWebhookService,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}
	space := &gitnesstypes.SpaceCore{ID: 2}
	var permissionChecks []gitnesstypes.PermissionCheck
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockMockWebhookService.On("ReTriggerWebhookExecution", ctx, int64(1)).Return(nil, fmt.Errorf("error"))

	r := api.ReTriggerWebhookExecutionRequestObject{
		RegistryRef:        "reg",
		WebhookIdentifier:  "webhook",
		WebhookExecutionId: "1",
	}

	response, err := controller.ReTriggerWebhookExecution(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ReTriggerWebhookExecution500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "failed to re-trigger execution: error")
	assert.Equal(t, "failed to re-trigger execution: error", response.(api.ReTriggerWebhookExecution500JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}

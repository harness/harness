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

	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//nolint:lll
func TestListWebhookExecutions_Success(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
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
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
	mockWebhooksRepository.On("GetByRegistryAndIdentifier", ctx, int64(3), "webhook").Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
	mockWebhooksExecutionRepository.On("ListForWebhook", ctx, int64(4), 10, 1, 10).Return([]*gitnesstypes.WebhookExecutionCore{
		{
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
			TriggerType:   enum.WebhookTriggerArtifactCreated,
		},
	}, nil)
	mockWebhooksExecutionRepository.On("CountForWebhook", ctx, int64(4)).Return(int64(1), nil)

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(1)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	listWebhookExecutions200JSONResponse, ok := response.(api.ListWebhookExecutions200JSONResponse)
	if !ok {
		t.Fatalf("expected api.ListWebhookExecutions200JSONResponse, got %T", response)
	}
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, api.StatusSUCCESS, listWebhookExecutions200JSONResponse.Status)
	assert.Len(t, listWebhookExecutions200JSONResponse.Data.Executions, 1)
	assert.Equal(t, int64(1), *listWebhookExecutions200JSONResponse.Data.Executions[0].Id)
	assert.Equal(t, "none", *listWebhookExecutions200JSONResponse.Data.Executions[0].Error)
	assert.Equal(t, "{}", *listWebhookExecutions200JSONResponse.Data.Executions[0].Request.Body)
	assert.Equal(t, "headers", *listWebhookExecutions200JSONResponse.Data.Executions[0].Request.Headers)
	assert.Equal(t, "http://example.com", *listWebhookExecutions200JSONResponse.Data.Executions[0].Request.Url)
	assert.Equal(t, "{}", *listWebhookExecutions200JSONResponse.Data.Executions[0].Response.Body)
	assert.Equal(t, "headers", *listWebhookExecutions200JSONResponse.Data.Executions[0].Response.Headers)
	assert.Equal(t, "200 OK", *listWebhookExecutions200JSONResponse.Data.Executions[0].Response.Status)
	assert.Equal(t, 200, *listWebhookExecutions200JSONResponse.Data.Executions[0].Response.StatusCode)
	assert.Equal(t, api.WebhookExecResultSUCCESS, *listWebhookExecutions200JSONResponse.Data.Executions[0].Result)
	assert.Equal(t, api.TriggerARTIFACTCREATION, *listWebhookExecutions200JSONResponse.Data.Executions[0].TriggerType)
	assert.Equal(t, int64(1), *listWebhookExecutions200JSONResponse.Data.ItemCount)
	assert.Equal(t, 1, *listWebhookExecutions200JSONResponse.Data.PageSize)
	assert.Equal(t, int64(1), *listWebhookExecutions200JSONResponse.Data.PageIndex)
	assert.Equal(t, int64(1), *listWebhookExecutions200JSONResponse.Data.PageCount)
	assert.Equal(t, api.StatusSUCCESS, listWebhookExecutions200JSONResponse.Status)

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockRegistryRepository.AssertExpectations(t)
	mockWebhooksRepository.AssertExpectations(t)
	mockWebhooksExecutionRepository.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_GetRegistryRequestBaseInfoError(t *testing.T) {
	ctx := context.Background()
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		RegistryMetadataHelper: mockRegistryMetadataHelper,
	}

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(nil, fmt.Errorf("error"))

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(11)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "error")

	mockRegistryMetadataHelper.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_FindByRefError(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:            mockSpaceFinder,
		RegistryMetadataHelper: mockRegistryMetadataHelper,
	}

	regInfo := &RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "reg",
		ParentRef:          "root/parent",
	}

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", ctx, "", "reg").Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", ctx, "root/parent").Return(nil, fmt.Errorf("error"))

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(11)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "error")

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_CheckPermissionsFails(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:            mockSpaceFinder,
		Authorizer:             mockAuthorizer,
		RegistryMetadataHelper: mockRegistryMetadataHelper,
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
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(false, nil)

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(11)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions403JSONResponse{}, response)
	assert.Contains(t, err.Error(), "not authorized")
	assert.Equal(t, "not authorized", response.(api.ListWebhookExecutions403JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_FailedToDetRegistry(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
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
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(nil, fmt.Errorf("error"))

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(1)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "failed to find registry: error")
	assert.Equal(t, "failed to find registry: error", response.(api.ListWebhookExecutions500JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockRegistryRepository.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_FailedToGetWebhook(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
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
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
	mockWebhooksRepository.On("GetByRegistryAndIdentifier", ctx, int64(3), "webhook").Return(nil, fmt.Errorf("error"))

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(1)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "failed to find webhook [webhook] : error")
	assert.Equal(t, "failed to find webhook [webhook] : error", response.(api.ListWebhookExecutions500JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockRegistryRepository.AssertExpectations(t)
	mockWebhooksRepository.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_FailedToGetWebhookExecutions(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
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
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
	mockWebhooksRepository.On("GetByRegistryAndIdentifier", ctx, int64(3), "webhook").Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
	mockWebhooksExecutionRepository.On("ListForWebhook", ctx, int64(4), 10, 1, 10).Return(nil, fmt.Errorf("error"))

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(1)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "failed to list webhook executions: error")
	assert.Equal(t, "failed to list webhook executions: error", response.(api.ListWebhookExecutions500JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockRegistryRepository.AssertExpectations(t)
	mockWebhooksRepository.AssertExpectations(t)
	mockWebhooksExecutionRepository.AssertExpectations(t)
}

//nolint:lll
func TestListWebhookExecutions_FailedToGetWebhooksCount(t *testing.T) {
	ctx := context.Background()
	mockSpaceFinder := new(MockSpaceFinder)
	mockRegistryRepository := new(MockRegistryRepository)
	mockWebhooksRepository := new(MockWebhooksRepository)
	mockWebhooksExecutionRepository := new(MockWebhooksExecutionRepository)
	mockAuthorizer := new(MockAuthorizer)
	mockRegistryMetadataHelper := new(MockRegistryMetadataHelper)
	controller := &APIController{
		SpaceFinder:                 mockSpaceFinder,
		RegistryRepository:          mockRegistryRepository,
		WebhooksRepository:          mockWebhooksRepository,
		WebhooksExecutionRepository: mockWebhooksExecutionRepository,
		Authorizer:                  mockAuthorizer,
		RegistryMetadataHelper:      mockRegistryMetadataHelper,
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
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryView).Return(permissionChecks)
	mockAuthorizer.On("CheckAll", ctx, mock.Anything, permissionChecks).Return(true, nil)
	mockRegistryRepository.On("GetByParentIDAndName", ctx, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
	mockWebhooksRepository.On("GetByRegistryAndIdentifier", ctx, int64(3), "webhook").Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
	mockWebhooksExecutionRepository.On("ListForWebhook", ctx, int64(4), 10, 1, 10).Return([]*gitnesstypes.WebhookExecutionCore{
		{
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
			TriggerType:   enum.WebhookTriggerArtifactCreated,
		},
	}, nil)
	mockWebhooksExecutionRepository.On("CountForWebhook", ctx, int64(4)).Return(nil, fmt.Errorf("error"))

	pageSize := api.PageSize(10)
	pageNumber := api.PageNumber(1)
	r := api.ListWebhookExecutionsRequestObject{
		RegistryRef:       "reg",
		WebhookIdentifier: "webhook",
		Params: api.ListWebhookExecutionsParams{
			Size: &pageSize,
			Page: &pageNumber,
		},
	}

	response, err := controller.ListWebhookExecutions(ctx, r)
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, api.ListWebhookExecutions500JSONResponse{}, response)
	assert.Contains(t, err.Error(), "failed to get webhook executions count: error")
	assert.Equal(t, "failed to get webhook executions count: error", response.(api.ListWebhookExecutions500JSONResponse).Message) //nolint:errcheck

	mockRegistryMetadataHelper.AssertExpectations(t)
	mockSpaceFinder.AssertExpectations(t)
	mockAuthorizer.AssertExpectations(t)
	mockRegistryRepository.AssertExpectations(t)
	mockWebhooksRepository.AssertExpectations(t)
	mockWebhooksExecutionRepository.AssertExpectations(t)
}

//nolint:lll
func TestMapToWebhookExecutionResponseEntity(t *testing.T) {
	execution := gitnesstypes.WebhookExecutionCore{
		Created:       1234567890,
		Duration:      100,
		ID:            1,
		Error:         "none",
		Request:       gitnesstypes.WebhookExecutionRequest{Body: "{}", Headers: "headers", URL: "http://example.com"},
		Response:      gitnesstypes.WebhookExecutionResponse{Body: "{}", Headers: "headers", Status: "200 OK", StatusCode: 200},
		RetriggerOf:   nil,
		Retriggerable: true,
		WebhookID:     4,
		Result:        enum.WebhookExecutionResultSuccess,
		TriggerType:   enum.WebhookTriggerArtifactCreated,
	}

	result := api.WebhookExecResultSUCCESS
	triggerType := api.TriggerARTIFACTCREATION
	expected := &api.WebhookExecution{
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
		Result:        &result,
		TriggerType:   &triggerType,
	}

	webhookExecution, err := MapToWebhookExecutionResponseEntity(execution)
	assert.NoError(t, err)
	assert.Equal(t, expected, webhookExecution)
}

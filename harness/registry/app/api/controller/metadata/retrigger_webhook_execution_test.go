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

//nolint:lll
package metadata_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	webhookExecution = &coretypes.WebhookExecutionCore{
		ID:            1,
		Created:       time.Now().Unix(),
		WebhookID:     1,
		Result:        enum.WebhookExecutionResultSuccess,
		Duration:      100,
		Error:         "",
		Retriggerable: false,
		Request: coretypes.WebhookExecutionRequest{
			URL:     "http://example.com",
			Headers: "{}",
			Body:    "{}",
		},
		Response: coretypes.WebhookExecutionResponse{
			StatusCode: 200,
			Status:     "OK",
			Headers:    "{}",
			Body:       "{}",
		},
		TriggerType: enum.WebhookTriggerArtifactCreated,
	}

	webhookExecutionEntity = &api.WebhookExecution{
		Id:            &webhookExecution.ID,
		Created:       &webhookExecution.Created,
		Duration:      &webhookExecution.Duration,
		RetriggerOf:   webhookExecution.RetriggerOf,
		Retriggerable: &webhookExecution.Retriggerable,
		Error:         &webhookExecution.Error,
		WebhookId:     &webhookExecution.WebhookID,
		Request: &api.WebhookExecRequest{
			Body:    &webhookExecution.Request.Body,
			Headers: &webhookExecution.Request.Headers,
			Url:     &webhookExecution.Request.URL,
		},
		Response: &api.WebhookExecResponse{
			Status:     &webhookExecution.Response.Status,
			StatusCode: &webhookExecution.Response.StatusCode,
			Body:       &webhookExecution.Response.Body,
			Headers:    &webhookExecution.Response.Headers,
		},
		Result:      &[]api.WebhookExecResult{api.WebhookExecResultSUCCESS}[0],
		TriggerType: &[]api.Trigger{api.TriggerARTIFACTCREATION}[0],
	}
)

func TestReTriggerWebhookExecution(t *testing.T) {
	// Create mocks that will be used across all tests
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockRegistryRepository := new(mocks.RegistryRepository)
	mockWebhooksRepository := new(mocks.WebhooksRepository)
	mockWebhooksExecutionRepository := new(mocks.WebhooksExecutionRepository)
	mockAuthorizer := new(mocks.Authorizer)
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockWebhookService := new(mocks.WebhookService)

	tests := []struct {
		name          string
		setupMocks    func(*metadata.APIController)
		request       api.ReTriggerWebhookExecutionRequestObject
		expectedResp  api.ReTriggerWebhookExecutionResponseObject
		expectedError error
	}{
		{
			name: "success_case",
			setupMocks: func(_ *metadata.APIController) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &coretypes.SpaceCore{ID: 2}
				var permissionChecks []coretypes.PermissionCheck

				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

				latestExecutionResult := enum.WebhookExecutionResultSuccess
				mockWebhookService.On("ReTriggerWebhookExecution", mock.Anything, int64(1)).Return(&gitnesswebhook.TriggerResult{
					Execution: webhookExecution,
					Webhook: &coretypes.WebhookCore{
						Identifier:            "webhook",
						DisplayName:           "webhook",
						URL:                   "http://example.com",
						Enabled:               true,
						Insecure:              false,
						Triggers:              []enum.WebhookTrigger{enum.WebhookTriggerArtifactCreated},
						Created:               webhookExecution.Created,
						Updated:               webhookExecution.Created,
						Description:           "Test webhook",
						SecretSpaceID:         1,
						ExtraHeaders:          []coretypes.ExtraHeader{{Key: "key", Value: "value"}},
						LatestExecutionResult: &latestExecutionResult,
					},
					TriggerType: enum.WebhookTriggerArtifactCreated,
				}, nil)
			},
			request: api.ReTriggerWebhookExecutionRequestObject{
				WebhookIdentifier:  "webhook",
				RegistryRef:        "reg",
				WebhookExecutionId: "1",
			},
			expectedResp: api.ReTriggerWebhookExecution200JSONResponse{
				WebhookExecutionResponseJSONResponse: api.WebhookExecutionResponseJSONResponse{
					Data:   *webhookExecutionEntity,
					Status: api.StatusSUCCESS,
				},
			},
		},
		{
			name: "permission_check_fails",
			setupMocks: func(_ *metadata.APIController) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &coretypes.SpaceCore{ID: 2}
				var permissionChecks []coretypes.PermissionCheck

				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
			},
			request: api.ReTriggerWebhookExecutionRequestObject{
				WebhookIdentifier:  "webhook",
				RegistryRef:        "reg",
				WebhookExecutionId: "1",
			},
			expectedResp: api.ReTriggerWebhookExecution403JSONResponse{
				UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{
					Code:    "403",
					Message: "not authorized",
				},
			},
		},
		{
			name: "invalid_execution_identifier",
			setupMocks: func(_ *metadata.APIController) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &coretypes.SpaceCore{ID: 2}
				var permissionChecks []coretypes.PermissionCheck

				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
			},
			request: api.ReTriggerWebhookExecutionRequestObject{
				WebhookIdentifier:  "webhook",
				RegistryRef:        "reg",
				WebhookExecutionId: "invalid",
			},
			expectedResp: api.ReTriggerWebhookExecution400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "invalid webhook execution identifier: invalid, err: strconv.ParseInt: parsing \"invalid\": invalid syntax",
				},
			},
		},
		{
			name: "retrigger_execution_error",
			setupMocks: func(_ *metadata.APIController) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &coretypes.SpaceCore{ID: 2}
				var permissionChecks []coretypes.PermissionCheck

				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, regInfo.RegistryIdentifier, enum.PermissionRegistryEdit).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockWebhookService.On("ReTriggerWebhookExecution", mock.Anything, int64(1)).Return(nil, fmt.Errorf("error"))
			},
			request: api.ReTriggerWebhookExecutionRequestObject{
				WebhookIdentifier:  "webhook",
				RegistryRef:        "reg",
				WebhookExecutionId: "1",
			},
			expectedResp: api.ReTriggerWebhookExecution500JSONResponse{
				InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{
					Code:    "500",
					Message: "failed to re-trigger execution: error",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear mock expectations
			mockSpaceFinder.ExpectedCalls = nil
			mockRegistryRepository.ExpectedCalls = nil
			mockWebhooksRepository.ExpectedCalls = nil
			mockWebhooksExecutionRepository.ExpectedCalls = nil
			mockAuthorizer.ExpectedCalls = nil
			mockRegistryMetadataHelper.ExpectedCalls = nil
			mockWebhookService.ExpectedCalls = nil

			controller := &metadata.APIController{
				SpaceFinder:                 mockSpaceFinder,
				RegistryRepository:          mockRegistryRepository,
				WebhooksRepository:          mockWebhooksRepository,
				WebhooksExecutionRepository: mockWebhooksExecutionRepository,
				Authorizer:                  mockAuthorizer,
				RegistryMetadataHelper:      mockRegistryMetadataHelper,
				WebhookService:              mockWebhookService,
			}

			// Setup mocks
			tt.setupMocks(controller)

			resp, err := controller.ReTriggerWebhookExecution(context.Background(), tt.request)
			assert.Equal(t, tt.expectedError, err)
			assert.NotNil(t, resp, "response should not be nil")

			switch tt.name {
			case "success_case":
				successResp, ok := resp.(api.ReTriggerWebhookExecution200JSONResponse)
				assert.True(t, ok, "expected 200 response")
				assert.Equal(t, api.StatusSUCCESS, successResp.Status)
				assert.NotNil(t, successResp.Data, "Data should not be nil")
				assert.NotNil(t, successResp.Data.Id, "Id should not be nil")
				assert.NotNil(t, successResp.Data.Error, "Error should not be nil")
				assert.NotNil(t, successResp.Data.Request, "Request should not be nil")
				assert.NotNil(t, successResp.Data.Response, "Response should not be nil")
				assert.NotNil(t, successResp.Data.Result, "Result should not be nil")
				assert.NotNil(t, successResp.Data.TriggerType, "TriggerType should not be nil")

				if assert.NotNil(t, successResp.Data.Request) {
					assert.Equal(t, "{}", *successResp.Data.Request.Body)
					assert.Equal(t, "{}", *successResp.Data.Request.Headers)
					assert.Equal(t, "http://example.com", *successResp.Data.Request.Url)
				}

				if assert.NotNil(t, successResp.Data.Response) {
					assert.Equal(t, "{}", *successResp.Data.Response.Body)
					assert.Equal(t, "{}", *successResp.Data.Response.Headers)
					assert.Equal(t, "OK", *successResp.Data.Response.Status)
					assert.Equal(t, 200, *successResp.Data.Response.StatusCode)
				}

				assert.Equal(t, int64(1), *successResp.Data.Id)
				assert.Equal(t, "", *successResp.Data.Error)
				assert.Equal(t, api.WebhookExecResultSUCCESS, *successResp.Data.Result)
				assert.Equal(t, api.TriggerARTIFACTCREATION, *successResp.Data.TriggerType)

			case "permission_check_fails":
				assert.IsType(t, api.ReTriggerWebhookExecution403JSONResponse{}, resp, "expected 403 response")
				errorResp, _ := resp.(api.ReTriggerWebhookExecution403JSONResponse) //nolint:errcheck
				assert.Equal(t, "403", errorResp.Code)
				assert.Equal(t, "forbidden", errorResp.Message)

			case "invalid_execution_identifier":
				assert.IsType(t, api.ReTriggerWebhookExecution400JSONResponse{}, resp, "expected 400 response")
				errorResp, _ := resp.(api.ReTriggerWebhookExecution400JSONResponse) //nolint:errcheck
				assert.Equal(t, "400", errorResp.Code)
				assert.Equal(t, "invalid webhook execution identifier: invalid, err: strconv.ParseInt: parsing \"invalid\": invalid syntax", errorResp.Message)

			case "retrigger_execution_error":
				assert.IsType(t, api.ReTriggerWebhookExecution500JSONResponse{}, resp, "expected 500 response")
				errorResp, _ := resp.(api.ReTriggerWebhookExecution500JSONResponse) //nolint:errcheck
				assert.Equal(t, "500", errorResp.Code)
				assert.Equal(t, "failed to re-trigger execution: error", errorResp.Message)
			}

			// Verify all mock expectations
			mockSpaceFinder.AssertExpectations(t)
			mockRegistryRepository.AssertExpectations(t)
			mockWebhooksRepository.AssertExpectations(t)
			mockWebhooksExecutionRepository.AssertExpectations(t)
			mockAuthorizer.AssertExpectations(t)
			mockRegistryMetadataHelper.AssertExpectations(t)
			mockWebhookService.AssertExpectations(t)
		})
	}
}

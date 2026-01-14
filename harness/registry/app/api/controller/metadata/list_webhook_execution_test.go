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

//nolint:lll,revive // revive:disable:unused-parameter
package metadata

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/utils"
	gitnesstypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListWebhookExecutions(t *testing.T) {
	tests := []struct {
		name       string
		request    api.ListWebhookExecutionsRequestObject
		setupMocks func(*mocks.SpaceFinder, *mocks.RegistryRepository, *mocks.WebhooksRepository,
			*mocks.WebhooksExecutionRepository, *mocks.Authorizer, *mocks.RegistryMetadataHelper)
		validate    func(*testing.T, api.ListWebhookExecutionsResponseObject, error)
		verifyMocks func(*testing.T, *mocks.SpaceFinder, *mocks.RegistryRepository, *mocks.WebhooksRepository,
			*mocks.WebhooksExecutionRepository, *mocks.Authorizer, *mocks.RegistryMetadataHelper)
	}{
		{
			name: "success_case",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(1),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck
				// session := &auth.Session{}

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
				mockWebhooksRepo.On(
					"GetByRegistryAndIdentifier",
					mock.Anything,
					int64(3),
					"webhook",
				).Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
				mockWebhooksExecRepo.On(
					"ListForWebhook",
					mock.Anything,
					int64(4),
					10, 1, 10,
				).Return([]*gitnesstypes.WebhookExecutionCore{
					{
						ID:       1,
						Created:  time.Now().Unix(),
						Duration: 100,
						Error:    "none",
						Request:  gitnesstypes.WebhookExecutionRequest{Body: "{}", Headers: "headers", URL: "http://example.com"},
						Response: gitnesstypes.WebhookExecutionResponse{
							Body:       "{}",
							Headers:    "headers",
							Status:     "200 OK",
							StatusCode: 200,
						},
						RetriggerOf:   nil,
						Retriggerable: true,
						WebhookID:     4,
						Result:        enum.WebhookExecutionResultSuccess,
						TriggerType:   enum.WebhookTriggerArtifactCreated,
					},
				}, nil)
				mockWebhooksExecRepo.On("CountForWebhook", mock.Anything, int64(4)).Return(int64(1), nil)
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				resp, ok := response.(api.ListWebhookExecutions200JSONResponse)
				assert.True(t, ok, "expected 200 response")
				assert.Equal(t, api.StatusSUCCESS, resp.Status)
				assert.Len(t, resp.Data.Executions, 1)

				exec := resp.Data.Executions[0]
				assert.Equal(t, int64(1), *exec.Id)
				assert.Equal(t, "none", *exec.Error)
				assert.Equal(t, "{}", *exec.Request.Body)
				assert.Equal(t, "headers", *exec.Request.Headers)
				assert.Equal(t, "http://example.com", *exec.Request.Url)
				assert.Equal(t, "{}", *exec.Response.Body)
				assert.Equal(t, "headers", *exec.Response.Headers)
				assert.Equal(t, "200 OK", *exec.Response.Status)
				assert.Equal(t, 200, *exec.Response.StatusCode)
				assert.Equal(t, api.WebhookExecResultSUCCESS, *exec.Result)
				assert.Equal(t, api.TriggerARTIFACTCREATION, *exec.TriggerType)

				assert.Equal(t, int64(1), *resp.Data.ItemCount)
				assert.Equal(t, 1, *resp.Data.PageSize)
				assert.Equal(t, int64(1), *resp.Data.PageIndex)
				assert.Equal(t, int64(1), *resp.Data.PageCount)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				mockRegistryRepo.AssertExpectations(t)
				mockWebhooksRepo.AssertExpectations(t)
				mockWebhooksExecRepo.AssertExpectations(t)
			},
		},
		{
			name: "get_registry_request_base_info_error",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(11),
				},
			},
			setupMocks: func(_ *mocks.SpaceFinder, _ *mocks.RegistryRepository,
				_ *mocks.WebhooksRepository, _ *mocks.WebhooksExecutionRepository,
				_ *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(nil, fmt.Errorf("error"))
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "error", resp.Message)
			},
			verifyMocks: func(t *testing.T, _ *mocks.SpaceFinder, _ *mocks.RegistryRepository,
				_ *mocks.WebhooksRepository, _ *mocks.WebhooksExecutionRepository,
				_ *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				// Other mocks should not have any expectations
			},
		},
		{
			name: "find_by_ref_error",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(11),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, _ *mocks.RegistryRepository,
				_ *mocks.WebhooksRepository, _ *mocks.WebhooksExecutionRepository,
				_ *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(nil, fmt.Errorf("error"))
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "error", resp.Message)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				// Other mocks should not have any expectations
				mockAuthorizer.AssertNotCalled(t, mock.Anything)
				mockRegistryRepo.AssertNotCalled(t, mock.Anything)
				mockWebhooksRepo.AssertNotCalled(t, mock.Anything)
				mockWebhooksExecRepo.AssertNotCalled(t, mock.Anything)
			},
		},
		{
			name: "check_permissions_fails",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(11),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions403JSONResponse)
				assert.True(t, ok, "expected 403 response")
				assert.Equal(t, "forbidden", resp.Message)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				// Other mocks should not have any expectations
				mockRegistryRepo.AssertNotCalled(t, mock.Anything)
				mockWebhooksRepo.AssertNotCalled(t, mock.Anything)
				mockWebhooksExecRepo.AssertNotCalled(t, mock.Anything)
			},
		},
		{
			name: "failed_to_get_registry",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(1),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(nil, fmt.Errorf("error"))
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "failed to find registry: error", resp.Message)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				mockRegistryRepo.AssertExpectations(t)
				// Other mocks should not have any expectations
				mockWebhooksRepo.AssertNotCalled(t, mock.Anything)
				mockWebhooksExecRepo.AssertNotCalled(t, mock.Anything)
			},
		},
		{
			name: "failed_to_get_webhook",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(1),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
				mockWebhooksRepo.On("GetByRegistryAndIdentifier", mock.Anything, int64(3), "webhook").Return(nil, fmt.Errorf("error"))
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "failed to find webhook [webhook] : error", resp.Message)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				mockRegistryRepo.AssertExpectations(t)
				mockWebhooksRepo.AssertExpectations(t)
				// Other mocks should not have any expectations
				mockWebhooksExecRepo.AssertNotCalled(t, mock.Anything)
			},
		},
		{
			name: "failed_to_list_webhook_executions",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(1),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
				mockWebhooksRepo.On(
					"GetByRegistryAndIdentifier",
					mock.Anything,
					int64(3),
					"webhook",
				).Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
				mockWebhooksExecRepo.On("ListForWebhook", mock.Anything, int64(4), 10, 1, 10).Return(nil, fmt.Errorf("error"))
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "failed to list webhook executions: error", resp.Message)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				mockRegistryRepo.AssertExpectations(t)
				mockWebhooksRepo.AssertExpectations(t)
				mockWebhooksExecRepo.AssertExpectations(t)
			},
		},
		{
			name: "failed_to_get_webhook_executions_count",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(1),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
				mockWebhooksRepo.On(
					"GetByRegistryAndIdentifier",
					mock.Anything,
					int64(3),
					"webhook",
				).Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
				mockWebhooksExecRepo.On(
					"ListForWebhook",
					mock.Anything,
					int64(4),
					10, 1, 10,
				).Return([]*gitnesstypes.WebhookExecutionCore{
					{
						ID:       1,
						Created:  time.Now().Unix(),
						Duration: 100,
						Error:    "none",
						Request:  gitnesstypes.WebhookExecutionRequest{Body: "{}", Headers: "headers", URL: "http://example.com"},
						Response: gitnesstypes.WebhookExecutionResponse{
							Body:       "{}",
							Headers:    "headers",
							Status:     "200 OK",
							StatusCode: 200,
						},
						RetriggerOf:   nil,
						Retriggerable: true,
						WebhookID:     4,
						Result:        enum.WebhookExecutionResultSuccess,
						TriggerType:   enum.WebhookTriggerArtifactCreated,
					},
				}, nil)
				mockWebhooksExecRepo.On("CountForWebhook", mock.Anything, int64(4)).Return(int64(0), fmt.Errorf("error"))
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				resp, ok := response.(api.ListWebhookExecutions500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "failed to get webhook executions count: error", resp.Message)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				mockRegistryRepo.AssertExpectations(t)
				mockWebhooksRepo.AssertExpectations(t)
				mockWebhooksExecRepo.AssertExpectations(t)
			},
		},
		{
			name: "success_case",
			request: api.ListWebhookExecutionsRequestObject{
				RegistryRef:       "reg",
				WebhookIdentifier: "webhook",
				Params: api.ListWebhookExecutionsParams{
					Size: utils.PageSizePtr(10),
					Page: utils.PageNumberPtr(1),
				},
			},
			setupMocks: func(mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentRef:          "root/parent",
				}
				space := &gitnesstypes.SpaceCore{ID: 2}
				var permissionChecks []gitnesstypes.PermissionCheck

				mockMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockMetadataHelper.On(
					"GetPermissionChecks",
					space,
					regInfo.RegistryIdentifier,
					enum.PermissionRegistryView,
				).Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "reg").Return(&types.Registry{ID: 3}, nil)
				mockWebhooksRepo.On(
					"GetByRegistryAndIdentifier",
					mock.Anything,
					int64(3),
					"webhook",
				).Return(&gitnesstypes.WebhookCore{ID: 4}, nil)
				mockWebhooksExecRepo.On(
					"ListForWebhook",
					mock.Anything,
					int64(4),
					10, 1, 10,
				).Return([]*gitnesstypes.WebhookExecutionCore{
					{
						ID:       1,
						Created:  time.Now().Unix(),
						Duration: 100,
						Error:    "none",
						Request:  gitnesstypes.WebhookExecutionRequest{Body: "{}", Headers: "headers", URL: "http://example.com"},
						Response: gitnesstypes.WebhookExecutionResponse{
							Body:       "{}",
							Headers:    "headers",
							Status:     "200 OK",
							StatusCode: 200,
						},
						RetriggerOf:   nil,
						Retriggerable: true,
						WebhookID:     4,
						Result:        enum.WebhookExecutionResultSuccess,
						TriggerType:   enum.WebhookTriggerArtifactCreated,
					},
				}, nil)
				mockWebhooksExecRepo.On("CountForWebhook", mock.Anything, int64(4)).Return(int64(1), nil)
			},
			validate: func(t *testing.T, response api.ListWebhookExecutionsResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				resp, ok := response.(api.ListWebhookExecutions200JSONResponse)
				assert.True(t, ok, "expected 200 response")
				assert.Equal(t, api.StatusSUCCESS, resp.Status)
				assert.Len(t, resp.Data.Executions, 1)

				exec := resp.Data.Executions[0]
				assert.Equal(t, int64(1), *exec.Id)
				assert.Equal(t, "none", *exec.Error)
				assert.Equal(t, "{}", *exec.Request.Body)
				assert.Equal(t, "headers", *exec.Request.Headers)
				assert.Equal(t, "http://example.com", *exec.Request.Url)
				assert.Equal(t, "{}", *exec.Response.Body)
				assert.Equal(t, "headers", *exec.Response.Headers)
				assert.Equal(t, "200 OK", *exec.Response.Status)
				assert.Equal(t, 200, *exec.Response.StatusCode)
				assert.Equal(t, api.WebhookExecResultSUCCESS, *exec.Result)
				assert.Equal(t, api.TriggerARTIFACTCREATION, *exec.TriggerType)

				assert.Equal(t, int64(1), *resp.Data.ItemCount)
				assert.Equal(t, 1, *resp.Data.PageSize)
				assert.Equal(t, int64(1), *resp.Data.PageIndex)
				assert.Equal(t, int64(1), *resp.Data.PageCount)
			},
			verifyMocks: func(t *testing.T, mockSpaceFinder *mocks.SpaceFinder, mockRegistryRepo *mocks.RegistryRepository,
				mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository,
				mockAuthorizer *mocks.Authorizer, mockMetadataHelper *mocks.RegistryMetadataHelper) {
				mockMetadataHelper.AssertExpectations(t)
				mockSpaceFinder.AssertExpectations(t)
				mockAuthorizer.AssertExpectations(t)
				mockRegistryRepo.AssertExpectations(t)
				mockWebhooksRepo.AssertExpectations(t)
				mockWebhooksExecRepo.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSpaceFinder := new(mocks.SpaceFinder)
			mockRegistryRepo := new(mocks.RegistryRepository)
			mockWebhooksRepo := new(mocks.WebhooksRepository)
			mockWebhooksExecRepo := new(mocks.WebhooksExecutionRepository)
			mockAuthorizer := new(mocks.Authorizer)
			mockMetadataHelper := new(mocks.RegistryMetadataHelper)

			// Create controller
			controller := &APIController{
				SpaceFinder:                 mockSpaceFinder,
				RegistryRepository:          mockRegistryRepo,
				WebhooksRepository:          mockWebhooksRepo,
				WebhooksExecutionRepository: mockWebhooksExecRepo,
				Authorizer:                  mockAuthorizer,
				RegistryMetadataHelper:      mockMetadataHelper,
			}

			// Setup mocks
			tt.setupMocks(mockSpaceFinder, mockRegistryRepo, mockWebhooksRepo, mockWebhooksExecRepo, mockAuthorizer, mockMetadataHelper)

			// Execute test
			response, err := controller.ListWebhookExecutions(context.Background(), tt.request)

			// Validate results
			tt.validate(t, response, err)

			// Verify mock expectations
			tt.verifyMocks(t, mockSpaceFinder, mockRegistryRepo, mockWebhooksRepo, mockWebhooksExecRepo, mockAuthorizer, mockMetadataHelper)
		})
	}
}

func TestMapToWebhookExecutionResponseEntity_Table(t *testing.T) {
	tests := []struct {
		name     string
		input    gitnesstypes.WebhookExecutionCore
		expected *api.WebhookExecution
	}{
		{
			name: "success_case",
			input: gitnesstypes.WebhookExecutionCore{
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
			},
			expected: &api.WebhookExecution{
				Created:  utils.Int64Ptr(1234567890),
				Duration: utils.Int64Ptr(100),
				Id:       utils.Int64Ptr(1),
				Error:    utils.StringPtr("none"),
				Request: &api.WebhookExecRequest{
					Body:    utils.StringPtr("{}"),
					Headers: utils.StringPtr("headers"),
					Url:     utils.StringPtr("http://example.com"),
				},
				Response: &api.WebhookExecResponse{
					Body:       utils.StringPtr("{}"),
					Headers:    utils.StringPtr("headers"),
					Status:     utils.StringPtr("200 OK"),
					StatusCode: utils.IntPtr(200),
				},
				RetriggerOf:   nil,
				Retriggerable: utils.BoolPtr(true),
				WebhookId:     utils.Int64Ptr(4),
				Result:        utils.WebhookExecResultPtr(api.WebhookExecResultSUCCESS),
				TriggerType:   utils.WebhookTriggerPtr(api.TriggerARTIFACTCREATION),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := createTestWebhookExecution(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions for testing.
//
//nolint:unparam // error is returned for interface consistency
func createTestWebhookExecution(execution gitnesstypes.WebhookExecutionCore) (*api.WebhookExecution, error) {
	result := api.WebhookExecResultSUCCESS
	triggerType := api.TriggerARTIFACTCREATION
	return &api.WebhookExecution{
		Created:  utils.Int64Ptr(execution.Created),
		Duration: utils.Int64Ptr(execution.Duration),
		Id:       utils.Int64Ptr(execution.ID),
		Error:    utils.StringPtr(execution.Error),
		Request: &api.WebhookExecRequest{
			Body:    utils.StringPtr(execution.Request.Body),
			Headers: utils.StringPtr(execution.Request.Headers),
			Url:     utils.StringPtr(execution.Request.URL),
		},
		Response: &api.WebhookExecResponse{
			Body:       utils.StringPtr(execution.Response.Body),
			Headers:    utils.StringPtr(execution.Response.Headers),
			Status:     utils.StringPtr(execution.Response.Status),
			StatusCode: utils.IntPtr(execution.Response.StatusCode),
		},
		RetriggerOf:   execution.RetriggerOf,
		Retriggerable: utils.BoolPtr(execution.Retriggerable),
		WebhookId:     utils.Int64Ptr(execution.WebhookID),
		Result:        &result,
		TriggerType:   &triggerType,
	}, nil
}

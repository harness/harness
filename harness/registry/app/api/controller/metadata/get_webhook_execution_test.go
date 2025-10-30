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
package metadata

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetWebhookExecution(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name       string
		setupMocks func(
			*APIController,
			*mocks.SpaceFinder,
			*mocks.Authorizer,
			*mocks.RegistryMetadataHelper,
			*mocks.WebhooksRepository,
			*mocks.WebhooksExecutionRepository)
		request  api.GetWebhookExecutionRequestObject
		validate func(*testing.T, api.GetWebhookExecutionResponseObject, error)
	}{
		{
			name: "invalid_execution_identifier",
			setupMocks: func(c *APIController, mockSpaceFinder *mocks.SpaceFinder, mockAuthorizer *mocks.Authorizer, mockRegistryMetadataHelper *mocks.RegistryMetadataHelper, mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository) {
				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, "reg", enum.PermissionRegistryView).Return([]coretypes.PermissionCheck{})
				mockAuthorizer.On("CheckAll", mock.Anything, (*auth.Session)(nil), mock.Anything).Return(true, nil)

				c.SpaceFinder = mockSpaceFinder
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.WebhooksRepository = mockWebhooksRepo
				c.WebhooksExecutionRepository = mockWebhooksExecRepo
			},
			request: api.GetWebhookExecutionRequestObject{
				RegistryRef:        "reg",
				WebhookIdentifier:  "webhook",
				WebhookExecutionId: "invalid",
			},
			validate: func(t *testing.T, response api.GetWebhookExecutionResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				resp, ok := response.(api.GetWebhookExecution400JSONResponse)
				assert.True(t, ok, "expected 400 response")
				assert.Equal(t, "400", resp.Code)
				assert.Equal(t, "invalid webhook execution identifier: invalid, err: strconv.ParseInt: parsing \"invalid\": invalid syntax", resp.Message)
			},
		},
		{
			name: "permission_check_fails",
			setupMocks: func(c *APIController, mockSpaceFinder *mocks.SpaceFinder, mockAuthorizer *mocks.Authorizer, mockRegistryMetadataHelper *mocks.RegistryMetadataHelper, mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository) {
				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, "reg", enum.PermissionRegistryView).Return([]coretypes.PermissionCheck{})
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

				c.SpaceFinder = mockSpaceFinder
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.WebhooksRepository = mockWebhooksRepo
				c.WebhooksExecutionRepository = mockWebhooksExecRepo
			},
			request: api.GetWebhookExecutionRequestObject{
				RegistryRef:        "reg",
				WebhookIdentifier:  "webhook",
				WebhookExecutionId: "1",
			},
			validate: func(t *testing.T, response api.GetWebhookExecutionResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				resp, ok := response.(api.GetWebhookExecution403JSONResponse)
				assert.True(t, ok, "expected 403 response")
				assert.Equal(t, "403", resp.Code)
				assert.Equal(t, "forbidden", resp.Message)
			},
		},
		{
			name: "success_case",
			setupMocks: func(c *APIController, mockSpaceFinder *mocks.SpaceFinder, mockAuthorizer *mocks.Authorizer, mockRegistryMetadataHelper *mocks.RegistryMetadataHelper, mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository) {
				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				execution := &coretypes.WebhookExecutionCore{
					ID:            1,
					Created:       now,
					WebhookID:     1,
					Result:        enum.WebhookExecutionResultSuccess,
					Duration:      0,
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
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, "reg", enum.PermissionRegistryView).Return([]coretypes.PermissionCheck{})
				mockAuthorizer.On("CheckAll", mock.Anything, (*auth.Session)(nil)).Return(true, nil)
				mockWebhooksExecRepo.On("Find", mock.Anything, int64(1)).Return(execution, nil)

				c.SpaceFinder = mockSpaceFinder
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.WebhooksRepository = mockWebhooksRepo
				c.WebhooksExecutionRepository = mockWebhooksExecRepo
			},
			request: api.GetWebhookExecutionRequestObject{
				RegistryRef:        "reg",
				WebhookIdentifier:  "webhook",
				WebhookExecutionId: "1",
			},
			validate: func(t *testing.T, response api.GetWebhookExecutionResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				resp, ok := response.(api.GetWebhookExecution200JSONResponse)
				assert.True(t, ok, "expected 200 response")
				assert.Equal(t, api.StatusSUCCESS, resp.Status)

				exec := resp.Data
				assert.Equal(t, int64(1), *exec.Id)
				assert.Equal(t, now, *exec.Created)
				assert.Equal(t, int64(1), *exec.WebhookId)
				assert.Equal(t, api.WebhookExecResultSUCCESS, *exec.Result)
				assert.Equal(t, int64(0), *exec.Duration)
				assert.Equal(t, "", *exec.Error)
				assert.False(t, *exec.Retriggerable)

				assert.Equal(t, "http://example.com", *exec.Request.Url)
				assert.Equal(t, "{}", *exec.Request.Headers)
				assert.Equal(t, "{}", *exec.Request.Body)

				assert.Equal(t, 200, *exec.Response.StatusCode)
				assert.Equal(t, "OK", *exec.Response.Status)
				assert.Equal(t, "{}", *exec.Response.Headers)
				assert.Equal(t, "{}", *exec.Response.Body)
			},
		},
		{
			name: "find_execution_error",
			setupMocks: func(c *APIController, mockSpaceFinder *mocks.SpaceFinder, mockAuthorizer *mocks.Authorizer, mockRegistryMetadataHelper *mocks.RegistryMetadataHelper, mockWebhooksRepo *mocks.WebhooksRepository, mockWebhooksExecRepo *mocks.WebhooksExecutionRepository) {
				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On(
					"GetRegistryRequestBaseInfo",
					mock.Anything,
					"",
					"reg",
				).Return(regInfo, nil)
				mockRegistryMetadataHelper.On(
					"GetPermissionChecks",
					space,
					"reg",
					enum.PermissionRegistryView,
				).Return([]coretypes.PermissionCheck{})
				mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
				mockWebhooksExecRepo.On(
					"Find",
					mock.Anything,
					int64(1),
				).Return(nil, fmt.Errorf("error finding execution"))

				c.SpaceFinder = mockSpaceFinder
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.WebhooksRepository = mockWebhooksRepo
				c.WebhooksExecutionRepository = mockWebhooksExecRepo
			},
			request: api.GetWebhookExecutionRequestObject{
				RegistryRef:        "reg",
				WebhookIdentifier:  "webhook",
				WebhookExecutionId: "1",
			},
			validate: func(t *testing.T, response api.GetWebhookExecutionResponseObject, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				resp, ok := response.(api.GetWebhookExecution500JSONResponse)
				assert.True(t, ok, "expected 500 response")
				assert.Equal(t, "500", resp.Code)
				assert.Equal(t, "failed to find webhook execution: error finding execution", resp.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockSpaceFinder := new(mocks.SpaceFinder)
			mockAuthorizer := new(mocks.Authorizer)
			mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
			mockWebhooksRepo := new(mocks.WebhooksRepository)
			mockWebhooksExecRepo := new(mocks.WebhooksExecutionRepository)

			// Create controller
			controller := &APIController{}

			// Setup mocks
			tt.setupMocks(controller, mockSpaceFinder, mockAuthorizer, mockRegistryMetadataHelper, mockWebhooksRepo, mockWebhooksExecRepo)

			// Call function
			resp, err := controller.GetWebhookExecution(context.Background(), tt.request)

			// Validate response
			tt.validate(t, resp, err)

			// Verify mock expectations
			mockSpaceFinder.AssertExpectations(t)
			mockAuthorizer.AssertExpectations(t)
			mockRegistryMetadataHelper.AssertExpectations(t)
			mockWebhooksRepo.AssertExpectations(t)
			mockWebhooksExecRepo.AssertExpectations(t)
		})
	}
}

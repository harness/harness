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
package metadata_test

import (
	"context"
	"fmt"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/utils"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testWebhookIdentifier = "test-webhook"
	testWebhookURL        = "http://example.com"
)

func TestCreateWebhook(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*metadata.APIController)
		request       api.CreateWebhookRequestObject
		expectedResp  interface{}
		expectedError error
	}{
		{
			name: "success_case",
			setupMocks: func(c *metadata.APIController) {
				// Mock registry metadata helper
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockWebhooksRepository := new(mocks.WebhooksRepository)
				mockAuthorizer := new(mocks.Authorizer)

				regInfo := &registrytypes.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					RegistryType:       api.RegistryTypeVIRTUAL,
				}

				space := &coretypes.SpaceCore{
					Path: "root/parent",
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope: coretypes.Scope{SpacePath: "root/parent"},
						Resource: coretypes.Resource{
							Type:       enum.ResourceTypeRegistry,
							Identifier: "reg",
						},
						Permission: enum.PermissionRegistryEdit,
					},
				}

				webhook := &coretypes.WebhookCore{
					ID:          1,
					DisplayName: testWebhookIdentifier,
					Identifier:  testWebhookIdentifier,
					URL:         "http://example.com",
					Enabled:     true,
					Insecure:    false,
					ParentID:    regInfo.RegistryID,
					Type:        enum.WebhookTypeExternal,
				}

				webhookResponseEntity := &api.Webhook{
					Identifier: "test-webhook",
					Name:       testWebhookIdentifier,
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
					CreatedAt:  utils.StringPtr("2023-01-01T00:00:00Z"),
					ModifiedAt: utils.StringPtr("2023-01-01T00:00:00Z"),
				}

				// Set up expectations.
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo,
					nil)
				mockRegistryMetadataHelper.On(
					"GetPermissionChecks",
					mock.Anything,
					"reg",
					enum.PermissionRegistryEdit,
				).Return(permissionChecks)
				// Set up authorizer to expect a session with Principal ID 123 for success case
				mockSession := &auth.Session{Principal: coretypes.Principal{ID: 123}}
				mockAuthorizer.On("CheckAll", mock.Anything, mockSession, permissionChecks[0]).Return(true, nil)

				mockRegistryMetadataHelper.On(
					"MapToWebhookCore",
					mock.Anything,
					mock.MatchedBy(func(req api.WebhookRequest) bool {
						return req.Identifier == testWebhookIdentifier &&
							req.Name == testWebhookIdentifier &&
							req.Url == testWebhookURL &&
							req.Enabled == true &&
							req.Insecure == false
					}),
					regInfo,
				).Return(webhook, nil)

				mockWebhooksRepository.On("Create", mock.Anything, webhook).Return(nil)
				mockWebhooksRepository.On(
					"GetByRegistryAndIdentifier",
					mock.Anything,
					regInfo.RegistryID,
					"test-webhook",
				).Return(webhook, nil)
				mockRegistryMetadataHelper.On(
					"MapToWebhookResponseEntity",
					mock.Anything,
					webhook,
				).Return(webhookResponseEntity, nil)

				// Assign mocks to controller
				c.SpaceFinder = mockSpaceFinder
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.Authorizer = mockAuthorizer
				c.WebhooksRepository = mockWebhooksRepository
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       testWebhookIdentifier,
					Identifier: testWebhookIdentifier,
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook201JSONResponse{
				WebhookResponseJSONResponse: api.WebhookResponseJSONResponse{
					Data: api.Webhook{
						Name:       "test-webhook",
						Identifier: "test-webhook",
						Url:        "http://example.com",
						Enabled:    true,
						Insecure:   false,
					},
					Status: api.StatusSUCCESS,
				},
			},
		},
		{
			name: "reserved_identifier",
			setupMocks: func(c *metadata.APIController) {
				// 1. Mock registry metadata helper for GetRegistryRequestBaseInfo
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				regInfo := &registrytypes.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					RegistryType:       api.RegistryTypeVIRTUAL,
				}
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").
					Return(regInfo, nil)
				c.RegistryMetadataHelper = mockRegistryMetadataHelper

				// 2. Mock spaceFinder
				mockSpaceFinder := new(mocks.SpaceFinder)
				space := &coretypes.SpaceCore{
					Path: "root/parent",
				}
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				c.SpaceFinder = mockSpaceFinder

				// 3. Mock GetPermissionChecks
				permissionCheck := coretypes.PermissionCheck{
					Scope: coretypes.Scope{SpacePath: "root/parent"},
					Resource: coretypes.Resource{
						Type:       enum.ResourceTypeRegistry,
						Identifier: "reg",
					},
					Permission: enum.PermissionRegistryEdit,
				}
				mockRegistryMetadataHelper.On("GetPermissionChecks", mock.Anything, "reg", enum.PermissionRegistryEdit).
					Return([]coretypes.PermissionCheck{permissionCheck})

				// 4. Mock Authorizer
				mockAuthorizer := new(mocks.Authorizer)
				mockAuthorizer.On("CheckAll", mock.Anything, (*auth.Session)(nil), permissionCheck).Return(true, nil)
				c.Authorizer = mockAuthorizer

				// 5. Mock MapToWebhookCore
				mockRegistryMetadataHelper.On("MapToWebhookCore", mock.Anything,
					mock.MatchedBy(func(req api.WebhookRequest) bool {
						return req.Identifier == "internal-webhook" &&
							req.Name == "internal-webhook" &&
							req.Url == testWebhookURL &&
							req.Enabled == true &&
							req.Insecure == false
					}), regInfo).Return(nil, fmt.Errorf("webhook identifier internal-webhook is reserved"))
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       "internal-webhook",
					Identifier: "internal-webhook",
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "failed to store webhook webhook identifier internal-webhook is reserved",
				},
			},
		},
		{
			name: "invalid_registry_reference",
			setupMocks: func(c *metadata.APIController) {
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "invalid-reg").
					Return(nil, fmt.Errorf("invalid registry reference"))
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "invalid-reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       "test-webhook",
					Identifier: "test-webhook",
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "invalid registry reference",
				},
			},
		},
		{
			name: "permission_check_fails",
			setupMocks: func(c *metadata.APIController) {
				// 1. Mock registry metadata helper for GetRegistryRequestBaseInfo
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				regInfo := &registrytypes.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					RegistryType:       api.RegistryTypeVIRTUAL,
				}
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").
					Return(regInfo, nil)

				// 2. Mock spaceFinder
				mockSpaceFinder := new(mocks.SpaceFinder)
				space := &coretypes.SpaceCore{
					Path: "root/parent",
				}
				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				c.SpaceFinder = mockSpaceFinder

				// 3. Mock GetPermissionChecks
				permissionCheck := coretypes.PermissionCheck{
					Scope: coretypes.Scope{SpacePath: "root/parent"},
					Resource: coretypes.Resource{
						Type:       enum.ResourceTypeRegistry,
						Identifier: "reg",
					},
					Permission: enum.PermissionRegistryEdit,
				}
				mockRegistryMetadataHelper.On("GetPermissionChecks", mock.Anything, "reg", enum.PermissionRegistryEdit).
					Return([]coretypes.PermissionCheck{permissionCheck})
				c.RegistryMetadataHelper = mockRegistryMetadataHelper

				// Mock authorizer with manual response
				mockAuthorizer := new(mocks.Authorizer)
				mockAuthorizer.On("CheckAll", mock.Anything, (*auth.Session)(nil), permissionCheck).Return(false,
					apiauth.ErrUnauthorized)
				c.Authorizer = mockAuthorizer
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       testWebhookIdentifier,
					Identifier: testWebhookIdentifier,
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook403JSONResponse{
				UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{
					Code:    "403",
					Message: "unauthorized",
				},
			},
		},
		{
			name: "non_virtual_registry",
			setupMocks: func(c *metadata.APIController) {
				// 1. Mock registry metadata helper for GetRegistryRequestBaseInfo
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				regInfo := &registrytypes.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					RegistryType:       api.RegistryTypeUPSTREAM,
				}
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").
					Return(regInfo, nil)
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       testWebhookIdentifier,
					Identifier: testWebhookIdentifier,
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "not allowed to create webhook for UPSTREAM registry",
				},
			},
		},
		{
			name: "permission_check_fails",
			setupMocks: func(c *metadata.APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockAuthorizer := new(mocks.Authorizer)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &registrytypes.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					RegistryType:       api.RegistryTypeVIRTUAL,
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope:      coretypes.Scope{SpacePath: "root/parent"},
						Resource:   coretypes.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
						Permission: enum.PermissionRegistryEdit,
					},
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").
					Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").
					Return(regInfo, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, "reg", enum.PermissionRegistryEdit).
					Return(permissionChecks)
				mockAuthorizer.On("CheckAll", mock.Anything, (*auth.Session)(nil), permissionChecks[0]).
					Return(false, fmt.Errorf("unauthorized"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.Authorizer = mockAuthorizer
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       testWebhookIdentifier,
					Identifier: testWebhookIdentifier,
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook403JSONResponse{
				UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{
					Code:    "403",
					Message: "unauthorized",
				},
			},
		},
		{
			name: "duplicate_webhook_identifier",
			setupMocks: func(c *metadata.APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockAuthorizer := new(mocks.Authorizer)
				mockWebhooksRepository := new(mocks.WebhooksRepository)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &registrytypes.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					RegistryType:       api.RegistryTypeVIRTUAL,
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope:      coretypes.Scope{SpacePath: "root/parent"},
						Resource:   coretypes.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
						Permission: enum.PermissionRegistryEdit,
					},
				}

				webhook := &coretypes.WebhookCore{
					ID:        1,
					Type:      enum.WebhookTypeExternal,
					CreatedBy: 0,
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").
					Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").
					Return(regInfo, nil)
				mockRegistryMetadataHelper.On("GetPermissionChecks", space, "reg", enum.PermissionRegistryEdit).
					Return(permissionChecks)
				// Set up authorizer to expect a session with Principal ID 123 for this test case
				mockSession := &auth.Session{Principal: coretypes.Principal{ID: 123}}
				mockAuthorizer.On("CheckAll", mock.Anything, mockSession, permissionChecks[0]).
					Return(true, nil)
				mockRegistryMetadataHelper.On("MapToWebhookCore", mock.Anything,
					mock.MatchedBy(func(req api.WebhookRequest) bool {
						return req.Identifier == testWebhookIdentifier &&
							req.Name == testWebhookIdentifier &&
							req.Url == testWebhookURL &&
							req.Enabled == true &&
							req.Insecure == false
					}), regInfo).
					Return(webhook, nil)
				mockWebhooksRepository.On("Create", mock.Anything, webhook).
					Return(fmt.Errorf("resource is a duplicate"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.Authorizer = mockAuthorizer
				c.WebhooksRepository = mockWebhooksRepository
			},
			request: api.CreateWebhookRequestObject{
				RegistryRef: "reg",
				Body: &api.CreateWebhookJSONRequestBody{
					Name:       testWebhookIdentifier,
					Identifier: testWebhookIdentifier,
					Url:        "http://example.com",
					Enabled:    true,
					Insecure:   false,
				},
			},
			expectedResp: api.CreateWebhook400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "failed to store webhook, Webhook with identifier test-webhook already exists",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create controller and setup mocks.
			controller := &metadata.APIController{}
			tt.setupMocks(controller)

			// Create a context with auth session for success case
			localCtx := context.Background()

			// Add auth session for tests that need it to set CreatedBy field
			if tt.name == "success_case" || tt.name == "duplicate_webhook_identifier" {
				mockPrincipal := coretypes.Principal{ID: 123}
				mockSession := &auth.Session{Principal: mockPrincipal}
				localCtx = request.WithAuthSession(localCtx, mockSession)
			}

			// Execute.
			resp, err := controller.CreateWebhook(localCtx, tt.request)

			// Verify response matches expected type and content.
			switch expected := tt.expectedResp.(type) {
			case api.CreateWebhook201JSONResponse:
				assert.NoError(t, err, "Expected no error but got one")
				actualResp, ok := resp.(api.CreateWebhook201JSONResponse)
				assert.True(t, ok, "Expected 201 response")

				// Verify webhook data fields individually.
				assert.Equal(t, expected.Data.Identifier, actualResp.Data.Identifier, "Webhook identifier should match")
				assert.Equal(t, expected.Data.Name, actualResp.Data.Name, "Webhook name should match")
				assert.Equal(t, expected.Data.Url, actualResp.Data.Url, "Webhook URL should match")
				assert.Equal(t, expected.Data.Enabled, actualResp.Data.Enabled, "Enabled status should match")
				assert.Equal(t, expected.Data.Insecure, actualResp.Data.Insecure, "Insecure status should match")
				assert.NotEmpty(t, actualResp.Data.CreatedAt, "CreatedAt should not be empty")
				assert.NotEmpty(t, actualResp.Data.ModifiedAt, "ModifiedAt should not be empty")
				assert.Equal(t, api.StatusSUCCESS, actualResp.Status, "Status should be SUCCESS")

			case api.CreateWebhook400JSONResponse:
				// For error responses, we don't need to check the err value since responses with error codes are still valid responses
				actualResp, ok := resp.(api.CreateWebhook400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
				assert.Equal(t, expected.Code, actualResp.Code, "Error code should match")
				assert.Equal(t, expected.Message, actualResp.Message, "Error message should match")

			case api.CreateWebhook403JSONResponse:
				// For error responses, we don't need to check the err value since responses with error codes are still valid responses
				actualResp, ok := resp.(api.CreateWebhook403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
				assert.Equal(t, expected.Code, actualResp.Code, "Error code should match")
				assert.Equal(t, expected.Message, actualResp.Message, "Error message should match")

			default:
				t.Fatalf("Unexpected response type: %T", tt.expectedResp)
			}

			// Verify mock expectations
			if mockSpaceFinder, ok := controller.SpaceFinder.(*mocks.SpaceFinder); ok {
				mockSpaceFinder.AssertExpectations(t)
			}
			if mockMetadataHelper, ok := controller.RegistryMetadataHelper.(*mocks.RegistryMetadataHelper); ok {
				mockMetadataHelper.AssertExpectations(t)
			}
			if mockWebhooksRepo, ok := controller.WebhooksRepository.(*mocks.WebhooksRepository); ok {
				mockWebhooksRepo.AssertExpectations(t)
			}
			if mockAuditService, ok := controller.AuditService.(*mocks.AuditService); ok {
				mockAuditService.AssertExpectations(t)
			}
		})
	}
}

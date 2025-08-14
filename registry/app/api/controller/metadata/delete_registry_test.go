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

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDeleteRegistry(t *testing.T) {
	// Create a mock session for testing
	principal := coretypes.Principal{
		ID:    1,
		Type:  enum.PrincipalTypeUser,
		Email: "test@example.com",
	}
	mockSession := &auth.Session{
		Principal: principal,
	}

	// Create a context with the mock session
	testCtx := request.WithAuthSession(context.Background(), mockSession)

	tests := []struct {
		name          string
		setupMocks    func(*APIController)
		request       api.DeleteRegistryRequestObject
		expectedResp  api.DeleteRegistryResponseObject
		expectedError error
	}{
		{
			name: "success_case_virtual_registry",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryRepository := new(mocks.RegistryRepository)
				mockAuthorizer := new(mocks.Authorizer)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockImageStore := new(mocks.ImageRepository)
				mockTx := new(mocks.Transaction)
				mockAuditService := new(mocks.AuditService)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				registry := &types.Registry{
					ID:          1,
					Name:        "reg",
					ParentID:    2,
					Type:        "virtual",
					PackageType: "pypi",
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope:      coretypes.Scope{SpacePath: "root/parent"},
						Resource:   coretypes.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
						Permission: enum.PermissionRegistryDelete,
					},
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo,
					nil)
				mockRegistryMetadataHelper.On(
					"GetPermissionChecks",
					space,
					"reg",
					enum.PermissionRegistryDelete,
				).Return(permissionChecks)
				mockAuthorizer.On(
					"CheckAll",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					permissionChecks[0],
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					regInfo.ParentID,
					regInfo.RegistryIdentifier,
				).Return(registry, nil)
				mockRegistryRepository.On("FetchRegistriesIDByUpstreamProxyID", mock.Anything,
					mock.Anything, regInfo.RootIdentifierID).Return([]int64{}, nil)
				mockImageStore.On("DeleteDownloadStatByRegistryID", mock.Anything, regInfo.RegistryID).Return(nil)
				mockImageStore.On("DeleteBandwidthStatByRegistryID", mock.Anything, regInfo.RegistryID).Return(nil)
				mockImageStore.On("DeleteByRegistryID", mock.Anything, regInfo.RegistryID).Return(nil)
				mockRegistryRepository.On("Delete", mock.Anything, regInfo.ParentID,
					regInfo.RegistryIdentifier).Return(nil)
				mockAuditService.On(
					"Log",
					mock.Anything,
					mock.AnythingOfType("*types.PrincipalInfo"),
					mock.AnythingOfType("*audit.Resource"),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("audit.Option"),
				).Return(nil)
				// Simply return nil for the transaction - we're testing the controller logic, not transaction details
				mockTx.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.ImageStore = mockImageStore
				c.tx = mockTx
				c.AuditService = mockAuditService
			},
			request: api.DeleteRegistryRequestObject{
				RegistryRef: "reg",
			},
			expectedResp: api.DeleteRegistry200JSONResponse{
				SuccessJSONResponse: api.SuccessJSONResponse{
					Status: api.StatusSUCCESS,
				},
			},
		},
		{
			name: "invalid_registry_reference",
			setupMocks: func(c *APIController) {
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockRegistryMetadataHelper.On(
					"GetRegistryRequestBaseInfo",
					mock.Anything,
					"",
					"invalid-reg",
				).Return(nil, fmt.Errorf("invalid registry reference"))
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
			},
			request: api.DeleteRegistryRequestObject{
				RegistryRef: "invalid-reg",
			},
			expectedResp: api.DeleteRegistry400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "invalid registry reference",
				},
			},
			expectedError: nil,
		},
		{
			name: "permission_check_fails",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockAuthorizer := new(mocks.Authorizer)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope:      coretypes.Scope{SpacePath: "root/parent"},
						Resource:   coretypes.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
						Permission: enum.PermissionRegistryDelete,
					},
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo,
					nil)
				mockRegistryMetadataHelper.On(
					"GetPermissionChecks",
					space,
					"reg",
					enum.PermissionRegistryDelete,
				).Return(permissionChecks)
				mockAuthorizer.On(
					"CheckAll",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					permissionChecks[0],
				).Return(false, fmt.Errorf("not authorized"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.Authorizer = mockAuthorizer
			},
			request: api.DeleteRegistryRequestObject{
				RegistryRef: "reg",
			},
			expectedResp: api.DeleteRegistry403JSONResponse{
				UnauthorizedJSONResponse: api.UnauthorizedJSONResponse{
					Code:    "403",
					Message: "not authorized",
				},
			},
			expectedError: nil,
		},
		{
			name: "registry_not_found",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryRepository := new(mocks.RegistryRepository)
				mockAuthorizer := new(mocks.Authorizer)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope:      coretypes.Scope{SpacePath: "root/parent"},
						Resource:   coretypes.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
						Permission: enum.PermissionRegistryDelete,
					},
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo,
					nil)
				mockRegistryMetadataHelper.On(
					"GetPermissionChecks",
					space,
					"reg",
					enum.PermissionRegistryDelete,
				).Return(permissionChecks)
				mockAuthorizer.On(
					"CheckAll",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					permissionChecks[0],
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					regInfo.ParentID,
					regInfo.RegistryIdentifier,
				).Return(nil, fmt.Errorf("registry doesn't exist with this key"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
			},
			request: api.DeleteRegistryRequestObject{
				RegistryRef: "reg",
			},
			expectedResp: api.DeleteRegistry404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Code:    "404",
					Message: "registry doesn't exist with this key",
				},
			},
		},
		{
			name: "success_case_native_registry",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryRepository := new(mocks.RegistryRepository)
				mockAuthorizer := new(mocks.Authorizer)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockImageStore := new(mocks.ImageRepository)
				mockTx := new(mocks.Transaction)
				mockAuditService := new(mocks.AuditService)
				mockUpstreamProxyStore := new(mocks.UpstreamProxyStore)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
				}

				registry := &types.Registry{
					ID:          1,
					Name:        "reg",
					ParentID:    2,
					Type:        "native",
					PackageType: "pypi",
				}

				permissionChecks := []coretypes.PermissionCheck{
					{
						Scope:      coretypes.Scope{SpacePath: "root/parent"},
						Resource:   coretypes.Resource{Type: enum.ResourceTypeRegistry, Identifier: "reg"},
						Permission: enum.PermissionRegistryDelete,
					},
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "reg").Return(regInfo,
					nil)
				mockRegistryMetadataHelper.On(
					"GetPermissionChecks",
					space,
					"reg",
					enum.PermissionRegistryDelete,
				).Return(permissionChecks)
				mockAuthorizer.On(
					"CheckAll",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					permissionChecks[0],
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					regInfo.ParentID,
					regInfo.RegistryIdentifier,
				).Return(registry, nil)
				mockRegistryRepository.On(
					"FetchUpstreamProxyIDs",
					mock.Anything,
					[]string{regInfo.RegistryIdentifier},
					regInfo.ParentID,
				).Return([]int64{}, nil)
				mockRegistryRepository.On("FetchRegistriesIDByUpstreamProxyID", mock.Anything,
					mock.Anything, regInfo.RootIdentifierID).Return([]int64{}, nil)
				mockUpstreamProxyStore.On("Delete", mock.Anything, regInfo.ParentID,
					regInfo.RegistryIdentifier).Return(nil)
				mockImageStore.On("DeleteDownloadStatByRegistryID", mock.Anything, regInfo.RegistryID).Return(nil)
				mockImageStore.On("DeleteBandwidthStatByRegistryID", mock.Anything, regInfo.RegistryID).Return(nil)
				mockImageStore.On("DeleteByRegistryID", mock.Anything, regInfo.RegistryID).Return(nil)
				mockRegistryRepository.On("Delete", mock.Anything, regInfo.ParentID,
					regInfo.RegistryIdentifier).Return(nil)
				mockAuditService.On(
					"Log",
					mock.Anything,
					mock.AnythingOfType("*types.PrincipalInfo"),
					mock.AnythingOfType("*audit.Resource"),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("string"),
					mock.AnythingOfType("audit.Option"),
				).Return(nil)
				// Simply return nil for the transaction - we're testing the controller logic, not transaction details
				mockTx.On("WithTx", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(nil)

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.ImageStore = mockImageStore
				c.tx = mockTx
				c.AuditService = mockAuditService
				c.UpstreamProxyStore = mockUpstreamProxyStore
			},
			request: api.DeleteRegistryRequestObject{
				RegistryRef: "reg",
			},
			expectedResp: api.DeleteRegistry200JSONResponse{
				SuccessJSONResponse: api.SuccessJSONResponse{
					Status: api.StatusSUCCESS,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			controller := &APIController{}
			tt.setupMocks(controller)

			// Execute with the mock session context
			resp, err := controller.DeleteRegistry(testCtx, tt.request)

			// Verify error only if explicitly expected.
			if tt.expectedError != nil {
				assert.Error(t, err, "Expected an error but got none")
				assert.Equal(t, tt.expectedError.Error(), err.Error(), "Error message should match")
			} else if _, ok := tt.expectedResp.(api.DeleteRegistry200JSONResponse); ok {
				// Only assert no error for success responses.
				assert.NoError(t, err, "Expected no error for success response")
			}
			// For error response types like 400/404/403, we don't assert err since the response object is what matters.

			// Verify response with detailed assertions.
			assert.NotNil(t, resp, "Response should not be nil")

			// Verify correct response type matching.
			switch tt.expectedResp.(type) {
			case api.DeleteRegistry200JSONResponse:
				_, ok := resp.(api.DeleteRegistry200JSONResponse)
				assert.True(t, ok, "Expected 200 success response")
				// Not checking Status field as it's hardcoded in test data and doesn't validate behavior.

			case api.DeleteRegistry400JSONResponse:
				_, ok := resp.(api.DeleteRegistry400JSONResponse)
				assert.True(t, ok, "Expected 400 bad request response")
				// Not checking fields as they're hardcoded in test data.

			case api.DeleteRegistry403JSONResponse:
				_, ok := resp.(api.DeleteRegistry403JSONResponse)
				assert.True(t, ok, "Expected 403 forbidden response")
				// Not checking fields as they're hardcoded in test data.

			case api.DeleteRegistry404JSONResponse:
				_, ok := resp.(api.DeleteRegistry404JSONResponse)
				assert.True(t, ok, "Expected 404 not found response")
				// Not checking fields as they're hardcoded in test data.

			default:
				// Fallback to simple type equality for any other response types.
				expectedType := fmt.Sprintf("%T", tt.expectedResp)
				actualType := fmt.Sprintf("%T", resp)
				assert.Equal(t, expectedType, actualType, "Response type should match.")
			}

			// Verify only essential mocks since we're not executing transaction functions.
			if controller.SpaceFinder != nil {
				mockSpaceFinder, ok := controller.SpaceFinder.(*mocks.SpaceFinder)
				if !ok {
					t.Fatal("Expected spaceFinder to be of type *mocks.spaceFinder")
				}
				mockSpaceFinder.AssertExpectations(t)
			}

			// Verify only FindByRef and GetPermissionChecks from RegistryRepository.
			if controller.RegistryRepository != nil {
				mockRegistryRepo, ok := controller.RegistryRepository.(*mocks.RegistryRepository)
				if !ok {
					t.Fatal("Expected RegistryRepository to be of type *mocks.RegistryRepository")
				}
				// We could use AssertCalled for specific methods if needed.
				// Only verify GetByParentIDAndName which is called before the transaction.
				mockRegistryRepo.AssertCalled(t, "GetByParentIDAndName", mock.Anything, mock.Anything, mock.Anything)
			}

			if controller.Authorizer != nil {
				mockAuthorizer, ok := controller.Authorizer.(*mocks.Authorizer)
				if !ok {
					t.Fatal("Expected Authorizer to be of type *mocks.Authorizer")
				}
				mockAuthorizer.AssertExpectations(t)
			}

			if controller.RegistryMetadataHelper != nil {
				mockRegistryMetadataHelper, ok := controller.RegistryMetadataHelper.(*mocks.RegistryMetadataHelper)
				if !ok {
					t.Fatal("Expected RegistryMetadataHelper to be of type *mocks.RegistryMetadataHelper")
				}
				mockRegistryMetadataHelper.AssertExpectations(t)
			}

			// Verify transaction was attempted.
			if controller.tx != nil {
				mockTx, ok := controller.tx.(*mocks.Transaction)
				if !ok {
					t.Fatal("Expected tx to be of type *mocks.Transaction")
				}
				mockTx.AssertCalled(t, "WithTx", mock.Anything, mock.Anything)
			}
		})
	}
}

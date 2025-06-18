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
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/utils"
	coretypes "github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	// TestRegistryName is the name used for test registries.
	testRegistryName = "test-registry"
)

var (
	// ErrTestConsumerNotNeeded is a sentinel error indicating consumer is not needed in tests.
	ErrTestConsumerNotNeeded = errors.New("consumer not needed in tests")
)

func TestCreateRegistry(t *testing.T) {
	space := &coretypes.SpaceCore{
		ID:   1,
		Path: "root",
	}

	// Create mock event system components.
	producer := &mocks.StreamProducer{}

	// Set up producer expectations.
	producer.On("Send", mock.Anything, "registry", mock.MatchedBy(func(payload map[string]interface{}) bool {
		return payload["action"] == "registry.create" &&
			payload["registry_id"] == int64(1) &&
			payload["registry_name"] == testRegistryName
	})).Return("", nil).Once()

	// Create events system.
	consumerFactory := func(_ string, _ string) (events.StreamConsumer, error) {
		// Return a sentinel error to indicate consumer is not needed in tests.
		return nil, ErrTestConsumerNotNeeded
	}
	eventsSystem, err := events.NewSystem(consumerFactory, producer)
	if err != nil {
		t.Fatalf("Failed to create events system: %v", err)
	}

	// Create event reporter.
	reporter, err := registryevents.NewReporter(eventsSystem)
	if err != nil {
		t.Fatalf("Failed to create event reporter: %v", err)
	}
	eventReporter := *reporter // Use value instead of pointer.

	// Helper mocks and setup complete.

	tests := []struct {
		name         string
		request      api.CreateRegistryRequestObject
		setupMocks   func() *metadata.APIController
		expectedResp interface{}
	}{
		{
			name: "create_virtual_registry_success",
			request: api.CreateRegistryRequestObject{
				Body: &api.CreateRegistryJSONRequestBody{
					Identifier: testRegistryName,
					ParentRef:  utils.StringPtr("root"),
					Config: func() *api.RegistryConfig {
						config := &api.RegistryConfig{Type: api.RegistryTypeVIRTUAL}
						_ = config.FromVirtualConfig(api.VirtualConfig{UpstreamProxies: &[]string{}})
						return config
					}(),
					PackageType:   api.PackageTypeDOCKER,
					CleanupPolicy: &[]api.CleanupPolicy{},
				},
			},
			expectedResp: api.CreateRegistry201JSONResponse{
				RegistryResponseJSONResponse: api.RegistryResponseJSONResponse{
					Data: api.Registry{
						Identifier: testRegistryName,
						Config: &api.RegistryConfig{
							Type: api.RegistryTypeVIRTUAL,
						},
						PackageType: api.PackageTypeDOCKER,
						Url:         "http://example.com/registry/test-registry",
						Description: utils.StringPtr(""),
						CreatedAt:   utils.StringPtr("-62135596800000"),
						ModifiedAt:  utils.StringPtr("-62135596800000"),
					},
					Status: api.StatusSUCCESS,
				},
			},
			setupMocks: func() *metadata.APIController {
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockRegistryRepo := new(mocks.RegistryRepository)
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockAuthorizer := new(mocks.Authorizer)
				mockAuditService := new(mocks.AuditService)
				mockCleanupPolicyRepo := new(mocks.CleanupPolicyRepository)
				mockTransactor := new(mocks.Transactor)
				mockGenericBlobRepo := new(mocks.GenericBlobRepository)
				// Create a mock URL provider.
				mockURLProvider := new(mocks.Provider)
				// Set up common URL provider expectations
				mockURLProvider.On("PackageURL", mock.Anything, mock.Anything, mock.Anything,
					mock.Anything).Return("http://example.com/registry/test-registry/docker")
				mockURLProvider.On("GenerateUIRegistryURL", mock.Anything, mock.Anything,
					mock.Anything).Return("http://example.com/registry/test-registry")
				mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything,
					mock.Anything).Return("http://example.com/registry/test-registry")

				// Setup base info mock.
				baseInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: testRegistryName,
					ParentRef:          "root",
					ParentID:           2,
					RootIdentifierID:   3,
					RootIdentifier:     "root",
					RegistryType:       api.RegistryTypeVIRTUAL,
					PackageType:        api.PackageTypeDOCKER,
				}
				// Create the registry entity that will be used in mocks.
				registry := &types.Registry{
					ID:           baseInfo.RegistryID,
					Name:         testRegistryName,
					ParentID:     baseInfo.ParentID,
					RootParentID: baseInfo.RootIdentifierID,
					Type:         api.RegistryTypeVIRTUAL,
					PackageType:  api.PackageTypeDOCKER,
				}

				// 1. Mock the initial registry metadata lookup.
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "root", "").Return(baseInfo, nil).Once()

				// 2. Mock the space lookup.
				mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil).Once()

				// 3. Mock the authorization check.
				mockAuthorizer.On("Check", mock.Anything, mock.Anything,
					mock.MatchedBy(func(scope *coretypes.Scope) bool { return scope.SpacePath == "root" }),
					mock.MatchedBy(func(r *coretypes.Resource) bool { return r.Type == gitnessenum.ResourceTypeRegistry }),
					gitnessenum.PermissionRegistryEdit).Return(true, nil).Once()

				// 4. Mock registry creation
				mockRegistryRepo.On("Create", mock.Anything,
					mock.MatchedBy(func(r *types.Registry) bool {
						return r.Name == testRegistryName &&
							r.ParentID == baseInfo.ParentID &&
							r.Type == api.RegistryTypeVIRTUAL &&
							r.PackageType == api.PackageTypeDOCKER
					})).Return(baseInfo.RegistryID, nil).Once()

				// 5. Mock registry retrieval.
				mockRegistryRepo.On("Get", mock.Anything, baseInfo.RegistryID).Return(registry, nil).Once()

				// 6. Mock cleanup policy retrieval.
				mockCleanupPolicyRepo.On("GetByRegistryID", mock.Anything,
					baseInfo.RegistryID).Return(&[]types.CleanupPolicy{}, nil).Once()

				// URL provider mock expectations set up above

				// Mock setup already done above.

				// Setup already covered above.

				fileManager := filemanager.NewFileManager(
					mockRegistryRepo,
					mockGenericBlobRepo,
					nil, // nodesRepo - not needed for this test.
					mockTransactor,
					nil, // reporter - not needed for this test.
					nil,
					nil,
				)

				// Setup audit service mock.
				mockAuditService.On("Log", mock.Anything,
					mock.MatchedBy(func(p coretypes.Principal) bool { return p.ID == 1 && p.Type == "user" }),
					mock.MatchedBy(func(r audit.Resource) bool {
						return r.Type == audit.ResourceTypeRegistry && r.Identifier == testRegistryName
					}),
					audit.ActionCreated,
					"root",
					mock.Anything,
				).Return(nil).Once()

				// Setup registry repo mock.
				mockRegistryRepo.On("FetchUpstreamProxyKeys", mock.Anything, mock.Anything).Return([]string{}, nil).Once()
				mockCleanupPolicyRepo.On("GetByRegistryID", mock.Anything,
					baseInfo.RegistryID).Return(&[]types.CleanupPolicy{}, nil).Once()

				// Authorizer mock already setup above.

				// Create controller.
				// Setup transactor mock.
				mockTransactor.On("WithTx", mock.Anything,
					mock.AnythingOfType("func(context.Context) error"), mock.Anything).Run(func(args mock.Arguments) {
					// Execute the transaction function.
					txFn, ok := args.Get(1).(func(context.Context) error)
					assert.True(t, ok, "Transaction function conversion failed")
					err := txFn(context.Background())
					// Check if an error occurs during transaction execution.
					assert.NoError(t, err, "Transaction function should not return an error")
				}).Return(nil)

				// Create controller.
				return metadata.NewAPIController(
					mockRegistryRepo,
					fileManager,
					nil, // blobStore.
					nil, // genericBlobStore.
					nil, // upstreamProxyStore.
					nil, // tagStore.
					nil, // manifestStore.
					mockCleanupPolicyRepo,
					nil, // imageStore.
					nil, // driver.
					mockSpaceFinder,
					mockTransactor,
					mockURLProvider,
					mockAuthorizer,
					mockAuditService,
					nil, // artifactStore.
					nil, // webhooksRepository.
					nil, // webhooksExecutionRepository.
					mockRegistryMetadataHelper,
					nil, // webhookService.
					eventReporter,
					nil, // downloadStatRepository.
					"",
					nil, // registryBlobStore - not needed for this test.
					nil, // PostProcessingReporter - not needed for this test.
					nil,
				)
			},
		},
		{
			name: "create_registry_invalid_parent",
			request: api.CreateRegistryRequestObject{
				Body: &api.CreateRegistryJSONRequestBody{
					Identifier: testRegistryName,
					ParentRef:  utils.StringPtr("invalid"),
					Config: func() *api.RegistryConfig {
						config := &api.RegistryConfig{Type: api.RegistryTypeVIRTUAL}
						_ = config.FromVirtualConfig(api.VirtualConfig{UpstreamProxies: &[]string{}})
						return config
					}(),
					PackageType:   api.PackageTypeDOCKER,
					CleanupPolicy: &[]api.CleanupPolicy{},
				},
			},
			expectedResp: api.CreateRegistry400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{
					Code:    "400",
					Message: "space not found",
				},
			},
			setupMocks: func() *metadata.APIController {
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockRegistryRepo := new(mocks.RegistryRepository)
				mockTransactor := new(mocks.Transactor)
				mockGenericBlobRepo := new(mocks.GenericBlobRepository)

				// Setup error case mock.
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "invalid", "").
					Return(nil, fmt.Errorf("space not found")).Once()

				fileManager := filemanager.NewFileManager(
					mockRegistryRepo,
					mockGenericBlobRepo,
					nil, // nodesRepo - not needed for this test.
					mockTransactor,
					nil, // reporter - not needed for this test.
					nil,
					nil,
				)

				return metadata.NewAPIController(
					mockRegistryRepo,
					fileManager,
					nil, // blobStore.
					nil, // genericBlobStore.
					nil, // upstreamProxyStore.
					nil, // tagStore.
					nil, // manifestStore.
					nil, // cleanupPolicyStore
					nil, // imageStore.
					nil, // driver.
					nil, // spaceFinder
					mockTransactor,
					nil, // urlProvider.
					nil, // authorizer.
					nil, // auditService.
					nil, // artifactStore.
					nil, // webhooksRepository.
					nil, // webhooksExecutionRepository.
					mockRegistryMetadataHelper,
					nil, // webhookService.
					eventReporter,
					nil, //
					"",  // downloadStatRepository.
					nil,
					nil, // PostProcessingReporter - not needed for this test.
					nil,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the controller.
			controller := tt.setupMocks()

			// Create context with auth session.
			ctx := context.Background()
			session := &auth.Session{
				Principal: coretypes.Principal{
					ID:   1,
					Type: "user",
				},
			}
			ctx = request.WithAuthSession(ctx, session)

			// Call the API.
			registryResp, err := controller.CreateRegistry(ctx, tt.request)

			// Verify response matches expected type and content.
			switch expected := tt.expectedResp.(type) {
			case api.CreateRegistry201JSONResponse:
				assert.NoError(t, err, "Expected no error but got one")
				actualResp, ok := registryResp.(api.CreateRegistry201JSONResponse)
				assert.True(t, ok, "Expected 201 response")
				assert.Equal(t, expected.Status, actualResp.Status, "Response status should match")

				// Verify registry data fields individually.
				assert.Equal(t, expected.Data.Identifier, actualResp.Data.Identifier, "Registry identifier should match")
				assert.Equal(t, expected.Data.PackageType, actualResp.Data.PackageType, "Package type should match")
				assert.Equal(t, expected.Data.Url, actualResp.Data.Url, "Registry URL should match")
				assert.Equal(t, expected.Data.Description, actualResp.Data.Description, "Description should match")
				assert.Equal(t, expected.Data.CreatedAt, actualResp.Data.CreatedAt, "CreatedAt should match")
				assert.Equal(t, expected.Data.ModifiedAt, actualResp.Data.ModifiedAt, "ModifiedAt should match")

				// Verify config type.
				assert.NotNil(t, actualResp.Data.Config, "Config should not be nil")
				assert.Equal(t, expected.Data.Config.Type, actualResp.Data.Config.Type, "Config type should match")

				// Verify mock expectations for success case.
				called := controller.RegistryRepository.(*mocks.RegistryRepository).AssertCalled(t, "Create", //nolint:errcheck
					mock.Anything, mock.MatchedBy(func(r *types.Registry) bool {
						return r.Name == tt.request.Body.Identifier &&
							r.Type == tt.request.Body.Config.Type &&
							r.PackageType == tt.request.Body.PackageType
					}))
				assert.True(t, called, "Expected Create call not made")

				// Verify all mocks expectations were met.
				mockRegistry, regOk := controller.RegistryRepository.(*mocks.RegistryRepository)
				assert.True(t, regOk, "Type assertion to RegistryRepository failed")
				regAssertOk := mockRegistry.AssertExpectations(t)
				assert.True(t, regAssertOk, "Mock expectations failed for RegistryRepository")

				mockAudit, auditOk := controller.AuditService.(*mocks.AuditService)
				assert.True(t, auditOk, "Type assertion to AuditService failed")
				auditAssertOk := mockAudit.AssertExpectations(t)
				assert.True(t, auditAssertOk, "Mock expectations failed for AuditService")

				// Verify audit expectations.
				logCalled := controller.AuditService.(*mocks.AuditService).AssertCalled(t, "Log", mock.Anything, //nolint:errcheck
					mock.Anything, mock.Anything, audit.ActionCreated, mock.Anything, mock.Anything)
				assert.True(t, logCalled, "Expected Log call not made")

			case api.CreateRegistry400JSONResponse:
				assert.Error(t, err, "Expected an error")
				actualResp, ok := registryResp.(api.CreateRegistry400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
				assert.Equal(t, expected.Code, actualResp.Code, "Error code should match")
				assert.Equal(t, expected.Message, actualResp.Message, "Error message should match")

				// Additional assertions for specific error cases.
				switch tt.name {
				case "create_registry_invalid_parent":
					assert.Contains(t, actualResp.Message, "space not found",
						"Error message should indicate invalid parent")
					notCalled := controller.RegistryRepository.(*mocks.RegistryRepository).AssertNotCalled(t, //nolint:errcheck
						"Create", mock.Anything, mock.Anything)
					assert.True(t, notCalled, "Unexpected Create call made")

					// Verify expectations for the error case.
					metaHelper, metaHelperOk := controller.RegistryMetadataHelper.(*mocks.RegistryMetadataHelper)
					assert.True(t, metaHelperOk, "Type assertion to RegistryMetadataHelper failed")
					assertMetaOk := metaHelper.AssertExpectations(t)
					assert.True(t, assertMetaOk, "Mock expectations failed for RegistryMetadataHelper")
				case "create_registry_duplicate":
					assert.Contains(t, actualResp.Message, "already defined",
						"Error message should indicate duplicate registry")
					assert.Contains(t, actualResp.Message, tt.request.Body.Identifier,
						"Error message should include the registry identifier")
					getByNameCalled := controller.RegistryRepository.(*mocks.RegistryRepository).AssertCalled(t, //nolint:errcheck
						"GetByRootParentIDAndName", mock.Anything, mock.Anything, tt.request.Body.Identifier)
					assert.True(t, getByNameCalled, "Expected GetByRootParentIDAndName call not made")
				}

			case api.CreateRegistry403JSONResponse:
				assert.Error(t, err, "Expected an error")
				actualResp, ok := registryResp.(api.CreateRegistry403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
				assert.Equal(t, expected.Code, actualResp.Code, "Error code should match")
				assert.Equal(t, expected.Message, actualResp.Message, "Error message should match")
				assert.Contains(t, actualResp.Message, "unauthorized",
					"Error message should indicate authorization failure")
				_ = controller.RegistryRepository.(*mocks.RegistryRepository).AssertNotCalled(t, //nolint:errcheck
					"Create", mock.Anything, mock.Anything)

			default:
				t.Fatalf("Unexpected response type: %T", tt.expectedResp)
			}

			// Verify common mock expectations.
			if controller.RegistryRepository != nil {
				registryRepo, regOk := controller.RegistryRepository.(*mocks.RegistryRepository)
				assert.True(t, regOk, "Type assertion to RegistryRepository failed")
				assertRegOk := registryRepo.AssertExpectations(t)
				assert.True(t, assertRegOk, "Mock expectations failed for RegistryRepository")
			}
			if controller.RegistryMetadataHelper != nil {
				metaHelper, metaHelperOk := controller.RegistryMetadataHelper.(*mocks.RegistryMetadataHelper)
				assert.True(t, metaHelperOk, "Type assertion to RegistryMetadataHelper failed")
				assertMetaOk := metaHelper.AssertExpectations(t)
				assert.True(t, assertMetaOk, "Mock expectations failed for RegistryMetadataHelper")
			}
			if controller.SpaceFinder != nil {
				spaceFinder, finderOk := controller.SpaceFinder.(*mocks.SpaceFinder)
				assert.True(t, finderOk, "Type assertion to SpaceFinder failed")
				spaceFinderOk := spaceFinder.AssertExpectations(t)
				assert.True(t, spaceFinderOk, "Mock expectations failed for SpaceFinder")
			}
			if controller.Authorizer != nil {
				auth, authOk := controller.Authorizer.(*mocks.Authorizer)
				assert.True(t, authOk, "Type assertion to Authorizer failed")
				authAssertOk := auth.AssertExpectations(t)
				assert.True(t, authAssertOk, "Mock expectations failed for Authorizer")
			}
			if controller.AuditService != nil {
				auditSvc, auditSvcOk := controller.AuditService.(*mocks.AuditService)
				assert.True(t, auditSvcOk, "Type assertion to AuditService failed")
				auditSvcAssertOk := auditSvc.AssertExpectations(t)
				assert.True(t, auditSvcAssertOk, "Mock expectations failed for AuditService")
			}
		})
	}
}

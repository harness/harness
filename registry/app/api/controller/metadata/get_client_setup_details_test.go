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

package metadata_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetClientSetupDetails(t *testing.T) {
	artifactParam := artifact.ArtifactParam("test-artifact")
	versionParam := artifact.VersionParam("v1.0.0")

	tests := []struct {
		name         string
		registryRef  string
		packageType  artifact.PackageType
		setupMocks   func() *metadata.APIController
		expectedResp any
		expectError  bool
	}{
		{
			name:        "docker_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeDOCKER, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "docker_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeDOCKER, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "helm_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeHELM,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeHELM, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "helm_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeHELM,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeHELM, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "maven_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeMAVEN, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "maven_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeMAVEN, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "generic_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeGENERIC,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeGENERIC, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "generic_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeGENERIC,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeGENERIC, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "python_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypePYTHON,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypePYTHON, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "python_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypePYTHON,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypePYTHON, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "npm_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeNPM,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeNPM, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "npm_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeNPM,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeNPM, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "nuget_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeNUGET,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeNUGET, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "nuget_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeNUGET,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeNUGET, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "go_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeGO,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeGO, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "go_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeGO,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeGO, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "rpm_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeRPM,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeRPM, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "rpm_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeRPM,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeRPM, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "huggingface_registry_authenticated",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeHUGGINGFACE,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeHUGGINGFACE, false)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "huggingface_registry_anonymous",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeHUGGINGFACE,
			setupMocks: func() *metadata.APIController {
				return setupControllerForPackageType(t, artifact.PackageTypeHUGGINGFACE, true)
			},
			expectedResp: artifact.GetClientSetupDetails200JSONResponse{},
			expectError:  false,
		},
		{
			name:        "registry_not_found",
			registryRef: "non-existent",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerForError(t, "registry_not_found")
			},
			expectedResp: artifact.GetClientSetupDetails404JSONResponse{},
			expectError:  true,
		},
		{
			name:        "space_not_found",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerForError(t, "space_not_found")
			},
			expectedResp: artifact.GetClientSetupDetails400JSONResponse{},
			expectError:  false,
		},
		{
			name:        "unauthorized_access",
			registryRef: "test-registry",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerForError(t, "unauthorized")
			},
			expectedResp: artifact.GetClientSetupDetails403JSONResponse{},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := tt.setupMocks()

			// Create context with appropriate auth session
			ctx := context.Background()
			if tt.name == "docker_registry_anonymous" || tt.name == "helm_registry_anonymous" ||
				tt.name == "maven_registry_anonymous" || tt.name == "generic_registry_anonymous" ||
				tt.name == "python_registry_anonymous" || tt.name == "npm_registry_anonymous" ||
				tt.name == "nuget_registry_anonymous" || tt.name == "go_registry_anonymous" ||
				tt.name == "rpm_registry_anonymous" || tt.name == "huggingface_registry_anonymous" {
				// Anonymous session
				session := &auth.Session{
					Principal: coretypes.Principal{
						ID:   0,
						Type: enum.PrincipalTypeUser,
					},
				}
				ctx = request.WithAuthSession(ctx, session)
			} else {
				// Authenticated session
				session := &auth.Session{
					Principal: coretypes.Principal{
						ID:    1,
						UID:   "test-user",
						Email: "test@example.com",
						Type:  enum.PrincipalTypeUser,
					},
				}
				ctx = request.WithAuthSession(ctx, session)
			}

			// Create request object
			req := artifact.GetClientSetupDetailsRequestObject{
				RegistryRef: artifact.RegistryRefPathParam(tt.registryRef),
				Params: artifact.GetClientSetupDetailsParams{
					Artifact: &artifactParam,
					Version:  &versionParam,
				},
			}

			// Call the API
			resp, err := controller.GetClientSetupDetails(ctx, req)

			// Verify response
			switch tt.expectedResp.(type) {
			case artifact.GetClientSetupDetails200JSONResponse:
				assert.NoError(t, err)
				actualResp, ok := resp.(artifact.GetClientSetupDetails200JSONResponse)
				assert.True(t, ok, "Expected 200 response")
				assert.Equal(t, artifact.StatusSUCCESS, actualResp.Status)
				assert.NotNil(t, actualResp.Data)
				assert.NotEmpty(t, actualResp.Data.MainHeader)
				assert.NotEmpty(t, actualResp.Data.SecHeader)
				assert.NotEmpty(t, actualResp.Data.Sections)

			case artifact.GetClientSetupDetails400JSONResponse:
				assert.NoError(t, err)
				actualResp, ok := resp.(artifact.GetClientSetupDetails400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
				assert.NotEmpty(t, actualResp.Message)

			case artifact.GetClientSetupDetails403JSONResponse:
				assert.NoError(t, err)
				actualResp, ok := resp.(artifact.GetClientSetupDetails403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
				assert.NotEmpty(t, actualResp.Message)

			case artifact.GetClientSetupDetails404JSONResponse:
				assert.Error(t, err)
				actualResp, ok := resp.(artifact.GetClientSetupDetails404JSONResponse)
				assert.True(t, ok, "Expected 404 response")
				assert.NotEmpty(t, actualResp.Message)
			}
		})
	}
}

func TestGenerateClientSetupDetails(t *testing.T) {
	artifactParam := artifact.ArtifactParam("test-artifact")
	versionParam := artifact.VersionParam("v1.0.0")

	tests := []struct {
		name         string
		packageType  string
		registryType artifact.RegistryType
		expectNil    bool
	}{
		{
			name:         "docker_virtual",
			packageType:  string(artifact.PackageTypeDOCKER),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "docker_upstream",
			packageType:  string(artifact.PackageTypeDOCKER),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
		{
			name:         "helm_virtual",
			packageType:  string(artifact.PackageTypeHELM),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "helm_upstream",
			packageType:  string(artifact.PackageTypeHELM),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
		{
			name:         "maven_virtual",
			packageType:  string(artifact.PackageTypeMAVEN),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "generic_virtual",
			packageType:  string(artifact.PackageTypeGENERIC),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "generic_upstream",
			packageType:  string(artifact.PackageTypeGENERIC),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
		{
			name:         "python_virtual",
			packageType:  string(artifact.PackageTypePYTHON),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "python_upstream",
			packageType:  string(artifact.PackageTypePYTHON),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
		{
			name:         "npm_virtual",
			packageType:  string(artifact.PackageTypeNPM),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "npm_upstream",
			packageType:  string(artifact.PackageTypeNPM),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
		{
			name:         "nuget_virtual",
			packageType:  string(artifact.PackageTypeNUGET),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "go_virtual",
			packageType:  string(artifact.PackageTypeGO),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "go_upstream",
			packageType:  string(artifact.PackageTypeGO),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
		{
			name:         "rpm_virtual",
			packageType:  string(artifact.PackageTypeRPM),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "huggingface_virtual",
			packageType:  string(artifact.PackageTypeHUGGINGFACE),
			registryType: artifact.RegistryTypeVIRTUAL,
			expectNil:    false,
		},
		{
			name:         "huggingface_upstream",
			packageType:  string(artifact.PackageTypeHUGGINGFACE),
			registryType: artifact.RegistryTypeUPSTREAM,
			expectNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := setupControllerForPackageType(t, artifact.PackageType(tt.packageType), false)

			ctx := context.Background()
			session := &auth.Session{
				Principal: coretypes.Principal{
					ID:    1,
					Email: "test@example.com",
					Type:  enum.PrincipalTypeUser,
				},
			}
			ctx = request.WithAuthSession(ctx, session)

			result := controller.GenerateClientSetupDetails(
				ctx,
				tt.packageType,
				&artifactParam,
				&versionParam,
				"root/test-registry",
				tt.registryType,
			)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, artifact.StatusSUCCESS, result.Status)
				assert.NotNil(t, result.Data)
				assert.NotEmpty(t, result.Data.MainHeader)
				assert.NotEmpty(t, result.Data.SecHeader)
				assert.NotEmpty(t, result.Data.Sections)
			}
		})
	}
}

func TestGenerateClientSetupDetails_WithUntaggedImages(t *testing.T) {
	artifactParam := artifact.ArtifactParam("test-artifact")
	versionParam := artifact.VersionParam("sha256:abc123")

	tests := []struct {
		name                  string
		packageType           string
		untaggedImagesEnabled bool
	}{
		{
			name:                  "docker_with_untagged_enabled",
			packageType:           string(artifact.PackageTypeDOCKER),
			untaggedImagesEnabled: true,
		},
		{
			name:                  "docker_with_untagged_disabled",
			packageType:           string(artifact.PackageTypeDOCKER),
			untaggedImagesEnabled: false,
		},
		{
			name:                  "helm_with_untagged_enabled",
			packageType:           string(artifact.PackageTypeHELM),
			untaggedImagesEnabled: true,
		},
		{
			name:                  "helm_with_untagged_disabled",
			packageType:           string(artifact.PackageTypeHELM),
			untaggedImagesEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockURLProvider := new(mocks.Provider)
			mockURLProvider.On("PackageURL", mock.Anything, mock.Anything, mock.Anything).
				Return("http://example.com/registry/test-registry")
			mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything).
				Return("http://example.com")

			fileManager := createFileManager()
			eventReporter := createEventReporter()

			controller := metadata.NewAPIController(
				nil,             // repositoryStore
				fileManager,     // fileManager
				nil,             // blobStore
				nil,             // genericBlobStore
				nil,             // upstreamProxyStore
				nil,             // tagStore
				nil,             // manifestStore
				nil,             // cleanupPolicyStore
				nil,             // imageStore
				nil,             // spaceFinder
				nil,             // tx
				mockURLProvider, // urlProvider
				nil,             // authorizer
				nil,             // auditService
				nil,             // artifactStore
				nil,             // webhooksRepository
				nil,             // webhooksExecutionRepository
				nil,             // registryMetadataHelper
				nil,             // webhookService
				eventReporter,   // artifactEventReporter
				nil,             // downloadStatRepository
				"",              // setupDetailsAuthHeaderPrefix
				nil,             // registryBlobStore
				nil,             // regFinder
				nil,             // postProcessingReporter
				nil,             // cargoRegistryHelper
				nil,             // spaceController
				nil,             // quarantineArtifactRepository
				nil,             // quarantineFinder
				nil,             // spaceStore
				func(_ context.Context) bool { return tt.untaggedImagesEnabled }, // untaggedImagesEnabled
				nil, // packageWrapper
				nil, // publicAccess
				nil, // storageService
			)

			ctx := context.Background()
			session := &auth.Session{
				Principal: coretypes.Principal{
					ID:    1,
					Email: "test@example.com",
					Type:  enum.PrincipalTypeUser,
				},
			}
			ctx = request.WithAuthSession(ctx, session)

			result := controller.GenerateClientSetupDetails(
				ctx,
				tt.packageType,
				&artifactParam,
				&versionParam,
				"root/test-registry",
				artifact.RegistryTypeVIRTUAL,
			)

			assert.NotNil(t, result)
			assert.Equal(t, artifact.StatusSUCCESS, result.Status)
		})
	}
}

func TestGenerateClientSetupDetails_MavenWithGroupID(t *testing.T) {
	artifactParam := artifact.ArtifactParam("com.example:test-artifact")
	versionParam := artifact.VersionParam("v1.0.0")

	mockURLProvider := new(mocks.Provider)
	mockURLProvider.On("PackageURL", mock.Anything, mock.Anything, mock.Anything).
		Return("http://example.com/registry/test-registry/maven")
	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything).
		Return("http://example.com")

	fileManager := createFileManager()
	eventReporter := createEventReporter()

	controller := metadata.NewAPIController(
		nil,             // repositoryStore
		fileManager,     // fileManager
		nil,             // blobStore
		nil,             // genericBlobStore
		nil,             // upstreamProxyStore
		nil,             // tagStore
		nil,             // manifestStore
		nil,             // cleanupPolicyStore
		nil,             // imageStore
		nil,             // spaceFinder
		nil,             // tx
		mockURLProvider, // urlProvider
		nil,             // authorizer
		nil,             // auditService
		nil,             // artifactStore
		nil,             // webhooksRepository
		nil,             // webhooksExecutionRepository
		nil,             // registryMetadataHelper
		nil,             // webhookService
		eventReporter,   // artifactEventReporter
		nil,             // downloadStatRepository
		"",              // setupDetailsAuthHeaderPrefix
		nil,             // registryBlobStore
		nil,             // regFinder
		nil,             // postProcessingReporter
		nil,             // cargoRegistryHelper
		nil,             // spaceController
		nil,             // quarantineArtifactRepository
		nil,             // quarantineFinder
		nil,             // spaceStore
		func(_ context.Context) bool { return false }, // untaggedImagesEnabled
		nil, // packageWrapper
		nil, // publicAccess
		nil, // storageService
	)

	ctx := context.Background()
	session := &auth.Session{
		Principal: coretypes.Principal{
			ID:    1,
			Email: "test@example.com",
			Type:  enum.PrincipalTypeUser,
		},
	}
	ctx = request.WithAuthSession(ctx, session)

	result := controller.GenerateClientSetupDetails(
		ctx,
		string(artifact.PackageTypeMAVEN),
		&artifactParam,
		&versionParam,
		"root/test-registry",
		artifact.RegistryTypeVIRTUAL,
	)

	assert.NotNil(t, result)
	assert.Equal(t, artifact.StatusSUCCESS, result.Status)
}

// Helper functions

func createFileManager() filemanager.FileManager {
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockGenericBlobRepo := new(mocks.GenericBlobRepository)
	mockTransactor := new(mocks.Transactor)

	return filemanager.NewFileManager(
		mockRegistryRepo,
		mockGenericBlobRepo,
		nil, // nodesRepo
		mockTransactor,
		nil, // config
		nil, // storageService
		nil, // bucketService
		nil, // replicationReporter
		nil, // blobCreationDBHook
	)
}

func createEventReporter() registryevents.Reporter {
	// Create a no-op reporter for testing
	return registryevents.Reporter{}
}

func setupControllerForPackageType(_ *testing.T, packageType artifact.PackageType, _ bool) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)
	mockURLProvider := new(mocks.Provider)

	space := &coretypes.SpaceCore{
		ID:   1,
		Path: "root",
	}

	baseInfo := &types.RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "test-registry",
		ParentRef:          "root",
		ParentID:           2,
		RootIdentifierID:   3,
		RootIdentifier:     "root",
		RegistryType:       artifact.RegistryTypeVIRTUAL,
		PackageType:        packageType,
		RegistryRef:        "root/test-registry",
	}

	registry := &types.Registry{
		ID:           1,
		Name:         "test-registry",
		ParentID:     2,
		RootParentID: 3,
		Type:         artifact.RegistryTypeVIRTUAL,
		PackageType:  packageType,
	}

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "test-registry").
		Return(baseInfo, nil)
	mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
		Return([]coretypes.PermissionCheck{})
	mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)
	mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "test-registry").
		Return(registry, nil)

	mockURLProvider.On("PackageURL", mock.Anything, mock.Anything, mock.Anything).
		Return("http://example.com/registry/test-registry/" + string(packageType))
	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything).
		Return("http://example.com")

	fileManager := createFileManager()
	eventReporter := createEventReporter()

	return metadata.NewAPIController(
		mockRegistryRepo,           // repositoryStore
		fileManager,                // fileManager
		nil,                        // blobStore
		nil,                        // genericBlobStore
		nil,                        // upstreamProxyStore
		nil,                        // tagStore
		nil,                        // manifestStore
		nil,                        // cleanupPolicyStore
		nil,                        // imageStore
		mockSpaceFinder,            // spaceFinder
		nil,                        // tx
		mockURLProvider,            // urlProvider
		mockAuthorizer,             // authorizer
		nil,                        // auditService
		nil,                        // artifactStore
		nil,                        // webhooksRepository
		nil,                        // webhooksExecutionRepository
		mockRegistryMetadataHelper, // registryMetadataHelper
		nil,                        // webhookService
		eventReporter,              // artifactEventReporter
		nil,                        // downloadStatRepository
		"Authorization: Bearer",    // setupDetailsAuthHeaderPrefix
		nil,                        // registryBlobStore
		nil,                        // regFinder
		nil,                        // postProcessingReporter
		nil,                        // cargoRegistryHelper
		nil,                        // spaceController
		nil,                        // quarantineArtifactRepository
		nil,                        // quarantineFinder
		nil,                        // spaceStore
		func(_ context.Context) bool { return false }, // untaggedImagesEnabled
		nil, // packageWrapper
		nil, // publicAccess
		nil, // storageService
	)
}

func setupControllerForError(_ *testing.T, errorType string) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)

	space := &coretypes.SpaceCore{
		ID:   1,
		Path: "root",
	}

	baseInfo := &types.RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "test-registry",
		ParentRef:          "root",
		ParentID:           2,
		RootIdentifierID:   3,
		RootIdentifier:     "root",
		RegistryType:       artifact.RegistryTypeVIRTUAL,
		PackageType:        artifact.PackageTypeDOCKER,
	}

	switch errorType {
	case errorTypeSpaceNotFound:
		mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "test-registry").
			Return(baseInfo, nil)
		mockSpaceFinder.On("FindByRef", mock.Anything, "root").
			Return(nil, http.ErrNotSupported)

	case errorTypeUnauthorized:
		mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "test-registry").
			Return(baseInfo, nil)
		mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
		mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
			Return([]coretypes.PermissionCheck{})
		mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
			Return(false, http.ErrNotSupported)

	case "registry_not_found":
		mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "non-existent").
			Return(baseInfo, nil)
		mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
		mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
			Return([]coretypes.PermissionCheck{})
		mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
			Return(true, nil)
		mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "test-registry").
			Return(nil, http.ErrNotSupported)
	}

	fileManager := createFileManager()
	eventReporter := createEventReporter()

	return metadata.NewAPIController(
		mockRegistryRepo,           // repositoryStore
		fileManager,                // fileManager
		nil,                        // blobStore
		nil,                        // genericBlobStore
		nil,                        // upstreamProxyStore
		nil,                        // tagStore
		nil,                        // manifestStore
		nil,                        // cleanupPolicyStore
		nil,                        // imageStore
		mockSpaceFinder,            // spaceFinder
		nil,                        // tx
		nil,                        // urlProvider
		mockAuthorizer,             // authorizer
		nil,                        // auditService
		nil,                        // artifactStore
		nil,                        // webhooksRepository
		nil,                        // webhooksExecutionRepository
		mockRegistryMetadataHelper, // registryMetadataHelper
		nil,                        // webhookService
		eventReporter,              // artifactEventReporter
		nil,                        // downloadStatRepository
		"",                         // setupDetailsAuthHeaderPrefix
		nil,                        // registryBlobStore
		nil,                        // regFinder
		nil,                        // postProcessingReporter
		nil,                        // cargoRegistryHelper
		nil,                        // spaceController
		nil,                        // quarantineArtifactRepository
		nil,                        // quarantineFinder
		nil,                        // spaceStore
		func(_ context.Context) bool { return false }, // untaggedImagesEnabled
		nil, // packageWrapper
		nil, // publicAccess
		nil, // storageService
	)
}

// TestGenerateClientSetupDetailsSnapshot tests that the generated client setup details
// match the expected snapshots for all package types. This ensures consistency across runs.
func TestGenerateClientSetupDetailsSnapshot(t *testing.T) {
	artifactParam := artifact.ArtifactParam("test-artifact")
	versionParam := artifact.VersionParam("v1.0.0")

	tests := []struct {
		name         string
		packageType  string
		registryType artifact.RegistryType
		anonymous    bool
	}{
		{
			name: "docker_authenticated", packageType: string(artifact.PackageTypeDOCKER),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "docker_anonymous", packageType: string(artifact.PackageTypeDOCKER),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "helm_authenticated", packageType: string(artifact.PackageTypeHELM),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "helm_anonymous", packageType: string(artifact.PackageTypeHELM),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "maven_authenticated", packageType: string(artifact.PackageTypeMAVEN),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "maven_anonymous", packageType: string(artifact.PackageTypeMAVEN),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "generic_authenticated", packageType: string(artifact.PackageTypeGENERIC),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "generic_anonymous", packageType: string(artifact.PackageTypeGENERIC),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "python_authenticated", packageType: string(artifact.PackageTypePYTHON),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "python_anonymous", packageType: string(artifact.PackageTypePYTHON),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "npm_authenticated", packageType: string(artifact.PackageTypeNPM),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "npm_anonymous", packageType: string(artifact.PackageTypeNPM),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "nuget_authenticated", packageType: string(artifact.PackageTypeNUGET),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "nuget_anonymous", packageType: string(artifact.PackageTypeNUGET),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "go_authenticated", packageType: string(artifact.PackageTypeGO),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "go_anonymous", packageType: string(artifact.PackageTypeGO),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "rpm_authenticated", packageType: string(artifact.PackageTypeRPM),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "rpm_anonymous", packageType: string(artifact.PackageTypeRPM),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "huggingface_authenticated", packageType: string(artifact.PackageTypeHUGGINGFACE),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: false,
		},
		{
			name: "huggingface_anonymous", packageType: string(artifact.PackageTypeHUGGINGFACE),
			registryType: artifact.RegistryTypeVIRTUAL, anonymous: true,
		},
		{
			name: "docker_upstream", packageType: string(artifact.PackageTypeDOCKER),
			registryType: artifact.RegistryTypeUPSTREAM, anonymous: false,
		},
		{
			name: "helm_upstream", packageType: string(artifact.PackageTypeHELM),
			registryType: artifact.RegistryTypeUPSTREAM, anonymous: false,
		},
		{
			name: "generic_upstream", packageType: string(artifact.PackageTypeGENERIC),
			registryType: artifact.RegistryTypeUPSTREAM, anonymous: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockURLProvider := new(mocks.Provider)
			mockURLProvider.On("PackageURL", mock.Anything, mock.Anything, mock.Anything).
				Return("http://example.com/registry/test-registry/" + tt.packageType)
			mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything).
				Return("http://example.com")

			fileManager := createFileManager()
			eventReporter := createEventReporter()

			controller := metadata.NewAPIController(
				nil,                     // repositoryStore
				fileManager,             // fileManager
				nil,                     // blobStore
				nil,                     // genericBlobStore
				nil,                     // upstreamProxyStore
				nil,                     // tagStore
				nil,                     // manifestStore
				nil,                     // cleanupPolicyStore
				nil,                     // imageStore
				nil,                     // spaceFinder
				nil,                     // tx
				mockURLProvider,         // urlProvider
				nil,                     // authorizer
				nil,                     // auditService
				nil,                     // artifactStore
				nil,                     // webhooksRepository
				nil,                     // webhooksExecutionRepository
				nil,                     // registryMetadataHelper
				nil,                     // webhookService
				eventReporter,           // artifactEventReporter
				nil,                     // downloadStatRepository
				"Authorization: Bearer", // setupDetailsAuthHeaderPrefix
				nil,                     // registryBlobStore
				nil,                     // regFinder
				nil,                     // postProcessingReporter
				nil,                     // cargoRegistryHelper
				nil,                     // spaceController
				nil,                     // quarantineArtifactRepository
				nil,                     // quarantineFinder
				nil,                     // spaceStore
				func(_ context.Context) bool { return false }, // untaggedImagesEnabled
				nil, // packageWrapper
				nil, // publicAccess
				nil, // storageService
			)

			ctx := context.Background()
			session := &auth.Session{
				Principal: coretypes.Principal{
					ID:   0,
					UID:  coretypes.AnonymousPrincipalUID,
					Type: enum.PrincipalTypeUser,
				},
			}
			if tt.anonymous {
				ctx = request.WithAuthSession(ctx, session)
			} else {
				authSession := &auth.Session{
					Principal: coretypes.Principal{
						ID:    1,
						UID:   "test-user",
						Email: "test@example.com",
						Type:  enum.PrincipalTypeUser,
					},
				}
				ctx = request.WithAuthSession(ctx, authSession)
			}

			result := controller.GenerateClientSetupDetails(
				ctx,
				tt.packageType,
				&artifactParam,
				&versionParam,
				"root/test-registry",
				tt.registryType,
			)

			require.NotNil(t, result)
			require.Equal(t, artifact.StatusSUCCESS, result.Status)

			// Verify snapshot
			verifySnapshot(t, tt.name, result.Data)
		})
	}
}

// verifySnapshot compares the actual data with a stored snapshot.
func verifySnapshot(t *testing.T, name string, actual artifact.ClientSetupDetails) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "snapshots", "client_setup_details")
	snapshotFile := filepath.Join(snapshotDir, name+".json")

	// Marshal actual data to JSON
	actualJSON, err := json.MarshalIndent(actual, "", "  ")
	require.NoError(t, err, "Failed to marshal actual data")

	// Check if UPDATE_SNAPSHOTS environment variable is set
	if os.Getenv(envUpdateSnapshots) == envValueTrue {
		// Create directory if it doesn't exist
		err := os.MkdirAll(snapshotDir, 0755)
		require.NoError(t, err, "Failed to create snapshot directory")

		// Write snapshot
		err = os.WriteFile(snapshotFile, actualJSON, snapshotFilePermissions)
		require.NoError(t, err, "Failed to write snapshot file")
		t.Logf("Updated snapshot: %s", snapshotFile)
		return
	}

	// Read existing snapshot
	expectedJSON, err := os.ReadFile(snapshotFile)
	if os.IsNotExist(err) {
		t.Fatalf("Snapshot file does not exist: %s\nRun tests with UPDATE_SNAPSHOTS=true to create it", snapshotFile)
	}
	require.NoError(t, err, "Failed to read snapshot file")

	// Compare JSON
	var expected, actualParsed any
	require.NoError(t, json.Unmarshal(expectedJSON, &expected), "Failed to unmarshal expected JSON")
	require.NoError(t, json.Unmarshal(actualJSON, &actualParsed), "Failed to unmarshal actual JSON")

	assert.Equal(t, expected, actualParsed,
		"Snapshot mismatch for %s.\nExpected:\n%s\n\nActual:\n%s\n\nRun tests with UPDATE_SNAPSHOTS=true to update snapshots",
		name, string(expectedJSON), string(actualJSON))
}

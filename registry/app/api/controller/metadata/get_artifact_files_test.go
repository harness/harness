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
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/registry/app/api/controller/metadata"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGetArtifactFiles tests the GetArtifactFiles endpoint for various package types.
//
// Note: This test is currently skipped because it requires a NodesRepository mock that doesn't exist yet.
// To enable this test:
// 1. Create a NodesRepository mock in registry/app/api/controller/mocks/
// 2. Mock the GetFilesMetadataByPathAndRegistryID and CountFilesByPath methods
// 3. Remove the t.Skip() call below
//
// Reference snapshots exist in testdata/snapshots/artifact_files/ for expected responses.
func TestGetArtifactFiles(t *testing.T) {
	t.Skip("Skipping - requires NodesRepository mock to be created. See comment above for details.")
	tests := []struct {
		name         string
		packageType  artifact.PackageType
		setupMocks   func() *metadata.APIController
		expectedResp any
	}{
		{
			name:        "maven_files_success",
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeMAVEN)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "generic_files_success",
			packageType: artifact.PackageTypeGENERIC,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeGENERIC)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "python_files_success",
			packageType: artifact.PackageTypePYTHON,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypePYTHON)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "npm_files_success",
			packageType: artifact.PackageTypeNPM,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeNPM)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "rpm_files_success",
			packageType: artifact.PackageTypeRPM,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeRPM)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "nuget_files_success",
			packageType: artifact.PackageTypeNUGET,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeNUGET)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "go_files_success",
			packageType: artifact.PackageTypeGO,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeGO)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        "huggingface_files_success",
			packageType: artifact.PackageTypeHUGGINGFACE,
			setupMocks: func() *metadata.APIController {
				return setupFilesController(t, artifact.PackageTypeHUGGINGFACE)
			},
			expectedResp: artifact.GetArtifactFiles200JSONResponse{},
		},
		{
			name:        errorTypeSpaceNotFound,
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupFilesControllerWithError(t, errorTypeSpaceNotFound)
			},
			expectedResp: artifact.GetArtifactFiles400JSONResponse{},
		},
		{
			name:        "unauthorized_access",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupFilesControllerWithError(t, errorTypeUnauthorized)
			},
			expectedResp: artifact.GetArtifactFiles403JSONResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := tt.setupMocks()

			ctx := context.Background()
			session := &auth.Session{
				Principal: coretypes.Principal{
					ID:    1,
					UID:   "test-user",
					Email: "test@example.com",
					Type:  enum.PrincipalTypeUser,
				},
			}
			ctx = request.WithAuthSession(ctx, session)

			req := artifact.GetArtifactFilesRequestObject{
				RegistryRef: artifact.RegistryRefPathParam("test-registry"),
				Artifact:    artifact.ArtifactPathParam("test-artifact"),
				Version:     artifact.VersionPathParam("v1.0.0"),
				Params:      artifact.GetArtifactFilesParams{},
			}

			resp, err := controller.GetArtifactFiles(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)

			// Verify response type matches expected
			switch tt.expectedResp.(type) {
			case artifact.GetArtifactFiles200JSONResponse:
				_, ok := resp.(artifact.GetArtifactFiles200JSONResponse)
				assert.True(t, ok, "Expected 200 response")
			case artifact.GetArtifactFiles400JSONResponse:
				_, ok := resp.(artifact.GetArtifactFiles400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
			case artifact.GetArtifactFiles403JSONResponse:
				_, ok := resp.(artifact.GetArtifactFiles403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
			case artifact.GetArtifactFiles500JSONResponse:
				_, ok := resp.(artifact.GetArtifactFiles500JSONResponse)
				assert.True(t, ok, "Expected 500 response")
			}
		})
	}
}

// TestGetArtifactFilesSnapshot tests that the generated artifact files responses
// match the expected snapshots for all package types. This ensures consistency across runs.
//
// Note: This test is currently skipped for the same reason as TestGetArtifactFiles.
// See TestGetArtifactFiles comment for details on how to enable.
func TestGetArtifactFilesSnapshot(t *testing.T) {
	t.Skip("Skipping - requires NodesRepository mock. See TestGetArtifactFiles comment for details.")
	tests := []struct {
		name        string
		packageType artifact.PackageType
	}{
		{name: "maven_files", packageType: artifact.PackageTypeMAVEN},
		{name: "generic_files", packageType: artifact.PackageTypeGENERIC},
		{name: "python_files", packageType: artifact.PackageTypePYTHON},
		{name: "npm_files", packageType: artifact.PackageTypeNPM},
		{name: "rpm_files", packageType: artifact.PackageTypeRPM},
		{name: "nuget_files", packageType: artifact.PackageTypeNUGET},
		{name: "go_files", packageType: artifact.PackageTypeGO},
		{name: "huggingface_files", packageType: artifact.PackageTypeHUGGINGFACE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := setupFilesSnapshotController(t, tt.packageType)

			ctx := context.Background()
			session := &auth.Session{
				Principal: coretypes.Principal{
					ID:    1,
					UID:   "test-user",
					Email: "test@example.com",
					Type:  enum.PrincipalTypeUser,
				},
			}
			ctx = request.WithAuthSession(ctx, session)

			req := artifact.GetArtifactFilesRequestObject{
				RegistryRef: artifact.RegistryRefPathParam("test-registry"),
				Artifact:    artifact.ArtifactPathParam("test-artifact"),
				Version:     artifact.VersionPathParam("v1.0.0"),
				Params:      artifact.GetArtifactFilesParams{},
			}

			resp, err := controller.GetArtifactFiles(ctx, req)
			require.NoError(t, err)

			actualResp, ok := resp.(artifact.GetArtifactFiles200JSONResponse)
			require.True(t, ok, "Expected 200 response")
			require.Equal(t, artifact.Status("SUCCESS"), actualResp.Status)

			// Verify snapshot
			verifyFilesSnapshot(t, tt.name, actualResp.FileDetailResponseJSONResponse)
		})
	}
}

// Helper functions

func setupFilesController(_ *testing.T, packageType artifact.PackageType) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)
	mockImageStore := new(mocks.ImageRepository)
	mockArtifactStore := new(mocks.ArtifactRepository)
	mockGenericBlobRepo := new(mocks.GenericBlobRepository)
	mockTransactor := new(mocks.Transactor)
	mockURLProvider := new(mocks.Provider)

	// Create FileManager with mocked dependencies
	// Note: NodesRepository mock doesn't exist yet, so passing nil
	// This will cause the test to fail if it tries to access file metadata
	mockFileManager := filemanager.NewFileManager(
		mockRegistryRepo,
		mockGenericBlobRepo,
		nil, // nodesRepo - TODO: Create NodesRepository mock
		mockTransactor,
		nil, // reporter
		nil, // config
		nil, // storageService
		nil, // bucketService
	)

	// TODO: Once NodesRepository mock is created, add these mocks:
	// mockNodesRepo.On("GetFilesMetadataByPathAndRegistryID", ...).Return(&fileMetadata, nil)
	// mockNodesRepo.On("CountFilesByPath", ...).Return(int64(1), nil)

	// Mock URLProvider methods
	mockURLProvider.On("PackageURL", mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry")
	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry")

	space := &coretypes.SpaceCore{
		ID:   1,
		Path: "root",
	}

	baseInfo := &types.RegistryRequestBaseInfo{
		RegistryID:         1,
		RegistryIdentifier: "test-registry",
		ParentRef:          "root",
		ParentID:           2,
	}

	registry := &types.Registry{
		ID:          1,
		Name:        "test-registry",
		ParentID:    2,
		PackageType: packageType,
	}

	image := &types.Image{
		ID:         1,
		Name:       "test-artifact",
		RegistryID: 1,
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	art := &types.Artifact{
		ID:        1,
		ImageID:   1,
		Version:   "v1.0.0",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
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
	mockImageStore.On("GetByNameAndType", mock.Anything, int64(1), "test-artifact", mock.Anything).
		Return(image, nil)
	mockArtifactStore.On("GetByName", mock.Anything, int64(1), "v1.0.0").
		Return(art, nil)

	eventReporter := createEventReporter()
	mockPackageWrapper := new(mocks.MockPackageWrapper)

	// Mock PackageWrapper methods for all package types
	mockPackageWrapper.On("GetFilePath", mock.Anything, mock.Anything, mock.Anything).
		Return("test-artifact/v1.0.0", nil)
	mockPackageWrapper.On("GetPackageURL",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry", nil)
	mockPackageWrapper.On("GetFileMetadata",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything).
		Return(&artifact.FileDetail{})

	return metadata.NewAPIController(mockRegistryRepo, mockFileManager, nil, mockGenericBlobRepo, nil, nil, nil, nil,
		mockImageStore, nil, mockSpaceFinder, nil, mockURLProvider, mockAuthorizer, nil, mockArtifactStore, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, nil, "", nil, nil, nil, nil, nil, nil, nil, nil,
		func(_ context.Context) bool { return false }, mockPackageWrapper, nil, nil)
}

func setupFilesControllerWithError(_ *testing.T, errorType string) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
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
	}

	fileManager := createFileManager()
	eventReporter := createEventReporter()

	return metadata.NewAPIController(nil, fileManager, nil, nil, nil, nil, nil, nil, nil, nil, mockSpaceFinder, nil,
		nil, mockAuthorizer, nil, nil, nil, nil, mockRegistryMetadataHelper, nil, eventReporter, nil, "", nil, nil, nil,
		nil, nil, nil, nil, nil, func(_ context.Context) bool { return false }, nil, nil)
}

func setupFilesSnapshotController(t *testing.T, packageType artifact.PackageType) *metadata.APIController {
	return setupFilesController(t, packageType)
}

// verifyFilesSnapshot compares the actual artifact files data with a stored snapshot.
func verifyFilesSnapshot(t *testing.T, name string, actual artifact.FileDetailResponseJSONResponse) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "snapshots", "artifact_files")
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

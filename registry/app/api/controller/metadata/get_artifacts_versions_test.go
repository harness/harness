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
	"github.com/harness/gitness/registry/types"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGetAllArtifactVersions tests the GetAllArtifactVersions endpoint for various package types.
//
// Note: Docker and Helm tests require TagRepository with correct mock signatures - skipped for now.
// Testing non-OCI package types and error scenarios.
func TestGetAllArtifactVersions(t *testing.T) {
	tests := []struct {
		name         string
		packageType  artifact.PackageType
		setupMocks   func() *metadata.APIController
		expectedResp any
	}{
		// Docker and Helm require TagRepository mocks - skipped
		// {name: "docker_versions_success", packageType: artifact.PackageTypeDOCKER, ...},
		// {name: "helm_versions_success", packageType: artifact.PackageTypeHELM, ...},
		{
			name:        "maven_versions_success",
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupVersionsController(t, artifact.PackageTypeMAVEN)
			},
			expectedResp: artifact.GetAllArtifactVersions200JSONResponse{},
		},
		{
			name:        "generic_versions_success",
			packageType: artifact.PackageTypeGENERIC,
			setupMocks: func() *metadata.APIController {
				return setupVersionsController(t, artifact.PackageTypeGENERIC)
			},
			expectedResp: artifact.GetAllArtifactVersions200JSONResponse{},
		},
		{
			name:        "python_versions_success",
			packageType: artifact.PackageTypePYTHON,
			setupMocks: func() *metadata.APIController {
				return setupVersionsController(t, artifact.PackageTypePYTHON)
			},
			expectedResp: artifact.GetAllArtifactVersions200JSONResponse{},
		},
		{
			name:        "npm_versions_success",
			packageType: artifact.PackageTypeNPM,
			setupMocks: func() *metadata.APIController {
				return setupVersionsController(t, artifact.PackageTypeNPM)
			},
			expectedResp: artifact.GetAllArtifactVersions200JSONResponse{},
		},
		{
			name:        errorTypeSpaceNotFound,
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupVersionsControllerWithError(t, errorTypeSpaceNotFound)
			},
			expectedResp: artifact.GetAllArtifactVersions400JSONResponse{},
		},
		{
			name:        "unauthorized_access",
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupVersionsControllerWithError(t, errorTypeUnauthorized)
			},
			expectedResp: artifact.GetAllArtifactVersions403JSONResponse{},
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

			req := artifact.GetAllArtifactVersionsRequestObject{
				RegistryRef: artifact.RegistryRefPathParam("test-registry"),
				Artifact:    artifact.ArtifactPathParam("test-artifact"),
				Params:      artifact.GetAllArtifactVersionsParams{},
			}

			resp, err := controller.GetAllArtifactVersions(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)

			// Verify response type matches expected
			switch tt.expectedResp.(type) {
			case artifact.GetAllArtifactVersions200JSONResponse:
				_, ok := resp.(artifact.GetAllArtifactVersions200JSONResponse)
				assert.True(t, ok, "Expected 200 response")
			case artifact.GetAllArtifactVersions400JSONResponse:
				_, ok := resp.(artifact.GetAllArtifactVersions400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
			case artifact.GetAllArtifactVersions403JSONResponse:
				_, ok := resp.(artifact.GetAllArtifactVersions403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
			case artifact.GetAllArtifactVersions404JSONResponse:
				_, ok := resp.(artifact.GetAllArtifactVersions404JSONResponse)
				assert.True(t, ok, "Expected 404 response")
			}
		})
	}
}

// TestGetAllArtifactVersionsSnapshot tests that the generated artifact versions responses
// match the expected snapshots for all package types. This ensures consistency across runs.
//
// Note: Docker and Helm tests require TagRepository with correct mock signatures.
// Currently testing non-OCI package types (Maven, Generic, Python, NPM, RPM, Nuget, Go, HuggingFace).
func TestGetAllArtifactVersionsSnapshot(t *testing.T) {
	tests := []struct {
		name        string
		packageType artifact.PackageType
	}{
		// Docker and Helm require TagRepository mocks - skipped for now
		// {name: "docker_versions", packageType: artifact.PackageTypeDOCKER},
		// {name: "helm_versions", packageType: artifact.PackageTypeHELM},
		{name: "maven_versions", packageType: artifact.PackageTypeMAVEN},
		{name: "generic_versions", packageType: artifact.PackageTypeGENERIC},
		{name: "python_versions", packageType: artifact.PackageTypePYTHON},
		{name: "npm_versions", packageType: artifact.PackageTypeNPM},
		{name: "rpm_versions", packageType: artifact.PackageTypeRPM},
		{name: "nuget_versions", packageType: artifact.PackageTypeNUGET},
		{name: "go_versions", packageType: artifact.PackageTypeGO},
		{name: "huggingface_versions", packageType: artifact.PackageTypeHUGGINGFACE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := setupVersionsSnapshotController(t, tt.packageType)

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

			req := artifact.GetAllArtifactVersionsRequestObject{
				RegistryRef: artifact.RegistryRefPathParam("test-registry"),
				Artifact:    artifact.ArtifactPathParam("test-artifact"),
				Params:      artifact.GetAllArtifactVersionsParams{},
			}

			resp, err := controller.GetAllArtifactVersions(ctx, req)
			require.NoError(t, err)

			actualResp, ok := resp.(artifact.GetAllArtifactVersions200JSONResponse)
			require.True(t, ok, "Expected 200 response")
			require.Equal(t, artifact.Status("SUCCESS"), actualResp.Status)

			// Verify snapshot
			verifyVersionsSnapshot(t, tt.name, actualResp.ListArtifactVersionResponseJSONResponse)
		})
	}
}

// Helper functions

func setupVersionsController(_ *testing.T, packageType artifact.PackageType) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)
	mockImageStore := new(mocks.ImageRepository)
	mockArtifactStore := new(mocks.MockArtifactRepository)
	mockURLProvider := new(mocks.Provider)

	space := &coretypes.SpaceCore{
		ID:   1,
		Path: "root",
	}

	regInfo := &metadata.RegistryRequestInfo{
		RegistryRequestBaseInfo: &types.RegistryRequestBaseInfo{
			RegistryID:         1,
			RegistryIdentifier: "test-registry",
			ParentRef:          "root",
			ParentID:           2,
			RootIdentifier:     "root",
		},
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

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "test-registry").
		Return(regInfo.RegistryRequestBaseInfo, nil)
	mockRegistryMetadataHelper.On("GetRegistryRequestInfo", mock.Anything, mock.Anything).
		Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
		Return([]coretypes.PermissionCheck{})
	mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)
	mockRegistryRepo.On("Get", mock.Anything, int64(1)).
		Return(registry, nil)
	mockImageStore.On("GetByNameAndType", mock.Anything, int64(1), "test-artifact", mock.Anything).
		Return(image, nil)

	// Mock for non-OCI types (Maven, Generic, Python, NPM, RPM, Nuget, Go, HuggingFace)
	var artifactType *artifact.ArtifactType
	if packageType == artifact.PackageTypeHUGGINGFACE {
		modelType := artifact.ArtifactTypeModel
		artifactType = &modelType
	}

	nonOCIVersions := &[]types.NonOCIArtifactMetadata{
		{
			Name:         "v1.0.0",
			ModifiedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			ArtifactType: artifactType,
		},
	}
	mockArtifactStore.On("GetAllVersionsByRepoAndImage",
		mock.Anything, int64(1), "test-artifact", mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nonOCIVersions, nil)
	mockArtifactStore.On("CountAllVersionsByRepoAndImage",
		mock.Anything, int64(2), "test-registry", "test-artifact",
		mock.Anything, mock.Anything).
		Return(int64(1), nil)

	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry")
	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry")

	eventReporter := createEventReporter()
	fileManager := createFileManager()
	mockPackageWrapper := new(mocks.MockPackageWrapper)

	// Mock PackageWrapper methods - return nil to use default implementation
	// which populates package-specific metadata
	mockPackageWrapper.On("GetArtifactVersionMetadata", mock.Anything, mock.Anything, mock.Anything).
		Return((*artifact.ArtifactVersionMetadata)(nil))

	return metadata.NewAPIController(
		mockRegistryRepo, fileManager, nil, nil, nil, nil, nil, nil, mockImageStore, nil,
		mockSpaceFinder, nil, mockURLProvider, mockAuthorizer, nil, mockArtifactStore, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, nil, "",
		nil, nil, nil, nil, nil, nil, nil,
		func(_ context.Context) bool { return false },
		mockPackageWrapper, nil,
	)
}

func setupVersionsControllerWithError(_ *testing.T, errorType string) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)

	space := &coretypes.SpaceCore{
		ID:   1,
		Path: "root",
	}

	regInfo := &metadata.RegistryRequestInfo{
		RegistryRequestBaseInfo: &types.RegistryRequestBaseInfo{
			RegistryID:         1,
			RegistryIdentifier: "test-registry",
			ParentRef:          "root",
			ParentID:           2,
		},
	}

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "test-registry").
		Return(regInfo.RegistryRequestBaseInfo, nil)

	switch errorType {
	case errorTypeSpaceNotFound:
		mockRegistryMetadataHelper.On("GetRegistryRequestInfo", mock.Anything, mock.Anything).
			Return(regInfo, nil)
		mockSpaceFinder.On("FindByRef", mock.Anything, "root").
			Return(nil, http.ErrNotSupported)

	case errorTypeUnauthorized:
		mockRegistryMetadataHelper.On("GetRegistryRequestInfo", mock.Anything, mock.Anything).
			Return(regInfo, nil)
		mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
		mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
			Return([]coretypes.PermissionCheck{})
		mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
			Return(false, http.ErrNotSupported)
	}

	eventReporter := createEventReporter()

	fileManager := createFileManager()

	return metadata.NewAPIController(
		nil, fileManager, nil, nil, nil, nil, nil, nil, nil, nil,
		mockSpaceFinder, nil, nil, mockAuthorizer, nil, nil, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, nil, "",
		nil, nil, nil, nil, nil, nil, nil,
		func(_ context.Context) bool { return false },
		nil, nil,
	)
}

func setupVersionsSnapshotController(t *testing.T, packageType artifact.PackageType) *metadata.APIController {
	return setupVersionsController(t, packageType)
}

// verifyVersionsSnapshot compares the actual artifact versions data with a stored snapshot.
func verifyVersionsSnapshot(t *testing.T, name string, actual artifact.ListArtifactVersionResponseJSONResponse) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "snapshots", "artifact_versions")
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

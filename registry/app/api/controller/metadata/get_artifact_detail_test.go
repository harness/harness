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

const (
	errorTypeSpaceNotFound  = "space_not_found"
	errorTypeUnauthorized   = "unauthorized"
	envUpdateSnapshots      = "UPDATE_SNAPSHOTS"
	envValueTrue            = "true"
	snapshotFilePermissions = 0600
)

func TestGetArtifactDetails(t *testing.T) {
	tests := []struct {
		name         string
		packageType  artifact.PackageType
		setupMocks   func() *metadata.APIController
		expectedResp any
	}{
		{
			name:        "maven_artifact_success",
			packageType: artifact.PackageTypeMAVEN,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "generic_artifact_success",
			packageType: artifact.PackageTypeGENERIC,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "python_artifact_success",
			packageType: artifact.PackageTypePYTHON,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "npm_artifact_success",
			packageType: artifact.PackageTypeNPM,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "rpm_artifact_success",
			packageType: artifact.PackageTypeRPM,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "nuget_artifact_success",
			packageType: artifact.PackageTypeNUGET,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "go_artifact_success",
			packageType: artifact.PackageTypeGO,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "huggingface_artifact_success",
			packageType: artifact.PackageTypeHUGGINGFACE,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "docker_artifact_success",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        "helm_artifact_success",
			packageType: artifact.PackageTypeHELM,
			setupMocks: func() *metadata.APIController {
				return setupBasicController(t)
			},
			expectedResp: artifact.GetArtifactDetails200JSONResponse{},
		},
		{
			name:        errorTypeSpaceNotFound,
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerWithError(t, errorTypeSpaceNotFound)
			},
			expectedResp: artifact.GetArtifactDetails400JSONResponse{},
		},
		{
			name:        "unauthorized_access",
			packageType: artifact.PackageTypeDOCKER,
			setupMocks: func() *metadata.APIController {
				return setupControllerWithError(t, errorTypeUnauthorized)
			},
			expectedResp: artifact.GetArtifactDetails403JSONResponse{},
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

			req := artifact.GetArtifactDetailsRequestObject{
				RegistryRef: artifact.RegistryRefPathParam("test-registry"),
				Artifact:    artifact.ArtifactPathParam("test-artifact"),
				Version:     artifact.VersionPathParam("v1.0.0"),
				Params:      artifact.GetArtifactDetailsParams{},
			}
			resp, err := controller.GetArtifactDetails(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)

			// Verify response type matches expected
			switch tt.expectedResp.(type) {
			case artifact.GetArtifactDetails200JSONResponse:
				actualResp, ok := resp.(artifact.GetArtifactDetails200JSONResponse)
				assert.True(t, ok, "Expected 200 response")
				if ok {
					assert.Equal(t, artifact.Status("SUCCESS"), actualResp.Status, "Expected SUCCESS status")
					assert.NotNil(t, actualResp.Data, "Expected artifact data to be present")
				}
			case artifact.GetArtifactDetails400JSONResponse:
				_, ok := resp.(artifact.GetArtifactDetails400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
			case artifact.GetArtifactDetails403JSONResponse:
				_, ok := resp.(artifact.GetArtifactDetails403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
			case artifact.GetArtifactDetails500JSONResponse:
				_, ok := resp.(artifact.GetArtifactDetails500JSONResponse)
				assert.True(t, ok, "Expected 500 response")
			}
		})
	}
}

// Helper functions

func setupBasicController(_ *testing.T) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)
	mockImageStore := new(mocks.ImageRepository)
	mockArtifactStore := new(mocks.ArtifactRepository)
	mockDownloadStatRepo := new(mocks.MockDownloadStatRepository)
	mockQuarantineRepo := new(mocks.MockQuarantineArtifactRepository)

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
		PackageType: artifact.PackageTypeDOCKER,
	}

	image := &types.Image{
		ID:         1,
		Name:       "test-artifact",
		RegistryID: 1,
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	metadataBytes, _ := json.Marshal(map[string]any{
		"name":        "test-artifact",
		"version":     "v1.0.0",
		"description": "Test artifact",
	})

	art := &types.Artifact{
		ID:        1,
		ImageID:   1,
		Version:   "v1.0.0",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Metadata:  metadataBytes,
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
	mockDownloadStatRepo.On("GetTotalDownloadsForArtifactID", mock.Anything, int64(1)).
		Return(int64(100), nil)
	mockQuarantineRepo.On("GetByFilePath", mock.Anything, "", int64(1), "test-artifact", "v1.0.0", mock.Anything).
		Return([]*types.QuarantineArtifact{}, nil)

	fileManager := createFileManager()
	eventReporter := createEventReporter()

	return metadata.NewAPIController(
		mockRegistryRepo, fileManager, nil, nil, nil, nil, nil, nil, mockImageStore, nil,
		mockSpaceFinder, nil, nil, mockAuthorizer, nil, mockArtifactStore, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, mockDownloadStatRepo, "",
		nil, nil, nil, nil, nil, mockQuarantineRepo, nil, nil,
		func(_ context.Context) bool { return false },
		nil, nil,
	)
}

func setupControllerWithError(_ *testing.T, errorType string) *metadata.APIController {
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

	return metadata.NewAPIController(
		nil, fileManager, nil, nil, nil, nil, nil, nil, nil, nil,
		mockSpaceFinder, nil, nil, mockAuthorizer, nil, nil, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, nil, "",
		nil, nil, nil, nil, nil, nil, nil, nil,
		func(_ context.Context) bool { return false },
		nil, nil,
	)
}

// TestGetArtifactDetailsSnapshot tests that the generated artifact details
// match the expected snapshots for all package types. This ensures consistency across runs.
func TestGetArtifactDetailsSnapshot(t *testing.T) {
	tests := []struct {
		name        string
		packageType artifact.PackageType
	}{
		{name: "docker_artifact", packageType: artifact.PackageTypeDOCKER},
		{name: "helm_artifact", packageType: artifact.PackageTypeHELM},
		{name: "maven_artifact", packageType: artifact.PackageTypeMAVEN},
		{name: "generic_artifact", packageType: artifact.PackageTypeGENERIC},
		{name: "python_artifact", packageType: artifact.PackageTypePYTHON},
		{name: "npm_artifact", packageType: artifact.PackageTypeNPM},
		{name: "rpm_artifact", packageType: artifact.PackageTypeRPM},
		{name: "nuget_artifact", packageType: artifact.PackageTypeNUGET},
		{name: "go_artifact", packageType: artifact.PackageTypeGO},
		{name: "huggingface_artifact", packageType: artifact.PackageTypeHUGGINGFACE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := setupSnapshotController(t, tt.packageType)

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

			req := artifact.GetArtifactDetailsRequestObject{
				RegistryRef: artifact.RegistryRefPathParam("test-registry"),
				Artifact:    artifact.ArtifactPathParam("test-artifact"),
				Version:     artifact.VersionPathParam("v1.0.0"),
				Params:      artifact.GetArtifactDetailsParams{},
			}

			resp, err := controller.GetArtifactDetails(ctx, req)
			require.NoError(t, err)

			actualResp, ok := resp.(artifact.GetArtifactDetails200JSONResponse)
			require.True(t, ok, "Expected 200 response")
			require.Equal(t, artifact.Status("SUCCESS"), actualResp.Status)

			// Verify snapshot
			verifyArtifactSnapshot(t, tt.name, actualResp.Data)
		})
	}
}

func setupSnapshotController(_ *testing.T, packageType artifact.PackageType) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)
	mockImageStore := new(mocks.ImageRepository)
	mockArtifactStore := new(mocks.ArtifactRepository)
	mockDownloadStatRepo := new(mocks.MockDownloadStatRepository)
	mockQuarantineRepo := new(mocks.MockQuarantineArtifactRepository)

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

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "", "test-registry").
		Return(baseInfo, nil)
	mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
		Return([]coretypes.PermissionCheck{})
	mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)
	mockRegistryRepo.On("GetByParentIDAndName", mock.Anything, int64(2), "test-registry").
		Return(registry, nil)
	// Create metadata based on package type
	var metadataBytes []byte
	switch packageType { //nolint:exhaustive // Only testing specific package types
	case artifact.PackageTypeMAVEN:
		metadataBytes, _ = json.Marshal(map[string]any{
			"groupId":    "com.example",
			"artifactId": "test-artifact",
			"version":    "v1.0.0",
		})
	case artifact.PackageTypeGENERIC:
		metadataBytes, _ = json.Marshal(map[string]any{
			"files": []map[string]any{
				{
					"name": "test-file.txt",
					"size": 1024,
				},
			},
			"file_count": 1,
			"size":       1024,
		})
	default:
		metadataBytes, _ = json.Marshal(map[string]any{
			"name":        "test-artifact",
			"version":     "v1.0.0",
			"description": "Test artifact for " + string(packageType),
		})
	}

	art := &types.Artifact{
		ID:        1,
		ImageID:   1,
		Version:   "v1.0.0",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Metadata:  metadataBytes,
	}

	mockImageStore.On("GetByNameAndType", mock.Anything, int64(1), "test-artifact", mock.Anything).
		Return(image, nil)
	mockArtifactStore.On("GetByName", mock.Anything, int64(1), "v1.0.0").
		Return(art, nil)
	mockDownloadStatRepo.On("GetTotalDownloadsForArtifactID", mock.Anything, int64(1)).
		Return(int64(100), nil)
	mockQuarantineRepo.On("GetByFilePath", mock.Anything, "", int64(1), "test-artifact", "v1.0.0", mock.Anything).
		Return([]*types.QuarantineArtifact{}, nil)

	fileManager := createFileManager()
	eventReporter := createEventReporter()

	return metadata.NewAPIController(
		mockRegistryRepo, fileManager, nil, nil, nil, nil, nil, nil, mockImageStore, nil,
		mockSpaceFinder, nil, nil, mockAuthorizer, nil, mockArtifactStore, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, mockDownloadStatRepo, "",
		nil, nil, nil, nil, nil, mockQuarantineRepo, nil, nil,
		func(_ context.Context) bool { return false },
		nil, nil,
	)
}

// verifyArtifactSnapshot compares the actual artifact detail data with a stored snapshot.
func verifyArtifactSnapshot(t *testing.T, name string, actual artifact.ArtifactDetail) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "snapshots", "artifact_details")
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

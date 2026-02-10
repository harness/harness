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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

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

// TestGetAllArtifacts tests the GetAllArtifacts endpoint for various scenarios.
//
// Note: This endpoint requires TagRepository which has complex mock signatures.
// Testing basic functionality and error scenarios.
func TestGetAllArtifacts(t *testing.T) {
	tests := []struct {
		name         string
		setupMocks   func() *metadata.APIController
		expectedResp any
	}{
		{
			name: errorTypeSpaceNotFound,
			setupMocks: func() *metadata.APIController {
				return setupArtifactsControllerWithError(t, errorTypeSpaceNotFound)
			},
			expectedResp: artifact.GetAllArtifacts400JSONResponse{},
		},
		{
			name: "unauthorized_access",
			setupMocks: func() *metadata.APIController {
				return setupArtifactsControllerWithError(t, errorTypeUnauthorized)
			},
			expectedResp: artifact.GetAllArtifacts403JSONResponse{},
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

			req := artifact.GetAllArtifactsRequestObject{
				SpaceRef: artifact.SpaceRefPathParam("root"),
				Params:   artifact.GetAllArtifactsParams{},
			}

			resp, err := controller.GetAllArtifacts(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)

			// Verify response type matches expected
			switch tt.expectedResp.(type) {
			case artifact.GetAllArtifacts200JSONResponse:
				_, ok := resp.(artifact.GetAllArtifacts200JSONResponse)
				assert.True(t, ok, "Expected 200 response")
			case artifact.GetAllArtifacts400JSONResponse:
				_, ok := resp.(artifact.GetAllArtifacts400JSONResponse)
				assert.True(t, ok, "Expected 400 response")
			case artifact.GetAllArtifacts403JSONResponse:
				_, ok := resp.(artifact.GetAllArtifacts403JSONResponse)
				assert.True(t, ok, "Expected 403 response")
			case artifact.GetAllArtifacts500JSONResponse:
				_, ok := resp.(artifact.GetAllArtifacts500JSONResponse)
				assert.True(t, ok, "Expected 500 response")
			}
		})
	}
}

// TestGetAllArtifactsSnapshot tests that the generated artifacts list responses
// match the expected snapshots. This ensures consistency across runs.
func TestGetAllArtifactsSnapshot(t *testing.T) {
	tests := []struct {
		name                  string
		includeQuarantine     bool
		untaggedImagesEnabled bool
		packageTypes          []artifact.PackageType
	}{
		{
			name:                  "all_artifacts",
			includeQuarantine:     false,
			untaggedImagesEnabled: false,
			packageTypes:          []artifact.PackageType{artifact.PackageTypeMAVEN, artifact.PackageTypeGENERIC},
		},
		{
			name:                  "all_artifacts_with_quarantine",
			includeQuarantine:     true,
			untaggedImagesEnabled: false,
			packageTypes:          []artifact.PackageType{artifact.PackageTypeMAVEN},
		},
		{
			name:                  "all_artifacts_untagged_images",
			includeQuarantine:     false,
			untaggedImagesEnabled: true,
			packageTypes:          []artifact.PackageType{artifact.PackageTypeMAVEN, artifact.PackageTypeGENERIC},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := setupArtifactsSnapshotController(t, tt.includeQuarantine, tt.untaggedImagesEnabled, tt.packageTypes)

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

			req := artifact.GetAllArtifactsRequestObject{
				SpaceRef: artifact.SpaceRefPathParam("root"),
				Params:   artifact.GetAllArtifactsParams{},
			}

			resp, err := controller.GetAllArtifacts(ctx, req)
			require.NoError(t, err)

			actualResp, ok := resp.(artifact.GetAllArtifacts200JSONResponse)
			if !ok {
				t.Logf("Response type: %T", resp)
				if errResp, isErr := resp.(artifact.GetAllArtifacts500JSONResponse); isErr {
					t.Logf("Error message: %v", errResp.Message)
				}
			}
			require.True(t, ok, "Expected 200 response")
			require.Equal(t, artifact.Status("SUCCESS"), actualResp.Status)

			// Verify snapshot
			verifyArtifactsSnapshot(t, tt.name, actualResp.ListArtifactResponseJSONResponse)
		})
	}
}

// Helper functions

func setupArtifactsControllerWithError(_ *testing.T, errorType string) *metadata.APIController {
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

	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "root", "").
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
		mockSpaceFinder, nil, nil, nil, mockAuthorizer, nil, nil, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, nil, "",
		nil, nil, nil, nil, nil, nil, nil, nil,
		func(_ context.Context) bool { return false },
		nil, nil,
	)
}

func setupArtifactsSnapshotController(
	_ *testing.T, includeQuarantine bool, untaggedImagesEnabled bool, _ []artifact.PackageType,
) *metadata.APIController {
	mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
	mockRegistryRepo := new(mocks.RegistryRepository)
	mockSpaceFinder := new(mocks.SpaceFinder)
	mockAuthorizer := new(mocks.Authorizer)
	mockTagStore := new(mocks.MockTagRepository)
	mockURLProvider := new(mocks.Provider)
	mockPackageWrapper := new(mocks.MockPackageWrapper)

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

	// Create sample artifacts for all package types
	allPackageTypes := []artifact.PackageType{
		artifact.PackageTypeDOCKER,
		artifact.PackageTypeHELM,
		artifact.PackageTypeMAVEN,
		artifact.PackageTypeGENERIC,
		artifact.PackageTypePYTHON,
		artifact.PackageTypeNPM,
		artifact.PackageTypeRPM,
		artifact.PackageTypeNUGET,
		artifact.PackageTypeGO,
		artifact.PackageTypeHUGGINGFACE,
	}

	artifacts := &[]types.ArtifactMetadata{}
	for i, pkgType := range allPackageTypes {
		// For untagged images, Docker and Helm need digest format (sha256:64hexchars)
		version := fmt.Sprintf("v%d.0.0", i+1)
		var tags []string
		if untaggedImagesEnabled && (pkgType == artifact.PackageTypeDOCKER || pkgType == artifact.PackageTypeHELM) {
			// Create a valid 64-character hex digest
			version = fmt.Sprintf("sha256:%064d", i+1)
			// Add tags array for untagged images - these are the tags that point to this digest
			tags = []string{fmt.Sprintf("v%d.0.0", i+1), "latest"}
		}

		art := types.ArtifactMetadata{
			Name:        fmt.Sprintf("test-artifact-%d", i+1),
			Version:     version,
			PackageType: pkgType,
			RepoName:    "test-registry",
			Tags:        tags,
		}

		// HuggingFace requires artifact type
		if pkgType == artifact.PackageTypeHUGGINGFACE {
			modelType := artifact.ArtifactTypeModel
			art.ArtifactType = &modelType
		}

		*artifacts = append(*artifacts, art)
	}

	// Add quarantine info if requested
	quarantineMap := make(map[types.ArtifactIdentifier]*types.QuarantineInfo)
	if includeQuarantine {
		reason := "Security vulnerability detected"
		quarantineMap[types.ArtifactIdentifier{
			Name:         "test-artifact-1",
			Version:      "v1.0.0",
			RegistryName: "test-registry",
		}] = &types.QuarantineInfo{
			Reason: reason,
		}
	}

	// Mock GetRegistryRequestBaseInfo which is called by GetRegistryRequestInfo
	mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo", mock.Anything, "root", "").
		Return(regInfo.RegistryRequestBaseInfo, nil)
	// Mock GetRegistryRequestInfo to return properly configured regInfo
	mockRegistryMetadataHelper.On("GetRegistryRequestInfo", mock.Anything, mock.Anything).
		Return(regInfo, nil)
	mockSpaceFinder.On("FindByRef", mock.Anything, "root").Return(space, nil)
	mockRegistryMetadataHelper.On("GetPermissionChecks", space, "test-registry", enum.PermissionRegistryView).
		Return([]coretypes.PermissionCheck{})
	mockAuthorizer.On("CheckAll", mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)

	// Mock TagStore methods - use mock.Anything for fields set by GetRegistryRequestInfo
	// GetAllArtifactsByParentID: ctx, parentID, registryIDs, sortByField,
	// sortByOrder, limit, offset, search, latestVersion, packageTypes
	mockTagStore.On("GetAllArtifactsByParentID",
		mock.Anything, int64(2), mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(artifacts, nil)
	// GetAllArtifactsByParentIDUntagged: ctx, parentID, registryIDs, sortByField,
	// sortByOrder, limit, offset, search, packageTypes (no latestVersion)
	mockTagStore.On("GetAllArtifactsByParentIDUntagged",
		mock.Anything, int64(2), mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(artifacts, nil)
	// CountAllArtifactsByParentID: ctx, parentID, registryIDs, search,
	// latestVersion, packageTypes, untaggedImagesEnabled
	mockTagStore.On("CountAllArtifactsByParentID",
		mock.Anything, int64(2), mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything).
		Return(int64(len(*artifacts)), nil)
	mockTagStore.On("GetQuarantineInfoForArtifacts", mock.Anything, mock.Anything, int64(2)).
		Return(quarantineMap, nil)

	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry")
	mockURLProvider.On("RegistryURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("https://registry.example.com/root/test-registry")

	// Mock PackageWrapper
	mockPackageWrapper.On("GetArtifactMetadata", mock.Anything).
		Return((*artifact.ArtifactMetadata)(nil))

	eventReporter := createEventReporter()
	fileManager := createFileManager()

	return metadata.NewAPIController(
		mockRegistryRepo, fileManager, nil, nil, nil, mockTagStore, nil, nil, nil, nil,
		mockSpaceFinder, nil, nil, mockURLProvider, mockAuthorizer, nil, nil, nil, nil,
		mockRegistryMetadataHelper, nil, eventReporter, nil, "",
		nil, nil, nil, nil, nil, nil, nil, nil,
		func(_ context.Context) bool { return untaggedImagesEnabled },
		mockPackageWrapper, nil,
	)
}

// verifyArtifactsSnapshot compares the actual artifacts list data with a stored snapshot.
func verifyArtifactsSnapshot(t *testing.T, name string, actual artifact.ListArtifactResponseJSONResponse) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "snapshots", "artifacts")
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

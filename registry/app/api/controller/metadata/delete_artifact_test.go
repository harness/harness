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
	"io"
	"mime/multipart"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/services/deletion"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// noopFileManager implements filemanager.FileManager with no-op methods.
// Only DeleteFile is functional for testing purposes.
type noopFileManager struct{}

var _ filemanager.FileManager = (*noopFileManager)(nil)

func (n *noopFileManager) UploadFile(
	_ context.Context, _ string, _ int64, _ int64, _ string, _ multipart.File, _ io.Reader, _ int64,
) (types.FileInfo, error) {
	return types.FileInfo{}, nil
}
func (n *noopFileManager) DownloadFileByPath(
	_ context.Context, _ string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	return nil, 0, "", nil
}
func (n *noopFileManager) DownloadFileByDigest(
	_ context.Context, _ string, _ types.FileInfo, _ int64, _ int64,
) (*storage.FileReader, error) {
	return &storage.FileReader{}, nil
}
func (n *noopFileManager) DeleteFile(_ context.Context, _ int64, _ string) error { return nil }
func (n *noopFileManager) HeadFile(_ context.Context, _ string, _ int64) (string, int64, error) {
	return "", 0, nil
}
func (n *noopFileManager) CopyNodes(_ context.Context, _ int64, _ int64, _ int64, _ []string) error {
	return nil
}
func (n *noopFileManager) FindLatestFilePath(_ context.Context, _ int64, _ string, _ string) (string, error) {
	return "", nil
}
func (n *noopFileManager) GetFilesMetadata(
	_ context.Context, _ string, _ int64, _ string, _ string, _ int, _ int, _ string,
) (*[]types.FileNodeMetadata, error) {
	result := []types.FileNodeMetadata{}
	return &result, nil
}
func (n *noopFileManager) CountFilesByPath(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (n *noopFileManager) PostFileUpload(
	_ context.Context, _ string, _ int64, _ int64, _ string, _ types.FileInfo, _ int64,
) error {
	return nil
}
func (n *noopFileManager) HeadByDigest(
	_ context.Context, _ string, _ types.FileInfo, _ int64, _ int64,
) (bool, int64, error) {
	return false, 0, nil
}
func (n *noopFileManager) SaveNodes(
	_ context.Context, _ string, _ int64, _ int64, _ int64, _ string,
) error {
	return nil
}
func (n *noopFileManager) CreateNodesWithoutFileNode(_ context.Context, _ string, _ int64, _ int64) error {
	return nil
}
func (n *noopFileManager) SaveNode(
	_ context.Context, _ string, _ string, _ int64, _ string, _ string, _ string, _ bool, _ int64,
) (string, error) {
	return "", nil
}
func (n *noopFileManager) GetFilePath(_ context.Context, _ string, _ int64, _ int64) (string, error) {
	return "", nil
}
func (n *noopFileManager) DeleteLeafNode(_ context.Context, _ int64, _ string) error { return nil }
func (n *noopFileManager) GetNode(_ context.Context, _ int64, _ string) (*types.Node, error) {
	return &types.Node{}, nil
}
func (n *noopFileManager) GetFileMetadata(_ context.Context, _ int64, _ string) (types.FileInfo, error) {
	return types.FileInfo{}, nil
}
func (n *noopFileManager) UploadFileNoDBUpdate(
	_ context.Context, _ string, _ multipart.File, _ io.Reader, _ int64, _ int64,
) (types.FileInfo, error) {
	return types.FileInfo{}, nil
}

func TestDeleteArtifact(t *testing.T) {
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
		request       api.DeleteArtifactRequestObject
		expectedResp  api.DeleteArtifactResponseObject
		expectedError error
	}{
		{
			name: "success_case",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryRepository := new(mocks.RegistryRepository)
				mockAuthorizer := new(mocks.Authorizer)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockImageStore := new(mocks.ImageRepository)
				mockArtifactStore := new(mocks.ArtifactRepository)
				mockAuditService := new(mocks.AuditService)

				// Create a transactor that executes the callback
				mockDeletionTx := new(mocks.Transactor)
				mockDeletionTx.On("WithTx", mock.Anything,
					mock.AnythingOfType("func(context.Context) error"),
					mock.Anything,
				).Run(func(args mock.Arguments) {
					fn, _ := args.Get(1).(func(context.Context) error)
					ctx, _ := args.Get(0).(context.Context)
					_ = fn(ctx)
				}).Return(nil)

				// No-op filemanager for deletion service
				mockFM := &noopFileManager{}

				// Mock artifact store deletion
				mockArtifactStore.On("DeleteByImageNameAndRegistryID",
					mock.Anything, int64(1), "test-artifact").Return(nil)

				// Mock image store deletion
				mockImageStore.On("DeleteByImageNameAndRegID",
					mock.Anything, int64(1), "test-artifact").Return(nil)

				deletionService := deletion.NewService(
					mockArtifactStore, mockImageStore, nil, nil, nil,
					mockFM, mockDeletionTx, nil, nil, nil, nil, nil,
				)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					PackageType:        api.PackageTypePYTHON,
				}

				registry := &types.Registry{
					ID:          1,
					Name:        "reg",
					ParentID:    2,
					Type:        "native",
					PackageType: "pypi",
				}

				artifact := &types.Image{
					ID:         1,
					Name:       "test-artifact",
					Enabled:    true,
					RegistryID: regInfo.RegistryID,
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo",
					mock.Anything, "", "reg").Return(regInfo, nil)
				mockAuthorizer.On(
					"Check",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					mock.AnythingOfType("*types.Scope"),
					mock.AnythingOfType("*types.Resource"),
					enum.PermissionArtifactsDelete,
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					int64(2),
					"reg",
					mock.AnythingOfType("types.QueryOption"),
				).Return(registry, nil)
				mockImageStore.On(
					"GetByName",
					mock.Anything,
					int64(1),
					"test-artifact",
					mock.AnythingOfType("types.QueryOption"),
				).Return(artifact, nil)
				mockAuditService.On(
					"Log",
					mock.Anything,
					mock.AnythingOfType("types.Principal"),
					mock.AnythingOfType("audit.Resource"),
					audit.ActionDeleted,
					"root/parent",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil)

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.ImageStore = mockImageStore
				c.DeletionService = deletionService
				c.AuditService = mockAuditService
			},
			request: api.DeleteArtifactRequestObject{
				RegistryRef: "reg",
				Artifact:    "test-artifact",
			},
			expectedResp: api.DeleteArtifact200JSONResponse{
				SuccessJSONResponse: api.SuccessJSONResponse{
					Status: api.StatusSUCCESS,
				},
			},
		},
		{
			name: "invalid_registry_reference",
			setupMocks: func(c *APIController) {
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo",
					mock.Anything, "", "invalid-reg").Return(nil, fmt.Errorf("invalid registry reference"))
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
			},
			request: api.DeleteArtifactRequestObject{
				RegistryRef: "invalid-reg",
				Artifact:    "test-artifact",
			},
			expectedResp: api.DeleteArtifact400JSONResponse{
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
					PackageType:        api.PackageTypePYTHON,
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo",
					mock.Anything, "", "reg").Return(regInfo, nil)
				mockAuthorizer.On("Check", mock.Anything, mock.AnythingOfType("*auth.Session"),
					mock.AnythingOfType("*types.Scope"),
					mock.AnythingOfType("*types.Resource"),
					enum.PermissionArtifactsDelete).Return(false, fmt.Errorf("not authorized"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.Authorizer = mockAuthorizer
			},
			request: api.DeleteArtifactRequestObject{
				RegistryRef: "reg",
				Artifact:    "test-artifact",
			},
			expectedResp: api.DeleteArtifact403JSONResponse{
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
					PackageType:        api.PackageTypePYTHON,
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo",
					mock.Anything, "", "reg").Return(regInfo, nil)
				mockAuthorizer.On(
					"Check",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					mock.AnythingOfType("*types.Scope"),
					mock.AnythingOfType("*types.Resource"),
					enum.PermissionArtifactsDelete,
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					int64(2),
					"reg",
					mock.AnythingOfType("types.QueryOption"),
				).Return(nil, store.ErrResourceNotFound)

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
			},
			request: api.DeleteArtifactRequestObject{
				RegistryRef: "reg",
				Artifact:    "test-artifact",
			},
			expectedResp: api.DeleteArtifact404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Code:    "404",
					Message: "registry reg doesn't exist",
				},
			},
		},
		{
			name: "artifact_not_found",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryRepository := new(mocks.RegistryRepository)
				mockAuthorizer := new(mocks.Authorizer)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockImageStore := new(mocks.ImageRepository)
				mockAuditService := new(mocks.AuditService)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					PackageType:        api.PackageTypePYTHON,
				}

				registry := &types.Registry{
					ID:          1,
					Name:        "reg",
					ParentID:    2,
					Type:        "native",
					PackageType: "pypi",
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo",
					mock.Anything, "", "reg").Return(regInfo, nil)
				mockAuthorizer.On(
					"Check",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					mock.AnythingOfType("*types.Scope"),
					mock.AnythingOfType("*types.Resource"),
					enum.PermissionArtifactsDelete,
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					int64(2),
					"reg",
					mock.AnythingOfType("types.QueryOption"),
				).Return(registry, nil)
				mockImageStore.On(
					"GetByName",
					mock.Anything,
					int64(1),
					"non-existent-artifact",
					mock.AnythingOfType("types.QueryOption"),
				).Return(nil, fmt.Errorf("artifact doesn't exist with this key"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.ImageStore = mockImageStore
				c.AuditService = mockAuditService
			},
			request: api.DeleteArtifactRequestObject{
				RegistryRef: "reg",
				Artifact:    "non-existent-artifact",
			},
			expectedResp: api.DeleteArtifact404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Code:    "404",
					Message: "artifact doesn't exist with this key",
				},
			},
		},
		{
			name: "artifact_already_deleted",
			setupMocks: func(c *APIController) {
				mockSpaceFinder := new(mocks.SpaceFinder)
				mockRegistryRepository := new(mocks.RegistryRepository)
				mockAuthorizer := new(mocks.Authorizer)
				mockRegistryMetadataHelper := new(mocks.RegistryMetadataHelper)
				mockImageStore := new(mocks.ImageRepository)
				mockAuditService := new(mocks.AuditService)

				space := &coretypes.SpaceCore{ID: 2}
				regInfo := &types.RegistryRequestBaseInfo{
					RegistryID:         1,
					RegistryIdentifier: "reg",
					ParentID:           2,
					ParentRef:          "root/parent",
					PackageType:        api.PackageTypePYTHON,
				}

				registry := &types.Registry{
					ID:          1,
					Name:        "reg",
					ParentID:    2,
					Type:        "native",
					PackageType: "pypi",
				}

				mockSpaceFinder.On("FindByRef", mock.Anything, "root/parent").Return(space, nil)
				mockRegistryMetadataHelper.On("GetRegistryRequestBaseInfo",
					mock.Anything, "", "reg").Return(regInfo, nil)
				mockAuthorizer.On(
					"Check",
					mock.Anything,
					mock.AnythingOfType("*auth.Session"),
					mock.AnythingOfType("*types.Scope"),
					mock.AnythingOfType("*types.Resource"),
					enum.PermissionArtifactsDelete,
				).Return(true, nil)
				mockRegistryRepository.On(
					"GetByParentIDAndName",
					mock.Anything,
					int64(2),
					"reg",
					mock.AnythingOfType("types.QueryOption"),
				).Return(registry, nil)
				mockImageStore.On(
					"GetByName",
					mock.Anything,
					int64(1),
					"deleted-artifact",
					mock.AnythingOfType("types.QueryOption"),
				).Return(nil, fmt.Errorf("artifact is already deleted"))

				c.SpaceFinder = mockSpaceFinder
				c.RegistryRepository = mockRegistryRepository
				c.Authorizer = mockAuthorizer
				c.RegistryMetadataHelper = mockRegistryMetadataHelper
				c.ImageStore = mockImageStore
				c.AuditService = mockAuditService
			},
			request: api.DeleteArtifactRequestObject{
				RegistryRef: "reg",
				Artifact:    "deleted-artifact",
			},
			expectedResp: api.DeleteArtifact404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{
					Code:    "404",
					Message: "artifact doesn't exist with this key",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &APIController{}
			tt.setupMocks(c)

			// Use the test context with the mock session
			resp, err := c.DeleteArtifact(testCtx, tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err, "Expected an error")
				assert.Equal(t, tt.expectedError.Error(), err.Error(), "Error message should match")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			// Check response type
			switch expected := tt.expectedResp.(type) {
			case api.DeleteArtifact200JSONResponse:
				actual, ok := resp.(api.DeleteArtifact200JSONResponse)
				assert.True(t, ok, "Expected 200 success response")
				assert.Equal(t, expected.Status, actual.Status, "Response status should match")
			case api.DeleteArtifact400JSONResponse:
				actual, ok := resp.(api.DeleteArtifact400JSONResponse)
				assert.True(t, ok, "Expected 400 bad request response")
				assert.Equal(t, expected.Message, actual.Message, "Error message should match")
			case api.DeleteArtifact403JSONResponse:
				actual, ok := resp.(api.DeleteArtifact403JSONResponse)
				assert.True(t, ok, "Expected 403 forbidden response")
				assert.Equal(t, expected.Message, actual.Message, "Error message should match")
			case api.DeleteArtifact404JSONResponse:
				actual, ok := resp.(api.DeleteArtifact404JSONResponse)
				assert.True(t, ok, "Expected 404 not found response")
				assert.Equal(t, expected.Message, actual.Message, "Error message should match")
			}

			// Full response should match expected value
			assert.Equal(t, tt.expectedResp, resp, "Full response should match expected value")

			// Verify all expectations were met
			// Only verify mocks that were actually set up in this test case
			if c.SpaceFinder != nil {
				mock.AssertExpectationsForObjects(t, c.SpaceFinder)
			}
			if c.RegistryRepository != nil {
				mock.AssertExpectationsForObjects(t, c.RegistryRepository)
			}
			if c.Authorizer != nil {
				mock.AssertExpectationsForObjects(t, c.Authorizer)
			}
			if c.RegistryMetadataHelper != nil {
				mock.AssertExpectationsForObjects(t, c.RegistryMetadataHelper)
			}
			if c.ImageStore != nil {
				mock.AssertExpectationsForObjects(t, c.ImageStore)
			}
			if c.tx != nil {
				mock.AssertExpectationsForObjects(t, c.tx)
			}
			if c.AuditService != nil {
				mock.AssertExpectationsForObjects(t, c.AuditService)
			}
		})
	}
}

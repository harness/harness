// Copyright 2023 Harness, Inc.
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

package python

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLocalRegistry struct {
	mock.Mock
}

func (m *MockLocalRegistry) GetArtifactType() artifact.RegistryType {
	args := m.Called()
	return args.Get(0).(artifact.RegistryType) //nolint:errcheck
}

func (m *MockLocalRegistry) GetPackageTypes() []artifact.PackageType {
	args := m.Called()
	return args.Get(0).([]artifact.PackageType) //nolint:errcheck
}

func (m *MockLocalRegistry) DownloadPackageFile(ctx context.Context, info python.ArtifactInfo) (
	*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error,
) {
	args := m.Called(ctx, info)
	//nolint:errcheck
	return args.Get(0).(*commons.ResponseHeaders), args.Get(1).(*storage.FileReader),
		args.Get(2).(io.ReadCloser), args.String(3), args.Error(4)
}

func (m *MockLocalRegistry) GetPackageMetadata(ctx context.Context, info python.ArtifactInfo) (
	python.PackageMetadata,
	error,
) {
	args := m.Called(ctx, info)
	return args.Get(0).(python.PackageMetadata), args.Error(1) //nolint:errcheck
}

func (m *MockLocalRegistry) UploadPackageFile(
	ctx context.Context,
	info python.ArtifactInfo,
	file multipart.File,
	filename string,
) (*commons.ResponseHeaders, string, error) {
	args := m.Called(ctx, info, file, filename)
	return args.Get(0).(*commons.ResponseHeaders), args.String(1), args.Error(2) //nolint:errcheck
}

func (m *MockLocalRegistry) UploadPackageFileReader(
	ctx context.Context,
	info python.ArtifactInfo,
	file io.ReadCloser,
	filename string,
) (*commons.ResponseHeaders, string, error) {
	args := m.Called(ctx, info, file, filename)
	return args.Get(0).(*commons.ResponseHeaders), args.String(1), args.Error(2) //nolint:errcheck
}

type MockLocalBase struct {
	mock.Mock
}

func (m *MockLocalBase) ExistsE(
	ctx context.Context,
	info pkg.PackageArtifactInfo,
	path string,
) (headers *commons.ResponseHeaders, err error) {
	args := m.Called(ctx, info, path)
	//nolint:errcheck
	return args.Get(0).(*commons.ResponseHeaders), args.Error(1)
}

func (m *MockLocalBase) DeleteFile(
	ctx context.Context,
	info pkg.PackageArtifactInfo,
	filePath string,
) (headers *commons.ResponseHeaders, err error) {
	args := m.Called(ctx, info, filePath)
	//nolint:errcheck
	return args.Get(0).(*commons.ResponseHeaders), args.Error(1)
}

func (m *MockLocalBase) UpdateFileManagerAndCreateArtifact(
	ctx context.Context,
	info pkg.ArtifactInfo,
	version, path string,
	metadata metadata.Metadata,
	fileInfo types.FileInfo,
	failOnConflict bool,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockLocalBase) CheckIfVersionExists(_ context.Context, _ pkg.PackageArtifactInfo) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (m *MockLocalBase) DeletePackage(_ context.Context, _ pkg.PackageArtifactInfo) error {
	// TODO implement me
	panic("implement me")
}

func (m *MockLocalBase) DeleteVersion(_ context.Context, _ pkg.PackageArtifactInfo) error {
	// TODO implement me
	panic("implement me")
}

func (m *MockLocalBase) Exists(ctx context.Context, info pkg.ArtifactInfo, path string) bool {
	args := m.Called(ctx, info, path)
	return args.Bool(0)
}

func (m *MockLocalBase) ExistsByFilePath(ctx context.Context, registryID int64, filePath string) (bool, error) {
	args := m.Called(ctx, registryID, filePath)
	return args.Bool(0), args.Error(1)
}

func (m *MockLocalBase) Download(ctx context.Context, info pkg.ArtifactInfo, version, filename string) (
	*commons.ResponseHeaders, *storage.FileReader, string, error,
) {
	args := m.Called(ctx, info, version, filename)
	//nolint:errcheck
	return args.Get(0).(*commons.ResponseHeaders), args.Get(1).(*storage.FileReader),
		args.String(2), args.Error(3)
}

func (m *MockLocalBase) Upload(
	ctx context.Context, info pkg.ArtifactInfo, filename, version, path string,
	reader io.ReadCloser, metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	args := m.Called(ctx, info, filename, version, path, reader, metadata)
	return args.Get(0).(*commons.ResponseHeaders), args.String(1), args.Error(2) //nolint:errcheck
}

func (m *MockLocalBase) UploadFile(
	ctx context.Context, info pkg.ArtifactInfo, filename, version, path string,
	file multipart.File,
	metadata metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	args := m.Called(ctx, info, filename, version, path, file, metadata)
	return args.Get(0).(*commons.ResponseHeaders), args.String(1), args.Error(2) //nolint:errcheck
}

func (m *MockLocalBase) MoveMultipleTempFilesAndCreateArtifact(
	ctx context.Context,
	info *pkg.ArtifactInfo,
	pathPrefix string,
	metadata metadata.Metadata,
	filesInfo *[]types.FileInfo,
	version string,
) error {
	args := m.Called(ctx, info, pathPrefix, metadata, filesInfo, version)
	return args.Error(0)
}

func (m *MockLocalBase) AuditPush(
	ctx context.Context,
	info pkg.ArtifactInfo,
	version string,
	imageUUID string,
	artifactUUID string,
) {
	m.Called(ctx, info, version, imageUUID, artifactUUID)
}

type MockReadCloser struct {
	mock.Mock
}

func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *MockReadCloser) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewLocalRegistryHelper(t *testing.T) {
	mockLocalRegistry := new(MockLocalRegistry)
	mockLocalBase := new(MockLocalBase)

	helper := NewLocalRegistryHelper(mockLocalRegistry, mockLocalBase)

	assert.NotNil(t, helper, "Helper should not be nil")
}

// Test for FileExists method.
func TestLocalRegistryHelper_FileExists(t *testing.T) {
	mockLocalRegistry := new(MockLocalRegistry)
	mockLocalBase := new(MockLocalBase)

	helper := NewLocalRegistryHelper(mockLocalRegistry, mockLocalBase)

	ctx := context.Background()
	artifactInfo := python.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				RootParentID:   1,
				RootIdentifier: "root",
				//RegParentID:    2,
			},
			RegIdentifier: "registry",
			Image:         "package",
		},
		Version:  "1.0.0",
		Filename: "package-1.0.0.whl",
	}

	mockLocalBase.On("Exists", ctx, artifactInfo.ArtifactInfo,
		artifactInfo.Image+"/"+artifactInfo.Version+"/"+artifactInfo.Filename).Return(true)

	exists := helper.FileExists(ctx, artifactInfo)

	assert.True(t, exists, "File should exist")
	mockLocalBase.AssertExpectations(t)
}

// Test for DownloadFile method.
func TestLocalRegistryHelper_DownloadFile(t *testing.T) {
	mockLocalRegistry := new(MockLocalRegistry)
	mockLocalBase := new(MockLocalBase)

	helper := NewLocalRegistryHelper(mockLocalRegistry, mockLocalBase)

	ctx := context.Background()

	artifactInfo := python.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				RootParentID:   1,
				RootIdentifier: "root",
				//RegParentID:    2,
			},
			RegIdentifier: "registry",
			Image:         "package",
		},
		Version:  "1.0.0",
		Filename: "package-1.0.0.whl",
	}

	expectedHeaders := &commons.ResponseHeaders{}
	expectedFileReader := &storage.FileReader{}
	expectedRedirectURL := "http://example.com/download"
	var expectedError error

	mockLocalBase.On("Download", ctx, artifactInfo.ArtifactInfo, artifactInfo.Version,
		artifactInfo.Filename).
		Return(expectedHeaders, expectedFileReader, expectedRedirectURL, expectedError)

	headers, fileReader, redirectURL, err := helper.DownloadFile(ctx, artifactInfo)

	assert.Equal(t, expectedHeaders, headers, "Headers should match")
	assert.Equal(t, expectedFileReader, fileReader, "FileReader should match")
	assert.Equal(t, expectedRedirectURL, redirectURL, "RedirectURL should match")
	assert.Nil(t, err, "Error should be nil")
	mockLocalBase.AssertExpectations(t)
}

// Test for DownloadFile method with error.
func TestLocalRegistryHelper_DownloadFile_Error(t *testing.T) {
	mockLocalRegistry := new(MockLocalRegistry)
	mockLocalBase := new(MockLocalBase)
	helper := NewLocalRegistryHelper(mockLocalRegistry, mockLocalBase)

	ctx := context.Background()
	artifactInfo := python.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				RootParentID:   1,
				RootIdentifier: "root",
				//RegParentID:    2,
			},
			RegIdentifier: "registry",
			Image:         "package",
		},
		Version:  "1.0.0",
		Filename: "package-1.0.0.whl",
	}

	expectedError := errors.New("download error")

	mockLocalBase.On("Download", ctx, artifactInfo.ArtifactInfo, artifactInfo.Version,
		artifactInfo.Filename).
		Return((*commons.ResponseHeaders)(nil), (*storage.FileReader)(nil), "", expectedError)

	headers, fileReader, redirectURL, err := helper.DownloadFile(ctx, artifactInfo)

	assert.Nil(t, headers, "Headers should be nil")
	assert.Nil(t, fileReader, "FileReader should be nil")
	assert.Equal(t, "", redirectURL, "RedirectURL should be empty")
	assert.Equal(t, expectedError, err, "Error should match expected error")
	mockLocalBase.AssertExpectations(t)
}

// Test for PutFile method.
func TestLocalRegistryHelper_UploadPackageFile(t *testing.T) {
	mockLocalRegistry := new(MockLocalRegistry)
	mockLocalBase := new(MockLocalBase)
	helper := NewLocalRegistryHelper(mockLocalRegistry, mockLocalBase)

	ctx := context.Background()
	artifactInfo := python.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				RootParentID:   1,
				RootIdentifier: "root",
				//RegParentID:    2,
			},
			RegIdentifier: "registry",
			Image:         "package",
		},
		Version:  "1.0.0",
		Filename: "package-1.0.0.whl",
	}

	mockReader := new(MockReadCloser)
	mockReader.On("Close").Return(nil)

	expectedHeaders := &commons.ResponseHeaders{}
	expectedSHA256 := "abc123"
	var expectedError error

	mockLocalRegistry.On("UploadPackageFileReader", ctx, artifactInfo,
		mock.AnythingOfType("*python.MockReadCloser"),
		"package-1.0.0.whl").
		Return(expectedHeaders, expectedSHA256, expectedError)

	headers, sha256, err := helper.UploadPackageFile(ctx, artifactInfo, mockReader, "package-1.0.0.whl")

	assert.Equal(t, expectedHeaders, headers, "Headers should match")
	assert.Equal(t, expectedSHA256, sha256, "SHA256 should match")
	assert.Nil(t, err, "Error should be nil")
	mockLocalRegistry.AssertExpectations(t)
}

// Test for PutFile method with error.
func TestLocalRegistryHelper_UploadPackageFile_Error(t *testing.T) {
	mockLocalRegistry := new(MockLocalRegistry)
	mockLocalBase := new(MockLocalBase)

	helper := NewLocalRegistryHelper(mockLocalRegistry, mockLocalBase)

	ctx := context.Background()
	artifactInfo := python.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				RootParentID:   1,
				RootIdentifier: "root",
				//RegParentID:    2,
			},
			RegIdentifier: "registry",
			Image:         "package",
		},
		Version:  "1.0.0",
		Filename: "package-1.0.0.whl",
	}

	mockReader := new(MockReadCloser)
	mockReader.On("Close").Return(nil)

	expectedError := errors.New("upload error")

	mockLocalRegistry.On("UploadPackageFileReader", ctx, artifactInfo,
		mock.AnythingOfType("*python.MockReadCloser"),
		"package-1.0.0.whl").
		Return((*commons.ResponseHeaders)(nil), "", expectedError)

	headers, sha256, err := helper.UploadPackageFile(ctx, artifactInfo, mockReader, "package-1.0.0.whl")

	assert.Nil(t, headers, "Headers should be nil")
	assert.Equal(t, "", sha256, "SHA256 should be empty")
	assert.Equal(t, expectedError, err, "Error should match expected error")
	mockLocalRegistry.AssertExpectations(t)
}

func TestMockLocalBase_MoveMultipleTempFilesAndCreateArtifact(t *testing.T) {
	mockLocalBase := new(MockLocalBase)
	ctx := context.Background()
	info := &pkg.ArtifactInfo{}
	pathPrefix := "test/path"
	var meta metadata.Metadata
	filesInfo := &[]types.FileInfo{}
	version := "1.0.0"

	mockLocalBase.On(
		"MoveMultipleTempFilesAndCreateArtifact",
		ctx, info, pathPrefix, meta, filesInfo, version,
	).Return(nil)

	err := mockLocalBase.MoveMultipleTempFilesAndCreateArtifact(ctx, info, pathPrefix, meta, filesInfo, version)
	assert.Nil(t, err, "Expected no error from mock implementation")
	mockLocalBase.AssertExpectations(t)
}

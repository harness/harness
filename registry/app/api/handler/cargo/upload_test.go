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

package cargo_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	cargo "github.com/harness/gitness/registry/app/api/controller/pkg/cargo"
	cargopkg "github.com/harness/gitness/registry/app/api/handler/cargo"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/request"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUploadPackage_Success(t *testing.T) {
	// Arrange
	resp := &cargo.UploadArtifactResponse{
		BaseResponse: cargo.BaseResponse{
			Error: nil,
			ResponseHeaders: &commons.ResponseHeaders{
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Code: http.StatusOK,
			},
		},
		Warnings: &cargo.UploadArtifactWarnings{},
	}

	mockCtrl := new(mockController)
	handler := cargopkg.NewHandler(mockCtrl, &fakePackagesHandler{})

	info := &cargotype.ArtifactInfo{FileName: "test-crate-1.0.0.crate"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	// Create a simple cargo package payload with metadata and file
	metadata := &cargometadata.VersionMetadata{
		Name:    "test-crate",
		Version: "1.0.0",
	}

	// Mock the parseDataFromPayload method by creating a simple payload
	payload := createMockCargoPayload(t, metadata, []byte("mock crate file content"))

	mockCtrl.On("UploadPackage", ctx, info, mock.AnythingOfType("*cargo.VersionMetadata"), mock.Anything).Return(resp, nil)

	req := httptest.NewRequest(http.MethodPut, "/cargo/upload", bytes.NewReader(payload)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()

	// Act
	handler.UploadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "application/json", result.Header.Get("Content-Type"))

	var response cargo.UploadArtifactResponse
	err := json.NewDecoder(result.Body).Decode(&response)
	require.NoError(t, err)

	mockCtrl.AssertExpectations(t)
}

func TestUploadPackage_InvalidArtifactInfo(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	// Context without proper artifact info
	ctx := context.Background()

	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errs[0].Error() == failedToFetchInfoFromContext
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodPut, "/cargo/upload", bytes.NewReader([]byte("test"))).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.UploadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockPkgHandler.AssertExpectations(t)
}

func TestUploadPackage_InvalidPayload(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{FileName: "test-crate-1.0.0.crate"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1
	}), mock.Anything)

	// Invalid payload (too short to contain proper cargo format)
	req := httptest.NewRequest(http.MethodPut, "/cargo/upload", bytes.NewReader([]byte("invalid"))).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.UploadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockPkgHandler.AssertExpectations(t)
}

func TestUploadPackage_ControllerError(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{FileName: "test-crate-1.0.0.crate"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	metadata := &cargometadata.VersionMetadata{
		Name:    "test-crate",
		Version: "1.0.0",
	}

	payload := createMockCargoPayload(t, metadata, []byte("mock crate file content"))

	expectedError := errors.New("controller error")
	mockCtrl.On("UploadPackage", ctx, info,
		mock.AnythingOfType("*cargo.VersionMetadata"), mock.Anything).Return(nil, expectedError)
	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errs[0].Error() == "failed to upload package: controller error"
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodPut, "/cargo/upload", bytes.NewReader(payload)).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.UploadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

func TestUploadPackage_JSONEncodingError(t *testing.T) {
	// Arrange
	resp := &cargo.UploadArtifactResponse{
		BaseResponse: cargo.BaseResponse{
			Error: nil,
			ResponseHeaders: &commons.ResponseHeaders{
				Headers: map[string]string{},
				Code:    http.StatusOK,
			},
		},
		Warnings: &cargo.UploadArtifactWarnings{},
	}

	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{FileName: "test-crate-1.0.0.crate"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	metadata := &cargometadata.VersionMetadata{
		Name:    "test-crate",
		Version: "1.0.0",
	}

	payload := createMockCargoPayload(t, metadata, []byte("mock crate file content"))

	mockCtrl.On("UploadPackage", ctx, info, mock.AnythingOfType("*cargo.VersionMetadata"), mock.Anything).Return(resp, nil)
	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodPut, "/cargo/upload", bytes.NewReader(payload)).WithContext(ctx)

	// Create a ResponseWriter that fails on Write to simulate JSON encoding error
	w := &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		shouldFail:       true,
	}

	// Act
	handler.UploadPackage(w, req)

	// Assert - JSON encoding error occurs but status was already written as 200 by WriteToResponse
	require.Equal(t, http.StatusOK, w.Code)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

// Helper function to create a mock cargo payload.
func createMockCargoPayload(t *testing.T, metadata *cargometadata.VersionMetadata, crateData []byte) []byte {
	// Simplified cargo format: metadata_len (4 bytes) + metadata + crate_len (4 bytes) + crate_data
	metadataJSON, err := json.Marshal(metadata)
	require.NoError(t, err)

	var buf bytes.Buffer

	// Write metadata length (little endian)
	metadataJSONLen := len(metadataJSON)
	if metadataJSONLen > 0xFFFFFFFF {
		t.Fatalf("metadata too large: %d bytes", metadataJSONLen)
	}
	metadataLen := uint32(metadataJSONLen)
	buf.WriteByte(byte(metadataLen))
	buf.WriteByte(byte(metadataLen >> 8))
	buf.WriteByte(byte(metadataLen >> 16))
	buf.WriteByte(byte(metadataLen >> 24))

	// Write metadata
	buf.Write(metadataJSON)

	// Write crate data length (little endian)
	crateDataLen := len(crateData)
	if crateDataLen > 0xFFFFFFFF {
		t.Fatalf("crate data too large: %d bytes", crateDataLen)
	}
	crateLen := uint32(crateDataLen)
	buf.WriteByte(byte(crateLen))
	buf.WriteByte(byte(crateLen >> 8))
	buf.WriteByte(byte(crateLen >> 16))
	buf.WriteByte(byte(crateLen >> 24))

	// Write crate data
	buf.Write(crateData)

	return buf.Bytes()
}

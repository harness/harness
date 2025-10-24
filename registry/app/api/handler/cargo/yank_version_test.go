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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	cargo "github.com/harness/gitness/registry/app/api/controller/pkg/cargo"
	cargopkg "github.com/harness/gitness/registry/app/api/handler/cargo"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/request"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestYankVersion_Success(t *testing.T) {
	// Arrange
	resp := &cargo.UpdateYankResponse{
		BaseResponse: cargo.BaseResponse{
			Error: nil,
			ResponseHeaders: &commons.ResponseHeaders{
				Headers: map[string]string{},
				Code:    http.StatusOK,
			},
		},
		Ok: true,
	}

	mockCtrl := new(mockController)
	handler := cargopkg.NewHandler(mockCtrl, &fakePackagesHandler{})

	info := &cargotype.ArtifactInfo{
		FileName: "test-crate",
		Version:  "1.0.0",
	}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("UpdateYank", ctx, info, true).Return(resp, nil)

	req := httptest.NewRequest(http.MethodDelete, "/cargo/yank", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.YankVersion(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "application/json", result.Header.Get("Content-Type"))

	var response cargo.UpdateYankResponse
	err := json.NewDecoder(result.Body).Decode(&response)
	require.NoError(t, err)
	require.True(t, response.Ok)

	mockCtrl.AssertExpectations(t)
}

func TestYankVersion_InvalidArtifactInfo(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	// Context without proper artifact info
	ctx := context.Background()

	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errs[0].Error() == failedToFetchInfoFromContext
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodDelete, "/cargo/yank", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.YankVersion(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockPkgHandler.AssertExpectations(t)
}

func TestYankVersion_ControllerError(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{
		FileName: "test-crate",
		Version:  "1.0.0",
	}
	ctx := request.WithArtifactInfo(context.Background(), info)

	expectedError := errors.New("controller error")
	mockCtrl.On("UpdateYank", ctx, info, true).Return(nil, expectedError)
	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errs[0].Error() == "failed to yank version: controller error"
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodDelete, "/cargo/yank", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.YankVersion(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

func TestYankVersion_JSONEncodingError(t *testing.T) {
	// Arrange
	resp := &cargo.UpdateYankResponse{
		BaseResponse: cargo.BaseResponse{
			Error: nil,
			ResponseHeaders: &commons.ResponseHeaders{
				Headers: map[string]string{},
				Code:    http.StatusOK,
			},
		},
		Ok: true,
	}

	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{
		FileName: "test-crate",
		Version:  "1.0.0",
	}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("UpdateYank", ctx, info, true).Return(resp, nil)
	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodDelete, "/cargo/yank", nil).WithContext(ctx)

	// Create a ResponseWriter that fails on Write to simulate JSON encoding error
	w := &failingResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		shouldFail:       true,
	}

	// Act
	handler.YankVersion(w, req)

	// Assert - JSON encoding error is handled but status code is already written
	require.Equal(t, http.StatusOK, w.Code)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

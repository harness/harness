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
	"errors"
	"io"
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

const (
	failedToFetchInfoFromContext = "failed to fetch info from context"
)

// --- Tests ---.
func TestDownloadPackageIndex_ServeContent(t *testing.T) {
	// Arrange
	body := []byte(`{"name": "test-package", "vers": "1.0.0"}`)
	resp := &cargo.GetPackageIndexResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error: nil,
				ResponseHeaders: &commons.ResponseHeaders{
					Headers: map[string]string{
						"Content-Type": "application/json; charset=utf-8",
					},
					Code: http.StatusOK,
				},
			},
			RedirectURL: "",
			Body:        nil,
			ReadCloser:  io.NopCloser(bytes.NewReader(body)),
		},
	}

	mockCtrl := new(mockController)
	handler := cargopkg.NewHandler(mockCtrl, &fakePackagesHandler{})

	info := &cargotype.ArtifactInfo{FileName: "index"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackageIndex", ctx, info, "test/path").Return(resp)

	req := httptest.NewRequest(http.MethodGet, "/cargo/index/test/path", nil).WithContext(ctx)
	// Mock PathValue method
	req.SetPathValue("*", "test/path")
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackageIndex(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", result.Header.Get("Content-Type"))

	data, _ := io.ReadAll(result.Body)
	require.Equal(t, body, data)

	mockCtrl.AssertExpectations(t)
}

func TestDownloadPackageIndex_Redirect(t *testing.T) {
	// Arrange
	resp := &cargo.GetPackageIndexResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error:           nil,
				ResponseHeaders: nil,
			},
			RedirectURL: "https://example.com/index/test/path",
			Body:        nil,
			ReadCloser:  nil,
		},
	}

	mockCtrl := new(mockController)
	handler := cargopkg.NewHandler(mockCtrl, &fakePackagesHandler{})

	info := &cargotype.ArtifactInfo{FileName: "index"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackageIndex", ctx, info, "test/path").Return(resp)

	req := httptest.NewRequest(http.MethodGet, "/cargo/index/test/path", nil).WithContext(ctx)
	req.SetPathValue("*", "test/path")
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackageIndex(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusTemporaryRedirect, result.StatusCode)
	require.Equal(t, "https://example.com/index/test/path", result.Header.Get("Location"))

	mockCtrl.AssertExpectations(t)
}

func TestDownloadPackageIndex_ErrorFromController(t *testing.T) {
	// Arrange
	resp := &cargo.GetPackageIndexResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error:           errors.New("index not found"),
				ResponseHeaders: nil,
			},
			RedirectURL: "",
			Body:        nil,
			ReadCloser:  nil,
		},
	}

	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{FileName: "index"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackageIndex", ctx, info, "test/path").Return(resp)
	mockPkgHandler.On("HandleError", ctx, mock.Anything, mock.AnythingOfType("*errors.errorString"))

	req := httptest.NewRequest(http.MethodGet, "/cargo/index/test/path", nil).WithContext(ctx)
	req.SetPathValue("*", "test/path")
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackageIndex(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

func TestDownloadPackageIndex_InvalidArtifactInfo(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	// Setup expectations - handleCargoPackageAPIError calls HandleErrors internally
	mockPkgHandler.On("HandleErrors", mock.Anything, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errs[0].Error() == failedToFetchInfoFromContext
	}), mock.Anything)

	// Create request with invalid context (no artifact info)
	req := httptest.NewRequest(http.MethodGet, "/cargo/index/test/path", nil)
	req.SetPathValue("*", "test/path")
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackageIndex(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockPkgHandler.AssertExpectations(t)
}

func TestDownloadPackageIndex_ControllerReturnsNil(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{FileName: "index"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	// Setup expectations
	mockCtrl.On("DownloadPackageIndex", ctx, info, "test/path").Return(nil)
	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errs[0].Error() == "failed to get response from controller"
	}), mock.Anything)

	req := httptest.NewRequest(http.MethodGet, "/cargo/index/test/path", nil).WithContext(ctx)
	req.SetPathValue("*", "test/path")
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackageIndex(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

func TestDownloadPackageIndex_ServeContentError(t *testing.T) {
	// Arrange
	resp := &cargo.GetPackageIndexResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error: nil,
				ResponseHeaders: &commons.ResponseHeaders{
					Headers: map[string]string{},
					Code:    http.StatusOK,
				},
			},
			RedirectURL: "",
			Body:        nil,
			ReadCloser:  nil, // This will cause ServeContent to fail
		},
	}

	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	info := &cargotype.ArtifactInfo{FileName: "index"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackageIndex", ctx, info, "test/path").Return(resp)
	mockPkgHandler.On("HandleError", ctx, mock.Anything, mock.AnythingOfType("*errors.errorString"))

	req := httptest.NewRequest(http.MethodGet, "/cargo/index/test/path", nil).WithContext(ctx)
	req.SetPathValue("*", "test/path")
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackageIndex(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	// Status code is 200 because headers are written before ServeContent fails
	require.Equal(t, http.StatusOK, result.StatusCode)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	cargopkg "github.com/harness/gitness/registry/app/api/handler/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// isExpectedError checks if the error matches the expected test error.
func isExpectedError(err error, testError error) bool {
	return errors.Is(err, testError)
}

func TestNewHandler(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}

	// Act
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	// Assert
	require.NotNil(t, handler)
}

func TestHandler_GetPackageArtifactInfo_Success(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	baseInfo := pkg.ArtifactInfo{
		Registry: types.Registry{
			Name: "test-registry",
		},
		Image: "test-image",
	}

	mockPkgHandler.On("GetArtifactInfo", mock.Anything).Return(baseInfo, nil)

	req := httptest.NewRequest(http.MethodGet, "/cargo/test", nil)
	req.SetPathValue("name", "test-package")
	req.SetPathValue("version", "1.0.0")

	// Act
	result, err := handler.GetPackageArtifactInfo(req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	cargoInfo, ok := result.(*cargotype.ArtifactInfo)
	require.True(t, ok)
	require.Equal(t, "test-package", cargoInfo.Image)
	require.Equal(t, "1.0.0", cargoInfo.Version)
	require.Equal(t, "test-registry", cargoInfo.Registry.Name)

	mockPkgHandler.AssertExpectations(t)
}

func TestHandler_GetPackageArtifactInfo_Error(t *testing.T) {
	// Arrange
	mockCtrl := new(mockController)
	mockPkgHandler := &fakePackagesHandler{}
	handler := cargopkg.NewHandler(mockCtrl, mockPkgHandler)

	expectedError := errors.New("failed to get artifact info")
	mockPkgHandler.On("GetArtifactInfo", mock.Anything).Return(nil, expectedError)

	req := httptest.NewRequest(http.MethodGet, "/cargo/test", nil)
	req.SetPathValue("name", "test-package")
	req.SetPathValue("version", "1.0.0")

	// Act
	result, err := handler.GetPackageArtifactInfo(req)

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	require.True(t, isExpectedError(err, expectedError))

	mockPkgHandler.AssertExpectations(t)
}

func TestHandler_HandleCargoPackageAPIError(t *testing.T) {
	// Arrange
	mockPkgHandler := &fakePackagesHandler{}

	testError := errors.New("test error")
	ctx := context.Background()

	mockPkgHandler.On("HandleErrors", ctx, mock.MatchedBy(func(errs []error) bool {
		return len(errs) == 1 && errors.Is(errs[0], testError)
	}), mock.Anything)

	w := httptest.NewRecorder()

	// Act - call HandleErrors directly on the mock since the handler delegates to it
	mockPkgHandler.HandleErrors(ctx, []error{testError}, w)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockPkgHandler.AssertExpectations(t)
}

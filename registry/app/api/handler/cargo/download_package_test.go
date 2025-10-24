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

	"github.com/harness/gitness/app/auth/authn"
	cargo "github.com/harness/gitness/registry/app/api/controller/pkg/cargo"
	cargopkg "github.com/harness/gitness/registry/app/api/handler/cargo"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/request"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock Controller ---.
type mockController struct {
	mock.Mock
}

func (m *mockController) DownloadPackage(ctx context.Context, info *cargotype.ArtifactInfo) *cargo.GetPackageResponse {
	args := m.Called(ctx, info)
	if resp := args.Get(0); resp != nil {
		if typedResp, ok := resp.(*cargo.GetPackageResponse); ok {
			return typedResp
		}
	}
	return nil
}

func (m *mockController) DownloadPackageIndex(ctx context.Context,
	info *cargotype.ArtifactInfo, path string) *cargo.GetPackageIndexResponse {
	args := m.Called(ctx, info, path)
	if resp := args.Get(0); resp != nil {
		if typedResp, ok := resp.(*cargo.GetPackageIndexResponse); ok {
			return typedResp
		}
	}
	return nil
}

func (m *mockController) GetRegistryConfig(ctx context.Context,
	info *cargotype.ArtifactInfo) (*cargo.GetRegistryConfigResponse, error) {
	args := m.Called(ctx, info)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if resp, ok := args.Get(0).(*cargo.GetRegistryConfigResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockController) SearchPackage(ctx context.Context, info *cargotype.ArtifactInfo,
	requestInfo *cargotype.SearchPackageRequestParams) (*cargo.SearchPackageResponse, error) {
	args := m.Called(ctx, info, requestInfo)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if resp, ok := args.Get(0).(*cargo.SearchPackageResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockController) RegeneratePackageIndex(ctx context.Context,
	info *cargotype.ArtifactInfo) (*cargo.RegeneratePackageIndexResponse, error) {
	args := m.Called(ctx, info)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if resp, ok := args.Get(0).(*cargo.RegeneratePackageIndexResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockController) UploadPackage(ctx context.Context, info *cargotype.ArtifactInfo,
	metadata *cargometadata.VersionMetadata, fileReader io.ReadCloser) (*cargo.UploadArtifactResponse, error) {
	args := m.Called(ctx, info, metadata, fileReader)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if resp, ok := args.Get(0).(*cargo.UploadArtifactResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockController) UpdateYank(ctx context.Context, info *cargotype.ArtifactInfo,
	yank bool) (*cargo.UpdateYankResponse, error) {
	args := m.Called(ctx, info, yank)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	if resp, ok := args.Get(0).(*cargo.UpdateYankResponse); ok {
		return resp, args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Fake packages.Handler ---.
type fakePackagesHandler struct {
	mock.Mock
}

func (f *fakePackagesHandler) GetRegistryCheckAccess(ctx context.Context, r *http.Request,
	reqPermissions ...enum.Permission) error {
	args := f.Called(ctx, r, reqPermissions)
	return args.Error(0)
}

func (f *fakePackagesHandler) GetArtifactInfo(r *http.Request) (pkg.ArtifactInfo, error) {
	args := f.Called(r)
	if args.Error(1) != nil {
		return pkg.ArtifactInfo{}, args.Error(1)
	}
	if args.Error(1) != nil {
		return pkg.ArtifactInfo{}, args.Error(1)
	}
	if info, ok := args.Get(0).(pkg.ArtifactInfo); ok {
		return info, args.Error(1)
	}
	return pkg.ArtifactInfo{}, args.Error(1)
}

func (f *fakePackagesHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	f.Called(w, r)
}

func (f *fakePackagesHandler) TrackDownloadStats(ctx context.Context, r *http.Request) error {
	args := f.Called(ctx, r)
	return args.Error(0)
}

func (f *fakePackagesHandler) GetPackageArtifactInfo(r *http.Request) (pkg.PackageArtifactInfo, error) {
	args := f.Called(r)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	if info, ok := args.Get(0).(pkg.PackageArtifactInfo); ok {
		return info, args.Error(1)
	}
	return nil, args.Error(1)
}

func (f *fakePackagesHandler) CheckQuarantineStatus(ctx context.Context) error {
	args := f.Called(ctx)
	return args.Error(0)
}

func (f *fakePackagesHandler) GetAuthenticator() authn.Authenticator {
	args := f.Called()
	if resp := args.Get(0); resp != nil {
		if auth, ok := resp.(authn.Authenticator); ok {
			return auth
		}
	}
	return nil
}

func (f *fakePackagesHandler) HandleErrors2(ctx context.Context, errors errcode.Error, w http.ResponseWriter) {
	f.Called(ctx, errors, w)
}

func (f *fakePackagesHandler) HandleErrors(ctx context.Context, errors errcode.Errors, w http.ResponseWriter) {
	f.Called(ctx, errors, w)
	// Actually write an error status code to simulate real error handling
	w.WriteHeader(http.StatusInternalServerError)
}

func (f *fakePackagesHandler) HandleError(ctx context.Context, w http.ResponseWriter, err error) {
	f.Called(ctx, w, err)
	// Actually write an error status code to simulate real error handling
	w.WriteHeader(http.StatusInternalServerError)
}

func (f *fakePackagesHandler) HandleCargoPackageAPIError(w http.ResponseWriter, r *http.Request, err error) {
	f.Called(w, r, err)
	// Actually write an error status code to simulate real error handling
	w.WriteHeader(http.StatusInternalServerError)
}

func (f *fakePackagesHandler) ServeContent(w http.ResponseWriter, r *http.Request,
	fileReader *storage.FileReader, filename string) {
	f.Called(w, r, fileReader, filename)
}

// --- Tests ---.
func TestDownloadPackage_ServeContent(t *testing.T) {
	// Arrange
	body := []byte("hello package")
	resp := &cargo.GetPackageResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error: nil,
				ResponseHeaders: &commons.ResponseHeaders{
					Headers: map[string]string{
						"Content-Type": "text/plain; charset=utf-8",
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

	info := &cargotype.ArtifactInfo{FileName: "test.txt"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackage", ctx, info).Return(resp)

	req := httptest.NewRequest(http.MethodGet, "/cargo/download", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", result.Header.Get("Content-Type"))

	data, _ := io.ReadAll(result.Body)
	require.Equal(t, body, data)

	mockCtrl.AssertExpectations(t)
}

func TestDownloadPackage_Redirect(t *testing.T) {
	// Arrange
	resp := &cargo.GetPackageResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error:           nil,
				ResponseHeaders: nil,
			},
			RedirectURL: "https://example.com/pkg",
			Body:        nil,
			ReadCloser:  nil,
		},
	}

	mockCtrl := new(mockController)
	handler := cargopkg.NewHandler(mockCtrl, &fakePackagesHandler{})

	info := &cargotype.ArtifactInfo{FileName: "test.txt"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackage", ctx, info).Return(resp)

	req := httptest.NewRequest(http.MethodGet, "/cargo/download", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusTemporaryRedirect, result.StatusCode)
	require.Equal(t, "https://example.com/pkg", result.Header.Get("Location"))

	mockCtrl.AssertExpectations(t)
}

func TestDownloadPackage_ErrorFromController(t *testing.T) {
	// Arrange
	resp := &cargo.GetPackageResponse{
		DownloadFileResponse: cargo.DownloadFileResponse{
			BaseResponse: cargo.BaseResponse{
				Error:           errors.New("something went wrong"),
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

	info := &cargotype.ArtifactInfo{FileName: "test.txt"}
	ctx := request.WithArtifactInfo(context.Background(), info)

	mockCtrl.On("DownloadPackage", ctx, info).Return(resp)
	mockPkgHandler.On("HandleError", ctx, mock.Anything, mock.AnythingOfType("*errors.errorString"))

	req := httptest.NewRequest(http.MethodGet, "/cargo/download", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	// Act
	handler.DownloadPackage(w, req)

	// Assert
	result := w.Result()
	defer func() { _ = result.Body.Close() }()

	require.Equal(t, http.StatusInternalServerError, result.StatusCode)

	mockCtrl.AssertExpectations(t)
	mockPkgHandler.AssertExpectations(t)
}

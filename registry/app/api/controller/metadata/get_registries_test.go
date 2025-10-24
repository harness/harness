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
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/controller/mocks"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	coretypes "github.com/harness/gitness/types"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAllRegistryResponse(t *testing.T) {
	controller := &APIController{}

	registries := []store.RegistryMetadata{
		{
			RegID:         "test-registry",
			RegIdentifier: "test-registry",
			Description:   "Test registry",
			PackageType:   api.PackageTypeDOCKER,
			Type:          api.RegistryTypeVIRTUAL,
			LastModified:  time.Now(),
			ArtifactCount: 10,
			DownloadCount: 500,
			Size:          10240,
			Labels:        pq.StringArray{"test"},
			ParentID:      2,
		},
	}

	mockURLProvider := new(mocks.Provider)
	mockURLProvider.On("GenerateUIRegistryURL", mock.Anything, "root", "test-registry").Return("http://test.com/registry")
	mockURLProvider.On("RegistryURL", mock.Anything, "root", "test-registry").
		Return("http://registry.test.com/root/test-registry")
	controller.URLProvider = mockURLProvider

	mockSpaceFinder := new(mocks.SpaceFinder)
	space := &coretypes.SpaceCore{ID: 2, Path: "root/parent"}
	mockSpaceFinder.On("FindByID", mock.Anything, int64(2)).Return(space, nil)
	controller.SpaceFinder = mockSpaceFinder

	mockPublicAccess := new(mocks.MockPublicAccess)
	mockPublicAccess.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	controller.PublicAccess = mockPublicAccess

	response, _ := controller.GetAllRegistryResponse(context.Background(), &registries, 1, 1, 10, "root", mockURLProvider)

	assert.NotNil(t, response)
	assert.Equal(t, api.StatusSUCCESS, response.Status)
	assert.Equal(t, int64(1), *response.Data.ItemCount)
	assert.Equal(t, int64(1), *response.Data.PageCount)
	assert.Equal(t, int64(1), *response.Data.PageIndex)
	assert.Equal(t, 10, *response.Data.PageSize)
	assert.Len(t, response.Data.Registries, 1)
	assert.Equal(t, "test-registry", response.Data.Registries[0].Identifier)
}

func TestGetRegistryMetadata(t *testing.T) {
	controller := &APIController{}

	registries := []store.RegistryMetadata{
		{
			RegID:         "test-registry",
			RegIdentifier: "test-registry",
			Description:   "Test registry",
			PackageType:   api.PackageTypeDOCKER,
			Type:          api.RegistryTypeVIRTUAL,
			LastModified:  time.Now(),
			ArtifactCount: 10,
			DownloadCount: 500,
			Size:          10240,
			Labels:        pq.StringArray{"test"},
			ParentID:      2,
		},
		{
			RegID:         "generic-registry",
			RegIdentifier: "generic-registry",
			Description:   "",
			PackageType:   api.PackageTypeGENERIC,
			Type:          api.RegistryTypeUPSTREAM,
			LastModified:  time.Now(),
			ArtifactCount: 0,
			DownloadCount: 0,
			Size:          0,
			Labels:        pq.StringArray{},
			ParentID:      2,
		},
	}

	mockURLProvider := new(mocks.Provider)
	mockURLProvider.On("GenerateUIRegistryURL", mock.Anything, "root", "test-registry").Return("http://test.com/registry")
	mockURLProvider.On("RegistryURL", mock.Anything, "root", "test-registry").
		Return("http://registry.test.com/root/test-registry")
	mockURLProvider.On("GenerateUIRegistryURL", mock.Anything, "root", "generic-registry").
		Return("http://test.com/generic/registry")
	mockURLProvider.On("RegistryURL", mock.Anything, "root", "generic-registry").
		Return("http://registry.test.com/root/generic/generic-registry")
	mockURLProvider.On("RegistryURL", mock.Anything, "root", "generic", "generic-registry").
		Return("http://registry.test.com/root/generic/generic-registry")

	mockSpaceFinder := new(mocks.SpaceFinder)
	space := &coretypes.SpaceCore{ID: 2, Path: "root/parent"}
	mockSpaceFinder.On("FindByID", mock.Anything, int64(2)).Return(space, nil)
	controller.SpaceFinder = mockSpaceFinder

	mockPublicAccess := new(mocks.MockPublicAccess)
	mockPublicAccess.On("Get", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
	controller.PublicAccess = mockPublicAccess

	metadata, _ := controller.GetRegistryMetadata(context.Background(), &registries, "root", mockURLProvider)

	assert.Len(t, metadata, 2)

	// Test first registry (Docker)
	assert.Equal(t, "test-registry", metadata[0].Identifier)
	assert.Equal(t, "Test registry", *metadata[0].Description)
	assert.Equal(t, api.PackageTypeDOCKER, metadata[0].PackageType)
	assert.NotNil(t, metadata[0].ArtifactsCount)
	assert.Equal(t, int64(10), *metadata[0].ArtifactsCount)
	assert.NotNil(t, metadata[0].DownloadsCount)
	assert.Equal(t, int64(500), *metadata[0].DownloadsCount)
	assert.NotNil(t, metadata[0].Labels)
	assert.Contains(t, *metadata[0].Labels, "test")
	assert.Equal(t, "http://registry.test.com/root/test-registry", metadata[0].Url)

	// Test second registry (Generic)
	assert.Equal(t, "generic-registry", metadata[1].Identifier)
	assert.Equal(t, "", *metadata[1].Description)
	assert.Equal(t, api.PackageTypeGENERIC, metadata[1].PackageType)
	assert.Nil(t, metadata[1].ArtifactsCount)
	assert.Nil(t, metadata[1].DownloadsCount)
	assert.Nil(t, metadata[1].Labels)
	assert.Equal(t, "http://registry.test.com/root/generic/generic-registry", metadata[1].Url)
}

func TestGetRegistryPath(t *testing.T) {
	controller := &APIController{}

	mockSpaceFinder := new(mocks.SpaceFinder)
	space := &coretypes.SpaceCore{ID: 2, Path: "root/parent"}
	mockSpaceFinder.On("FindByID", mock.Anything, int64(2)).Return(space, nil)
	mockSpaceFinder.On("FindByID", mock.Anything, int64(0)).Return(nil, fmt.Errorf("space not found"))
	mockSpaceFinder.On("FindByID", mock.Anything, int64(999)).Return(nil, fmt.Errorf("space not found"))
	controller.SpaceFinder = mockSpaceFinder

	// Test with valid parent ID
	path := controller.GetRegistryPath(context.Background(), 2, "test-registry")
	assert.Equal(t, "root/parent/test-registry", path)

	// Test with zero parent ID
	path = controller.GetRegistryPath(context.Background(), 0, "test-registry")
	assert.Equal(t, "", path)

	// Test with error finding space
	path = controller.GetRegistryPath(context.Background(), 999, "test-registry")
	assert.Equal(t, "", path)
}

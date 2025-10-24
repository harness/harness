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
	"testing"
	"time"

	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
)

func TestGetAllArtifactByRegistryResponse(t *testing.T) {
	artifacts := &[]registrytypes.ArtifactMetadata{
		{
			Name:          "test-artifact",
			LatestVersion: "1.0.0",
			PackageType:   api.PackageTypeDOCKER,
			ID:            1,
			RepoName:      "test-registry",
			CreatedAt:     time.Now(),
			ModifiedAt:    time.Now(),
			DownloadCount: 100,
			Labels:        []string{"test"},
		},
		{
			Name:          "another-artifact",
			LatestVersion: "2.0.0",
			PackageType:   api.PackageTypeGENERIC,
			ID:            2,
			RepoName:      "test-registry",
			CreatedAt:     time.Now(),
			ModifiedAt:    time.Now(),
			DownloadCount: 50,
			Labels:        nil,
		},
	}

	response := GetAllArtifactByRegistryResponse(artifacts, 2, 1, 10)

	assert.NotNil(t, response)
	assert.Equal(t, api.StatusSUCCESS, response.Status)
	assert.Equal(t, int64(2), *response.Data.ItemCount)
	assert.Equal(t, int64(1), *response.Data.PageCount)
	assert.Equal(t, int64(1), *response.Data.PageIndex)
	assert.Equal(t, 10, *response.Data.PageSize)
	assert.Len(t, response.Data.Artifacts, 2)

	// Test first artifact
	assert.Equal(t, "test-artifact", response.Data.Artifacts[0].Name)
	assert.Equal(t, "1.0.0", response.Data.Artifacts[0].LatestVersion)
	assert.Equal(t, api.PackageTypeDOCKER, response.Data.Artifacts[0].PackageType)
	assert.NotNil(t, response.Data.Artifacts[0].DownloadsCount)
	assert.Equal(t, int64(100), *response.Data.Artifacts[0].DownloadsCount)
	assert.NotNil(t, response.Data.Artifacts[0].Labels)
	assert.Contains(t, *response.Data.Artifacts[0].Labels, "test")

	// Test second artifact
	assert.Equal(t, "another-artifact", response.Data.Artifacts[1].Name)
	assert.Equal(t, "2.0.0", response.Data.Artifacts[1].LatestVersion)
	assert.Equal(t, api.PackageTypeGENERIC, response.Data.Artifacts[1].PackageType)
	assert.NotNil(t, response.Data.Artifacts[1].DownloadsCount)
	assert.Equal(t, int64(50), *response.Data.Artifacts[1].DownloadsCount)
	assert.NotNil(t, response.Data.Artifacts[1].Labels)
	assert.Empty(t, *response.Data.Artifacts[1].Labels)
}

func TestGetRegistryArtifactMetadata(t *testing.T) {
	artifacts := []registrytypes.ArtifactMetadata{
		{
			Name:          "test-artifact",
			LatestVersion: "1.0.0",
			PackageType:   api.PackageTypeDOCKER,
			ID:            1,
			RepoName:      "test-registry",
			CreatedAt:     time.Now(),
			ModifiedAt:    time.Now(),
			DownloadCount: 100,
			Labels:        []string{"test", "artifact"},
		},
		{
			Name:          "empty-artifact",
			LatestVersion: "0.1.0",
			PackageType:   api.PackageTypeGENERIC,
			ID:            2,
			RepoName:      "test-registry",
			CreatedAt:     time.Now(),
			ModifiedAt:    time.Now(),
			DownloadCount: 0,
			Labels:        nil,
		},
	}

	metadata := GetRegistryArtifactMetadata(artifacts)

	assert.Len(t, metadata, 2)

	// Test first artifact with data
	assert.Equal(t, "test-artifact", metadata[0].Name)
	assert.Equal(t, "1.0.0", metadata[0].LatestVersion)
	assert.Equal(t, api.PackageTypeDOCKER, metadata[0].PackageType)
	assert.NotNil(t, metadata[0].DownloadsCount)
	assert.Equal(t, int64(100), *metadata[0].DownloadsCount)
	assert.NotNil(t, metadata[0].Labels)
	assert.Len(t, *metadata[0].Labels, 2)
	assert.Contains(t, *metadata[0].Labels, "test")
	assert.Contains(t, *metadata[0].Labels, "artifact")

	// Test second artifact with zero/nil values
	assert.Equal(t, "empty-artifact", metadata[1].Name)
	assert.Equal(t, "0.1.0", metadata[1].LatestVersion)
	assert.Equal(t, api.PackageTypeGENERIC, metadata[1].PackageType)
	assert.NotNil(t, metadata[1].DownloadsCount)
	assert.Equal(t, int64(0), *metadata[1].DownloadsCount)
	assert.NotNil(t, metadata[1].Labels)
	assert.Empty(t, *metadata[1].Labels)
}

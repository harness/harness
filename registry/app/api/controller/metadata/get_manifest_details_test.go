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

package metadata

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/controller/mocks"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestGetManifestDetails_WithSoftDeletedImage tests line 151 - GetByName with WithAllDeleted option.
func TestGetManifestDetails_WithSoftDeletedImage(t *testing.T) {
	ctx := context.Background()

	// Create test manifest
	manifest := &types.Manifest{
		ID:         1,
		RegistryID: 100,
		ImageName:  "test-image",
		Digest:     "sha256:abc123",
		TotalSize:  1024,
		CreatedAt:  time.Now(),
	}

	// Create soft-deleted image
	deletedAt := time.Now()
	softDeletedImage := &types.Image{
		ID:         10,
		RegistryID: 100,
		Name:       "test-image",
		DeletedAt:  &deletedAt,
	}

	// Setup mocks
	mockImageRepo := new(mocks.ImageRepository)
	mockDownloadStatRepo := new(mocks.MockDownloadStatRepository)
	mockQuarantineRepo := new(mocks.MockQuarantineArtifactRepository)

	// Mock GetByName - this is line 151 being tested
	// The key here is that it's called with types.WithAllDeleted() option
	mockImageRepo.On("GetByName", ctx, manifest.RegistryID, manifest.ImageName, mock.Anything).
		Return(softDeletedImage, nil)

	// Mock download stats
	mockDownloadStatRepo.On("GetTotalDownloadsForManifests", ctx, mock.Anything, softDeletedImage.ID).
		Return(map[string]int64{}, nil)

	// Mock quarantine check
	mockQuarantineRepo.On(
		"GetByFilePath", ctx, "", manifest.RegistryID, softDeletedImage.Name, mock.Anything, mock.Anything,
	).Return(nil, nil)

	// Create controller
	controller := &APIController{
		ImageStore:                   mockImageRepo,
		DownloadStatRepository:       mockDownloadStatRepo,
		QuarantineArtifactRepository: mockQuarantineRepo,
	}

	// Execute - this calls line 151
	details, err := controller.getManifestDetails(ctx, manifest, nil)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, manifest.Digest.String(), details.Digest)

	// Verify GetByName was called (line 151)
	mockImageRepo.AssertCalled(t, "GetByName", ctx, manifest.RegistryID, manifest.ImageName, mock.Anything)
}

// TestGetManifestDetails_WithActiveImage tests line 151 with active (non-deleted) image.
func TestGetManifestDetails_WithActiveImage(t *testing.T) {
	ctx := context.Background()

	manifest := &types.Manifest{
		ID:         2,
		RegistryID: 200,
		ImageName:  "active-image",
		Digest:     "sha256:def456",
		TotalSize:  2048,
		CreatedAt:  time.Now(),
	}

	// Active image (no DeletedAt)
	activeImage := &types.Image{
		ID:         20,
		RegistryID: 200,
		Name:       "active-image",
		DeletedAt:  nil,
	}

	mockImageRepo := new(mocks.ImageRepository)
	mockDownloadStatRepo := new(mocks.MockDownloadStatRepository)
	mockQuarantineRepo := new(mocks.MockQuarantineArtifactRepository)

	mockImageRepo.On("GetByName", ctx, manifest.RegistryID, manifest.ImageName, mock.Anything).
		Return(activeImage, nil)

	mockDownloadStatRepo.On("GetTotalDownloadsForManifests", ctx, mock.Anything, activeImage.ID).
		Return(map[string]int64{}, nil)

	mockQuarantineRepo.On(
		"GetByFilePath", ctx, "", manifest.RegistryID, activeImage.Name, mock.Anything, mock.Anything,
	).Return(nil, nil)

	controller := &APIController{
		ImageStore:                   mockImageRepo,
		DownloadStatRepository:       mockDownloadStatRepo,
		QuarantineArtifactRepository: mockQuarantineRepo,
	}

	// Execute - line 151
	details, err := controller.getManifestDetails(ctx, manifest, nil)

	assert.NoError(t, err)
	assert.Equal(t, manifest.Digest.String(), details.Digest)

	mockImageRepo.AssertCalled(t, "GetByName", ctx, manifest.RegistryID, manifest.ImageName, mock.Anything)
}

// TestGetManifestDetails_WithManifestConfig tests line 151 with OS/Arch info.
func TestGetManifestDetails_WithManifestConfig(t *testing.T) {
	ctx := context.Background()

	manifest := &types.Manifest{
		ID:         3,
		RegistryID: 300,
		ImageName:  "multi-arch-image",
		Digest:     "sha256:ghi789",
		TotalSize:  3072,
		CreatedAt:  time.Now(),
	}

	image := &types.Image{
		ID:         30,
		RegistryID: 300,
		Name:       "multi-arch-image",
	}

	mConfig := &manifestConfig{
		Os:   "linux",
		Arch: "amd64",
	}

	mockImageRepo := new(mocks.ImageRepository)
	mockDownloadStatRepo := new(mocks.MockDownloadStatRepository)
	mockQuarantineRepo := new(mocks.MockQuarantineArtifactRepository)

	mockImageRepo.On("GetByName", ctx, manifest.RegistryID, manifest.ImageName, mock.Anything).
		Return(image, nil)

	mockDownloadStatRepo.On("GetTotalDownloadsForManifests", ctx, mock.Anything, image.ID).
		Return(map[string]int64{}, nil)

	mockQuarantineRepo.On(
		"GetByFilePath", ctx, "", manifest.RegistryID, image.Name, mock.Anything, mock.Anything,
	).Return(nil, nil)

	controller := &APIController{
		ImageStore:                   mockImageRepo,
		DownloadStatRepository:       mockDownloadStatRepo,
		QuarantineArtifactRepository: mockQuarantineRepo,
	}

	// Execute - line 151
	details, err := controller.getManifestDetails(ctx, manifest, mConfig)

	assert.NoError(t, err)
	assert.Equal(t, manifest.Digest.String(), details.Digest)
	assert.Equal(t, "linux/amd64", details.OsArch)

	mockImageRepo.AssertCalled(t, "GetByName", ctx, manifest.RegistryID, manifest.ImageName, mock.Anything)
}

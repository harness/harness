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

package gopackage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/gopackage/utils"
	"github.com/harness/gitness/registry/app/store"
)

type RegistryHelper interface {
	UpdatePackageIndex(
		ctx context.Context, principalID int64, rootParentID int64,
		registryID int64, image string,
	) error
}

type registryHelper struct {
	fileManager filemanager.FileManager
	artifactDao store.ArtifactRepository
	spaceFinder refcache.SpaceFinder
}

func NewRegistryHelper(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
	spaceFinder refcache.SpaceFinder,
) RegistryHelper {
	return &registryHelper{
		fileManager: fileManager,
		artifactDao: artifactDao,
		spaceFinder: spaceFinder,
	}
}

func (h *registryHelper) UpdatePackageIndex(
	ctx context.Context, principalID int64, rootParentID int64,
	registryID int64, image string,
) error {
	rootSpace, err := h.spaceFinder.FindByID(ctx, rootParentID)
	if err != nil {
		return fmt.Errorf("failed to find root space by ID: %w", err)
	}
	versionList, err := h.regeneratePackageIndex(ctx, registryID, image)
	if err != nil {
		return fmt.Errorf("failed to regenerate package index: %w", err)
	}
	return h.uploadIndexMetadata(
		ctx, principalID, rootSpace.Identifier, rootParentID, registryID,
		image, versionList,
	)
}

func (h *registryHelper) regeneratePackageIndex(
	ctx context.Context, registryID int64, image string,
) ([]string, error) {
	lastArtifactID := int64(0)
	artifactBatchLimit := 50
	versionList := []string{}
	for {
		artifacts, err := h.artifactDao.GetArtifactsByRepoAndImageBatch(
			ctx, registryID, image, artifactBatchLimit, lastArtifactID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get artifacts: %w", err)
		}

		for _, a := range *artifacts {
			versionList = append(versionList, a.Version)
			if a.ID > lastArtifactID {
				lastArtifactID = a.ID
			}
		}
		if len(*artifacts) < artifactBatchLimit {
			break
		}
	}
	return versionList, nil
}

func (h *registryHelper) uploadIndexMetadata(
	ctx context.Context, principalID int64, rootIdentifier string,
	rootParentID int64, registryID int64, image string,
	versionList []string,
) error {
	fileName := image
	filePath := utils.GetIndexFilePath(image)

	_, err := h.fileManager.UploadFile(
		ctx, filePath, registryID, rootParentID, rootIdentifier, nil,
		io.NopCloser(strings.NewReader(strings.Join(versionList, "\n"))),
		fileName, principalID,
	)
	if err != nil {
		return fmt.Errorf("failed to upload package index metadata: %w", err)
	}
	return nil
}

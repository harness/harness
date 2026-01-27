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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	apiutils "github.com/harness/gitness/registry/app/api/utils"
	gopackagemetadata "github.com/harness/gitness/registry/app/metadata/gopackage"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/gopackage/utils"
	refcache2 "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
)

type RegistryHelper interface {
	UpdatePackageIndex(
		ctx context.Context, principalID int64, rootParentID int64,
		registryID int64, image string,
	) error
	UpdatePackageMetadata(
		ctx context.Context, rootParentID int64,
		registryID int64, image string, version string,
	) error
}

type registryHelper struct {
	fileManager    filemanager.FileManager
	artifactDao    store.ArtifactRepository
	spaceFinder    refcache.SpaceFinder
	registryFinder refcache2.RegistryFinder
}

func NewRegistryHelper(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
	spaceFinder refcache.SpaceFinder,
	registryFinder refcache2.RegistryFinder,
) RegistryHelper {
	return &registryHelper{
		fileManager:    fileManager,
		artifactDao:    artifactDao,
		spaceFinder:    spaceFinder,
		registryFinder: registryFinder,
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
		if err != nil && errors.Is(err, gitnessstore.ErrResourceNotFound) {
			break
		}
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
	filePath := utils.GetIndexFilePath(image)

	_, err := h.fileManager.UploadFile(ctx, filePath, registryID, rootParentID, rootIdentifier, nil,
		io.NopCloser(strings.NewReader(strings.Join(versionList, "\n"))), principalID)
	if err != nil {
		return fmt.Errorf("failed to upload package index metadata: %w", err)
	}
	return nil
}

func (h *registryHelper) UpdatePackageMetadata(
	ctx context.Context, rootParentID int64,
	registryID int64, image string, version string,
) error {
	rootSpace, err := h.spaceFinder.FindByID(ctx, rootParentID)
	if err != nil {
		return fmt.Errorf("failed to find root space by ID: %w", err)
	}

	registry, err := h.registryFinder.FindByID(ctx, registryID)
	if err != nil {
		return fmt.Errorf("failed to find registry by ID: %w", err)
	}

	artifact, err := h.artifactDao.GetByRegistryImageAndVersion(ctx, registryID, image, version)
	if err != nil {
		return fmt.Errorf("failed to get artifact by registry, image and version: %w", err)
	}

	var metadata gopackagemetadata.VersionMetadataDB
	// convert artifact metadata to version metadata using json
	err = json.Unmarshal(artifact.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("failed to convert artifact metadata to version metadata: %w", err)
	}
	// regenerate package metadata
	err = h.regeneratePackageMetadata(
		ctx, rootSpace.Identifier, registry, image, version, &metadata.VersionMetadata,
	)
	if err != nil {
		return fmt.Errorf("failed to regenerate package metadata: %w", err)
	}
	// convert version metadata to artifact metadata using json
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	// update artifact
	err = h.artifactDao.UpdateArtifactMetadata(ctx, rawMetadata, artifact.ID)
	if err != nil {
		return fmt.Errorf("failed to create or update artifact: %w", err)
	}
	return nil
}

func (h *registryHelper) regeneratePackageMetadata(
	ctx context.Context, rootIdentifier string, registry *types.Registry,
	image string, version string, metadata *gopackagemetadata.VersionMetadata,
) error {
	path, err := apiutils.GetFilePath(artifact.PackageTypeGO, image, version)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}

	metadata.Name = image
	metadata.Version = version

	err = h.updatePackageMetadataFromInfoFile(
		ctx, path, rootIdentifier, registry, version, metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update package metadata from info file: %w", err)
	}

	err = h.updatePackageMetadataFromModFile(
		ctx, path, rootIdentifier, registry, version, metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update package metadata from mod file: %w", err)
	}

	err = h.updatePackageMetadataFromZipFile(
		ctx, path, rootIdentifier, registry, version, metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update package metadata from zip file: %w", err)
	}
	return nil
}

func (h *registryHelper) updatePackageMetadataFromInfoFile(
	ctx context.Context, path string, rootIdentifier string, registry *types.Registry,
	version string, metadata *gopackagemetadata.VersionMetadata,
) error {
	infoFileName := version + ".info"
	infoFilePath := filepath.Join(path, infoFileName)
	reader, _, _, err := h.fileManager.DownloadFileByPath(
		ctx, infoFilePath, registry.ID, registry.Name, rootIdentifier, false,
	)
	if err != nil {
		return fmt.Errorf("failed to download package metadata: %w", err)
	}
	infoBytes := &bytes.Buffer{}
	if _, err := io.Copy(infoBytes, reader); err != nil {
		return fmt.Errorf("error reading 'info': %w", err)
	}
	reader.Close()

	infometadata, err := utils.GetPackageMetadataFromInfoFile(infoBytes)
	if err != nil {
		return fmt.Errorf("failed to get package metadata from info file: %w", err)
	}

	metadata.Time = infometadata.Time
	metadata.Origin = infometadata.Origin
	return nil
}

func (h *registryHelper) updatePackageMetadataFromModFile(
	ctx context.Context, path string, rootIdentifier string, registry *types.Registry,
	version string, metadata *gopackagemetadata.VersionMetadata,
) error {
	modFileName := version + ".mod"
	modFilePath := filepath.Join(path, modFileName)
	reader, _, _, err := h.fileManager.DownloadFileByPath(
		ctx, modFilePath, registry.ID, registry.Name, rootIdentifier, false,
	)
	if err != nil {
		return fmt.Errorf("failed to download package metadata: %w", err)
	}
	modBytes := &bytes.Buffer{}
	if _, err := io.Copy(modBytes, reader); err != nil {
		return fmt.Errorf("error reading 'mod': %w", err)
	}
	reader.Close()

	err = utils.UpdateMetadataFromModFile(modBytes, metadata)
	if err != nil {
		return fmt.Errorf("failed to update metadata from mod file: %w", err)
	}
	return nil
}

func (h *registryHelper) updatePackageMetadataFromZipFile(
	ctx context.Context, path string, rootIdentifier string, registry *types.Registry,
	version string, metadata *gopackagemetadata.VersionMetadata,
) error {
	zipFileName := version + ".zip"
	zipFilePath := filepath.Join(path, zipFileName)
	reader, _, _, err := h.fileManager.DownloadFileByPath(
		ctx, zipFilePath, registry.ID, registry.Name, rootIdentifier, false,
	)
	if err != nil {
		return fmt.Errorf("failed to download package metadata: %w", err)
	}
	defer reader.Close()

	err = utils.UpdateMetadataFromZipFile(reader, metadata)
	if err != nil {
		return fmt.Errorf("failed to update metadata from zip file: %w", err)
	}
	return nil
}

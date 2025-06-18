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

package cargo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/app/api/request"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/store"
)

type RegistryHelper interface {
	GetIndexFilePathFromImageName(imageName string) string
	UpdatePackageIndex(
		ctx context.Context, rootIdentifier string, rootParentID int64,
		registryID int64, image string,
	) error
}

type registryHelper struct {
	fileManager filemanager.FileManager
	artifactDao store.ArtifactRepository
}

func NewRegistryHelper(
	fileManager filemanager.FileManager,
	artifactDao store.ArtifactRepository,
) RegistryHelper {
	return &registryHelper{
		fileManager: fileManager,
		artifactDao: artifactDao,
	}
}

func (h *registryHelper) GetIndexFilePathFromImageName(imageName string) string {
	length := len(imageName)
	switch {
	case length == 0:
		return imageName
	case length == 1:
		return fmt.Sprintf("index/1/%s", imageName)
	case length == 2:
		return fmt.Sprintf("index/2/%s", imageName)
	case length == 3:
		return fmt.Sprintf("index/3/%c/%s", imageName[0], imageName)
	default:
		return fmt.Sprintf("index/%s/%s/%s", imageName[0:2], imageName[2:4], imageName)
	}
}

func (h *registryHelper) UpdatePackageIndex(
	ctx context.Context, rootIdentifier string, rootParentID int64,
	registryID int64, image string,
) error {
	indexMetadataList, err := h.regeneratePackageIndex(ctx, registryID, image)
	if err != nil {
		return fmt.Errorf("failed to regenerate package index: %w", err)
	}
	return h.uploadIndexMetadata(
		ctx, rootIdentifier, rootParentID, registryID, image, indexMetadataList,
	)
}

func (h *registryHelper) regeneratePackageIndex(
	ctx context.Context, registryID int64, image string,
) ([]*cargometadata.IndexMetadata, error) {
	lastArtifactID := int64(0)
	artifactBatchLimit := 50
	versionMetadataList := []*cargometadata.IndexMetadata{}
	for {
		artifacts, err := h.artifactDao.GetArtifactsByRepoAndImageBatch(
			ctx, registryID, image, artifactBatchLimit, lastArtifactID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get artifacts: %w", err)
		}

		for _, a := range *artifacts {
			metadata := cargometadata.VersionMetadataDB{}
			err := json.Unmarshal(a.Metadata, &metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata for artifact %s: %w", a.Name, err)
			}

			deps := h.mapVersionDependenciesToIndexDependencies(metadata.Dependencies)

			features := metadata.Features
			// If there are no features, we initialize it to an empty map
			if features == nil {
				features = make(map[string][]string)
			}

			if len(metadata.GetFiles()) > 0 {
				versionMetadataList = append(versionMetadataList,
					&cargometadata.IndexMetadata{
						Name:         a.Name,
						Version:      a.Version,
						Checksum:     metadata.GetFiles()[0].Sha256,
						Dependencies: deps,
						Features:     features,
						Yanked:       metadata.Yanked,
					})
			}
			if a.ID > lastArtifactID {
				lastArtifactID = a.ID
			}
		}
		if len(*artifacts) < artifactBatchLimit {
			break
		}
	}
	return versionMetadataList, nil
}

func (h *registryHelper) uploadIndexMetadata(
	ctx context.Context, rootIdentifier string,
	rootParentID int64, registryID int64, image string,
	indexMetadataList []*cargometadata.IndexMetadata,
) error {
	fileName := image
	filePath := h.GetIndexFilePathFromImageName(image)

	metadataList := []string{}

	for _, metadata := range indexMetadataList {
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataList = append(metadataList, string(metadataJSON))
	}
	session, _ := request.AuthSessionFrom(ctx)
	_, err := h.fileManager.UploadFile(
		ctx, filePath, registryID, rootParentID, rootIdentifier, nil,
		io.NopCloser(strings.NewReader(strings.Join(metadataList, "\n"))), fileName, session.Principal.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to upload package index metadata: %w", err)
	}
	return nil
}

func (h *registryHelper) mapVersionDependenciesToIndexDependencies(
	dependencies []cargometadata.VersionDependency,
) []cargometadata.IndexDependency {
	// If there are no dependencies, we initialize it to an empty slice
	if dependencies == nil {
		return []cargometadata.IndexDependency{}
	}
	deps := make([]cargometadata.IndexDependency, len(dependencies))
	for i, dep := range dependencies {
		deps[i] = cargometadata.IndexDependency{
			VersionRequired: dep.VersionRequired,
			Dependency: cargometadata.Dependency{
				Name:               dep.Name,
				Features:           dep.Features,
				IsOptional:         dep.IsOptional,
				DefaultFeatures:    dep.DefaultFeatures,
				Target:             dep.Target,
				Kind:               dep.Kind,
				Registry:           dep.Registry,
				ExplicitNameInToml: dep.ExplicitNameInToml,
			},
		}
	}
	return deps
}

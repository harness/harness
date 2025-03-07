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

package pypi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/harness/gitness/registry/app/store/database"
)

// Metadata represents the metadata of a PyPI package.
func (c *controller) GetPackageMetadata(ctx context.Context, info ArtifactInfo, packageName string) (
	PackageMetadata,
	error,
) {
	registry, err := c.registryDao.GetByRootParentIDAndName(ctx, info.RootParentID, info.RegIdentifier)
	packageMetadata := PackageMetadata{}
	packageMetadata.Name = packageName
	packageMetadata.Files = []File{}

	if err != nil {
		return packageMetadata, err
	}

	artifacts, err := c.artifactDao.GetByRegistryIDAndImage(ctx, registry.ID, packageName)
	if err != nil {
		return packageMetadata, err
	}

	for _, artifact := range *artifacts {
		metadata := &database.PyPiMetadata{}
		err = json.Unmarshal(artifact.Metadata, metadata)
		if err != nil {
			return packageMetadata, err
		}

		for _, file := range metadata.Files {
			fileInfo := File{
				Name: file.Filename,
				FileURL: c.urlProvider.RegistryURL(ctx) + fmt.Sprintf(
					"/pkg/%s/%s/pypi/files/%s/%s/%s",
					info.RootIdentifier,
					info.RegIdentifier,
					packageName,
					artifact.Version,
					file.Filename,
				),
				RequiresPython: metadata.RequiresPython,
			}
			packageMetadata.Files = append(packageMetadata.Files, fileInfo)
		}
	}

	// Sort files by Name
	sort.Slice(packageMetadata.Files, func(i, j int) bool {
		return packageMetadata.Files[i].Name < packageMetadata.Files[j].Name
	})

	return packageMetadata, nil
}

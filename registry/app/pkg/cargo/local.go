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
	"fmt"
	"io"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/utils/cargo"
	"github.com/harness/gitness/store/database/dbtx"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type localRegistry struct {
	localBase      base.LocalBase
	fileManager    filemanager.FileManager
	proxyStore     store.UpstreamProxyConfigRepository
	tx             dbtx.Transactor
	registryDao    store.RegistryRepository
	imageDao       store.ImageRepository
	artifactDao    store.ArtifactRepository
	urlProvider    urlprovider.Provider
	registryHelper cargo.RegistryHelper
}

type LocalRegistry interface {
	Registry
}

func NewLocalRegistry(
	localBase base.LocalBase,
	fileManager filemanager.FileManager,
	proxyStore store.UpstreamProxyConfigRepository,
	tx dbtx.Transactor,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	urlProvider urlprovider.Provider,
	registryHelper cargo.RegistryHelper,
) LocalRegistry {
	return &localRegistry{
		localBase:      localBase,
		fileManager:    fileManager,
		proxyStore:     proxyStore,
		tx:             tx,
		registryDao:    registryDao,
		imageDao:       imageDao,
		artifactDao:    artifactDao,
		urlProvider:    urlProvider,
		registryHelper: registryHelper,
	}
}

func (c *localRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeCARGO}
}

func (c *localRegistry) UploadPackage(
	ctx context.Context, info cargotype.ArtifactInfo,
	metadata *cargometadata.VersionMetadata, crateFile io.ReadCloser,
) (*commons.ResponseHeaders, error) {
	// upload crate file
	response, err := c.uploadFile(ctx, info, metadata, crateFile)
	if err != nil {
		return response, fmt.Errorf("failed to upload crate file: %w", err)
	}

	// regenerate package index for cargo client to consume
	err = c.regeneratePackageIndex(ctx, info)
	if err != nil {
		return nil, fmt.Errorf("failed to update package index: %w", err)
	}
	return response, nil
}

func (c *localRegistry) uploadFile(
	ctx context.Context, info cargotype.ArtifactInfo,
	metadata *cargometadata.VersionMetadata, fileReader io.ReadCloser,
) (responseHeaders *commons.ResponseHeaders, err error) {
	fileName := getCrateFileName(info.Image, info.Version)
	path := getCrateFilePath(info.Image, info.Version)

	response, _, err := c.localBase.Upload(
		ctx, info.ArtifactInfo, fileName, info.Version, path, fileReader,
		&cargometadata.VersionMetadataDB{
			VersionMetadata: *metadata,
		})
	if err != nil {
		return response, fmt.Errorf("failed to upload file: %w", err)
	}

	return response, nil
}

func (c *localRegistry) regeneratePackageIndex(
	ctx context.Context, info cargotype.ArtifactInfo,
) error {
	err := c.registryHelper.UpdatePackageIndex(
		ctx, info.RootIdentifier, info.RootParentID, info.RegistryID, info.Image,
	)
	if err != nil {
		return fmt.Errorf("failed to update package index: %w", err)
	}
	return nil
}

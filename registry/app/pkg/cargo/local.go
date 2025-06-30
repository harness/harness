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
	"net/http"

	"github.com/harness/gitness/app/api/request"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/utils/cargo"
	"github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/store/database/dbtx"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type localRegistry struct {
	localBase             base.LocalBase
	fileManager           filemanager.FileManager
	proxyStore            store.UpstreamProxyConfigRepository
	tx                    dbtx.Transactor
	registryDao           store.RegistryRepository
	imageDao              store.ImageRepository
	artifactDao           store.ArtifactRepository
	urlProvider           urlprovider.Provider
	registryHelper        cargo.RegistryHelper
	artifactEventReporter *registryevents.Reporter
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
	artifactEventReporter *registryevents.Reporter,
) LocalRegistry {
	return &localRegistry{
		localBase:             localBase,
		fileManager:           fileManager,
		proxyStore:            proxyStore,
		tx:                    tx,
		registryDao:           registryDao,
		imageDao:              imageDao,
		artifactDao:           artifactDao,
		urlProvider:           urlProvider,
		registryHelper:        registryHelper,
		artifactEventReporter: artifactEventReporter,
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

	// publish artifact created event
	session, _ := request.AuthSessionFrom(ctx)
	payload := webhook.GetArtifactCreatedPayloadForCommonArtifacts(
		session.Principal.ID,
		info.RegistryID,
		artifact.PackageTypeCARGO,
		info.Image,
		info.Version,
	)
	c.artifactEventReporter.ArtifactCreated(ctx, &payload)

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

func (c *localRegistry) DownloadPackageIndex(
	ctx context.Context, info cargotype.ArtifactInfo,
	filePath string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	path := getPackageIndexFilePath(filePath)
	response, fileReader, redirectURL, err := c.downloadFileInternal(ctx, info, path)
	if err != nil {
		return response, nil, nil, "", fmt.Errorf("failed to download package index file: %w", err)
	}
	return response, fileReader, nil, redirectURL, nil
}

func (c *localRegistry) DownloadPackage(
	ctx context.Context, info cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	path := getCrateFilePath(info.Image, info.Version)
	response, fileReader, redirectURL, err := c.downloadFileInternal(ctx, info, path)
	if err != nil {
		return response, nil, nil, "", fmt.Errorf("failed to download package file: %w", err)
	}
	return response, fileReader, nil, redirectURL, nil
}

func (c *localRegistry) downloadFileInternal(
	ctx context.Context, info cargotype.ArtifactInfo, path string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	fileReader, _, redirectURL, err := c.fileManager.DownloadFile(ctx, path, info.RegistryID,
		info.RegIdentifier, info.RootIdentifier, true)
	if err != nil {
		return responseHeaders, nil, "", fmt.Errorf("failed to download file %s: %w", path, err)
	}
	return responseHeaders, fileReader, redirectURL, nil
}

func (c *localRegistry) UpdateYank(
	ctx context.Context, info cargotype.ArtifactInfo,
	yanked bool,
) (*commons.ResponseHeaders, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	// update yanked status in the database
	err := c.updateYankInternal(ctx, info, yanked)
	if err != nil {
		return responseHeaders, fmt.Errorf("failed to update yank status: %w", err)
	}

	// regenerate package index for cargo client to consume
	err = c.regeneratePackageIndex(ctx, info)
	if err != nil {
		return responseHeaders, fmt.Errorf("failed to update package index: %w", err)
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil
}

func (c *localRegistry) updateYankInternal(ctx context.Context, info cargotype.ArtifactInfo, yanked bool) error {
	a, err := c.artifactDao.GetByRegistryImageAndVersion(
		ctx, info.RegistryID, info.Image, info.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to get artifact by image and version: %w", err)
	}

	metadata := cargometadata.VersionMetadataDB{}
	err = json.Unmarshal(a.Metadata, &metadata)
	if err != nil {
		return fmt.Errorf("failed to unmarshal metadata for artifact %s: %w", a.Version, err)
	}
	// Mark the version as yanked
	metadata.Yanked = yanked
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	// Update the artifact with the new metadata
	a.Metadata = metadataJSON
	_, err = c.artifactDao.CreateOrUpdate(ctx, a)
	if err != nil {
		return fmt.Errorf("failed to update artifact: %w", err)
	}
	return nil
}

func (c *localRegistry) RegeneratePackageIndex(
	ctx context.Context, info cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	// regenerate package index for cargo client to consume
	err := c.regeneratePackageIndex(ctx, info)
	if err != nil {
		return responseHeaders, fmt.Errorf("failed to update package index: %w", err)
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil
}

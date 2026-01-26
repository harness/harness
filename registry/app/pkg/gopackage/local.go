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
	"net/http"
	"path/filepath"

	"github.com/harness/gitness/app/api/request"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	gopackagemetadata "github.com/harness/gitness/registry/app/metadata/gopackage"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	gopackageutils "github.com/harness/gitness/registry/app/pkg/gopackage/utils"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/services/webhook"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type localRegistry struct {
	localBase              base.LocalBase
	fileManager            filemanager.FileManager
	proxyStore             store.UpstreamProxyConfigRepository
	tx                     dbtx.Transactor
	registryDao            store.RegistryRepository
	imageDao               store.ImageRepository
	artifactDao            store.ArtifactRepository
	urlProvider            urlprovider.Provider
	artifactEventReporter  *registryevents.Reporter
	postProcessingReporter *asyncprocessing.Reporter
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
	artifactEventReporter *registryevents.Reporter,
	postProcessingReporter *asyncprocessing.Reporter,
) LocalRegistry {
	return &localRegistry{
		localBase:              localBase,
		fileManager:            fileManager,
		proxyStore:             proxyStore,
		tx:                     tx,
		registryDao:            registryDao,
		imageDao:               imageDao,
		artifactDao:            artifactDao,
		urlProvider:            urlProvider,
		artifactEventReporter:  artifactEventReporter,
		postProcessingReporter: postProcessingReporter,
	}
}

func (c *localRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeGO}
}

func (c *localRegistry) UploadPackage(
	ctx context.Context, info gopackagetype.ArtifactInfo,
	modfile io.ReadCloser, zipfile io.ReadCloser,
) (*commons.ResponseHeaders, error) {
	// Check if version exists
	checkIfVersionExists, err := c.localBase.CheckIfVersionExists(ctx, info)
	if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to check if version exists: %w", err)
	}
	if checkIfVersionExists {
		return nil, fmt.Errorf("version %s already exists", info.Version)
	}

	// Get file path
	filePath, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, info.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get file path: %w", err)
	}
	// upload .zip
	zipFileName := info.Version + ".zip"
	zipFilePath := filepath.Join(filePath, zipFileName)

	response, err := c.uploadFile(ctx, info, &info.Metadata, zipfile, zipFileName, zipFilePath)
	if err != nil {
		return response, fmt.Errorf("failed to upload zip file: %w", err)
	}
	// upload .mod
	modFileName := info.Version + ".mod"
	modFilePath := filepath.Join(filePath, modFileName)
	response, err = c.uploadFile(ctx, info, &info.Metadata, modfile, modFileName, modFilePath)
	if err != nil {
		return response, fmt.Errorf("failed to upload mod file: %w", err)
	}
	// upload .info
	infoFileName := info.Version + ".info"
	infoFilePath := filepath.Join(filePath, infoFileName)
	infoFile, err := metadataToReadCloser(info.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to convert metadata to io.ReadCloser: %w", err)
	}
	response, err = c.uploadFile(ctx, info, &info.Metadata, infoFile, infoFileName, infoFilePath)
	if err != nil {
		return response, fmt.Errorf("failed to upload info file: %w", err)
	}

	// publish artifact created event
	c.publishArtifactCreatedEvent(ctx, info)

	// regenerate package index for go package
	c.regeneratePackageIndex(ctx, info)

	// regenerate package metadata for go package
	c.regeneratePackageMetadata(ctx, info)
	return response, nil
}

func metadataToReadCloser(meta gopackagemetadata.VersionMetadata) (io.ReadCloser, error) {
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (c *localRegistry) uploadFile(
	ctx context.Context, info gopackagetype.ArtifactInfo,
	metadata *gopackagemetadata.VersionMetadata, fileReader io.ReadCloser,
	filename string, path string,
) (responseHeaders *commons.ResponseHeaders, err error) {
	response, _, err := c.localBase.Upload(
		ctx, info.ArtifactInfo, filename, info.Version, path, fileReader,
		&gopackagemetadata.VersionMetadataDB{
			VersionMetadata: *metadata,
		})
	if err != nil {
		return response, fmt.Errorf("failed to upload file %s: %w", filename, err)
	}

	return response, nil
}

func (c *localRegistry) publishArtifactCreatedEvent(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) {
	session, _ := request.AuthSessionFrom(ctx)
	payload := webhook.GetArtifactCreatedPayloadForCommonArtifacts(
		session.Principal.ID,
		info.RegistryID,
		artifact.PackageTypeGO,
		info.Image,
		info.Version,
	)
	c.artifactEventReporter.ArtifactCreated(ctx, &payload)
}

func (c *localRegistry) regeneratePackageIndex(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) {
	c.postProcessingReporter.BuildPackageIndex(ctx, info.RegistryID, info.Image)
}

func (c *localRegistry) regeneratePackageMetadata(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) {
	c.postProcessingReporter.BuildPackageMetadata(
		ctx, info.RegistryID, info.Image, info.Version,
	)
}

func (c *localRegistry) DownloadPackageFile(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	if info.Version == "" {
		// If version is empty, it is an index file
		return c.DownloadPackageIndex(ctx, info)
	}
	if info.Version == LatestVersionKey {
		// Download latest version
		return c.DownloadPackageLatestVersionInfo(ctx, info)
	}
	path, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, info.Version)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get file path: %w", err)
	}
	filePath := filepath.Join(path, info.FileName)
	response, fileReader, redirectURL, err := c.downloadFileInternal(ctx, info, filePath)
	if err != nil {
		return response, nil, nil, "", fmt.Errorf("failed to download package file: %w", err)
	}
	return response, fileReader, nil, redirectURL, nil
}

func (c *localRegistry) DownloadPackageIndex(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	filePath := gopackageutils.GetIndexFilePath(info.Image)
	response, fileReader, redirectURL, err := c.downloadFileInternal(ctx, info, filePath)
	if err != nil {
		return response, nil, nil, "", fmt.Errorf("failed to download package index: %w", err)
	}
	return response, fileReader, nil, redirectURL, nil
}

func (c *localRegistry) DownloadPackageLatestVersionInfo(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	image, err := c.imageDao.GetByName(ctx, info.RegistryID, info.Image)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get image: %w", err)
	}

	artifact, err := c.artifactDao.GetLatestByImageID(ctx, image.ID)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get latest artifact: %w", err)
	}

	info.Version = artifact.Version
	info.FileName = artifact.Version + ".info"

	// Prevent infinite recursion
	if artifact.Version == LatestVersionKey {
		return nil, nil, nil, "", fmt.Errorf("artifact version cannot be %s", LatestVersionKey)
	}

	return c.DownloadPackageFile(ctx, info)
}

func (c *localRegistry) downloadFileInternal(
	ctx context.Context, info gopackagetype.ArtifactInfo, path string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	fileReader, _, redirectURL, err := c.fileManager.DownloadFileByPath(ctx, path, info.RegistryID,
		info.RegIdentifier, info.RootIdentifier, true)
	if err != nil {
		return responseHeaders, nil, "", fmt.Errorf("failed to download file %s: %w", path, err)
	}
	return responseHeaders, fileReader, redirectURL, nil
}

func (c *localRegistry) RegeneratePackageIndex(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	c.regeneratePackageIndex(ctx, info)
	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil
}

func (c *localRegistry) RegeneratePackageMetadata(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	c.regeneratePackageMetadata(ctx, info)
	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil
}

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

package generic

import (
	"context"
	"fmt"
	"io"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	generic2 "github.com/harness/gitness/registry/app/metadata/generic"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type localRegistry struct {
	localBase   base.LocalBase
	fileManager filemanager.FileManager
	proxyStore  store.UpstreamProxyConfigRepository
	tx          dbtx.Transactor
	registryDao store.RegistryRepository
	imageDao    store.ImageRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
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
) LocalRegistry {
	return &localRegistry{
		localBase:   localBase,
		fileManager: fileManager,
		proxyStore:  proxyStore,
		tx:          tx,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		urlProvider: urlProvider,
	}
}

func (c *localRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeGENERIC}
}

func (c *localRegistry) PutFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	reader io.ReadCloser,
	contentType string,
) (*commons.ResponseHeaders, string, error) {
	completePath := pkg.JoinWithSeparator("/", info.Image, info.Version, info.FilePath)
	headers, sha256, err := c.localBase.Upload(ctx, info.ArtifactInfo, info.FileName, info.Version, completePath,
		reader, &generic2.GenericMetadata{})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to upload file: %q, %q, %q", info.FileName, info.Version,
			completePath)
		return nil, "", fmt.Errorf("failed to upload file: %w", err)
	}
	log.Ctx(ctx).Info().Str("sha256", sha256).Msg("Successfully uploaded file. content type: " + contentType)
	return headers, sha256, nil
}

func (c *localRegistry) DownloadFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	filePath string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	download, reader, url, err := c.localBase.Download(ctx, info.ArtifactInfo, info.Version, filePath)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("Failed to download file: %q, %q, %q", info.FileName, info.Version, filePath)
		return nil, nil, nil, "", fmt.Errorf("failed to download file: %w", err)
	}
	return download, reader, nil, url, err
}

func (c *localRegistry) DeleteFile(ctx context.Context, info generic.ArtifactInfo) (*commons.ResponseHeaders, error) {
	return c.localBase.DeleteFile(ctx, info, info.FilePath)
}

func (c *localRegistry) HeadFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	filePath string,
) (headers *commons.ResponseHeaders, err error) {
	return c.localBase.ExistsE(ctx, info, filePath)
}

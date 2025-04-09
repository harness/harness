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

package npm

import (
	"context"
	"io"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	npm2 "github.com/harness/gitness/registry/app/pkg/npm"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

// Controller handles PyPI package operations.
type controller struct {
	fileManager  filemanager.FileManager
	proxyStore   store.UpstreamProxyConfigRepository
	tx           dbtx.Transactor
	registryDao  store.RegistryRepository
	imageDao     store.ImageRepository
	artifactDao  store.ArtifactRepository
	urlProvider  urlprovider.Provider
	downloadStat store.DownloadStatRepository
	local        npm2.LocalRegistry
	proxy        npm2.Proxy
}

type Controller interface {
	UploadPackageFile(
		ctx context.Context,
		info npm.ArtifactInfo,
		file io.ReadCloser,
	) *PutArtifactResponse

	DownloadPackageFile(
		ctx context.Context,
		info npm.ArtifactInfo,
	) *GetArtifactResponse

	GetPackageMetadata(ctx context.Context, info npm.ArtifactInfo) *GetMetadataResponse

	HeadPackageFileByName(ctx context.Context, info npm.ArtifactInfo) *HeadMetadataResponse

	ListTags(
		ctx context.Context,
		info npm.ArtifactInfo,
	) *ListTagResponse

	AddTag(
		ctx context.Context,
		info npm.ArtifactInfo,
	) *ListTagResponse

	DeleteTag(
		ctx context.Context,
		info npm.ArtifactInfo,
	) *ListTagResponse

	DeleteVersion(
		ctx context.Context,
		info npm.ArtifactInfo,
	) *DeleteEntityResponse

	DeletePackage(
		ctx context.Context,
		info npm.ArtifactInfo,
	) *DeleteEntityResponse
}

// NewController creates a new PyPI controller.
func NewController(
	proxyStore store.UpstreamProxyConfigRepository,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	fileManager filemanager.FileManager,
	downloadStatsDao store.DownloadStatRepository,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	local npm2.LocalRegistry,
	proxy npm2.Proxy,
) Controller {
	return &controller{
		proxyStore:   proxyStore,
		registryDao:  registryDao,
		imageDao:     imageDao,
		artifactDao:  artifactDao,
		downloadStat: downloadStatsDao,
		fileManager:  fileManager,
		tx:           tx,
		urlProvider:  urlProvider,
		local:        local,
		proxy:        proxy,
	}
}

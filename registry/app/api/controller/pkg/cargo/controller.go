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
	"io"

	"github.com/harness/gitness/app/services/publicaccess"
	refcache2 "github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg/cargo"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

type Controller interface {
	GetRegistryConfig(ctx context.Context, info *cargotype.ArtifactInfo) (*GetRegistryConfigResponse, error)
	SearchPackage(
		ctx context.Context,
		info *cargotype.ArtifactInfo,
		requestInfo *cargotype.SearchPackageRequestParams,
	) (*SearchPackageResponse, error)
	DownloadPackageIndex(
		ctx context.Context, info *cargotype.ArtifactInfo, filePath string,
	) *GetPackageIndexResponse
	RegeneratePackageIndex(
		ctx context.Context, info *cargotype.ArtifactInfo,
	) (*RegeneratePackageIndexResponse, error)
	DownloadPackage(
		ctx context.Context, info *cargotype.ArtifactInfo,
	) *GetPackageResponse
	UploadPackage(
		ctx context.Context,
		info *cargotype.ArtifactInfo,
		metadata *cargometadata.VersionMetadata,
		fileReader io.ReadCloser,
	) (*UploadArtifactResponse, error)
	UpdateYank(ctx context.Context, info *cargotype.ArtifactInfo, yank bool) (*UpdateYankResponse, error)
}

// Controller handles Cargo package operations.
type controller struct {
	fileManager         filemanager.FileManager
	proxyStore          store.UpstreamProxyConfigRepository
	tx                  dbtx.Transactor
	registryDao         store.RegistryRepository
	registryFinder      refcache.RegistryFinder
	imageDao            store.ImageRepository
	artifactDao         store.ArtifactRepository
	urlProvider         urlprovider.Provider
	local               cargo.LocalRegistry
	proxy               cargo.Proxy
	publicAccessService publicaccess.Service
	spaceFinder         refcache2.SpaceFinder
}

// NewController creates a new Cargo controller.
func NewController(
	proxyStore store.UpstreamProxyConfigRepository,
	registryDao store.RegistryRepository,
	registryFinder refcache.RegistryFinder,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	local cargo.LocalRegistry,
	proxy cargo.Proxy,
	publicAccessService publicaccess.Service,
	spaceFinder refcache2.SpaceFinder,
) Controller {
	return &controller{
		proxyStore:          proxyStore,
		registryDao:         registryDao,
		registryFinder:      registryFinder,
		imageDao:            imageDao,
		artifactDao:         artifactDao,
		fileManager:         fileManager,
		tx:                  tx,
		urlProvider:         urlProvider,
		local:               local,
		proxy:               proxy,
		publicAccessService: publicAccessService,
		spaceFinder:         spaceFinder,
	}
}

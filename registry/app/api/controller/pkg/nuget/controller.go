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

package nuget

import (
	"context"
	"io"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/nuget"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

type Controller interface {
	UploadPackage(ctx context.Context, info nugettype.ArtifactInfo, fileReader io.ReadCloser,
		fileBundleType nuget.FileBundleType) *PutArtifactResponse

	DownloadPackage(ctx context.Context, info nugettype.ArtifactInfo) *GetArtifactResponse

	DeletePackage(ctx context.Context, info nugettype.ArtifactInfo) *DeleteArtifactResponse

	GetServiceEndpoint(ctx context.Context,
		info nugettype.ArtifactInfo) *GetServiceEndpointResponse

	GetServiceEndpointV2(ctx context.Context,
		info nugettype.ArtifactInfo) *GetServiceEndpointV2Response

	ListPackageVersion(ctx context.Context, info nugettype.ArtifactInfo) *ListPackageVersionResponse

	ListPackageVersionV2(ctx context.Context, info nugettype.ArtifactInfo) *ListPackageVersionV2Response

	GetPackageMetadata(ctx context.Context, info nugettype.ArtifactInfo) *GetPackageMetadataResponse

	GetPackageVersionMetadataV2(ctx context.Context, info nugettype.ArtifactInfo) *GetPackageVersionMetadataV2Response

	GetPackageVersionMetadata(ctx context.Context, info nugettype.ArtifactInfo) *GetPackageVersionMetadataResponse

	CountPackageVersionV2(ctx context.Context, info nugettype.ArtifactInfo) *EntityCountResponse

	SearchPackage(ctx context.Context, info nugettype.ArtifactInfo, searchTerm string,
		limit, offset int) *SearchPackageResponse

	SearchPackageV2(ctx context.Context, info nugettype.ArtifactInfo, searchTerm string,
		limit, offset int) *SearchPackageV2Response

	CountPackageV2(ctx context.Context, info nugettype.ArtifactInfo, searchTerm string) *EntityCountResponse

	GetServiceMetadataV2(ctx context.Context, info nugettype.ArtifactInfo) *GetServiceMetadataV2Response
}

// Controller handles Python package operations.
type controller struct {
	fileManager filemanager.FileManager
	proxyStore  store.UpstreamProxyConfigRepository
	tx          dbtx.Transactor
	registryDao store.RegistryRepository
	imageDao    store.ImageRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
	local       nuget.LocalRegistry
	proxy       nuget.Proxy
}

// NewController creates a new Python controller.
func NewController(
	proxyStore store.UpstreamProxyConfigRepository,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	local nuget.LocalRegistry,
	proxy nuget.Proxy,
) Controller {
	return &controller{
		proxyStore:  proxyStore,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		fileManager: fileManager,
		tx:          tx,
		urlProvider: urlProvider,
		local:       local,
		proxy:       proxy,
	}
}

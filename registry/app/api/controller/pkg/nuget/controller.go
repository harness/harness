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
	UploadPackage(
		ctx context.Context,
		info nugettype.ArtifactInfo,
		fileReader io.ReadCloser,
	) *PutArtifactResponse

	DownloadPackage(ctx context.Context, info nugettype.ArtifactInfo) *GetArtifactResponse

	GetServiceEndpoint(ctx context.Context,
		info nugettype.ArtifactInfo) *GetServiceEndpointResponse

	ListPackageVersion(ctx context.Context, info nugettype.ArtifactInfo) *ListPackageVersionResponse

	GetPackageMetadata(ctx context.Context, info nugettype.ArtifactInfo) *GetPackageMetadataResponse

	GetPackageVersionMetadata(ctx context.Context, info nugettype.ArtifactInfo) *GetPackageVersionMetadataResponse
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
	}
}

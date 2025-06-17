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

package rpm

import (
	"context"
	"mime/multipart"

	urlprovider "github.com/harness/gitness/app/url"
	registrypostprocessingevents "github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/rpm"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

type Controller interface {
	UploadPackageFile(
		ctx context.Context,
		info rpmtype.ArtifactInfo,
		file multipart.Part,
		fileName string,
	) *PutArtifactResponse

	DownloadPackageFile(
		ctx context.Context,
		info rpmtype.ArtifactInfo,
	) *GetArtifactResponse

	GetRepoData(
		ctx context.Context,
		info rpmtype.ArtifactInfo,
		fileName string,
	) *GetRepoDataResponse
}

// Controller handles RPM package operations.
type controller struct {
	fileManager            filemanager.FileManager
	proxyStore             store.UpstreamProxyConfigRepository
	tx                     dbtx.Transactor
	registryDao            store.RegistryRepository
	imageDao               store.ImageRepository
	artifactDao            store.ArtifactRepository
	urlProvider            urlprovider.Provider
	local                  rpm.LocalRegistry
	proxy                  rpm.Proxy
	upstreamProxyStore     store.UpstreamProxyConfigRepository
	postProcessingReporter *registrypostprocessingevents.Reporter
}

// NewController creates a new RPM controller.
func NewController(
	proxyStore store.UpstreamProxyConfigRepository,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	fileManager filemanager.FileManager,
	tx dbtx.Transactor,
	urlProvider urlprovider.Provider,
	local rpm.LocalRegistry,
	proxy rpm.Proxy,
	upstreamProxyStore store.UpstreamProxyConfigRepository,
	postProcessingReporter *registrypostprocessingevents.Reporter,
) Controller {
	return &controller{
		proxyStore:             proxyStore,
		registryDao:            registryDao,
		imageDao:               imageDao,
		artifactDao:            artifactDao,
		fileManager:            fileManager,
		tx:                     tx,
		urlProvider:            urlProvider,
		local:                  local,
		proxy:                  proxy,
		upstreamProxyStore:     upstreamProxyStore,
		postProcessingReporter: postProcessingReporter,
	}
}

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
	"io"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
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
	registryHelper RegistryHelper
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
	registryHelper RegistryHelper,
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
	return []artifact.PackageType{artifact.PackageTypeRPM}
}

func (c *localRegistry) UploadPackageFile(
	ctx context.Context,
	info rpmtype.ArtifactInfo,
	file io.Reader,
	fileName string,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	return c.registryHelper.UploadPackage(ctx, info, file, fileName)
}

func (c *localRegistry) GetRepoData(
	ctx context.Context,
	info rpmtype.ArtifactInfo,
	fileName string,
) (*commons.ResponseHeaders,
	*storage.FileReader,
	io.ReadCloser,
	string,
	error,
) {
	return getRepoData(ctx, info, fileName, c.fileManager)
}

func (c *localRegistry) DownloadPackageFile(
	ctx context.Context,
	info rpmtype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return downloadPackageFile(ctx, info, c.localBase)
}

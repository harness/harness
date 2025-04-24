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
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	rpmutil "github.com/harness/gitness/registry/app/utils/rpm"
	"github.com/harness/gitness/registry/services/index"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

type localRegistry struct {
	localBase            base.LocalBase
	fileManager          filemanager.FileManager
	proxyStore           store.UpstreamProxyConfigRepository
	tx                   dbtx.Transactor
	registryDao          store.RegistryRepository
	imageDao             store.ImageRepository
	artifactDao          store.ArtifactRepository
	urlProvider          urlprovider.Provider
	registryIndexService index.Service
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
	registryIndexService index.Service,
) LocalRegistry {
	return &localRegistry{
		localBase:            localBase,
		fileManager:          fileManager,
		proxyStore:           proxyStore,
		tx:                   tx,
		registryDao:          registryDao,
		imageDao:             imageDao,
		artifactDao:          artifactDao,
		urlProvider:          urlProvider,
		registryIndexService: registryIndexService,
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
	file multipart.Part,
	fileName string,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	fileInfo, tempFileName, err := c.fileManager.UploadTempFile(ctx, info.RootIdentifier, nil, fileName, &file)
	if err != nil {
		return nil, "", err
	}
	r, _, err := c.fileManager.DownloadTempFile(ctx, fileInfo.Size, tempFileName, info.RootIdentifier)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	p, err := rpmutil.ParsePackage(r)
	if err != nil {
		log.Printf("failded to parse rpm package: %v", err)
		return nil, "", err
	}

	info.Image = p.Name
	info.Version = p.Version + "." + p.FileMetadata.Architecture
	info.Metadata = rpmmetadata.Metadata{
		VersionMetadata: *p.VersionMetadata,
		FileMetadata:    *p.FileMetadata,
	}

	rpmFileName := fmt.Sprintf("%s-%s.%s.rpm", p.Name, p.Version, p.FileMetadata.Architecture)
	path := fmt.Sprintf("%s/%s/%s/%s", p.Name, p.Version, p.FileMetadata.Architecture, rpmFileName)
	fileInfo.Filename = rpmFileName
	rs, sha256, err := c.localBase.MoveTempFile(ctx, info.ArtifactInfo, tempFileName, info.Version, path,
		&rpmmetadata.RpmMetadata{
			Metadata: info.Metadata,
		}, fileInfo)

	if err != nil {
		return nil, "", err
	}

	//TODO: make it async / atomic operation, implement artifact status (sync successful, sync failed..... statuses)
	err = c.registryIndexService.RegenerateRpmRepoData(ctx, info.RegistryID, info.RootParentID, info.RootIdentifier)
	if err != nil {
		return nil, "", err
	}
	return rs, sha256, err
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
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	fileReader, _, redirectURL, err := c.fileManager.DownloadFile(
		ctx, "/"+rpmutil.RepoDataPrefix+fileName, info.RegistryID, info.RegIdentifier, info.RootIdentifier,
	)
	if err != nil {
		return responseHeaders, nil, nil, "", err
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, nil, redirectURL, nil
}

func (c *localRegistry) DownloadPackageFile(
	ctx context.Context,
	info rpmtype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	headers, fileReader, redirectURL, err := c.localBase.Download(
		ctx, info.ArtifactInfo,
		fmt.Sprintf("%s/%s", info.Version, info.Arch),
		info.FileName,
	)
	if err != nil {
		return nil, nil, nil, "", err
	}
	return headers, fileReader, nil, redirectURL, nil
}

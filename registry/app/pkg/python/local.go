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

package python

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"sort"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	pythonmetadata "github.com/harness/gitness/registry/app/metadata/python"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"
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
	return []artifact.PackageType{artifact.PackageTypePYTHON}
}

func (c *localRegistry) DownloadPackageFile(ctx context.Context, info pythontype.ArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	[]error,
) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	path := "/" + info.Image + "/" + info.Version + "/" + info.Filename

	fileReader, _, redirectURL, err := c.fileManager.DownloadFile(ctx, path, types.Registry{
		ID:   info.RegistryID,
		Name: info.RegIdentifier,
	}, info.RootIdentifier)
	if err != nil {
		return responseHeaders, nil, "", []error{err}
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, redirectURL, nil
}

// Metadata represents the metadata of a Python package.
func (c *localRegistry) GetPackageMetadata(
	ctx context.Context,
	info pythontype.ArtifactInfo,
) (pythontype.PackageMetadata, error) {
	registry, err := c.registryDao.GetByRootParentIDAndName(ctx, info.RootParentID, info.RegIdentifier)
	packageMetadata := pythontype.PackageMetadata{}
	packageMetadata.Name = info.Image
	packageMetadata.Files = []pythontype.File{}

	if err != nil {
		return packageMetadata, err
	}

	artifacts, err := c.artifactDao.GetByRegistryIDAndImage(ctx, registry.ID, info.Image)
	if err != nil {
		return packageMetadata, err
	}

	for _, artifact := range *artifacts {
		metadata := &pythonmetadata.PythonMetadata{}
		err = json.Unmarshal(artifact.Metadata, metadata)
		if err != nil {
			return packageMetadata, err
		}

		for _, file := range metadata.Files {
			fileInfo := pythontype.File{
				Name: file.Filename,
				FileURL: c.urlProvider.RegistryURL(ctx) + fmt.Sprintf(
					"/pkg/%s/%s/python/files/%s/%s/%s",
					info.RootIdentifier,
					info.RegIdentifier,
					info.Image,
					artifact.Version,
					file.Filename,
				),
				RequiresPython: metadata.RequiresPython,
			}
			packageMetadata.Files = append(packageMetadata.Files, fileInfo)
		}
	}

	// Sort files by Name
	sort.Slice(packageMetadata.Files, func(i, j int) bool {
		return packageMetadata.Files[i].Name < packageMetadata.Files[j].Name
	})

	return packageMetadata, nil
}

func (c *localRegistry) UploadPackageFile(
	ctx context.Context,
	info pythontype.ArtifactInfo,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (headers *commons.ResponseHeaders, sha256 string, err errcode.Error) {
	path := info.Image + "/" + info.Metadata.Version + "/" + fileHeader.Filename
	return c.localBase.Upload(ctx, info.ArtifactInfo, fileHeader.Filename, info.Metadata.Version, path, file,
		&pythonmetadata.PythonMetadata{
			Metadata: info.Metadata,
		})
}

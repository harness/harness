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
	"io"
	"mime/multipart"
	"sort"
	"strings"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	pythonmetadata "github.com/harness/gitness/registry/app/metadata/python"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/remote/adapter/commons/pypi"
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
	return []artifact.PackageType{artifact.PackageTypePYTHON}
}

func (c *localRegistry) DownloadPackageFile(
	ctx context.Context,
	info pythontype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	headers, fileReader, redirectURL, err := c.localBase.Download(ctx, info.ArtifactInfo, info.Version,
		info.Filename)
	if err != nil {
		return nil, nil, nil, "", err
	}
	return headers, fileReader, nil, redirectURL, nil
}

// GetPackageMetadata Metadata represents the metadata of a Python package.
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

	if len(*artifacts) == 0 {
		return packageMetadata, errors.NotFoundf("no artifacts found for registry %s and image %s", info.RegIdentifier,
			info.Image)
	}

	for _, artifact := range *artifacts {
		metadata := &pythonmetadata.PythonMetadata{}
		err = json.Unmarshal(artifact.Metadata, metadata)
		if err != nil {
			return packageMetadata, err
		}

		for _, file := range metadata.Files {
			pkgURL := c.urlProvider.PackageURL(
				ctx,
				info.RootIdentifier+"/"+info.RegIdentifier,
				"python",
			)
			fileInfo := pythontype.File{
				Name: file.Filename,
				FileURL: fmt.Sprintf(
					"%s/files/%s/%s/%s",
					pkgURL,
					info.Image,
					artifact.Version,
					file.Filename,
				),
				RequiresPython: metadata.RequiresPython,
			}
			packageMetadata.Files = append(packageMetadata.Files, fileInfo)
		}
	}

	sortPackageMetadata(ctx, packageMetadata)
	return packageMetadata, nil
}

func sortPackageMetadata(ctx context.Context, metadata pythontype.PackageMetadata) {
	sort.Slice(metadata.Files, func(i, j int) bool {
		version1 := pypi.GetPyPIVersion(metadata.Files[i].Name)
		version2 := pypi.GetPyPIVersion(metadata.Files[j].Name)
		if version1 == "" || version2 == "" || version1 == version2 {
			return metadata.Files[i].Name < metadata.Files[j].Name
		}

		vi := parseVersion(ctx, version1)
		vj := parseVersion(ctx, version2)

		for k := 0; k < len(vi) && k < len(vj); k++ {
			if vi[k] != vj[k] {
				return vi[k] < vj[k]
			}
		}

		return len(vi) < len(vj)
	})
}

func parseVersion(ctx context.Context, version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))
	for i, part := range parts {
		num, err := pkg.ExtractFirstNumber(part)
		if err != nil {
			log.Debug().Ctx(ctx).Msgf("failed to parse version %s, part %s: %v", version, part, err)
			continue
		}
		result[i] = num
	}
	return result
}

func (c *localRegistry) UploadPackageFile(
	ctx context.Context,
	info pythontype.ArtifactInfo,
	file multipart.File,
	filename string,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	path := pkg.JoinWithSeparator("/", info.Image, info.Metadata.Version, filename)
	return c.localBase.UploadFile(ctx, info.ArtifactInfo, filename, info.Metadata.Version, path, file,
		&pythonmetadata.PythonMetadata{
			Metadata: info.Metadata,
		})
}

func (c *localRegistry) UploadPackageFileReader(
	ctx context.Context,
	info pythontype.ArtifactInfo,
	file io.ReadCloser,
	filename string,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	path := pkg.JoinWithSeparator("/", info.Image, info.Metadata.Version, filename)
	return c.localBase.Upload(ctx, info.ArtifactInfo, filename, info.Metadata.Version, path, file,
		&pythonmetadata.PythonMetadata{
			Metadata: info.Metadata,
		})
}

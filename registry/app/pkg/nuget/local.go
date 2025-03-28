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
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

var IDMatch = regexp.MustCompile(`\A\w+(?:[.-]\w+)*\z`)

type PackageType int

const (
	// DependencyPackage represents a package (*.nupkg).
	DependencyPackage PackageType = iota + 1
	// SymbolsPackage represents a symbol package (*.snupkg).
	SymbolsPackage
)

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

func (c *localRegistry) GetServiceEndpoint(ctx context.Context,
	info nugettype.ArtifactInfo) *nugettype.ServiceEndpoint {
	baseURL := c.urlProvider.RegistryURL(ctx, "pkg", info.RootIdentifier, info.RegIdentifier, "nuget")
	serviceEndpoints := buildServiceEndpoint(baseURL)
	return serviceEndpoints
}

func (c *localRegistry) UploadPackage(ctx context.Context,
	info nugettype.ArtifactInfo,
	fileReader io.ReadCloser,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	metadata, err := c.buildMetadata(info, fileReader)
	if err != nil {
		return headers, "", err
	}
	fileName := strings.ToLower(fmt.Sprintf("%s.%s.nupkg",
		metadata.PackageMetadata.ID, metadata.PackageMetadata.Version))
	info.Metadata = metadata
	info.Filename = fileName
	info.Version = metadata.PackageMetadata.Version
	info.Image = metadata.PackageMetadata.ID
	path := info.Image + "/" + info.Version + "/" + fileName

	return c.localBase.Upload(ctx, info.ArtifactInfo, fileName, info.Version, path, fileReader,
		&nugetmetadata.NugetMetadata{
			Metadata: info.Metadata,
		})
}

func (c *localRegistry) buildMetadata(info nugettype.ArtifactInfo,
	fileReader io.Reader) (metadata nugetmetadata.Metadata, err error) {
	pathUUID := uuid.NewString()
	tmpFile, err2 := os.CreateTemp(os.TempDir(), info.RootIdentifier+"-"+pathUUID+"*")
	if err2 != nil {
		return metadata, err2
	}
	defer os.Remove(tmpFile.Name()) // Cleanup

	_, err2 = io.Copy(tmpFile, fileReader)
	if err2 != nil {
		return metadata, err2
	}
	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		return nugetmetadata.Metadata{}, err
	}

	stat, err2 := tmpFile.Stat()
	if err2 != nil {
		return metadata, err2
	}

	archive, err2 := zip.NewReader(tmpFile, stat.Size())
	if err2 != nil {
		return metadata, err2
	}

	for _, file := range archive.File {
		if filepath.Dir(file.Name) != "." {
			continue
		}
		if strings.HasSuffix(strings.ToLower(file.Name), ".nuspec") {
			f, err3 := archive.Open(file.Name)
			if err3 != nil {
				return metadata, err3
			}
			defer f.Close()

			metadata, err = c.parseMetadata(f)
			return metadata, err
		}
	}
	return metadata, nil
}

func (c *localRegistry) parseMetadata(f io.Reader) (metadata nugetmetadata.Metadata, err error) {
	var p nugetmetadata.Metadata
	if err = xml.NewDecoder(f).Decode(&p); err != nil {
		return metadata, err
	}

	if !IDMatch.MatchString(p.PackageMetadata.ID) {
		return metadata, fmt.Errorf("invalid package id: %s", p.PackageMetadata.ID)
	}
	return p, nil
}

func (c *localRegistry) DownloadPackage(ctx context.Context,
	info nugettype.ArtifactInfo) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
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
		return responseHeaders, nil, "", err
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, fileReader, redirectURL, nil
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
	return []artifact.PackageType{artifact.PackageTypeNUGET}
}

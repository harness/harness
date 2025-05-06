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
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	urlprovider "github.com/harness/gitness/app/url"
	apicontract "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	zs "github.com/harness/gitness/registry/app/pkg/commons/zipreader"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
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
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	serviceEndpoints := buildServiceEndpoint(packageURL)
	return serviceEndpoints
}

func (c *localRegistry) ListPackageVersion(ctx context.Context,
	info nugettype.ArtifactInfo) (response *nugettype.PackageVersion, err error) {
	artifacts, err2 := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	}
	var versions []string
	for _, artifact := range *artifacts {
		versions = append(versions, artifact.Version)
	}
	return &nugettype.PackageVersion{
		Versions: versions,
	}, nil
}

func (c *localRegistry) GetPackageMetadata(ctx context.Context,
	info nugettype.ArtifactInfo) (*nugettype.RegistrationIndexResponse, error) {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	artifacts, err2 := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	}

	return createRegistrationIndexResponse(packageURL, info, artifacts)
}

func (c *localRegistry) GetPackageVersionMetadata(ctx context.Context,
	info nugettype.ArtifactInfo) (*nugettype.RegistrationLeafResponse, error) {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	image, err2 := c.imageDao.GetByName(ctx, info.RegistryID, info.Image)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get image for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	}
	artifact, err2 := c.artifactDao.GetByName(ctx, image.ID, info.Version)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	}

	return createRegistrationLeafResponse(packageURL, info, artifact), nil
}

func (c *localRegistry) UploadPackage(ctx context.Context,
	info nugettype.ArtifactInfo,
	fileReader io.ReadCloser,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	tmpFileName := info.RootIdentifier + "-" + uuid.NewString()

	fileInfo, tempFileName, err := c.fileManager.UploadTempFile(ctx, info.RootIdentifier,
		nil, tmpFileName, fileReader)
	if err != nil {
		return headers, "", fmt.Errorf(
			"failed to upload file: %s with registry: %d with error: %w", tmpFileName, info.RegistryID, err)
	}
	r, _, err := c.fileManager.DownloadTempFile(ctx, fileInfo.Size, tempFileName, info.RootIdentifier)
	if err != nil {
		return headers, "", fmt.Errorf(
			"failed to download file: %s with registry: %d with error: %w", tempFileName,
			info.RegistryID, err)
	}
	defer r.Close()

	metadata, err := c.buildMetadata(r)
	if err != nil {
		return headers, "", fmt.Errorf(
			"failed to build metadata for file: %s with registry: %d with error: %w", tempFileName,
			info.RegistryID, err)
	}
	fileName := strings.ToLower(fmt.Sprintf("%s.%s.nupkg",
		metadata.PackageMetadata.ID, metadata.PackageMetadata.Version))
	info.Metadata = metadata
	info.Filename = fileName
	info.Version = metadata.PackageMetadata.Version
	info.Image = strings.ToLower(metadata.PackageMetadata.ID)
	path := info.Image + "/" + info.Version + "/" + fileName

	return c.localBase.MoveTempFile(ctx, info.ArtifactInfo, tempFileName, info.Version, path,
		&nugetmetadata.NugetMetadata{
			Metadata: info.Metadata,
		}, fileInfo)
}

func (c *localRegistry) buildMetadata(fileReader io.Reader) (metadata nugetmetadata.Metadata, err error) {
	zr := zs.NewReader(fileReader)

	for {
		header, err2 := zr.Next()
		if errors.Is(err2, io.EOF) {
			break
		}
		if err2 != nil {
			return metadata, fmt.Errorf("failed to read zip file with error: %w", err2)
		}

		if strings.HasSuffix(header.Name, ".nuspec") {
			metadata, err = c.parseMetadata(zr)
			return metadata, err
		}
	}
	return metadata, nil
}

func (c *localRegistry) parseMetadata(f io.Reader) (metadata nugetmetadata.Metadata, err error) {
	var p nugetmetadata.Metadata
	if err = xml.NewDecoder(f).Decode(&p); err != nil {
		return metadata, fmt.Errorf("failed to parse .nuspec file with error: %w", err)
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

	fileReader, size, redirectURL, err := c.fileManager.DownloadFile(ctx, path,
		info.RegistryID,
		info.RegIdentifier, info.RootIdentifier)
	if err != nil {
		return responseHeaders, nil, "", fmt.Errorf("failed to download file with path: %s, "+
			"registry: %d with error: %w", path, info.RegistryID, err)
	}
	responseHeaders.Code = http.StatusOK
	responseHeaders.Headers["Content-Type"] = "application/octet-stream"
	responseHeaders.Headers["Content-Length"] = strconv.FormatInt(size, 10)
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

func (c *localRegistry) GetArtifactType() apicontract.RegistryType {
	return apicontract.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []apicontract.PackageType {
	return []apicontract.PackageType{apicontract.PackageTypeNUGET}
}

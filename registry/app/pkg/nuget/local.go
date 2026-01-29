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
	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*localRegistry)(nil)
var _ Registry = (*localRegistry)(nil)

var IDMatch = regexp.MustCompile(`\A\w+(?:[.-]\w+)*\z`)

type FileBundleType int

const (
	DependencyPackageExtension = ".nupkg"
	SymbolsPackageExtension    = ".snupkg"
)

const (
	DependencyFile FileBundleType = iota + 1
	SymbolsFile
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

func (c *localRegistry) GetServiceEndpoint(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) *nugettype.ServiceEndpoint {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	serviceEndpoints := buildServiceEndpoint(packageURL)
	return serviceEndpoints
}

func (c *localRegistry) GetServiceEndpointV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) *nugettype.ServiceEndpointV2 {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	serviceEndpoints := buildServiceV2Endpoint(packageURL)
	return serviceEndpoints
}

func (c *localRegistry) GetServiceMetadataV2(
	_ context.Context,
	_ nugettype.ArtifactInfo,
) *nugettype.ServiceMetadataV2 {
	return getServiceMetadataV2()
}

func (c *localRegistry) ListPackageVersion(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (response *nugettype.PackageVersion, err error) {
	artifacts, err2 := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	} else if artifacts == nil || len(*artifacts) == 0 {
		return nil, fmt.Errorf(
			"no artifacts found for registry: %d and image: %s", info.RegistryID, info.Image)
	}
	var versions []string
	for _, artifact := range *artifacts {
		versions = append(versions, artifact.Version)
	}
	return &nugettype.PackageVersion{
		Versions: versions,
	}, nil
}

func (c *localRegistry) ListPackageVersionV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (response *nugettype.FeedResponse, err error) {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	artifacts, err2 := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	} else if artifacts == nil || len(*artifacts) == 0 {
		return nil, fmt.Errorf(
			"no artifacts found for registry: %d and image: %s", info.RegistryID, info.Image)
	}
	return createFeedResponse(packageURL, info, artifacts)
}

func (c *localRegistry) CountPackageVersionV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (count int64, err error) {
	count, err = c.artifactDao.CountByImageName(ctx, info.RegistryID, info.Image)
	if err != nil {
		return 0, fmt.Errorf(
			"failed to get artifacts count for registry: %d and image: %s: %w", info.RegistryID, info.Image, err)
	}
	return count, nil
}

func (c *localRegistry) CountPackageV2(
	ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string,
) (count int64, err error) {
	count, err = c.artifactDao.CountByImageName(ctx, info.RegistryID, strings.ToLower(searchTerm))
	if err != nil {
		return 0, fmt.Errorf(
			"failed to get artifacts count for registry: %d and image: %s: %w", info.RegistryID, searchTerm, err)
	}
	return count, nil
}

func (c *localRegistry) SearchPackageV2(
	ctx context.Context, info nugettype.ArtifactInfo,
	searchTerm string, limit int, offset int,
) (*nugettype.FeedResponse, error) {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	artifacts, err := c.artifactDao.SearchByImageName(ctx, info.RegistryID, strings.ToLower(searchTerm), limit, offset)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, searchTerm, err)
	}
	return createSearchV2Response(packageURL, artifacts, searchTerm, limit, offset)
}

func (c *localRegistry) SearchPackage(
	ctx context.Context,
	info nugettype.ArtifactInfo,
	searchTerm string, limit int, offset int,
) (*nugettype.SearchResultResponse, error) {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	artifacts, err := c.artifactDao.SearchByImageName(ctx, info.RegistryID, strings.ToLower(searchTerm), limit, offset)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, searchTerm, err)
	}
	count, err2 := c.artifactDao.CountByImageName(ctx, info.RegistryID, strings.ToLower(searchTerm))
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts count for registry: %d and image: %s: %w",
			info.RegistryID, info.Image, err2)
	}
	return createSearchResponse(packageURL, artifacts, count)
}

func (c *localRegistry) GetPackageMetadata(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (nugettype.RegistrationResponse, error) {
	packageURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "nuget")
	artifacts, err2 := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err2 != nil {
		return nil, fmt.Errorf(
			"failed to get artifacts for registry: %d and image: %s: %w", info.RegistryID, info.Image, err2)
	} else if artifacts == nil || len(*artifacts) == 0 {
		return nil, fmt.Errorf(
			"no artifacts found for registry: %d and image: %s", info.RegistryID, info.Image)
	}
	return createRegistrationIndexResponse(packageURL, info, artifacts)
}

func (c *localRegistry) GetPackageVersionMetadataV2(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*nugettype.FeedEntryResponse, error) {
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
	return createFeedEntryResponse(packageURL, info, artifact)
}

func (c *localRegistry) GetPackageVersionMetadata(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*nugettype.RegistrationLeafResponse, error) {
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

func (c *localRegistry) UploadPackage(
	ctx context.Context, info nugettype.ArtifactInfo,
	fileReader io.ReadCloser, fileBundleType FileBundleType,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	tmpFileName := info.RootIdentifier + "-" + uuid.NewString()
	var fileExtension string
	metadata := nugetmetadata.Metadata{}

	fileInfo, err := c.fileManager.UploadFileNoDBUpdate(ctx, info.RootIdentifier, nil, fileReader, info.RootParentID,
		info.RegistryID)
	if err != nil {
		return headers, "", fmt.Errorf(
			"failed to upload file: %s with registry: %d with error: %w", tmpFileName, info.RegistryID, err)
	}
	r, err := c.fileManager.DownloadFileByDigest(ctx, info.RootIdentifier, fileInfo, 0, 0)
	if err != nil {
		return headers, "", fmt.Errorf(
			"failed to download file with registry: %d with error: %w",
			info.RegistryID, err)
	}
	defer r.Close()

	metadata, err = c.buildMetadata(r)
	if err != nil {
		return headers, "", fmt.Errorf(
			"failed to build metadata for registry: %d with error: %w",
			info.RegistryID, err)
	}
	info.Image = strings.ToLower(metadata.PackageMetadata.ID)
	info.Version = metadata.PackageMetadata.Version
	normalisedVersion, err2 := validateAndNormaliseVersion(info.Version)
	if err2 != nil {
		return headers, "", fmt.Errorf("nuspec file contains an invalid version: %s with "+
			"package name: %s, registry name: %s", info.Version, info.Image, info.RegIdentifier)
	}
	info.Version = normalisedVersion
	info.Metadata = metadata
	if fileBundleType == SymbolsFile {
		versionExists, err3 := c.localBase.CheckIfVersionExists(ctx, info)
		if err3 != nil {
			return headers, "", fmt.Errorf(
				"failed to check package version existence for id: %s , version: %s "+
					"with registry: %d with error: %w", info.Image, info.Version, info.RegistryID, err)
		} else if !versionExists {
			return headers, "", fmt.Errorf(
				"can't push symbol package as package doesn't exists for id: %s , version: %s "+
					"with registry: %d with error: %w", info.Image, info.Version, info.RegistryID, err)
		}
		fileExtension = SymbolsPackageExtension
	} else {
		fileExtension = DependencyPackageExtension
	}
	fileName := strings.ToLower(fmt.Sprintf("%s.%s%s",
		metadata.PackageMetadata.ID, metadata.PackageMetadata.Version, fileExtension))
	info.Filename = fileName
	fileInfo.Filename = fileName
	var path string

	if info.NestedPath != "" {
		path = info.Image + "/" + info.Version + "/" + info.NestedPath + "/" + fileName
	} else {
		path = info.Image + "/" + info.Version + "/" + fileName
	}

	h, checkSum, _, _, err := c.localBase.UpdateFileManagerAndCreateArtifact(ctx, info.ArtifactInfo, info.Version, path,
		&nugetmetadata.NugetMetadata{
			Metadata: info.Metadata,
		}, fileInfo, false)
	return h, checkSum, err
}

func (c *localRegistry) buildMetadata(fileReader io.Reader) (metadata nugetmetadata.Metadata, err error) {
	var readme string
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
			if err != nil {
				return metadata, fmt.Errorf("failed to parse metadata from .nuspec file: %w", err2)
			}
		} else if strings.HasSuffix(header.Name, "README.md") {
			readme, err2 = c.parseReadme(zr)
			if err2 != nil {
				return metadata, fmt.Errorf("failed to parse metadata from README.md file: %w", err2)
			}
		}
	}
	if readme != "" {
		metadata.PackageMetadata.Readme = readme
	} else if metadata.PackageMetadata.Description != "" {
		metadata.PackageMetadata.Readme = metadata.PackageMetadata.Description
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

func (c *localRegistry) parseReadme(f io.Reader) (readme string, err error) {
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *localRegistry) DownloadPackage(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, io.ReadCloser, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	path, err := c.fileManager.FindLatestFilePath(ctx, info.RegistryID,
		"/"+info.Image+"/"+info.Version, info.Filename)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to find file node for id: %s , version: %s "+
			"with registry: %d with error: %v", info.Image, info.Version, info.RegistryID, err)
		return responseHeaders, nil, "", nil, fmt.Errorf("failed to find file node for id: %s , version: %s "+
			"with registry: %d with error: %w", info.Image, info.Version, info.RegistryID, err)
	}

	fileReader, size, redirectURL, err := c.fileManager.DownloadFileByPath(ctx, path,
		info.RegistryID,
		info.RegIdentifier, info.RootIdentifier, true)
	if err != nil {
		return responseHeaders, nil, "", nil, err
	}
	responseHeaders.Code = http.StatusOK
	responseHeaders.Headers["Content-Type"] = "application/octet-stream"
	responseHeaders.Headers["Content-Length"] = strconv.FormatInt(size, 10)
	return responseHeaders, fileReader, redirectURL, nil, nil
}

func (c *localRegistry) DeletePackage(
	ctx context.Context,
	info nugettype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	err := c.localBase.DeleteVersion(ctx, info)
	if err != nil {
		return responseHeaders, fmt.Errorf("failed to delete package version with package: %s, version: %s and "+
			"registry: %d with error: %w", info.Image, info.Version, info.RegistryID, err)
	}
	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil
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

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

package npm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/usererror"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	npm2 "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
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
	tagsDao     store.PackageTagRepository
	nodesDao    store.NodesRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
}

func (c *localRegistry) HeadPackageMetadata(ctx context.Context, info npm.ArtifactInfo) (bool, error) {
	return c.localBase.CheckIfVersionExists(ctx, info)
}

func (c *localRegistry) DownloadPackageFile(ctx context.Context,
	info npm.ArtifactInfo) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	headers, fileReader, redirectURL, err :=
		c.localBase.Download(ctx, info.ArtifactInfo, info.Version,
			info.Filename)
	if err != nil {
		return nil, nil, "", err
	}
	return headers, fileReader, redirectURL, nil
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
	tagDao store.PackageTagRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	nodesDao store.NodesRepository,
	urlProvider urlprovider.Provider,
) LocalRegistry {
	return &localRegistry{
		localBase:   localBase,
		fileManager: fileManager,
		proxyStore:  proxyStore,
		tx:          tx,
		tagsDao:     tagDao,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		nodesDao:    nodesDao,
		urlProvider: urlProvider,
	}
}

func (c *localRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (c *localRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeNPM}
}

func (c *localRegistry) UploadPackageFile(
	ctx context.Context,
	info npm.ArtifactInfo,
	file io.ReadCloser,
) (headers *commons.ResponseHeaders, sha256 string, err error) {
	defer file.Close()
	path := pkg.JoinWithSeparator("/", info.Image, info.Version, info.Filename)
	response, sha, err := c.localBase.Upload(ctx, info.ArtifactInfo, info.Filename, info.Version, path, file,
		&npm2.NpmMetadata{
			PackageMetadata: info.Metadata,
		})
	if !commons.IsEmpty(err) {
		return nil, "", err
	}
	_, err = c.AddTag(ctx, info)
	if err != nil {
		return nil, "", err
	}
	return response, sha, nil
}

func (c *localRegistry) GetPackageMetadata(ctx context.Context, info npm.ArtifactInfo) (npm2.PackageMetadata, error) {
	packageMetadata := npm2.PackageMetadata{}
	versions := make(map[string]*npm2.PackageMetadataVersion)
	artifacts, err := c.artifactDao.GetByRegistryIDAndImage(ctx, info.RegistryID, info.Image)
	if err != nil {
		log.Warn().Msgf("Failed to fetch artifact for image:[%s], Reg:[%s]",
			info.BaseArtifactInfo().Image, info.BaseArtifactInfo().RegIdentifier)
		return packageMetadata, usererror.ErrInternal
	}

	if len(*artifacts) == 0 {
		return packageMetadata,
			usererror.NotFound(fmt.Sprintf("no artifacts found for registry %s and image %s", info.Registry.Name, info.Image))
	}
	regURL := c.urlProvider.PackageURL(ctx, info.RootIdentifier+"/"+info.RegIdentifier, "npm")

	for _, artifact := range *artifacts {
		metadata := &npm2.NpmMetadata{}
		err = json.Unmarshal(artifact.Metadata, metadata)
		if err != nil {
			return packageMetadata, err
		}
		if packageMetadata.Name == "" {
			packageMetadata = metadata.PackageMetadata
		}
		for _, versionMetadata := range metadata.Versions {
			versions[artifact.Version] = CreatePackageMetadataVersion(regURL, versionMetadata)
		}
	}
	distTags, err := c.ListTags(ctx, info)
	if !commons.IsEmpty(err) {
		return npm2.PackageMetadata{}, err
	}
	packageMetadata.Versions = versions
	packageMetadata.DistTags = distTags
	return packageMetadata, nil
}

func CreatePackageMetadataVersion(registryURL string,
	metadata *npm2.PackageMetadataVersion) *npm2.PackageMetadataVersion {
	return &npm2.PackageMetadataVersion{
		ID:                   fmt.Sprintf("%s@%s", metadata.Name, metadata.Version),
		Name:                 metadata.Name,
		Version:              metadata.Version,
		Description:          metadata.Description,
		Author:               metadata.Author,
		Homepage:             registryURL,
		License:              metadata.License,
		Dependencies:         metadata.Dependencies,
		BundleDependencies:   metadata.BundleDependencies,
		DevDependencies:      metadata.DevDependencies,
		PeerDependencies:     metadata.PeerDependencies,
		OptionalDependencies: metadata.OptionalDependencies,
		Readme:               metadata.Readme,
		Bin:                  metadata.Bin,
		Dist: npm2.PackageDistribution{
			Shasum:    metadata.Dist.Shasum,
			Integrity: metadata.Dist.Integrity,
			Tarball: fmt.Sprintf("http://localhost:3000/pkg/test/npm1/npm/%s/-/%s/%s", metadata.Name, metadata.Version,
				metadata.Name+"-"+metadata.Version+".tgz"),
		},
	}
}

func (c *localRegistry) ListTags(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error) {
	tags, err := c.tagsDao.FindByImageNameAndRegID(ctx, info.Image, info.RegistryID)
	if err != nil {
		return nil, err
	}

	pkgTags := make(map[string]string)

	for _, tag := range tags {
		pkgTags[tag.Name] = tag.Version
	}
	return pkgTags, nil
}

func (c *localRegistry) AddTag(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error) {
	image, err := c.imageDao.GetByRepoAndName(ctx, info.ParentID, info.RegIdentifier, info.Image)
	if err != nil {
		return nil, err
	}
	version, err := c.artifactDao.GetByName(ctx, image.ID, info.Version)

	if err != nil {
		return nil, err
	}

	if len(info.DistTags) == 0 {
		return nil, err
	}
	packageTag := &types.PackageTag{
		ID:         uuid.NewString(),
		Name:       info.DistTags[0],
		ArtifactID: version.ID,
	}
	_, err = c.tagsDao.Create(ctx, packageTag)
	if err != nil {
		return nil, err
	}
	return c.ListTags(ctx, info)
}

func (c *localRegistry) DeleteTag(ctx context.Context, info npm.ArtifactInfo) (map[string]string, error) {
	if len(info.DistTags) == 0 {
		return nil, usererror.BadRequest("Delete tag error: distTags are empty")
	}
	err := c.tagsDao.DeleteByTagAndImageName(ctx, info.DistTags[0], info.Image, info.RegistryID)
	if err != nil {
		return nil, err
	}
	return c.ListTags(ctx, info)
}

func (c *localRegistry) DeletePackage(ctx context.Context, info npm.ArtifactInfo) error {
	return c.localBase.DeletePackage(ctx, info)
}

func (c *localRegistry) DeleteVersion(ctx context.Context, info npm.ArtifactInfo) error {
	return c.localBase.DeleteVersion(ctx, info)
}

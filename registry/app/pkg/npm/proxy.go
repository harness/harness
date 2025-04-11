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
	"fmt"
	"io"

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	npm2 "github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*proxy)(nil)
var _ Registry = (*proxy)(nil)

type proxy struct {
	fileManager         filemanager.FileManager
	proxyStore          store.UpstreamProxyConfigRepository
	tx                  dbtx.Transactor
	registryDao         store.RegistryRepository
	imageDao            store.ImageRepository
	artifactDao         store.ArtifactRepository
	urlProvider         urlprovider.Provider
	spaceFinder         refcache.SpaceFinder
	service             secret.Service
	localRegistryHelper LocalRegistryHelper
}

func (r *proxy) UploadPackageFileReader(_ context.Context,
	_ npm2.ArtifactInfo) (*commons.ResponseHeaders, string, error) {
	return nil, " ", commons.ErrNotSupported
}

func (r *proxy) HeadPackageMetadata(_ context.Context, _ npm2.ArtifactInfo) (bool, error) {
	return false, commons.ErrNotSupported
}

func (r *proxy) ListTags(_ context.Context, _ npm2.ArtifactInfo) (map[string]string, error) {
	return nil, commons.ErrNotSupported
}

func (r *proxy) AddTag(_ context.Context, _ npm2.ArtifactInfo) (map[string]string, error) {
	return nil, commons.ErrNotSupported
}

func (r *proxy) DeleteTag(_ context.Context, _ npm2.ArtifactInfo) (map[string]string, error) {
	return nil, commons.ErrNotSupported
}

func (r *proxy) DeletePackage(_ context.Context, _ npm2.ArtifactInfo) error {
	return commons.ErrNotSupported
}

func (r *proxy) DeleteVersion(_ context.Context, _ npm2.ArtifactInfo) error {
	return commons.ErrNotSupported
}

func (r *proxy) GetPackageMetadata(ctx context.Context, info npm2.ArtifactInfo) (npm.PackageMetadata, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return npm.PackageMetadata{}, err
	}

	helper, _ := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	result, err := helper.GetPackageMetadata(ctx, info.Image)
	if err != nil {
		return npm.PackageMetadata{}, err
	}
	regURL := r.urlProvider.PackageURL(ctx, info.RootIdentifier, info.RegIdentifier, "npm")

	versions := make(map[string]*npm.PackageMetadataVersion)
	for _, version := range result.Versions {
		versions[version.Version] = CreatePackageMetadataVersion(regURL, version)
	}

	result.Versions = versions
	return *result, nil
}

type Proxy interface {
	Registry
}

func NewProxy(
	fileManager filemanager.FileManager,
	proxyStore store.UpstreamProxyConfigRepository,
	tx dbtx.Transactor,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	urlProvider urlprovider.Provider,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
	localRegistryHelper LocalRegistryHelper,
) Proxy {
	return &proxy{
		proxyStore:          proxyStore,
		registryDao:         registryDao,
		imageDao:            imageDao,
		artifactDao:         artifactDao,
		fileManager:         fileManager,
		tx:                  tx,
		urlProvider:         urlProvider,
		spaceFinder:         spaceFinder,
		service:             service,
		localRegistryHelper: localRegistryHelper,
	}
}

func (r *proxy) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeUPSTREAM
}

func (r *proxy) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeNPM}
}

func (r *proxy) DownloadPackageFile(ctx context.Context, info npm2.ArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	io.ReadCloser,
	string,
	error,
) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return nil, nil, nil, "", err
	}

	exists := r.localRegistryHelper.FileExists(ctx, info)
	if exists {
		headers, fileReader, redirectURL, err := r.localRegistryHelper.DownloadFile(ctx, info)
		if err == nil {
			return headers, fileReader, nil, redirectURL, nil
		}
		// If file exists in local registry, but download failed, we should try to download from remote
		log.Warn().Ctx(ctx).Msgf("failed to pull from local, attempting streaming from remote, %v", err)
	}

	remote, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return nil, nil, nil, "", err
	}

	file, err := remote.GetPackage(ctx, info.Image, info.Version)
	if err != nil {
		return nil, nil, nil, "", errcode.ErrCodeUnknown.WithDetail(err)
	}

	go func(info npm2.ArtifactInfo) {
		ctx2 := context.WithoutCancel(ctx)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")
		err = r.putFileToLocal(ctx2, info, remote)
		if err != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated file: %s, registry: %s", info.Filename, info.RegIdentifier)
	}(info)

	return nil, nil, file, "", nil
}

// UploadPackageFile FIXME: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (r *proxy) UploadPackageFile(
	ctx context.Context,
	_ npm2.ArtifactInfo,
	_ io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) putFileToLocal(ctx context.Context, info npm2.ArtifactInfo, remote RemoteRegistryHelper) error {
	versionMetadata, err := remote.GetVersionMetadata(ctx, info.Image, info.GetVersion())
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("fetching metadata of pkg with name %s,"+
			" version %s failed, %v", info.Image, info.Version, err)
		return err
	}
	file, err := remote.GetPackage(ctx, info.Image, info.Version)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("fetching pkg with name %s,"+
			" version %s failed, %v", info.Image, info.Version, err)
		return err
	}
	defer file.Close()

	info.Metadata = *versionMetadata
	_, sha256, err2 := r.localRegistryHelper.UploadPackageFile(ctx, info, file)
	if err2 != nil {
		log.Ctx(ctx).Error().Stack().Err(err2).Msgf("uploading file %s failed, %v", info.Filename, err)
		return err2
	}
	log.Info().Msgf("Successfully uploaded %s with SHA256: %s", info.Filename, sha256)
	return nil
}

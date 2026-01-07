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
	"fmt"
	"io"
	"mime/multipart"

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/remote/adapter/commons/pypi"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	cfg "github.com/harness/gitness/registry/config"
	request2 "github.com/harness/gitness/registry/request"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"

	_ "github.com/harness/gitness/registry/app/remote/adapter/pypi" // This is required to init pypi adapter
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
		fileManager:         fileManager,
		proxyStore:          proxyStore,
		tx:                  tx,
		registryDao:         registryDao,
		imageDao:            imageDao,
		artifactDao:         artifactDao,
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
	return []artifact.PackageType{artifact.PackageTypePYTHON}
}

func (r *proxy) DownloadPackageFile(ctx context.Context, info pythontype.ArtifactInfo) (
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

	// TODO: Extract out to Path Utils for all package types
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

	file, err := remote.GetFile(ctx, info.Image, info.Filename)
	if err != nil {
		return nil, nil, nil, "", errcode.ErrCodeUnknown.WithDetail(err)
	}

	go func(info pythontype.ArtifactInfo) {
		ctx2 := context.WithoutCancel(ctx)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")
		err = r.putFileToLocal(ctx2, info.Image, info.Filename, remote)
		if err != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated file: %s, registry: %s", info.Filename, info.RegIdentifier)
	}(info)

	return nil, nil, file, "", nil
}

// GetPackageMetadata Returns metadata from remote.
func (r *proxy) GetPackageMetadata(
	ctx context.Context,
	info pythontype.ArtifactInfo,
) (pythontype.PackageMetadata, error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return pythontype.PackageMetadata{}, err
	}

	helper, _ := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	result, err := helper.GetMetadata(ctx, info.Image)
	if err != nil {
		return pythontype.PackageMetadata{}, err
	}

	var files []pythontype.File
	for _, file := range result.Packages {
		files = append(files, pythontype.File{
			Name: file.Name,
			FileURL: r.urlProvider.RegistryURL(ctx) + fmt.Sprintf(
				"/pkg/%s/%s/python/files/%s/%s/%s",
				info.RootIdentifier,
				info.RegIdentifier,
				info.Image,
				file.Version(),
				file.Name),
			RequiresPython: file.RequiresPython(),
		})
	}

	metadata := pythontype.PackageMetadata{
		Name:  info.Image,
		Files: files,
	}
	sortPackageMetadata(ctx, metadata)
	return metadata, nil
}

func (r *proxy) putFileToLocal(ctx context.Context, pkg string, filename string, remote RemoteRegistryHelper) error {
	version := pypi.GetPyPIVersion(filename)
	metadata, err := remote.GetJSON(ctx, pkg, version)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("fetching metadata for %s failed, %v", filename, err)
		return err
	}
	file, err := remote.GetFile(ctx, pkg, filename)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("fetching file %s failed, %v", filename, err)
		return err
	}
	defer file.Close()
	info, ok := request2.ArtifactInfoFrom(ctx).(*pythontype.ArtifactInfo)
	if !ok {
		log.Ctx(ctx).Error().Msgf("failed to cast artifact info to python artifact info")
		return errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("failed to cast artifact info to python artifact info"))
	}
	info.Metadata = *metadata
	info.Filename = filename

	_, sha256, err2 := r.localRegistryHelper.UploadPackageFile(ctx, *info, file, filename)
	if err2 != nil {
		log.Ctx(ctx).Error().Stack().Err(err2).Msgf("uploading file %s failed, %v", filename, err)
		return err2
	}
	log.Ctx(ctx).Info().Msgf("Successfully uploaded %s with SHA256: %s", filename, sha256)
	return nil
}

// UploadPackageFile TODO: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (r *proxy) UploadPackageFile(
	ctx context.Context,
	_ pythontype.ArtifactInfo,
	_ multipart.File,
	_ string,
) (*commons.ResponseHeaders, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

// UploadPackageFile TODO: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (r *proxy) UploadPackageFileReader(
	ctx context.Context,
	_ pythontype.ArtifactInfo,
	_ io.ReadCloser,
	_ string,
) (*commons.ResponseHeaders, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

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

package gopackage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/api/utils"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	gopackagemetadata "github.com/harness/gitness/registry/app/metadata/gopackage"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
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
	localBase             base.LocalBase
	fileManager           filemanager.FileManager
	proxyStore            store.UpstreamProxyConfigRepository
	tx                    dbtx.Transactor
	registryDao           store.RegistryRepository
	imageDao              store.ImageRepository
	artifactDao           store.ArtifactRepository
	urlProvider           urlprovider.Provider
	spaceFinder           refcache.SpaceFinder
	service               secret.Service
	artifactEventReporter *registryevents.Reporter
	localRegistryHelper   LocalRegistryHelper
}

type Proxy interface {
	Registry
}

func NewProxy(
	localBase base.LocalBase,
	fileManager filemanager.FileManager,
	proxyStore store.UpstreamProxyConfigRepository,
	tx dbtx.Transactor,
	registryDao store.RegistryRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	urlProvider urlprovider.Provider,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
	artifactEventReporter *registryevents.Reporter,
	localRegistryHelper LocalRegistryHelper,
) Proxy {
	return &proxy{
		localBase:             localBase,
		fileManager:           fileManager,
		proxyStore:            proxyStore,
		tx:                    tx,
		registryDao:           registryDao,
		imageDao:              imageDao,
		artifactDao:           artifactDao,
		urlProvider:           urlProvider,
		spaceFinder:           spaceFinder,
		service:               service,
		artifactEventReporter: artifactEventReporter,
		localRegistryHelper:   localRegistryHelper,
	}
}

func (r *proxy) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeUPSTREAM
}

func (r *proxy) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeGO}
}

func (r *proxy) UploadPackage(
	ctx context.Context, _ gopackagetype.ArtifactInfo,
	_ io.ReadCloser, _ io.ReadCloser,
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) DownloadPackageFile(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	if info.Version == "" {
		// If version is empty, it is an index file
		response, reader, fileReader, err := r.DownloadPackageIndex(ctx, info)
		return response, reader, fileReader, "", err
	}
	if info.Version == LatestVersionKey {
		// Download latest version
		response, reader, fileReader, err := r.DownloadPackageLatestVersionInfo(ctx, info)
		return response, reader, fileReader, "", err
	}
	path, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, info.Version)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get file path: %w", err)
	}
	filePath := filepath.Join(path, info.FileName)
	// Check if the file exists in the local registry
	exists := r.localRegistryHelper.FileExists(ctx, info, filePath)
	if exists {
		headers, fileReader, reader, redirectURL, err := r.localRegistryHelper.DownloadFile(ctx, info)
		if err == nil {
			return headers, fileReader, reader, redirectURL, nil
		}
		// If file exists in local registry, but download failed, we should try to download from remote
		log.Warn().Ctx(ctx).Msgf("failed to pull from local, attempting streaming from remote, %v", err)
	}
	// fetch it from upstream
	pathForUpstream, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, "")
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get file path: %w", err)
	}
	filePathForUpstream := filepath.Join(pathForUpstream, info.FileName)
	response, helper, fileReader, err := r.downloadFileFromUpstream(ctx, info, filePathForUpstream)
	if err != nil {
		return response, nil, nil, "", fmt.Errorf("failed to download package file from upstream: %w", err)
	}
	// cache all package files if its zip file
	if filepath.Ext(info.FileName) == ".zip" {
		go func(info gopackagetype.ArtifactInfo) {
			ctx2 := context.WithoutCancel(ctx)
			ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")
			err = r.putFileToLocal(ctx2, &info, helper)
			if err != nil {
				log.Ctx(ctx2).Error().Stack().Err(err).Msgf(
					"error while putting cargo file to localRegistry, %v", err,
				)
				return
			}
			log.Ctx(ctx2).Info().Msgf(
				"Successfully updated for image: %s, version: %s in registry: %s",
				info.Image, info.Version, info.RegIdentifier,
			)
		}(info)
	}
	return response, nil, fileReader, "", nil
}

// do not cache package index always get it from upstream.
func (r *proxy) DownloadPackageIndex(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, error) {
	path, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get file path: %w", err)
	}
	filePath := filepath.Join(path, "list")
	response, _, reader, err := r.downloadFileFromUpstream(ctx, info, filePath)
	if err != nil {
		return response, nil, nil, fmt.Errorf("failed to download package index: %w", err)
	}
	return response, nil, reader, nil
}

// do not cache latest version always get it from upstream.
func (r *proxy) DownloadPackageLatestVersionInfo(
	ctx context.Context, info gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, error) {
	filePath := filepath.Join(info.Image, LatestVersionKey)
	response, _, reader, err := r.downloadFileFromUpstream(ctx, info, filePath)
	if err != nil {
		return response, nil, nil, fmt.Errorf("failed to download package latest version: %w", err)
	}
	return response, nil, reader, nil
}

func (r *proxy) downloadFileFromUpstream(
	ctx context.Context, info gopackagetype.ArtifactInfo, path string,
) (*commons.ResponseHeaders, RemoteRegistryHelper, io.ReadCloser, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, nil, nil, fmt.Errorf("failed to get upstream proxy: %w", err)
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return responseHeaders, nil, nil, fmt.Errorf("failed to create remote registry helper: %w", err)
	}

	file, err := helper.GetPackageFile(ctx, info.Image, path)
	if err != nil {
		return responseHeaders, nil, nil, fmt.Errorf("failed to get package file: %w", err)
	}

	responseHeaders.Code = http.StatusOK
	return responseHeaders, helper, file, nil
}

func (r *proxy) putFileToLocal(
	ctx context.Context, info *gopackagetype.ArtifactInfo,
	remote RemoteRegistryHelper,
) error {
	// path for upstream proxy file
	path, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, "")
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}

	// cache .info file
	info.FileName = info.Version + ".info"
	infoFilePath := filepath.Join(path, info.FileName)
	err = r.cacheFileAndCreateOrUpdateVersion(ctx, info, infoFilePath, remote)
	if err != nil {
		return fmt.Errorf("failed to cache info file: %w", err)
	}

	// cache .mod file
	info.FileName = info.Version + ".mod"
	modFilePath := filepath.Join(path, info.FileName)
	err = r.cacheFileAndCreateOrUpdateVersion(ctx, info, modFilePath, remote)
	if err != nil {
		return fmt.Errorf("failed to cache mod file: %w", err)
	}

	// cache .zip file
	info.FileName = info.Version + ".zip"
	zipFilePath := filepath.Join(path, info.FileName)
	err = r.cacheFileAndCreateOrUpdateVersion(ctx, info, zipFilePath, remote)
	if err != nil {
		return fmt.Errorf("failed to cache zip file: %w", err)
	}

	// regenerate package index
	r.localRegistryHelper.RegeneratePackageIndex(ctx, *info)
	// regenerate package metadata
	r.localRegistryHelper.RegeneratePackageMetadata(ctx, *info)
	return nil
}

func (r *proxy) cacheFileAndCreateOrUpdateVersion(
	ctx context.Context, info *gopackagetype.ArtifactInfo,
	filePath string, remote RemoteRegistryHelper,
) error {
	// Get pacakage from upstream source
	file, err := remote.GetPackageFile(ctx, info.Image, filePath)
	if err != nil {
		return fmt.Errorf("failed to get package file: %w", err)
	}
	defer file.Close()

	// upload to file path
	path, err := utils.GetFilePath(artifact.PackageTypeGO, info.Image, info.Version)
	if err != nil {
		return fmt.Errorf("failed to get file path: %w", err)
	}
	filePathForLocal := filepath.Join(path, info.FileName)
	_, _, err = r.localBase.Upload(
		ctx, info.ArtifactInfo, info.FileName, info.Version, filePathForLocal, file,
		&gopackagemetadata.VersionMetadataDB{
			VersionMetadata: gopackagemetadata.VersionMetadata{
				Name:    info.Image,
				Version: info.Version,
				Time:    time.Now().Format(time.RFC3339), // get time in "2025-07-15T14:37:08.552428Z" format
			},
		})
	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", filePath, err)
	}

	return nil
}

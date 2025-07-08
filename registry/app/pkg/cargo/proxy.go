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

package cargo

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/registry/services/webhook"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*proxy)(nil)
var _ Registry = (*proxy)(nil)

type proxy struct {
	fileManager           filemanager.FileManager
	proxyStore            store.UpstreamProxyConfigRepository
	tx                    dbtx.Transactor
	registryDao           store.RegistryRepository
	imageDao              store.ImageRepository
	artifactDao           store.ArtifactRepository
	urlProvider           urlprovider.Provider
	spaceFinder           refcache.SpaceFinder
	service               secret.Service
	localRegistryHelper   LocalRegistryHelper
	artifactEventReporter *registryevents.Reporter
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
	artifactEventReporter *registryevents.Reporter,
) Proxy {
	return &proxy{
		fileManager:           fileManager,
		proxyStore:            proxyStore,
		tx:                    tx,
		registryDao:           registryDao,
		imageDao:              imageDao,
		artifactDao:           artifactDao,
		urlProvider:           urlProvider,
		spaceFinder:           spaceFinder,
		service:               service,
		localRegistryHelper:   localRegistryHelper,
		artifactEventReporter: artifactEventReporter,
	}
}

func (r *proxy) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeUPSTREAM
}

func (r *proxy) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeCARGO}
}

func (r *proxy) UploadPackage(
	ctx context.Context, _ cargotype.ArtifactInfo,
	_ *cargometadata.VersionMetadata, _ io.ReadCloser,
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) DownloadPackageIndex(
	ctx context.Context, info cargotype.ArtifactInfo,
	filePath string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, nil, nil, "", fmt.Errorf("failed to get upstream proxy: %w", err)
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service, "")
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to create remote registry helper: %w", err)
	}
	result, err := helper.GetPackageIndex(info.Image, filePath)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get package index file: %w", err)
	}

	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil, result, "", nil
}

func (r *proxy) DownloadPackage(
	ctx context.Context, info cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}

	// Get the upstream proxy configuration for the registry
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, nil, nil, "", fmt.Errorf("failed to get upstream proxy: %w", err)
	}

	// Check if the file exists in the local registry
	exists := r.localRegistryHelper.FileExists(ctx, info)
	if exists {
		headers, fileReader, reader, redirectURL, err := r.localRegistryHelper.DownloadFile(ctx, info)
		if err == nil {
			return headers, fileReader, reader, redirectURL, nil
		}
		// If file exists in local registry, but download failed, we should try to download from remote
		log.Warn().Ctx(ctx).Msgf("failed to pull from local, attempting streaming from remote, %v", err)
	}

	// get registry config json from upstream proxy
	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service, "")
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to create remote registry helper: %w", err)
	}
	registryConfig, err := helper.GetRegistryConfig()
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get registry config: %w", err)
	}

	// download the package file from the upstream proxy
	helper, _ = NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service, registryConfig.DownloadURL)
	result, err := helper.GetPackageFile(ctx, info.Image, info.Version)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to get registry config: %w", err)
	}

	go func(info cargotype.ArtifactInfo) {
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

	responseHeaders.Code = http.StatusOK
	return responseHeaders, nil, result, "", nil
}

func (r *proxy) UpdateYank(
	ctx context.Context, _ cargotype.ArtifactInfo, _ bool,
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) RegeneratePackageIndex(
	ctx context.Context, _ cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) putFileToLocal(ctx context.Context, info *cargotype.ArtifactInfo,
	remote RemoteRegistryHelper) error {
	// Get pacakage from upstream source
	file, err := remote.GetPackageFile(ctx, info.Image, info.Version)
	if err != nil {
		return fmt.Errorf("failed to get registry config: %w", err)
	}
	defer file.Close()
	// upload to temporary path
	tmpFileName := info.RootIdentifier + "-" + uuid.NewString()
	fileInfo, tempFileName, err := r.fileManager.UploadTempFile(ctx, info.RootIdentifier,
		nil, tmpFileName, file)
	if err != nil {
		return fmt.Errorf(
			"failed to upload file: %s with registry: %d with error: %w", tmpFileName, info.RegistryID, err)
	}
	// download the temporary file
	tempFile, _, err := r.fileManager.DownloadTempFile(ctx, fileInfo.Size, tempFileName, info.RootIdentifier)
	if err != nil {
		return fmt.Errorf(
			"failed to download file: %s with registry: %d with error: %w", tempFileName,
			info.RegistryID, err)
	}
	defer tempFile.Close()
	// generate the metadata
	metadata, err := generateMetadataFromFile(info, tempFile)
	if err != nil {
		return fmt.Errorf("failed to generate metadata: %w", err)
	}

	// move temporary file to correct location
	fileInfo.Filename = getCrateFileName(info.Image, info.Version)
	_, _, _, _, err = r.localRegistryHelper.MoveTempFile(
		ctx, info, fileInfo, tempFileName, metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	// publish artifact created event
	session, _ := request.AuthSessionFrom(ctx)
	payload := webhook.GetArtifactCreatedPayloadForCommonArtifacts(
		session.Principal.ID,
		info.RegistryID,
		artifact.PackageTypeCARGO,
		info.Image,
		info.Version,
	)
	r.artifactEventReporter.ArtifactCreated(ctx, &payload)

	// regenerate package index
	r.localRegistryHelper.UpdatePackageIndex(ctx, *info)

	log.Info().Msgf("Successfully uploaded file for pkg: %s , version: %s",
		info.Image, info.Version)
	return nil
}

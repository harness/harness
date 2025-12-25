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

package generic

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"

	_ "github.com/harness/gitness/registry/app/remote/adapter/generic" // This is required to init generic adapter
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
	return []artifact.PackageType{artifact.PackageTypeGENERIC}
}

func (r *proxy) PutFile(
	_ context.Context,
	_ generic.ArtifactInfo,
	_ io.ReadCloser,
	_ string,
) (*commons.ResponseHeaders, string, error) {
	return nil, "", usererror.MethodNotAllowed("generic upload to upstream is not allowed")
}

func (r *proxy) DownloadFile(ctx context.Context, info generic.ArtifactInfo, filePath string) (
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

	_, err = r.localRegistryHelper.FileExists(ctx, info)
	if err == nil {
		headers, fileReader, redirectURL, err := r.localRegistryHelper.DownloadFile(ctx, info)
		if err == nil {
			return headers, fileReader, nil, redirectURL, nil
		}
		// If file exists in local registry, but download failed, we should try to download from remote
		log.Error().Ctx(ctx).Msgf("failed to pull from local, attempting to stream from remote, %v", err)
	}

	remote, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return nil, nil, nil, "", err
	}

	readCloser, err := remote.GetFile(ctx, filePath)
	if err != nil {
		return nil, nil, nil, "", usererror.NotFoundf("filepath not found %q, %v", filePath, err)
	}

	go func(info generic.ArtifactInfo) {
		ctx2 := context.WithoutCancel(ctx)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")

		log.Ctx(ctx2).Info().Msgf("Downloading generic artifact %q:%q, filepath: %q for registry: %d",
			info.ArtifactInfo.Image, info.Version, filePath, info.RegistryID)
		remote2, err2 := NewRemoteRegistryHelper(ctx2, r.spaceFinder, *upstreamProxy, r.service)
		if err2 != nil {
			log.Ctx(ctx2).Error().Msgf("failed to create remote in goroutine: %v", err2)
			return
		}

		err2 = r.putFileToLocal(ctx2, info, filePath, remote2)
		if err2 != nil {
			log.Ctx(ctx2).Error().Stack().Err(err2).Msgf("error while putting file to localRegistry %q, %v",
				info.RegIdentifier, err2)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated file: %s, registry: %s", filePath, info.RegIdentifier)
	}(info)

	return &commons.ResponseHeaders{
		Headers: map[string]string{},
		Code:    http.StatusOK,
	}, nil, readCloser, "", nil
}

func (r *proxy) putFileToLocal(
	ctx context.Context,
	info generic.ArtifactInfo,
	filePath string,
	remote RemoteRegistryHelper,
) error {
	readCloser, err := remote.GetFile(ctx, filePath)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("fetching file %s failed, %v", filePath, err)
		return err
	}
	defer readCloser.Close()

	_, sha256, err2 := r.localRegistryHelper.PutFile(ctx, info, readCloser, "")
	if err2 != nil {
		log.Ctx(ctx).Error().Stack().Err(err2).Msgf("uploading file %s failed", filePath)
		return err2
	}
	log.Ctx(ctx).Info().Msgf("Successfully uploaded %s with SHA256: %s", filePath, sha256)
	return nil
}

func (r *proxy) DeleteFile(ctx context.Context, info generic.ArtifactInfo) (*commons.ResponseHeaders, error) {
	return r.localRegistryHelper.DeleteFile(ctx, info)
}

func (r *proxy) HeadFile(
	ctx context.Context,
	info generic.ArtifactInfo,
	filePath string,
) (headers *commons.ResponseHeaders, err error) {
	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return nil, err
	}

	headers, err = r.localRegistryHelper.FileExists(ctx, info)
	if err == nil {
		return headers, nil
	}

	remote, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstreamProxy, r.service)
	if err != nil {
		return nil, err
	}

	headers, err = remote.HeadFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("HeadFile Failure: %w", err)
	}

	return headers, nil
}

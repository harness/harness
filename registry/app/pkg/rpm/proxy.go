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

package rpm

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"

	_ "github.com/harness/gitness/registry/app/remote/adapter/rpm" // This is required to init rpm adapter
)

var _ pkg.Artifact = (*proxy)(nil)
var _ Registry = (*proxy)(nil)

type proxy struct {
	fileManager    filemanager.FileManager
	proxyStore     store.UpstreamProxyConfigRepository
	tx             dbtx.Transactor
	registryDao    store.RegistryRepository
	imageDao       store.ImageRepository
	artifactDao    store.ArtifactRepository
	urlProvider    urlprovider.Provider
	localBase      base.LocalBase
	registryHelper RegistryHelper
	spaceFinder    refcache.SpaceFinder
	service        secret.Service
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
	localBase base.LocalBase,
	registryHelper RegistryHelper,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
) Proxy {
	return &proxy{
		proxyStore:     proxyStore,
		registryDao:    registryDao,
		imageDao:       imageDao,
		artifactDao:    artifactDao,
		fileManager:    fileManager,
		tx:             tx,
		urlProvider:    urlProvider,
		localBase:      localBase,
		registryHelper: registryHelper,
		spaceFinder:    spaceFinder,
		service:        service,
	}
}

func (r *proxy) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeUPSTREAM
}

func (r *proxy) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeRPM}
}

func (r *proxy) DownloadPackageFile(
	ctx context.Context,
	info rpmtype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	headers, fileReader, readcloser, redirect, err := downloadPackageFile(ctx, info, r.localBase)
	if err == nil {
		return headers, fileReader, readcloser, redirect, err
	}
	log.Warn().Ctx(ctx).Msgf("failed to download from local, err: %v", err)

	if info.PackagePath == "" {
		log.Ctx(ctx).Error().Msgf("Package path is empty for registry %s", info.RegIdentifier)
		return nil, nil, nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("package path is empty"))
	}

	upstream, err := r.proxyStore.Get(ctx, info.RegistryID)
	if err != nil {
		return nil, nil, nil, "", err
	}

	helper, err := NewRemoteRegistryHelper(ctx, r.spaceFinder, *upstream, r.service)
	if err != nil {
		return nil, nil, nil, "", err
	}

	closer, err := helper.GetPackage(ctx, info.PackagePath)
	if err != nil {
		return nil, nil, nil, "", err
	}
	go func() {
		ctx2 := context.WithoutCancel(ctx)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "goRoutine")
		closer2, err2 := helper.GetPackage(ctx2, info.PackagePath)
		if err2 != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		_, _, err := paths.DisectLeaf(info.PackagePath)
		if err != nil {
			log.Ctx(ctx2).Error().Msgf("error while disecting file name for [%s]: %v", info.PackagePath, err)
			return
		}
		_, _, err = r.registryHelper.UploadPackage(ctx2, info, closer2)
		if err != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).Msgf("error while putting file to localRegistry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated file: %s, registry: %s", info.FileName, info.RegIdentifier)
	}()

	return &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    http.StatusOK,
	}, nil, closer, "", nil
}

// GetRepoData returns the metadata of a RPM package.
func (r *proxy) GetRepoData(
	ctx context.Context,
	info rpmtype.ArtifactInfo,
	fileName string,
) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	io.ReadCloser,
	string,
	error,
) {
	return getRepoData(ctx, info, fileName, r.fileManager)
}

// UploadPackageFile FIXME: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (r *proxy) UploadPackageFile(
	ctx context.Context,
	_ rpmtype.ArtifactInfo,
	_ io.Reader,
	_ string,
) (*commons.ResponseHeaders, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

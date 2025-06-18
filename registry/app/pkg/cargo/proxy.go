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

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*proxy)(nil)
var _ Registry = (*proxy)(nil)

type proxy struct {
	fileManager filemanager.FileManager
	proxyStore  store.UpstreamProxyConfigRepository
	tx          dbtx.Transactor
	registryDao store.RegistryRepository
	imageDao    store.ImageRepository
	artifactDao store.ArtifactRepository
	urlProvider urlprovider.Provider
	spaceFinder refcache.SpaceFinder
	service     secret.Service
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
) Proxy {
	return &proxy{
		fileManager: fileManager,
		proxyStore:  proxyStore,
		tx:          tx,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		urlProvider: urlProvider,
		spaceFinder: spaceFinder,
		service:     service,
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

// TODO: Implement download package functionality
func (r *proxy) DownloadPackage(
	ctx context.Context, _ cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, nil, nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) UpdateYank(
	ctx context.Context, _ cargotype.ArtifactInfo, _ bool,
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

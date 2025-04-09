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
}

func (r *proxy) HeadPackageMetadata(_ context.Context, _ npm2.ArtifactInfo) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (r *proxy) ListTags(_ context.Context, _ npm2.ArtifactInfo) (map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (r *proxy) AddTag(_ context.Context, _ npm2.ArtifactInfo) (map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (r *proxy) DeleteTag(_ context.Context, _ npm2.ArtifactInfo) (map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (r *proxy) DeletePackage(_ context.Context, _ npm2.ArtifactInfo) error {
	// TODO implement me
	panic("implement me")
}

func (r *proxy) DeleteVersion(_ context.Context, _ npm2.ArtifactInfo) error {
	// TODO implement me
	panic("implement me")
}

func (r *proxy) GetPackageMetadata(_ context.Context, _ npm2.ArtifactInfo) (npm.PackageMetadata, error) {
	// TODO implement me
	panic("implement me")
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
) Proxy {
	return &proxy{
		proxyStore:  proxyStore,
		registryDao: registryDao,
		imageDao:    imageDao,
		artifactDao: artifactDao,
		fileManager: fileManager,
		tx:          tx,
		urlProvider: urlProvider,
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
	string,
	error,
) {
	headers, body, _, url, errs := r.fetchFile(ctx, info, true)
	return headers, body, url, errs
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

func (r *proxy) fetchFile(_ context.Context, _ npm2.ArtifactInfo, _ bool) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser,
	redirectURL string, err error,
) {
	// TODO implement me
	panic("implement me")
}

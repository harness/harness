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

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	"github.com/harness/gitness/registry/app/remote/controller/proxy/python"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
)

var _ pkg.Artifact = (*proxy)(nil)
var _ Registry = (*proxy)(nil)

type proxy struct {
	fileManager     filemanager.FileManager
	proxyStore      store.UpstreamProxyConfigRepository
	tx              dbtx.Transactor
	registryDao     store.RegistryRepository
	imageDao        store.ImageRepository
	artifactDao     store.ArtifactRepository
	urlProvider     urlprovider.Provider
	proxyController python.Controller
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
	return []artifact.PackageType{artifact.PackageTypePYTHON}
}

func (r *proxy) DownloadPackageFile(ctx context.Context, info pythontype.ArtifactInfo) (
	*commons.ResponseHeaders,
	*storage.FileReader,
	string,
	[]error,
) {
	headers, body, _, url, errs := r.fetchFile(ctx, info, true)
	return headers, body, url, errs
}

// Metadata represents the metadata of a Python package.
func (r *proxy) GetPackageMetadata(
	_ context.Context,
	_ pythontype.ArtifactInfo,
) (pythontype.PackageMetadata, error) {
	return pythontype.PackageMetadata{}, nil
}

// UploadPackageFile FIXME: Extract this upload function for all types of packageTypes
// uploads the package file to the storage.
func (r *proxy) UploadPackageFile(
	ctx context.Context,
	_ pythontype.ArtifactInfo,
	_ multipart.File,
	_ *multipart.FileHeader,
) (*commons.ResponseHeaders, string, errcode.Error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) fetchFile(ctx context.Context, info pythontype.ArtifactInfo, serveFile bool) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	log.Ctx(ctx).Info().Msgf("Maven Proxy: %s", info.RegIdentifier)

	responseHeaders, body, redirectURL, useLocal := r.proxyController.UseLocalFile(ctx, info)
	if useLocal {
		return responseHeaders, body, readCloser, redirectURL, errs
	}

	upstreamProxy, err := r.proxyStore.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return responseHeaders, nil, nil, "", []error{errcode.ErrCodeUnknown.WithDetail(err)}
	}

	// This is start of proxy Code.
	responseHeaders, readCloser, err = r.proxyController.ProxyFile(ctx, info, *upstreamProxy, serveFile)
	if err != nil {
		return responseHeaders, nil, nil, "", []error{errcode.ErrCodeUnknown.WithDetail(err)}
	}
	return responseHeaders, nil, readCloser, "", errs
}

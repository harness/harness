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

	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/secret"
	"github.com/harness/gitness/store/database/dbtx"

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
		artifactEventReporter: artifactEventReporter,
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
) (*commons.ResponseHeaders, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

func (r *proxy) DownloadPackageFile(
	ctx context.Context, _ gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, nil, nil, "", errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

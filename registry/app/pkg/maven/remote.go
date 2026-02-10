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

package maven

import (
	"context"
	"io"

	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/remote/controller/proxy/maven"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/rs/zerolog/log"
)

const (
	ArtifactTypeRemoteRegistry = "Remote Registry"
)

func NewRemoteRegistry(
	dBStore *DBStore, tx dbtx.Transactor, local *LocalRegistry,
	proxyController maven.Controller,
) Registry {
	return &RemoteRegistry{
		DBStore:         dBStore,
		tx:              tx,
		local:           local,
		proxyController: proxyController,
	}
}

type RemoteRegistry struct {
	local           *LocalRegistry
	proxyController maven.Controller
	DBStore         *DBStore
	tx              dbtx.Transactor
}

func (r *RemoteRegistry) GetMavenArtifactType() string {
	return ArtifactTypeRemoteRegistry
}

func (r *RemoteRegistry) HeadArtifact(ctx context.Context, info pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, errs []error,
) {
	responseHeaders, _, _, _, errs = r.fetchArtifact(ctx, info, false)
	return responseHeaders, errs
}

func (r *RemoteRegistry) GetArtifact(ctx context.Context, info pkg.MavenArtifactInfo) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	return r.fetchArtifact(ctx, info, true)
}

func (r *RemoteRegistry) PutArtifact(ctx context.Context, _ pkg.MavenArtifactInfo, _ io.Reader) (
	responseHeaders *commons.ResponseHeaders, errs []error,
) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, nil
}

func (r *RemoteRegistry) fetchArtifact(ctx context.Context, info pkg.MavenArtifactInfo, serveFile bool) (
	responseHeaders *commons.ResponseHeaders, body *storage.FileReader, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	log.Ctx(ctx).Info().Msgf("Maven Proxy: %s", info.RegIdentifier)

	responseHeaders, body, redirectURL, useLocal := r.proxyController.UseLocalFile(ctx, info)
	if useLocal {
		return responseHeaders, body, readCloser, redirectURL, errs
	}

	upstreamProxy, err := r.DBStore.UpstreamProxyDao.GetByRegistryIdentifier(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return processError(err)
	}

	// This is start of proxy Code.
	responseHeaders, readCloser, err = r.proxyController.ProxyFile(ctx, info, *upstreamProxy, serveFile)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to proxy file: %s", info.RegIdentifier)
		return responseHeaders, nil, nil, "", []error{err}
	}
	return responseHeaders, nil, readCloser, "", errs
}

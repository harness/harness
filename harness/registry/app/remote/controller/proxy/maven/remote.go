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

	"github.com/harness/gitness/app/services/refcache"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"

	_ "github.com/harness/gitness/registry/app/remote/adapter/maven" // This is required to init maven adapter
)

const MavenCentralURL = "https://repo1.maven.org/maven2"

// RemoteInterface defines operations related to remote repository under proxy.
type RemoteInterface interface {
	// Download the file
	GetFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, io.ReadCloser, error)

	// Check existence of file
	HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, error)
}

type remoteHelper struct {
	registry      adapter.ArtifactRegistry
	upstreamProxy types.UpstreamProxy
	URL           string
	secretService secret.Service
}

// NewRemoteHelper create a remote interface.
func NewRemoteHelper(
	ctx context.Context, spaceFinder refcache.SpaceFinder, secretService secret.Service,
	proxy types.UpstreamProxy,
) (RemoteInterface, error) {
	if proxy.Source == string(api.UpstreamConfigSourceMavenCentral) {
		proxy.RepoURL = MavenCentralURL
	}
	r := &remoteHelper{
		upstreamProxy: proxy,
		secretService: secretService,
	}
	if err := r.init(ctx, spaceFinder, string(api.UpstreamConfigSourceMavenCentral)); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *remoteHelper) init(ctx context.Context, spaceFinder refcache.SpaceFinder, proxyType string) error {
	if r.registry != nil {
		return nil
	}

	factory, err := adapter.GetFactory(proxyType)
	if err != nil {
		return err
	}
	adp, err := factory.Create(ctx, spaceFinder, r.upstreamProxy, r.secretService)
	if err != nil {
		return err
	}
	reg, ok := adp.(adapter.ArtifactRegistry)
	if !ok {
		log.Ctx(ctx).Warn().Msgf("Error: adp is not of type adapter.ArtifactRegistry")
	}
	r.registry = reg
	return nil
}

func (r *remoteHelper) GetFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, io.ReadCloser, error) {
	return r.registry.GetFile(ctx, filePath)
}

func (r *remoteHelper) HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, error) {
	return r.registry.HeadFile(ctx, filePath)
}

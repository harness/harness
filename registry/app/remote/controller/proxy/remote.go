// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"io"

	"github.com/harness/gitness/app/store"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"

	_ "github.com/harness/gitness/registry/app/remote/adapter/awsecr"    // This is required to init aws ecr adapter
	_ "github.com/harness/gitness/registry/app/remote/adapter/dockerhub" // This is required to init docker adapter
)

const DockerHubURL = "https://registry-1.docker.io"

// RemoteInterface defines operations related to remote repository under proxy.
type RemoteInterface interface {
	// BlobReader create a reader for remote blob.
	BlobReader(registry, dig string) (int64, io.ReadCloser, error)
	// Manifest get manifest by reference.
	Manifest(registry string, ref string) (manifest.Manifest, string, error)
	// ManifestExist checks manifest exist, if exist, return digest.
	ManifestExist(registry string, ref string) (bool, *manifest.Descriptor, error)
	// ListTags returns all tags of the registry.
	ListTags(registry string) ([]string, error)
}

type remoteHelper struct {
	repoKey string
	// TODO: do we need image name here also?
	registry      adapter.ArtifactRegistry
	upstreamProxy types.UpstreamProxy
	URL           string
	secretService secret.Service
}

// NewRemoteHelper create a remote interface.
func NewRemoteHelper(
	ctx context.Context, spacePathStore store.SpacePathStore, secretService secret.Service, repoKey string,
	proxy types.UpstreamProxy,
) (RemoteInterface, error) {
	if proxy.Source == string(api.UpstreamConfigSourceDockerhub) {
		proxy.RepoURL = DockerHubURL
	}
	r := &remoteHelper{
		repoKey:       repoKey,
		upstreamProxy: proxy,
		secretService: secretService,
	}
	if err := r.init(ctx, spacePathStore, proxy.Source); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *remoteHelper) init(ctx context.Context, spacePathStore store.SpacePathStore, proxyType string) error {
	if r.registry != nil {
		return nil
	}

	// TODO add health check.
	factory, err := adapter.GetFactory(proxyType)
	if err != nil {
		return err
	}
	adp, err := factory.Create(ctx, spacePathStore, r.upstreamProxy, r.secretService)
	if err != nil {
		return err
	}
	reg, ok := adp.(adapter.ArtifactRegistry)
	if !ok {
		log.Warn().Msgf("Error: adp is not of type adapter.ArtifactRegistry")
	}
	r.registry = reg
	return nil
}

func (r *remoteHelper) BlobReader(registry, dig string) (int64, io.ReadCloser, error) {
	return r.registry.PullBlob(registry, dig)
}

func (r *remoteHelper) Manifest(registry string, ref string) (manifest.Manifest, string, error) {
	return r.registry.PullManifest(registry, ref)
}

func (r *remoteHelper) ManifestExist(registry string, ref string) (bool, *manifest.Descriptor, error) {
	return r.registry.ManifestExist(registry, ref)
}

func (r *remoteHelper) ListTags(registry string) ([]string, error) {
	return r.registry.ListTags(registry)
}

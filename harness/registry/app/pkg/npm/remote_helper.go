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
	"io"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/npmjs"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type RemoteRegistryHelper interface {
	// GetFile Downloads the file for the given package and filename
	GetPackage(ctx context.Context, pkg string, version string) (io.ReadCloser, error)

	// GetMetadata Fetches the metadata for the given package for all versions
	GetPackageMetadata(ctx context.Context, pkg string) (*npm.PackageMetadata, error)

	GetVersionMetadata(ctx context.Context, pkg string, version string) (*npm.PackageMetadata, error)
}

type remoteRegistryHelper struct {
	adapter  registry.NpmRegistry
	registry types.UpstreamProxy
}

func NewRemoteRegistryHelper(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	registry types.UpstreamProxy,
	service secret.Service,
) (RemoteRegistryHelper, error) {
	r := &remoteRegistryHelper{
		registry: registry,
	}
	if err := r.init(ctx, spaceFinder, service); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to init remote registry for remote: %s", registry.RepoKey)
		return nil, err
	}
	return r, nil
}

func (r *remoteRegistryHelper) init(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
) error {
	key := string(artifact.PackageTypeNPM)
	if r.registry.Source == string(artifact.UpstreamConfigSourceNpmJs) {
		r.registry.RepoURL = npmjs.NpmjsURL
	}

	factory, err := adapter.GetFactory(key)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get factory " + key)
		return err
	}

	adpt, err := factory.Create(ctx, spaceFinder, r.registry, service)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to create factory " + key)
		return err
	}

	npmReg, ok := adpt.(registry.NpmRegistry)
	if !ok {
		log.Ctx(ctx).Error().Msg("failed to cast factory to npm registry")
		return err
	}
	r.adapter = npmReg
	return nil
}

func (r *remoteRegistryHelper) GetPackage(ctx context.Context, pkg string, version string) (io.ReadCloser, error) {
	v2, err := r.adapter.GetPackage(ctx, pkg, version)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get pkg: %s, version: %s", pkg, version)
	}
	return v2, err
}

func (r *remoteRegistryHelper) GetPackageMetadata(ctx context.Context, pkg string) (*npm.PackageMetadata, error) {
	packages, err := r.adapter.GetPackageMetadata(ctx, pkg)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get metadata for pkg: %s", pkg)
		return nil, err
	}
	return packages, nil
}

func (r *remoteRegistryHelper) GetVersionMetadata(ctx context.Context,
	pkg string, version string) (*npm.PackageMetadata, error) {
	metadata, err := r.adapter.GetPackageMetadata(ctx, pkg)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get metadata for pkg: %s", pkg)
		return nil, err
	}
	for _, p := range metadata.Versions {
		if p.Version == version {
			metadata.Versions = map[string]*npm.PackageMetadataVersion{p.Version: p}
			break
		}
	}
	return metadata, nil
}

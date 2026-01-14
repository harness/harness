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
	"io"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type RemoteRegistryHelper interface {
	GetMetadataFile(ctx context.Context, filePath string) (io.ReadCloser, error)
	GetPackage(ctx context.Context, pkg string) (io.ReadCloser, error)
}

type remoteRegistryHelper struct {
	adapter  registry.RpmRegistry
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
	key := string(artifact.PackageTypeRPM)

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

	rpmReg, ok := adpt.(registry.RpmRegistry)
	if !ok {
		log.Ctx(ctx).Error().Msg("failed to cast factory to rpm registry")
		return err
	}
	r.adapter = rpmReg
	return nil
}

func (r *remoteRegistryHelper) GetMetadataFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	v2, err := r.adapter.GetMetadataFile(ctx, filePath)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get metadata file: %s", filePath)
	}
	return v2, err
}

func (r *remoteRegistryHelper) GetPackage(ctx context.Context, pkg string) (io.ReadCloser, error) {
	packages, err := r.adapter.GetPackage(ctx, pkg)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to get metadata for pkg: %s", pkg)
		return nil, err
	}
	return packages, nil
}

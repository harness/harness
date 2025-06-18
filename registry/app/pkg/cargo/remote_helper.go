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

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/remote/adapter"
	crates "github.com/harness/gitness/registry/app/remote/adapter/crates"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type RemoteRegistryHelper interface {
	// GetFile Downloads the file for the given package and filename
	GetPackageFile(ctx context.Context, pkg string, version string) (io.ReadCloser, error)

	// GetPackageIndex Fetches the package index for the given package
	GetPackageIndex(pkg string, filePath string) (io.ReadCloser, error)
}

type remoteRegistryHelper struct {
	adapter  registry.CargoRegistry
	registry types.UpstreamProxy
}

func NewRemoteRegistryHelper(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	registry types.UpstreamProxy,
	service secret.Service,
	remoteURL string,
) (RemoteRegistryHelper, error) {
	r := &remoteRegistryHelper{
		registry: registry,
	}
	if err := r.init(ctx, spaceFinder, service, remoteURL); err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to init remote registry for remote: %s", registry.RepoKey)
		return nil, err
	}
	return r, nil
}

func (r *remoteRegistryHelper) init(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
	remoteURL string,
) error {
	key := string(artifact.PackageTypeCARGO)
	if r.registry.Source == string(artifact.UpstreamConfigSourceCrates) {
		r.registry.RepoURL = crates.CratesURL
	}
	if remoteURL != "" {
		r.registry.RepoURL = remoteURL
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

	cargoReg, ok := adpt.(registry.CargoRegistry)
	if !ok {
		log.Ctx(ctx).Error().Msg("failed to cast factory to cargo registry")
		return err
	}
	r.adapter = cargoReg
	return nil
}

func (r *remoteRegistryHelper) GetPackageIndex(
	pkg string, filePath string,
) (io.ReadCloser, error) {
	data, err := r.adapter.GetPackageFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get package index: %s", pkg)
	}
	if data == nil {
		return nil, fmt.Errorf("index metadata not found for package: %s", pkg)
	}
	return data, err
}

func (r *remoteRegistryHelper) GetPackageFile(
	ctx context.Context, _ string, _ string,
) (io.ReadCloser, error) {
	log.Error().Ctx(ctx).Msg("Not implemented")
	return nil, errcode.ErrCodeInvalidRequest.WithDetail(fmt.Errorf("not implemented"))
}

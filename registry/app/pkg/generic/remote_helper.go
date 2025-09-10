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

package generic

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

type RemoteRegistryHelper interface {
	// GetFile Downloads the file for the given package and filename
	GetFile(ctx context.Context, filePath string) (io.ReadCloser, error)
	HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, error)
}

type remoteRegistryHelper struct {
	adapter  registry.GenericRegistry
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
		return nil, fmt.Errorf("failed to init remote registry: %w", err)
	}
	return r, nil
}

func (r *remoteRegistryHelper) init(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	service secret.Service,
) error {
	key := string(artifact.PackageTypeGENERIC)
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

	genericReg, ok := adpt.(registry.GenericRegistry)
	if !ok {
		log.Ctx(ctx).Error().Msg("failed to cast factory to generic registry")
		return fmt.Errorf("adapter not a GenericRegistry (got %T)", adpt)
	}
	r.adapter = genericReg
	return nil
}

func (r *remoteRegistryHelper) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	headers, readCloser, err := r.adapter.GetFile(ctx, filePath)
	if err != nil {
		evt := log.Ctx(ctx).Error().Err(err).Str("file", filePath)
		if headers != nil {
			evt = evt.Int("code", headers.Code).Interface("headers", headers.Headers)
		} else {
			evt = evt.Str("headers", "<nil>")
		}
		evt.Msgf("GetFile failed")
		return nil, fmt.Errorf("GetFile Failure: %w", err)
	}
	return readCloser, nil
}

func (r *remoteRegistryHelper) HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, error) {
	headers, err := r.adapter.HeadFile(ctx, filePath)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to head file: %s", filePath)
		return nil, fmt.Errorf("HeadFile Failure: %w", err)
	}
	return headers, nil
}

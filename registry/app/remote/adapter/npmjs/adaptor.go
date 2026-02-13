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

package npmjs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/metadata/npm"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

var _ registry.NpmRegistry = (*adapter)(nil)
var _ adp.Adapter = (*adapter)(nil)

const (
	NpmjsURL = "https://registry.npmjs.org"
)

type adapter struct {
	*native.Adapter
	registry types.UpstreamProxy
	client   *client
}

func newAdapter(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	registry types.UpstreamProxy,
	service secret.Service,
) (adp.Adapter, error) {
	nativeAdapter, err := native.NewAdapter(ctx, spaceFinder, service, registry)
	if err != nil {
		return nil, err
	}
	c, err := newClient(ctx, registry, spaceFinder, service)
	if err != nil {
		return nil, err
	}

	return &adapter{
		Adapter:  nativeAdapter,
		registry: registry,
		client:   c,
	}, nil
}

type factory struct {
}

func (f *factory) Create(
	ctx context.Context, spaceFinder refcache.SpaceFinder, record types.UpstreamProxy, service secret.Service,
) (adp.Adapter, error) {
	return newAdapter(ctx, spaceFinder, record, service)
}

func init() {
	adapterType := string(artifact.PackageTypeNPM)
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Failed to register adapter factory for %s", adapterType)
		return
	}
}

func (a *adapter) GetPackageMetadata(ctx context.Context, pkg string) (*npm.PackageMetadata, error) {
	_, readCloser, err := a.GetFile(ctx, pkg)
	if err != nil {
		code := errors.ErrCode(err)
		if code == errors.NotFoundCode {
			return nil, usererror.NotFoundf("failed to get package metadata %s", pkg)
		}
		if code == errors.ForbiddenCode {
			return nil, usererror.Forbidden(fmt.Sprintf("failed to get package metadata %s", pkg))
		}
	}
	defer readCloser.Close()
	response, err := ParseNPMMetadata(readCloser)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (a *adapter) GetPackage(ctx context.Context, pkg string, version string) (io.ReadCloser, error) {
	metadata, err := a.GetPackageMetadata(ctx, pkg)
	if err != nil {
		return nil, err
	}

	downloadURL := ""

	for _, p := range metadata.Versions {
		if p.Version == version {
			downloadURL = p.Dist.Tarball
			break
		}
	}

	if downloadURL == "" {
		return nil, fmt.Errorf("pkg: %s, version: %s not found", pkg, version)
	}

	log.Ctx(ctx).Info().Msgf("Download URL: %s", downloadURL)
	_, closer, err := a.GetFileFromURL(ctx, downloadURL)
	if err != nil {
		code := errors.ErrCode(err)
		if code == errors.NotFoundCode {
			return nil, usererror.NotFoundf("failed to get package file %s", pkg+version)
		}
		if code == errors.ForbiddenCode {
			return nil, usererror.Forbidden(fmt.Sprintf("failed to get package file %s", pkg+version))
		}
	}
	return closer, nil
}

// ParseNPMMetadata parses the given Json and returns a SimpleMetadata DTO.
func ParseNPMMetadata(r io.ReadCloser) (npm.PackageMetadata, error) {
	var metadata npm.PackageMetadata
	if err := json.NewDecoder(r).Decode(&metadata); err != nil {
		return npm.PackageMetadata{}, err
	}

	return metadata, nil
}

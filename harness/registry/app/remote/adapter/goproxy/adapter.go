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

package goproxy

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

var _ registry.GoPackageRegistry = (*adapter)(nil)
var _ adp.Adapter = (*adapter)(nil)

const (
	GoProxyURL = "https://proxy.golang.org"
)

type adapter struct {
	*native.Adapter
}

func newAdapter(
	ctx context.Context,
	spaceFinder refcache.SpaceFinder,
	registry types.UpstreamProxy,
	service secret.Service,
) (adp.Adapter, error) {
	nativeAdapter := native.NewAdapter(ctx, spaceFinder, service, registry)
	return &adapter{
		Adapter: nativeAdapter,
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
	adapterType := string(artifact.PackageTypeGO)
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Failed to register adapter factory for %s", adapterType)
		return
	}
}

func (a *adapter) GetPackageFile(ctx context.Context, filepath string) (io.ReadCloser, error) {
	_, readCloser, err := a.GetFile(ctx, filepath)
	if err != nil {
		code := errors.ErrCode(err)
		if code == errors.NotFoundCode {
			return nil, usererror.NotFoundf("failed to get package file %s", filepath)
		}
		if code == errors.ForbiddenCode {
			return nil, usererror.Forbidden(fmt.Sprintf("failed to get package file %s", filepath))
		}
		return nil, err
	}
	return readCloser, nil
}

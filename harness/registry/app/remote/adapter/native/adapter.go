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

package native

import (
	"context"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/common/lib"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/commons"
	"github.com/harness/gitness/registry/app/remote/clients/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

var _ adp.Adapter = &Adapter{}

var (
	_ adp.Adapter          = (*Adapter)(nil)
	_ adp.ArtifactRegistry = (*Adapter)(nil)
)

// Adapter implements an adapter for Docker proxy. It can be used to all registries
// that implement the proxy V2 API.
type Adapter struct {
	proxy types.UpstreamProxy
	registry.Client
}

// NewAdapter returns an instance of the Adapter.
func NewAdapter(
	ctx context.Context, spaceFinder refcache.SpaceFinder, service secret.Service, reg types.UpstreamProxy,
) *Adapter {
	adapter := &Adapter{
		proxy: reg,
	}

	url := reg.RepoURL
	accessKey, secretKey, _, err := commons.GetCredentials(ctx, spaceFinder, service, reg)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("error getting credentials for registry: %s", reg.RepoKey)
		return nil
	}

	adapter.Client = registry.NewClient(url, accessKey, secretKey, false,
		reg.PackageType == artifact.PackageTypeDOCKER || reg.PackageType == artifact.PackageTypeHELM)
	return adapter
}

// NewAdapterWithAuthorizer returns an instance of the Adapter with provided authorizer.
func NewAdapterWithAuthorizer(reg types.UpstreamProxy, authorizer lib.Authorizer) *Adapter {
	return &Adapter{
		proxy:  reg,
		Client: registry.NewClientWithAuthorizer(reg.RepoURL, authorizer, false),
	}
}

// HealthCheck checks health status of a proxy.
func (a *Adapter) HealthCheck() (string, error) {
	return "Not implemented", nil
}

// PingSimple checks whether the proxy is available. It checks the connectivity and certificate (if TLS enabled)
// only, regardless of 401/403 error.
func (a *Adapter) PingSimple(ctx context.Context) error {
	err := a.Ping(ctx)
	if err == nil {
		return nil
	}
	if errors.IsErr(err, errors.UnAuthorizedCode) || errors.IsErr(err, errors.ForbiddenCode) {
		return nil
	}
	return err
}

// DeleteTag isn't supported for docker proxy.
func (a *Adapter) DeleteTag(_ context.Context, _, _ string) error {
	return errors.New("the tag deletion isn't supported")
}

// CanBeMount isn't supported for docker proxy.
func (a *Adapter) CanBeMount(_ context.Context, _ string) (mount bool, repository string, err error) {
	return false, "", nil
}

func (a *Adapter) GetImageName(imageName string) (string, error) {
	return imageName, nil
}

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

package nuget

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/types/nuget"
	adp "github.com/harness/gitness/registry/app/remote/adapter"
	"github.com/harness/gitness/registry/app/remote/adapter/native"
	"github.com/harness/gitness/registry/app/remote/registry"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/rs/zerolog/log"
)

var _ registry.NugetRegistry = (*adapter)(nil)
var _ adp.Adapter = (*adapter)(nil)

const (
	NugetOrgURL         = "https://api.nuget.org/v3/index.json"
	RegistrationBaseURL = "RegistrationsBaseUrl"
	PackageBaseAddress  = "PackageBaseAddress"
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
	nativeAdapter := native.NewAdapter(ctx, spaceFinder, service, registry)
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
	adapterType := string(artifact.PackageTypeNUGET)
	if err := adp.RegisterFactory(adapterType, new(factory)); err != nil {
		log.Error().Stack().Err(err).Msgf("Failed to register adapter factory for %s", adapterType)
		return
	}
}

func (a adapter) GetServiceEndpoint() (*nuget.ServiceEndpoint, error) {
	_, readCloser, err := a.GetFileFromURL(a.client.url)
	if err != nil {
		return nil, err
	}
	defer readCloser.Close()
	response, err := ParseServiceEndpointResponse(readCloser)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (a adapter) GetPackageMetadata(ctx context.Context, pkg, proxyEndpoint string) (io.ReadCloser, error) {
	var packageMetadataEndpoint string
	if proxyEndpoint != "" {
		packageMetadataEndpoint = proxyEndpoint
	} else {
		svcEndpoints, err := a.GetServiceEndpoint()
		if err != nil {
			return nil, err
		}
		baseURL, err := getResourceByTypePrefix(svcEndpoints, RegistrationBaseURL)
		if err != nil {
			return nil, err
		}
		packageMetadataEndpoint = fmt.Sprintf("%s/%s/index.json", strings.TrimRight(baseURL, "/"), pkg)
	}
	log.Ctx(ctx).Info().Msgf("Package Metadata URL: %s", packageMetadataEndpoint)

	_, readCloser, err := a.GetFileFromURL(packageMetadataEndpoint)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file from URL: %s", packageMetadataEndpoint)
		return nil, err
	}
	return readCloser, nil
}

func (a adapter) GetPackageVersionMetadataV2(ctx context.Context, pkg, version string) (io.ReadCloser, error) {
	baseURL := a.client.url
	packageVersionEndpoint := fmt.Sprintf("%s/Packages(Id='%s',Version='%s')", baseURL, pkg, version)
	log.Ctx(ctx).Info().Msgf("Package Version V2 Metadata URL: %s", packageVersionEndpoint)

	_, readCloser, err := a.GetFileFromURL(packageVersionEndpoint)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file from URL: %s", packageVersionEndpoint)
		return nil, err
	}
	return readCloser, nil
}

func (a adapter) GetPackage(ctx context.Context, pkg, version, proxyEndpoint, fileName string) (io.ReadCloser, error) {
	var packageEndpoint string
	if proxyEndpoint != "" {
		packageEndpoint = proxyEndpoint
	} else {
		svcEndpoints, err := a.GetServiceEndpoint()
		if err != nil {
			return nil, err
		}
		baseURL, err := getResourceByTypePrefix(svcEndpoints, PackageBaseAddress)
		if err != nil {
			return nil, err
		}
		packageEndpoint = fmt.Sprintf("%s/%s/%s/%s.%s.%s", strings.TrimRight(baseURL, "/"), pkg,
			version, pkg, version, fileName)
	}

	log.Ctx(ctx).Info().Msgf("Package URL: %s", packageEndpoint)
	_, closer, err := a.GetFileFromURL(packageEndpoint)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file from URL: %s", packageEndpoint)
		return nil, err
	}
	return closer, nil
}

func (a adapter) ListPackageVersion(ctx context.Context, pkg string) (io.ReadCloser, error) {
	svcEndpoints, err := a.GetServiceEndpoint()
	if err != nil {
		return nil, err
	}
	baseURL, err := getResourceByTypePrefix(svcEndpoints, PackageBaseAddress)
	if err != nil {
		return nil, err
	}
	versionEndpoint := fmt.Sprintf("%s/%s/index.json", baseURL, pkg)
	log.Ctx(ctx).Info().Msgf("List Version URL: %s", versionEndpoint)
	_, closer, err := a.GetFileFromURL(versionEndpoint)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file from URL: %s", versionEndpoint)
		return nil, err
	}
	return closer, nil
}

func (a adapter) ListPackageVersionV2(ctx context.Context, pkg string) (io.ReadCloser, error) {
	baseURL := a.client.url
	versionEndpoint := fmt.Sprintf("%s/FindPackagesById()?id='%s'", baseURL, pkg)
	log.Ctx(ctx).Info().Msgf("List Version V2 URL: %s", versionEndpoint)
	_, closer, err := a.GetFileFromURL(versionEndpoint)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get file from URL: %s", versionEndpoint)
		return nil, err
	}
	return closer, nil
}

func ParseServiceEndpointResponse(r io.ReadCloser) (nuget.ServiceEndpoint, error) {
	var result nuget.ServiceEndpoint
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return nuget.ServiceEndpoint{}, err
	}
	return result, nil
}

func getResourceByTypePrefix(endpoints *nuget.ServiceEndpoint, typePrefix string) (string, error) {
	var resource *nuget.Resource
	for _, r := range endpoints.Resources {
		if strings.HasPrefix(r.Type, typePrefix) {
			if resource == nil || r.Type > resource.Type {
				resource = &r
			}
		}
	}
	if resource == nil {
		return "", fmt.Errorf("resource %s not found", typePrefix)
	}
	return resource.ID, nil
}

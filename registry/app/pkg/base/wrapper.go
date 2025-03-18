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

package base

import (
	"context"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/response"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

var TypeRegistry = map[string]pkg.Artifact{}

func Register(registries ...pkg.Artifact) {
	for _, r := range registries {
		for _, packageType := range r.GetPackageTypes() {
			log.Info().Msgf("Registering package type %s with artifact type %s", packageType, r.GetArtifactType())
			key := getFactoryKey(packageType, r.GetArtifactType())
			TypeRegistry[key] = r
		}
	}
}

func NoProxyWrapper(
	ctx context.Context,
	registryDao store.RegistryRepository,
	f func(registry registrytypes.Registry, a pkg.Artifact) response.Response,
	info pkg.ArtifactInfo,
) response.Response {
	var result response.Response
	registry, err := registryDao.Get(ctx, info.RegistryID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to get registry by ID %d", info.RegistryID)
		return result
	}

	log.Ctx(ctx).Info().Msgf("Using Repository: %s, Type: %s", registry.Name, registry.Type)
	art := getArtifactRegistry(*registry)
	result = f(*registry, art)
	if pkg.IsEmpty(result.GetErrors()) {
		return result
	}
	log.Ctx(ctx).Warn().Msgf("Repository: %s, Type: %s, errors: %v", registry.Name, registry.Type,
		result.GetErrors())

	return result
}

func ProxyWrapper(
	ctx context.Context,
	registryDao store.RegistryRepository,
	f func(registry registrytypes.Registry, a pkg.Artifact) response.Response,
	info pkg.ArtifactInfo,
) response.Response {
	var response response.Response
	requestRepoKey := info.RegIdentifier
	if repos, err := getOrderedRepos(ctx, registryDao, requestRepoKey, *info.BaseInfo); err == nil {
		for _, registry := range repos {
			log.Ctx(ctx).Info().Msgf("Using Repository: %s, Type: %s", registry.Name, registry.Type)
			reg := getArtifactRegistry(registry)
			if reg != nil {
				response = f(registry, reg)
				if pkg.IsEmpty(response.GetErrors()) {
					return response
				}
				log.Ctx(ctx).Warn().Msgf("Repository: %s, Type: %s, errors: %v", registry.Name, registry.Type,
					response.GetErrors())
			}
		}
	}
	return response
}

func factory(key string) pkg.Artifact {
	return TypeRegistry[key]
}

func getFactoryKey(packageType artifact.PackageType, registryType artifact.RegistryType) string {
	return string(packageType) + ":" + string(registryType)
}

func getArtifactRegistry(registry registrytypes.Registry) pkg.Artifact {
	key := getFactoryKey(registry.PackageType, registry.Type)
	return factory(key)
}

func getOrderedRepos(
	ctx context.Context,
	registryDao store.RegistryRepository,
	repoKey string,
	artInfo pkg.BaseInfo,
) ([]registrytypes.Registry, error) {
	var result []registrytypes.Registry
	if registry, err := registryDao.GetByParentIDAndName(ctx, artInfo.ParentID, repoKey); err == nil {
		result = append(result, *registry)
		proxies := registry.UpstreamProxies
		if len(proxies) > 0 {
			upstreamRepos, _ := registryDao.GetByIDIn(ctx, proxies)
			result = append(result, *upstreamRepos...)
		}
	} else {
		return result, err
	}

	return result, nil
}

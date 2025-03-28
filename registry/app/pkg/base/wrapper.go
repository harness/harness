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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/registry/app/api/handler/utils"
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
	info pkg.PackageArtifactInfo,
) (response.Response, error) {
	return proxyInternal(ctx, registryDao, f, info, false)
}

func ProxyWrapper(
	ctx context.Context,
	registryDao store.RegistryRepository,
	f func(registry registrytypes.Registry, a pkg.Artifact) response.Response,
	info pkg.PackageArtifactInfo,
) (response.Response, error) {
	return proxyInternal(ctx, registryDao, f, info, true)
}

func proxyInternal(
	ctx context.Context,
	registryDao store.RegistryRepository,
	f func(registry registrytypes.Registry, a pkg.Artifact) response.Response,
	info pkg.PackageArtifactInfo,
	useUpstream bool,
) (response.Response, error) {
	var r response.Response
	requestRepoKey := info.BaseArtifactInfo().RegIdentifier
	registries, skipped, err := filterRegs(ctx, registryDao, requestRepoKey, info, useUpstream)
	if err != nil {
		return r, err
	}

	for _, registry := range registries {
		log.Ctx(ctx).Info().Msgf("Using Registry: %s, Type: %s", registry.Name, registry.Type)
		art := getArtifactRegistry(registry)
		if art != nil {
			r = f(registry, art)
			if r.GetError() == nil {
				return r, nil
			}
			log.Ctx(ctx).Warn().Msgf("Repository: %s, Type: %s, error: %v", registry.Name, registry.Type,
				r.GetError())
		}
	}

	if !pkg.IsEmpty(skipped) {
		var skippedRegNames []string
		for _, registry := range skipped {
			skippedRegNames = append(skippedRegNames, registry.Name)
		}
		return r, errors.NotFound("no matching artifacts found in registry %s, skipped registries: [%s] "+
			"due to allowed/blocked policies",
			requestRepoKey, pkg.JoinWithSeparator(", ", skippedRegNames...))
	}

	return r, errors.NotFound("no matching artifacts found in registry %s", requestRepoKey)
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

func filterRegs(
	ctx context.Context,
	registryDao store.RegistryRepository,
	repoKey string,
	info pkg.PackageArtifactInfo,
	upstream bool,
) (included []registrytypes.Registry, skipped []registrytypes.Registry, err error) {
	registries, err := getOrderedRepos(ctx, registryDao, repoKey, info.BaseArtifactInfo().ParentID, upstream)
	if err != nil {
		return nil, nil, err
	}

	exists, imageVersion := info.GetImageVersion()
	if !exists {
		log.Debug().Msgf("image version not ready for %s, skipping filter", repoKey)
		return registries, nil, nil
	}

	for _, repo := range registries {
		allowedPatterns := repo.AllowedPattern
		blockedPatterns := repo.BlockedPattern
		isAllowed, err := utils.IsPatternAllowed(allowedPatterns, blockedPatterns, imageVersion)
		if !isAllowed || err != nil {
			log.Debug().Ctx(ctx).Msgf("Skipping repository %s", repo.Name)
			skipped = append(skipped, repo)
			continue
		}
		included = append(included, repo)
	}
	return included, skipped, nil
}

func getOrderedRepos(
	ctx context.Context,
	registryDao store.RegistryRepository,
	repoKey string,
	parentID int64,
	upstream bool,
) ([]registrytypes.Registry, error) {
	var result []registrytypes.Registry
	registry, err := registryDao.GetByParentIDAndName(ctx, parentID, repoKey)
	if err != nil {
		return result, errors.NotFound("registry %s not found", repoKey)
	}
	result = append(result, *registry)
	if !upstream {
		return result, nil
	}
	proxies := registry.UpstreamProxies
	if len(proxies) > 0 {
		upstreamRepos, err2 := registryDao.GetByIDIn(ctx, proxies)
		if err2 != nil {
			log.Error().Msgf("Failed to get upstream proxies for %s: %v", repoKey, err2)
			return result, err2
		}
		repoMap := make(map[int64]registrytypes.Registry)
		for _, repo := range *upstreamRepos {
			repoMap[repo.ID] = repo
		}
		for _, proxyID := range proxies {
			if repo, ok := repoMap[proxyID]; ok {
				result = append(result, repo)
			}
		}
	}
	return result, nil
}

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

	var lastError error

	for _, registry := range registries {
		log.Ctx(ctx).Info().Msgf("Using Registry: %s, Type: %s", registry.Name, registry.Type)
		art := GetArtifactRegistry(registry)
		if art != nil {
			r = f(registry, art)
			if r.GetError() == nil {
				return r, nil
			}
			lastError = r.GetError()
			log.Ctx(ctx).Warn().Msgf("Repository: %s, Type: %s, error: %v", registry.Name, registry.Type,
				r.GetError())
		}
	}

	if !pkg.IsEmpty(skipped) {
		var skippedRegNames []string
		for _, registry := range skipped {
			skippedRegNames = append(skippedRegNames, registry.Name)
		}
		return r, errors.InvalidArgumentf("no matching artifacts found in registry %s, skipped registries: [%s] "+
			"due to allowed/blocked policies",
			requestRepoKey, pkg.JoinWithSeparator(", ", skippedRegNames...))
	}

	if lastError != nil {
		return r, lastError
	}

	return r, errors.NotFoundf("no matching artifacts found in registry %s", requestRepoKey)
}

func factory(key string) pkg.Artifact {
	return TypeRegistry[key]
}

func getFactoryKey(packageType artifact.PackageType, registryType artifact.RegistryType) string {
	return string(packageType) + ":" + string(registryType)
}

func GetArtifactRegistry(registry registrytypes.Registry) pkg.Artifact {
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
	registries, err := GetOrderedRepos(ctx, registryDao, repoKey, info.BaseArtifactInfo().ParentID, upstream)
	if err != nil {
		return nil, nil, err
	}

	exists, imageVersion := info.GetImageVersion()
	if !exists {
		log.Ctx(ctx).Debug().Msgf("image version not ready for %s, skipping filter", repoKey)
		return registries, nil, nil
	}
	for _, repo := range registries {
		allowedPatterns := repo.AllowedPattern
		blockedPatterns := repo.BlockedPattern
		err2 := utils.PatternAllowed(allowedPatterns, blockedPatterns, imageVersion)
		if err2 != nil {
			log.Ctx(ctx).Debug().Msgf("Skipping repository %s", repo.Name)
			skipped = append(skipped, repo)
			continue
		}
		included = append(included, repo)
	}
	return included, skipped, nil
}

func GetOrderedRepos(
	ctx context.Context,
	registryDao store.RegistryRepository,
	repoKey string,
	parentID int64,
	upstream bool,
) ([]registrytypes.Registry, error) {
	var result []registrytypes.Registry
	registry, err := registryDao.GetByParentIDAndName(ctx, parentID, repoKey)
	if err != nil {
		return result, errors.NotFoundf("registry %s not found", repoKey)
	}
	result = append(result, *registry)
	if !upstream {
		return result, nil
	}
	proxies := registry.UpstreamProxies
	if len(proxies) > 0 {
		upstreamRepos, err2 := registryDao.GetByIDIn(ctx, proxies)
		if err2 != nil {
			log.Ctx(ctx).Error().Msgf("Failed to get upstream proxies for %s: %v", repoKey, err2)
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

func SearchPackagesProxyWrapper(
	ctx context.Context,
	registryDao store.RegistryRepository,
	f func(registry registrytypes.Registry, a pkg.Artifact, limit, offset int) response.Response,
	extractResponseDataFunc func(searchResponse response.Response) ([]any, int64),
	info pkg.PackageArtifactInfo,
	limit int,
	offset int,
) ([]any, int64, error) {
	requestRepoKey := info.BaseArtifactInfo().RegIdentifier

	// Get all registries (local + proxies) - Z = [x, p1, p2, p3...]
	registries, skipped, err := filterRegs(ctx, registryDao, requestRepoKey, info, true)
	if err != nil {
		return nil, 0, err
	}

	// Pre-allocate slice for aggregated results (generic interface{})
	var aggregatedResults []any
	var totalCount int64

	currentOffset := offset
	remainingLimit := limit

	// Loop through each registry R in Z
	for _, registry := range registries {
		if remainingLimit <= 0 {
			break
		}

		log.Ctx(ctx).Info().Msgf("Searching in Registry: %s, Type: %s", registry.Name, registry.Type)
		art := GetArtifactRegistry(registry)
		if art == nil {
			log.Ctx(ctx).Warn().Msgf("No artifact registry found for: %s", registry.Name)
			continue
		}

		// 1. Call f with remainingLimit and currentOffset
		searchResponse := f(registry, art, remainingLimit, currentOffset)
		if searchResponse.GetError() != nil {
			log.Ctx(ctx).Warn().Msgf("Search failed for registry %s: %v", registry.Name, searchResponse.GetError())
			continue
		}

		// Extract response data using the provided function
		nativeResults, totalHits := extractResponseDataFunc(searchResponse)
		resultsSize := len(nativeResults)
		totalCount += totalHits

		if resultsSize == 0 {
			continue
		}

		// 2. Append search results (preserve native types)
		aggregatedResults = append(aggregatedResults, nativeResults...)

		// 3. Update currentOffset = max(0, currentOffset - registryResults.size)
		currentOffset = max(0, currentOffset-resultsSize)

		// 4. Update remainingLimit = remainingLimit - registryResults.size
		remainingLimit -= resultsSize

		// Early exit if we have enough results
		if remainingLimit <= 0 {
			break
		}
	}

	// Log skipped registries if any
	if !pkg.IsEmpty(skipped) {
		skippedRegNames := make([]string, 0, len(skipped))
		for _, registry := range skipped {
			skippedRegNames = append(skippedRegNames, registry.Name)
		}
		log.Ctx(ctx).Warn().Msgf("Skipped registries due to policies: [%s]",
			pkg.JoinWithSeparator(", ", skippedRegNames...))
	}

	return aggregatedResults, totalCount, nil
}

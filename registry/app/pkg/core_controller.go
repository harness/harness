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

package pkg

import (
	"context"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	store2 "github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/rs/zerolog/log"
)

type ArtifactType int

const (
	LocalRegistry ArtifactType = 1 << iota
	RemoteRegistry
)

var TypeRegistry = map[ArtifactType]Artifact{}

type CoreController struct {
	RegistryDao store2.RegistryRepository
}

func NewCoreController(registryDao store2.RegistryRepository) *CoreController {
	return &CoreController{
		RegistryDao: registryDao,
	}
}

func (c *CoreController) factory(t ArtifactType) Artifact {
	switch t {
	case LocalRegistry:
		return TypeRegistry[t]
	case RemoteRegistry:
		return TypeRegistry[t]
	default:
		log.Error().Stack().Msgf("Invalid artifact type %v", t)
		return nil
	}
}

func (c *CoreController) GetArtifact(registry types.Registry) Artifact {
	if string(registry.Type) == string(artifact.RegistryTypeVIRTUAL) {
		return c.factory(LocalRegistry)
	}
	return c.factory(RemoteRegistry)
}

func (c *CoreController) GetOrderedRepos(
	ctx context.Context,
	repoKey string,
	artInfo RegistryInfo,
) ([]types.Registry, error) {
	var result []types.Registry
	if registry, err := c.RegistryDao.GetByParentIDAndName(ctx, artInfo.ParentID, repoKey); err == nil {
		result = append(result, *registry)
		proxies := registry.UpstreamProxies
		if len(proxies) > 0 {
			upstreamRepos, _ := c.RegistryDao.GetByIDIn(ctx, proxies)
			result = append(result, *upstreamRepos...)
		}
	} else {
		return result, err
	}

	return result, nil
}

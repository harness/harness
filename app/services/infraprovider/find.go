// Copyright 2023 Harness, Inc.
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

package infraprovider

import (
	"context"
	"fmt"
	"slices"

	"github.com/harness/gitness/types"
)

func (c *Service) Find(
	ctx context.Context,
	space *types.SpaceCore,
	identifier string,
) (*types.InfraProviderConfig, error) {
	infraProviderConfig, err := c.infraProviderConfigStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider config: %q %w", identifier, err)
	}
	resources, err := c.infraProviderResourceStore.List(ctx, infraProviderConfig.ID, types.ListQueryFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider resources for config: %q %w",
			infraProviderConfig.Identifier, err)
	}
	infraProviderConfig.SpacePath = space.Path
	if len(resources) > 0 {
		providerResources := make([]types.InfraProviderResource, len(resources))
		for i, resource := range resources {
			if resource != nil {
				providerResources[i] = *resource
				providerResources[i].SpacePath = space.Path
			}
		}
		slices.SortFunc(providerResources, types.CompareInfraProviderResource)
		infraProviderConfig.Resources = providerResources
	}

	setupYAML, err := c.getSetupYAML(infraProviderConfig)
	if err != nil {
		return nil, err
	}

	infraProviderConfig.SetupYAML = setupYAML

	return infraProviderConfig, nil
}

func (c *Service) getSetupYAML(infraProviderConfig *types.InfraProviderConfig) (string, error) {
	provider, err := c.infraProviderFactory.GetInfraProvider(infraProviderConfig.Type)
	if err != nil {
		return "", fmt.Errorf("failed to get infra provider of type %s before getting setup yaml for infra "+
			"config %s: %w", infraProviderConfig.Type, infraProviderConfig.Identifier, err)
	}

	setupYAML, err := provider.GenerateSetupYAML(infraProviderConfig)
	if err != nil {
		return "", fmt.Errorf("failed to generate setup yaml for infra provider config %s: %w",
			infraProviderConfig.Identifier, err)
	}

	return setupYAML, nil
}

func (c *Service) FindTemplate(
	ctx context.Context,
	space *types.SpaceCore,
	identifier string,
) (*types.InfraProviderTemplate, error) {
	infraProviderTemplate, err := c.infraProviderTemplateStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider template: %q %w", identifier, err)
	}
	return infraProviderTemplate, nil
}

func (c *Service) FindResourceByConfigAndIdentifier(
	ctx context.Context,
	spaceID int64,
	infraProviderConfigIdentifier string,
	identifier string,
) (*types.InfraProviderResource, error) {
	infraProviderConfig, err := c.infraProviderConfigStore.FindByIdentifier(ctx, spaceID, infraProviderConfigIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider config %s: %w", infraProviderConfigIdentifier, err)
	}
	return c.infraProviderResourceStore.FindByConfigAndIdentifier(ctx, spaceID, infraProviderConfig.ID, identifier)
}

func (c *Service) FindResource(ctx context.Context, id int64) (*types.InfraProviderResource, error) {
	return c.infraProviderResourceStore.Find(ctx, id)
}

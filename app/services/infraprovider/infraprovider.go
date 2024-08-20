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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
)

func NewService(
	tx dbtx.Transactor,
	resourceStore store.InfraProviderResourceStore,
	configStore store.InfraProviderConfigStore,
	templateStore store.InfraProviderTemplateStore,
	factory infraprovider.Factory,
	spaceStore store.SpaceStore,
) *Service {
	return &Service{
		tx:                         tx,
		infraProviderResourceStore: resourceStore,
		infraProviderConfigStore:   configStore,
		infraProviderTemplateStore: templateStore,
		infraProviderFactory:       factory,
		spaceStore:                 spaceStore,
	}
}

type Service struct {
	infraProviderResourceStore store.InfraProviderResourceStore
	infraProviderConfigStore   store.InfraProviderConfigStore
	infraProviderTemplateStore store.InfraProviderTemplateStore
	infraProviderFactory       infraprovider.Factory
	spaceStore                 store.SpaceStore
	tx                         dbtx.Transactor
}

func (c *Service) Find(
	ctx context.Context,
	space *types.Space,
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
		infraProviderConfig.Resources = providerResources
	}
	return infraProviderConfig, nil
}

func (c *Service) FindTemplate(
	ctx context.Context,
	space *types.Space,
	identifier string,
) (*types.InfraProviderTemplate, error) {
	infraProviderTemplate, err := c.infraProviderTemplateStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider template: %q %w", identifier, err)
	}
	return infraProviderTemplate, nil
}

func (c *Service) FindResourceByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string) (*types.InfraProviderResource, error) {
	return c.infraProviderResourceStore.FindByIdentifier(ctx, spaceID, identifier)
}

func (c *Service) FindResource(ctx context.Context, id int64) (*types.InfraProviderResource, error) {
	return c.infraProviderResourceStore.Find(ctx, id)
}

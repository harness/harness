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
	values := make([]types.InfraProviderResource, len(infraProviderConfig.Resources))
	for i, ptr := range resources {
		if ptr != nil {
			values[i] = *ptr
		}
	}
	infraProviderConfig.Resources = values
	return infraProviderConfig, nil
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

func (c *Service) CreateInfraProvider(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		err := c.createConfig(ctx, infraProviderConfig)
		if err != nil {
			return fmt.Errorf("could not autocreate the config: %q %w", infraProviderConfig.Identifier, err)
		}
		err = c.createResources(ctx, infraProviderConfig.Resources, infraProviderConfig.ID)
		if err != nil {
			return fmt.Errorf("could not autocreate the resources: %v %w", infraProviderConfig.Resources, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider %w", err)
	}
	return nil
}

func (c *Service) createConfig(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error {
	err := c.infraProviderConfigStore.Create(ctx, infraProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to create infraprovider config for : %q %w", infraProviderConfig.Identifier, err)
	}
	return nil
}

func (c *Service) CreateResources(ctx context.Context, resources []types.InfraProviderResource, configID int64) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		return c.createResources(ctx, resources, configID)
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider resource %w", err)
	}
	return nil
}

func (c *Service) createResources(ctx context.Context, resources []types.InfraProviderResource, configID int64) error {
	for idx := range resources {
		resource := &resources[idx]
		resource.InfraProviderConfigID = configID
		infraProvider, err := c.infraProviderFactory.GetInfraProvider(resource.InfraProviderType)
		if err != nil {
			return fmt.Errorf("failed to fetch infrastructure impl for type : %q %w", resource.InfraProviderType, err)
		}
		if len(infraProvider.TemplateParams()) > 0 {
			err = c.validateTemplates(ctx, infraProvider, *resource)
			if err != nil {
				return err
			}
		}
		err = c.infraProviderResourceStore.Create(ctx, resource)
		if err != nil {
			return fmt.Errorf("failed to create infraprovider resource for : %q %w", resource.Identifier, err)
		}
	}
	return nil
}

func (c *Service) validateTemplates(
	ctx context.Context,
	infraProvider infraprovider.InfraProvider,
	res types.InfraProviderResource,
) error {
	templateParams := infraProvider.TemplateParams()
	for _, param := range templateParams {
		key := param.Name
		if res.Metadata[key] != "" {
			templateIdentifier := res.Metadata[key]
			_, err := c.infraProviderTemplateStore.FindByIdentifier(
				ctx, res.SpaceID, templateIdentifier)
			if err != nil {
				return fmt.Errorf("unable to get template params for ID : %s %w",
					res.Metadata[key], err)
			}
		}
	}
	return nil
}

func (c *Service) CreateTemplate(
	ctx context.Context,
	template *types.InfraProviderTemplate,
) error {
	return c.infraProviderTemplateStore.Create(ctx, template)
}

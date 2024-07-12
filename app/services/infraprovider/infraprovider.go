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
	infraProviderResourceStore store.InfraProviderResourceStore,
	infraProviderConfigStore store.InfraProviderConfigStore,
	factory infraprovider.Factory,
	spaceStore store.SpaceStore,
) ProviderService {
	return ProviderService{
		tx:                         tx,
		infraProviderResourceStore: infraProviderResourceStore,
		infraProviderConfigStore:   infraProviderConfigStore,
		infraProviderFactory:       factory,
		spaceStore:                 spaceStore,
	}
}

type ProviderService struct {
	infraProviderResourceStore store.InfraProviderResourceStore
	infraProviderConfigStore   store.InfraProviderConfigStore
	infraProviderFactory       infraprovider.Factory
	spaceStore                 store.SpaceStore
	tx                         dbtx.Transactor
}

func (c ProviderService) Find(
	ctx context.Context,
	space *types.Space,
	identifier string,
) (*types.InfraProviderConfig, error) {
	infraProviderConfig, err := c.infraProviderConfigStore.FindByIdentifier(ctx, space.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider config: %w", err)
	}
	resources, err := c.infraProviderResourceStore.List(ctx, infraProviderConfig.ID, types.ListQueryFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to find infraprovider resources: %w", err)
	}
	infraProviderConfig.SpacePath = space.Path
	infraProviderConfig.Resources = resources
	return infraProviderConfig, nil
}

func (c ProviderService) FindResourceByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string) (*types.InfraProviderResource, error) {
	return c.infraProviderResourceStore.FindByIdentifier(ctx, spaceID, identifier)
}

func (c ProviderService) FindResource(ctx context.Context, id int64) (*types.InfraProviderResource, error) {
	return c.infraProviderResourceStore.Find(ctx, id)
}

func (c ProviderService) CreateInfraProvider(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		err := c.infraProviderConfigStore.Create(ctx, infraProviderConfig)
		if err != nil {
			return fmt.Errorf("failed to create infraprovider config for : %q %w", infraProviderConfig.Identifier, err)
		}
		infraProvider, err := c.infraProviderFactory.GetInfraProvider(infraProviderConfig.Type)
		if err != nil {
			return fmt.Errorf("failed to fetch infrastructure impl for type : %q %w", infraProviderConfig.Type, err)
		}
		if len(infraProvider.TemplateParams()) > 0 {
			return fmt.Errorf("failed to fetch templates") // TODO Implement
		}
		parameters := []infraprovider.Parameter{}
		// TODO logic to populate paramteters as per the provider type
		err = infraProvider.ValidateParams(parameters)
		if err != nil {
			return fmt.Errorf("failed to validate infraprovider templates")
		}
		for _, res := range infraProviderConfig.Resources {
			res.InfraProviderConfigID = infraProviderConfig.ID
			err = c.infraProviderResourceStore.Create(ctx, res)
			if err != nil {
				return fmt.Errorf("failed to create infraprovider resource for : %q %w", res.Identifier, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider resource : %w", err)
	}
	return nil
}

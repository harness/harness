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
	"net/http"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types"
)

func (c *Service) CreateConfig(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		err := c.areNewConfigsAllowed(ctx, infraProviderConfig)
		if err != nil {
			return err
		}

		_, err = c.createConfig(ctx, infraProviderConfig)
		if err != nil {
			return fmt.Errorf("could not create the config: %q %w", infraProviderConfig.Identifier, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider %w", err)
	}
	return nil
}

func (c *Service) areNewConfigsAllowed(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error {
	existingConfigs, err := c.fetchExistingConfigs(ctx, infraProviderConfig)
	if err != nil {
		return err
	}
	if len(existingConfigs) > 0 {
		return usererror.NewWithPayload(http.StatusForbidden, fmt.Sprintf(
			"%d infra configs for provider %s exist for this account. Only 1 is allowed",
			len(existingConfigs), infraProviderConfig.Type))
	}
	return nil
}

func (c *Service) fetchExistingConfigs(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
) ([]*types.InfraProviderConfig, error) {
	existingConfigs, err := c.infraProviderConfigStore.List(ctx, &types.InfraProviderConfigFilter{
		SpaceIDs: []int64{infraProviderConfig.SpaceID},
		Type:     infraProviderConfig.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find existing infraprovider config for type %s & space %d: %w",
			infraProviderConfig.Type, infraProviderConfig.SpaceID, err)
	}
	return existingConfigs, nil
}

func (c *Service) createConfig(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) (int64, error) {
	err := c.validateConfig(infraProviderConfig)
	if err != nil {
		return 0, err
	}

	infraProviderConfig, err = c.updateConfig(infraProviderConfig)
	if err != nil {
		return 0, err
	}

	err = c.infraProviderConfigStore.Create(ctx, infraProviderConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to create infraprovider config for %s: %w", infraProviderConfig.Identifier, err)
	}

	newInfraProviderConfig, err := c.infraProviderConfigStore.FindByIdentifier(ctx, infraProviderConfig.SpaceID,
		infraProviderConfig.Identifier)
	if err != nil {
		return 0, fmt.Errorf("failed to find newly created infraprovider config %s in space %d: %w",
			infraProviderConfig.Identifier, infraProviderConfig.SpaceID, err)
	}
	return newInfraProviderConfig.ID, nil
}

func (c *Service) validateConfig(infraProviderConfig *types.InfraProviderConfig) error {
	infraProvider, err := c.infraProviderFactory.GetInfraProvider(infraProviderConfig.Type)
	if err != nil {
		return fmt.Errorf("failed to fetch infra provider for type %s: %w", infraProviderConfig.Type, err)
	}

	err = infraProvider.ValidateConfig(infraProviderConfig)
	if err != nil {
		return err
	}

	return nil
}

func (c *Service) updateConfig(infraProviderConfig *types.InfraProviderConfig) (*types.InfraProviderConfig, error) {
	infraProvider, err := c.infraProviderFactory.GetInfraProvider(infraProviderConfig.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch infra provider for type %s: %w", infraProviderConfig.Type, err)
	}

	updatedConfig, err := infraProvider.UpdateConfig(infraProviderConfig)
	if err != nil {
		return nil, err
	}

	return updatedConfig, nil
}

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
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Service) CreateTemplate(
	ctx context.Context,
	template *types.InfraProviderTemplate,
) error {
	return c.infraProviderTemplateStore.Create(ctx, template)
}

func (c *Service) CreateInfraProvider(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		err := c.areNewConfigsAllowed(ctx, infraProviderConfig)
		if err != nil {
			return err
		}

		configID, err := c.createConfig(ctx, infraProviderConfig)
		if err != nil {
			return fmt.Errorf("could not create the config: %q %w", infraProviderConfig.Identifier, err)
		}
		err = c.createResources(ctx, infraProviderConfig.Resources, configID, infraProviderConfig.Identifier)
		if err != nil {
			return fmt.Errorf("could not create the resources: %v %w", infraProviderConfig.Resources, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider %w", err)
	}
	return nil
}

func (c *Service) validateConfigAndResources(infraProviderConfig *types.InfraProviderConfig) error {
	infraProvider, err := c.infraProviderFactory.GetInfraProvider(infraProviderConfig.Type)
	if err != nil {
		return fmt.Errorf("failed to fetch infra provider for type %s: %w", infraProviderConfig.Type, err)
	}

	err = infraProvider.ValidateConfigAndResources(infraProviderConfig)
	if err != nil {
		return err
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
	existingConfigs, err := c.infraProviderConfigStore.FindByType(ctx, infraProviderConfig.SpaceID,
		infraProviderConfig.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing infraprovider config for type %s & space %d: %w",
			infraProviderConfig.Type, infraProviderConfig.SpaceID, err)
	}
	return existingConfigs, nil
}

func (c *Service) createConfig(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) (int64, error) {
	err := c.validateConfigAndResources(infraProviderConfig)
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

func (c *Service) CreateResources(
	ctx context.Context,
	resources []types.InfraProviderResource,
	configID int64,
	infraProviderConfigIdentifier string,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		return c.createResources(ctx, resources, configID, infraProviderConfigIdentifier)
	})
	if err != nil {
		return fmt.Errorf("failed to complete create txn for the infraprovider resource %w", err)
	}
	return nil
}

func (c *Service) createResources(
	ctx context.Context,
	resources []types.InfraProviderResource,
	configID int64,
	infraProviderConfigIdentifier string,
) error {
	for idx := range resources {
		resource := &resources[idx]
		resource.InfraProviderConfigID = configID
		resource.InfraProviderConfigIdentifier = infraProviderConfigIdentifier

		err := c.validate(ctx, resource)
		if err != nil {
			return err
		}

		err = c.infraProviderResourceStore.Create(ctx, resource)
		if err != nil {
			return fmt.Errorf("failed to create infraprovider resource for : %q %w", resource.UID, err)
		}
	}
	return nil
}

func (c *Service) validate(ctx context.Context, resource *types.InfraProviderResource) error {
	infraProvider, err := c.infraProviderFactory.GetInfraProvider(resource.InfraProviderType)
	if err != nil {
		return fmt.Errorf("failed to fetch infra impl for type : %q %w", resource.InfraProviderType, err)
	}

	if len(infraProvider.TemplateParams()) > 0 {
		err = c.validateTemplates(ctx, infraProvider, *resource)
		if err != nil {
			return err
		}
	}

	err = c.validateResourceParams(infraProvider, *resource)
	if err != nil {
		return err
	}

	return err
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
				log.Warn().Msgf("unable to get template params for ID : %s",
					res.Metadata[key])
			}
		}
	}
	return nil
}

func (c *Service) validateResourceParams(
	infraProvider infraprovider.InfraProvider,
	res types.InfraProviderResource,
) error {
	infraResourceParams := make([]types.InfraProviderParameter, 0)
	for key, value := range res.Metadata {
		infraResourceParams = append(infraResourceParams, types.InfraProviderParameter{
			Name:  key,
			Value: value,
		})
	}
	return infraProvider.ValidateParams(infraResourceParams)
}

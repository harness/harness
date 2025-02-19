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
	"errors"
	"fmt"

	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Service) CreateResources(
	ctx context.Context,
	spaceID int64,
	resources []types.InfraProviderResource,
	configID int64,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		return c.createMissingResources(ctx, resources, configID, spaceID)
	})
	if err != nil {
		return fmt.Errorf("failed to complete create txn for the infraprovider resource %w", err)
	}
	return nil
}

func (c *Service) createMissingResources(
	ctx context.Context,
	resources []types.InfraProviderResource,
	configID int64,
	spaceID int64,
) error {
	for idx := range resources {
		resource := &resources[idx]
		resource.InfraProviderConfigID = configID
		resource.SpaceID = spaceID
		if err := c.validateResource(ctx, resource); err != nil {
			return err
		}
		existingResource, err := c.infraProviderResourceStore.FindByConfigAndIdentifier(ctx, resource.SpaceID,
			configID, resource.UID)
		if (err != nil && errors.Is(err, store.ErrResourceNotFound)) || existingResource == nil {
			if err = c.infraProviderResourceStore.Create(ctx, resource); err != nil {
				return fmt.Errorf("failed to create infraprovider resource for %s: %w", resource.UID, err)
			}
			log.Info().Msgf("created new resource %s/%s", resource.InfraProviderConfigIdentifier, resource.UID)
		}
	}
	return nil
}

func (c *Service) validateResource(ctx context.Context, resource *types.InfraProviderResource) error {
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

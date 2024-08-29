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
	"time"

	"github.com/harness/gitness/types"
)

func (c *Service) UpdateInfraProvider(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		err := c.updateConfig(ctx, infraProviderConfig)
		if err != nil {
			return fmt.Errorf("could not update the config: %q %w", infraProviderConfig.Identifier, err)
		}
		for _, resource := range infraProviderConfig.Resources {
			if err = c.UpdateResource(ctx, resource); err != nil {
				return fmt.Errorf("could not update the resources: %v %w", infraProviderConfig.Resources, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider %w", err)
	}
	return nil
}

func (c *Service) updateConfig(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error {
	infraProviderConfig.Updated = time.Now().UnixMilli()
	err := c.infraProviderConfigStore.Update(ctx, infraProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to update infraprovider config for : %q %w", infraProviderConfig.Identifier, err)
	}
	return nil
}

func (c *Service) UpdateResource(ctx context.Context, resource types.InfraProviderResource) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		space, err := c.spaceStore.FindByRef(ctx, resource.SpacePath)
		if err != nil {
			return err
		}
		infraProviderResource, err := c.FindResourceByIdentifier(ctx, space.ID, resource.Identifier)
		if err != nil {
			return err
		}
		resource.ID = infraProviderResource.ID
		resource.Updated = time.Now().UnixMilli()
		if err = c.infraProviderResourceStore.Update(ctx, &resource); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete update txn for the infraprovider resource %w", err)
	}
	return nil
}

func (c *Service) UpdateTemplate(ctx context.Context, template types.InfraProviderTemplate) error {
	err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		space, err := c.spaceStore.FindByRef(ctx, template.SpacePath)
		if err != nil {
			return err
		}
		templateInDB, err := c.infraProviderTemplateStore.FindByIdentifier(ctx, space.ID, template.Identifier)
		if err != nil {
			return err
		}
		template.ID = templateInDB.ID
		template.SpaceID = space.ID
		if err = c.infraProviderTemplateStore.Update(ctx, &template); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to complete update txn for the infraprovider template %w", err)
	}
	return nil
}

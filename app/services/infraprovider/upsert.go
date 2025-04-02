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

	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (c *Service) UpsertConfigAndResources(
	ctx context.Context,
	infraProviderConfig *types.InfraProviderConfig,
	infraProviderResources []types.InfraProviderResource,
) error {
	space, err := c.spaceFinder.FindByRef(ctx, infraProviderConfig.SpacePath)
	if err != nil {
		return fmt.Errorf("failed to find space by ref: %w", err)
	}

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		return c.upsertConfigAndResources(ctx, space, infraProviderConfig, infraProviderResources)
	})
	if err != nil {
		return fmt.Errorf("failed to complete txn for the infraprovider: %w", err)
	}
	return nil
}

func (c *Service) upsertConfigAndResources(
	ctx context.Context,
	space *types.SpaceCore,
	infraProviderConfig *types.InfraProviderConfig,
	infraProviderResources []types.InfraProviderResource,
) error {
	providerConfigInDB, err := c.Find(ctx, space, infraProviderConfig.Identifier)
	var infraProviderConfigID int64
	if errors.Is(err, store.ErrResourceNotFound) { // nolint:gocritic
		configID, createErr := c.createConfig(ctx, infraProviderConfig)
		if createErr != nil {
			return fmt.Errorf("could not create the config: %q %w", infraProviderConfig.Identifier, err)
		}
		infraProviderConfigID = configID
		log.Info().Msgf("created new infraconfig %s", infraProviderConfig.Identifier)
	} else if err != nil {
		return err
	} else {
		infraProviderConfigID = providerConfigInDB.ID
	}

	infraProviderConfig.ID = infraProviderConfigID
	if err = c.UpdateConfig(ctx, infraProviderConfig); err != nil {
		return fmt.Errorf("could not update the config %s: %w", infraProviderConfig.Identifier, err)
	}

	log.Info().Msgf("updated infraconfig %s", infraProviderConfig.Identifier)
	if err = c.createMissingResources(ctx, infraProviderResources, infraProviderConfigID, space.ID,
		infraProviderConfig.Identifier); err != nil {
		return err
	}
	return nil
}

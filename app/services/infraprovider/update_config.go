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

func (c *Service) UpdateConfig(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error {
	err := c.validateConfig(infraProviderConfig)
	if err != nil {
		return err
	}

	infraProviderConfig, err = c.updateConfig(infraProviderConfig)
	if err != nil {
		return err
	}

	existingConfig, err := c.infraProviderConfigStore.FindByIdentifier(ctx, infraProviderConfig.SpaceID,
		infraProviderConfig.Identifier)
	if err != nil {
		return fmt.Errorf("could not find infraprovider config %s before updating: %w",
			infraProviderConfig.Identifier, err)
	}

	infraProviderConfig.ID = existingConfig.ID
	infraProviderConfig.Updated = time.Now().UnixMilli()
	err = c.infraProviderConfigStore.Update(ctx, infraProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to update infraprovider config for %s: %w", infraProviderConfig.Identifier, err)
	}

	return nil
}

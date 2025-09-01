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

	"github.com/harness/gitness/types"
)

func (c *Service) List(
	ctx context.Context,
	filter *types.InfraProviderConfigFilter,
) ([]*types.InfraProviderConfig, error) {
	infraProviderConfigs, err := c.infraProviderConfigStore.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list infraprovider configs: %w", err)
	}
	for _, infraProviderConfig := range infraProviderConfigs {
		err = c.populateDetails(ctx, infraProviderConfig)
		if err != nil {
			return nil, err
		}
	}
	return infraProviderConfigs, nil
}

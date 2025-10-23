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

	"github.com/rs/zerolog/log"
)

func (c *Service) DeleteAllForSpaces(ctx context.Context, spaces []*types.Space) error {
	spaceIDsMap := make(map[int64]*types.SpaceCore)
	spaceIDs := make([]int64, 0, len(spaces))
	for _, space := range spaces {
		spaceIDs = append(spaceIDs, space.ID)
		spaceIDsMap[space.ID] = space.Core()
	}

	log.Debug().Msgf("Deleting all infra providers for spaces %+v", spaceIDs)

	infraProviderConfigFilter := types.InfraProviderConfigFilter{SpaceIDs: spaceIDs}
	infraProviderConfigs, err := c.List(ctx, &infraProviderConfigFilter)
	if err != nil {
		return fmt.Errorf("error while listing infra provider entities before deleting all for spaces: %w", err)
	}

	for _, infraProviderConfig := range infraProviderConfigs {
		for _, infraProviderResource := range infraProviderConfig.Resources {
			log.Debug().Msgf("Deleting infra resource %s for space %d", infraProviderResource.UID,
				infraProviderResource.SpaceID)

			err = c.DeleteResource(ctx, infraProviderConfig.SpaceID, infraProviderConfig.Identifier,
				infraProviderResource.UID, false)
			if err != nil {
				return fmt.Errorf("error while deleting infra resource %s while deleting all for spaces: %w",
					infraProviderResource.UID, err)
			}

			log.Debug().Msgf("Deleted infra resource %s for space %d", infraProviderResource.UID,
				infraProviderResource.SpaceID)
		}

		log.Debug().Msgf("Deleting infra config %s for space %d", infraProviderConfig.Identifier,
			infraProviderConfig.SpaceID)

		err = c.DeleteConfig(ctx, spaceIDsMap[infraProviderConfig.SpaceID], infraProviderConfig.Identifier, false)
		if err != nil {
			return fmt.Errorf("error while deleting infra config %s while deleting all for spaces: %w",
				infraProviderConfig.Identifier, err)
		}

		log.Debug().Msgf("Deleted infra config %s for space %d", infraProviderConfig.Identifier,
			infraProviderConfig.SpaceID)
	}

	log.Debug().Msgf("Deleted all infra providers for spaces %+v", spaceIDs)

	return nil
}

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

package gitspace

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

func (c *Service) DeleteAllForSpaces(ctx context.Context, spaces []*types.Space) error {
	spaceIDs := make([]int64, 0, len(spaces))
	for _, space := range spaces {
		spaceIDs = append(spaceIDs, space.ID)
	}

	log.Debug().Msgf("Deleting all gitspaces for spaces %+v", spaceIDs)

	var gitspaceFilter = types.GitspaceFilter{}
	gitspaceFilter.SpaceIDs = spaceIDs
	gitspaceFilter.Deleted = ptr.Bool(false)
	gitspaceFilter.MarkedForDeletion = ptr.Bool(false)

	gitspaces, _, _, err := c.ListGitspacesWithInstance(ctx, gitspaceFilter, false)
	if err != nil {
		return fmt.Errorf("error while listing gitspaces with instance before deleting all for spaces: %w", err)
	}

	for _, gitspace := range gitspaces {
		log.Debug().Msgf("Deleting gitspace %s for space %d", gitspace.Identifier, gitspace.SpaceID)
		err = c.deleteGitspace(ctx, gitspace)
		if err != nil {
			return fmt.Errorf("error while deleting gitspace %s while deleting all for spaces: %w",
				gitspace.Identifier, err)
		}
		log.Debug().Msgf("Deleted gitspace %s for space %d", gitspace.Identifier, gitspace.SpaceID)
	}

	log.Debug().Msgf("Deleted all gitspaces for spaces %+v", spaceIDs)

	return nil
}

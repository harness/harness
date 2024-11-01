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
	"errors"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const spaceIsDeleted = "Failed to find space: resource not found"

func (c *Controller) ListAllGitspaces( // nolint:gocognit
	ctx context.Context,
	session *auth.Session,
) ([]*types.GitspaceConfig, error) {
	filter := &types.GitspaceFilter{
		GitspaceInstanceFilter: types.GitspaceInstanceFilter{UserIdentifier: session.Principal.UID},
	}

	var result []*types.GitspaceConfig
	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		allGitspaceConfigs, err := c.gitspaceConfigStore.ListWithLatestInstance(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}

		var spacesMap = make(map[int64]string)

		for idx := 0; idx < len(allGitspaceConfigs); idx++ {
			if spacesMap[allGitspaceConfigs[idx].SpaceID] == "" {
				space, findSpaceErr := c.spaceStore.Find(ctx, allGitspaceConfigs[idx].SpaceID)
				if findSpaceErr != nil {
					if findSpaceErr.Error() != spaceIsDeleted {
						return fmt.Errorf(
							"error fetching space %d: %w", allGitspaceConfigs[idx].SpaceID, findSpaceErr)
					}
					continue
				}
				spacesMap[allGitspaceConfigs[idx].SpaceID] = space.Path
			}
			allGitspaceConfigs[idx].SpacePath = spacesMap[allGitspaceConfigs[idx].SpaceID]
		}

		authorizedSpaceIDs, err := c.getAuthorizedSpaces(ctx, session, spacesMap)
		if err != nil {
			return err
		}

		finalGitspaceConfigs := c.filter(allGitspaceConfigs, authorizedSpaceIDs)

		result = finalGitspaceConfigs

		return nil
	}, dbtx.TxDefaultReadOnly)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Controller) filter(
	allGitspaceConfigs []*types.GitspaceConfig,
	authorizedSpaceIDs map[int64]bool,
) []*types.GitspaceConfig {
	return c.getAuthorizedGitspaceConfigs(allGitspaceConfigs, authorizedSpaceIDs)
}

func (c *Controller) getAuthorizedGitspaceConfigs(
	allGitspaceConfigs []*types.GitspaceConfig,
	authorizedSpaceIDs map[int64]bool,
) []*types.GitspaceConfig {
	var authorizedGitspaceConfigs = make([]*types.GitspaceConfig, 0)
	for idx := 0; idx < len(allGitspaceConfigs); idx++ {
		if authorizedSpaceIDs[allGitspaceConfigs[idx].SpaceID] {
			authorizedGitspaceConfigs = append(authorizedGitspaceConfigs, allGitspaceConfigs[idx])
		}
	}
	return authorizedGitspaceConfigs
}

func (c *Controller) getAuthorizedSpaces(
	ctx context.Context,
	session *auth.Session,
	spacesMap map[int64]string,
) (map[int64]bool, error) {
	var authorizedSpaceIDs = make(map[int64]bool, 0)

	for spaceID, spacePath := range spacesMap {
		authErr := apiauth.CheckGitspace(ctx, c.authorizer, session, spacePath, "", enum.PermissionGitspaceView)
		if authErr != nil && !errors.Is(authErr, apiauth.ErrNotAuthorized) {
			return nil, fmt.Errorf("failed to check gitspace auth for space ID %d: %w", spaceID, authErr)
		}

		authorizedSpaceIDs[spaceID] = true
	}

	return authorizedSpaceIDs, nil
}

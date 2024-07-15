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

func (c *Controller) ListAllGitspaces( // nolint:gocognit
	ctx context.Context,
	session *auth.Session,
) ([]*types.GitspaceConfig, error) {
	var result []*types.GitspaceConfig

	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		allGitspaceConfigs, err := c.gitspaceConfigStore.ListAll(ctx, session.Principal.UID)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}

		var spacesMap = make(map[int64]string)

		for idx := 0; idx < len(allGitspaceConfigs); idx++ {
			if spacesMap[allGitspaceConfigs[idx].SpaceID] == "" {
				space, findSpaceErr := c.spaceStore.Find(ctx, allGitspaceConfigs[idx].SpaceID)
				if findSpaceErr != nil {
					return fmt.Errorf(
						"failed to find space for ID %d: %w", allGitspaceConfigs[idx].SpaceID, findSpaceErr)
				}
				spacesMap[allGitspaceConfigs[idx].SpaceID] = space.Path
			}

			allGitspaceConfigs[idx].SpacePath = spacesMap[allGitspaceConfigs[idx].SpaceID]
		}

		authorizedSpaceIDs, err := c.getAuthorizedSpaces(ctx, session, spacesMap)
		if err != nil {
			return err
		}

		finalGitspaceConfigs, err := c.filterAndPopulateInstanceDetails(ctx, allGitspaceConfigs, authorizedSpaceIDs)
		if err != nil {
			return err
		}

		result = finalGitspaceConfigs

		return nil
	}, dbtx.TxDefaultReadOnly)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Controller) filterAndPopulateInstanceDetails(
	ctx context.Context,
	allGitspaceConfigs []*types.GitspaceConfig,
	authorizedSpaceIDs map[int64]bool,
) ([]*types.GitspaceConfig, error) {
	authorizedGitspaceConfigs := c.getAuthorizedGitspaceConfigs(allGitspaceConfigs, authorizedSpaceIDs)

	gitspaceInstancesMap, err := c.getLatestInstanceMap(ctx, authorizedGitspaceConfigs)
	if err != nil {
		return nil, err
	}

	var result []*types.GitspaceConfig

	for _, gitspaceConfig := range authorizedGitspaceConfigs {
		instance := gitspaceInstancesMap[gitspaceConfig.ID]

		gitspaceConfig.GitspaceInstance = instance

		if instance != nil {
			gitspaceStateType, stateErr := enum.GetGitspaceStateFromInstance(instance.State)
			if stateErr != nil {
				return nil, stateErr
			}

			gitspaceConfig.State = gitspaceStateType

			instance.SpacePath = gitspaceConfig.SpacePath
		} else {
			gitspaceConfig.State = enum.GitspaceStateUninitialized
		}

		result = append(result, gitspaceConfig)
	}
	return result, nil
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

func (c *Controller) getLatestInstanceMap(
	ctx context.Context,
	authorizedGitspaceConfigs []*types.GitspaceConfig,
) (map[int64]*types.GitspaceInstance, error) {
	var authorizedConfigIDs = make([]int64, 0)
	for _, config := range authorizedGitspaceConfigs {
		authorizedConfigIDs = append(authorizedConfigIDs, config.ID)
	}

	var gitspaceInstances, err = c.gitspaceInstanceStore.FindAllLatestByGitspaceConfigID(ctx, authorizedConfigIDs)
	if err != nil {
		return nil, err
	}

	var gitspaceInstancesMap = make(map[int64]*types.GitspaceInstance)

	for _, gitspaceEntry := range gitspaceInstances {
		gitspaceInstancesMap[gitspaceEntry.GitSpaceConfigID] = gitspaceEntry
	}

	return gitspaceInstancesMap, nil
}

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
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListAllGitspaces all the gitspace with given filter.
// DO NOT USE allSpaceIDs = true for cde-manager. This arg is used only in gitness to list all the gitspaces in gitness
// for all. This is useful to list all the gitspaces in OSS for IDE plugins.
func (c *Controller) ListAllGitspaces( // nolint:gocognit
	ctx context.Context,
	session *auth.Session,
	filter types.GitspaceFilter,
	allSpaceIDs bool,
) ([]*types.GitspaceConfig, error) {
	if allSpaceIDs {
		leafSpaceIDs, err := c.fetchAllLeafSpaceIDs(ctx)
		if err != nil {
			return nil, err
		}
		filter.SpaceIDs = leafSpaceIDs
	}
	var result []*types.GitspaceConfig
	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		allGitspaceConfigs, _, _, err := c.gitspaceSvc.ListGitspacesWithInstance(ctx, filter, false)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}

		var spacesMap = make(map[int64]string)
		for idx := range allGitspaceConfigs {
			if spacesMap[allGitspaceConfigs[idx].SpaceID] == "" {
				space, findSpaceErr := c.spaceFinder.FindByRef(ctx, allGitspaceConfigs[idx].SpacePath)
				if findSpaceErr != nil {
					if !errors.Is(findSpaceErr, store.ErrResourceNotFound) {
						return fmt.Errorf(
							"error fetching space %d: %w", allGitspaceConfigs[idx].SpaceID, findSpaceErr)
					}
					continue
				}
				spacesMap[allGitspaceConfigs[idx].SpaceID] = space.Path
			}
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

func (c *Controller) fetchAllLeafSpaceIDs(ctx context.Context) ([]int64, error) {
	opts := &types.SpaceFilter{}
	rootSpaces, err := c.spaceStore.GetAllRootSpaces(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get root spaces: %w", err)
	}
	var leafSpaceIDs []int64
	for _, rootSpace := range rootSpaces {
		spaceIDs, err := c.spaceStore.GetDescendantsIDs(ctx, rootSpace.ID)
		if err != nil {
			if !errors.Is(err, store.ErrResourceNotFound) {
				return nil, fmt.Errorf("failed to get descendants ids: %w", err)
			}
		}
		leafSpaceIDs = append(leafSpaceIDs, spaceIDs...)
	}

	return leafSpaceIDs, nil
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
	for idx := range allGitspaceConfigs {
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
		err := apiauth.CheckGitspace(
			ctx, c.authorizer, session, spacePath, "", enum.PermissionGitspaceView,
		)
		if err != nil && !apiauth.IsNoAccess(err) {
			return nil, fmt.Errorf("failed to check gitspace auth for space ID %d: %w", spaceID, err)
		}

		authorizedSpaceIDs[spaceID] = true
	}

	return authorizedSpaceIDs, nil
}

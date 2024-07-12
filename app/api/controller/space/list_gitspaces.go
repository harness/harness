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

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) ListGitspaces(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.ListQueryFilter,
) ([]*types.GitspaceConfig, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, "", enum.PermissionGitspaceView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize gitspace: %w", err)
	}
	gitspaceFilter := &types.GitspaceFilter{
		QueryFilter: filter,
		UserID:      session.Principal.UID,
		SpaceIDs:    []int64{space.ID},
	}
	var gitspaceConfigs []*types.GitspaceConfig
	var count int64
	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		gitspaceConfigs, err = c.gitspaceConfigStore.List(ctx, gitspaceFilter)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}
		count, err = c.gitspaceConfigStore.Count(ctx, gitspaceFilter)
		if err != nil {
			return fmt.Errorf("failed to count gitspaces in space: %w", err)
		}
		var gitspaceConfigIDs = make([]int64, 0)
		for idx := 0; idx < len(gitspaceConfigs); idx++ {
			if gitspaceConfigs[idx].IsDeleted {
				continue
			}
			gitspaceConfigs[idx].SpacePath = space.Path // As the API is for a space, this will remain same
			gitspaceConfigIDs = append(gitspaceConfigIDs, gitspaceConfigs[idx].ID)
		}
		gitspaceInstancesMap, err := c.getLatestInstanceMap(ctx, gitspaceConfigIDs)
		if err != nil {
			return err
		}
		for _, gitspaceConfig := range gitspaceConfigs {
			instance := gitspaceInstancesMap[gitspaceConfig.ID]
			gitspaceConfig.GitspaceInstance = instance
			if instance != nil {
				gitspaceStateType, err := enum.GetGitspaceStateFromInstance(instance.State)
				if err != nil {
					return err
				}
				gitspaceConfig.State = gitspaceStateType
				instance.SpacePath = gitspaceConfig.SpacePath
			} else {
				gitspaceConfig.State = enum.GitspaceStateUninitialized
			}
		}
		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}
	return gitspaceConfigs, count, nil
}

func (c *Controller) getLatestInstanceMap(
	ctx context.Context,
	gitspaceConfigIDs []int64,
) (map[int64]*types.GitspaceInstance, error) {
	var gitspaceInstances, err = c.gitspaceInstanceStore.FindAllLatestByGitspaceConfigID(ctx, gitspaceConfigIDs)
	if err != nil {
		return nil, err
	}
	var gitspaceInstancesMap = make(map[int64]*types.GitspaceInstance)
	for _, gitspaceEntry := range gitspaceInstances {
		gitspaceInstancesMap[gitspaceEntry.GitSpaceConfigID] = gitspaceEntry
	}
	return gitspaceInstancesMap, nil
}

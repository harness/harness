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

	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func NewService(
	tx dbtx.Transactor,
	gitspaceStore store.GitspaceConfigStore,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	spaceStore store.SpaceStore,
	infraProviderSvc *infraprovider.Service,
) *Service {
	return &Service{
		tx:                    tx,
		gitspaceConfigStore:   gitspaceStore,
		gitspaceInstanceStore: gitspaceInstanceStore,
		spaceStore:            spaceStore,
		infraProviderSvc:      infraProviderSvc,
	}
}

type Service struct {
	gitspaceConfigStore   store.GitspaceConfigStore
	gitspaceInstanceStore store.GitspaceInstanceStore
	spaceStore            store.SpaceStore
	tx                    dbtx.Transactor
	infraProviderSvc      *infraprovider.Service
}

func (c *Service) ListGitspacesForSpace(
	ctx context.Context,
	space *types.Space,
	userIdentifier string,
	filter types.ListQueryFilter,
) ([]*types.GitspaceConfig, int64, error) {
	gitspaceFilter := &types.GitspaceFilter{
		QueryFilter: filter,
		UserID:      userIdentifier,
		SpaceIDs:    []int64{space.ID},
	}
	var gitspaceConfigs []*types.GitspaceConfig
	var count int64
	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		gitspaceConfigs, err = c.gitspaceConfigStore.List(ctx, gitspaceFilter)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}
		if len(gitspaceConfigs) >= gitspaceFilter.QueryFilter.Size {
			count, err = c.gitspaceConfigStore.Count(ctx, gitspaceFilter)
			if err != nil {
				return fmt.Errorf("failed to count gitspaces in space: %w", err)
			}
		} else {
			count = int64(len(gitspaceConfigs))
		}
		gitspaceInstancesMap, err := c.getLatestInstanceMap(ctx, gitspaceConfigs)
		if err != nil {
			return err
		}
		for _, gitspaceConfig := range gitspaceConfigs {
			instance := gitspaceInstancesMap[gitspaceConfig.ID]
			gitspaceConfig.GitspaceInstance = instance
			gitspaceConfig.SpacePath = space.Path
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

func (c *Service) getLatestInstanceMap(
	ctx context.Context,
	gitspaceConfigs []*types.GitspaceConfig,
) (map[int64]*types.GitspaceInstance, error) {
	var gitspaceConfigIDs = make([]int64, 0)
	for idx := 0; idx < len(gitspaceConfigs); idx++ {
		gitspaceConfigIDs = append(gitspaceConfigIDs, gitspaceConfigs[idx].ID)
	}
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

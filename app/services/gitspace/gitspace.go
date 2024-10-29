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

	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func NewService(
	tx dbtx.Transactor,
	gitspaceStore store.GitspaceConfigStore,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	eventReporter *gitspaceevents.Reporter,
	gitspaceEventStore store.GitspaceEventStore,
	spaceStore store.SpaceStore,
	infraProviderSvc *infraprovider.Service,
	orchestrator orchestrator.Orchestrator,
	scm *scm.SCM,
) *Service {
	return &Service{
		tx:                    tx,
		gitspaceConfigStore:   gitspaceStore,
		gitspaceInstanceStore: gitspaceInstanceStore,
		eventReporter:         eventReporter,
		gitspaceEventStore:    gitspaceEventStore,
		spaceStore:            spaceStore,
		infraProviderSvc:      infraProviderSvc,
		orchestrator:          orchestrator,
		scm:                   scm,
	}
}

type Service struct {
	gitspaceConfigStore   store.GitspaceConfigStore
	gitspaceInstanceStore store.GitspaceInstanceStore
	eventReporter         *gitspaceevents.Reporter
	gitspaceEventStore    store.GitspaceEventStore
	spaceStore            store.SpaceStore
	tx                    dbtx.Transactor
	infraProviderSvc      *infraprovider.Service
	orchestrator          orchestrator.Orchestrator
	scm                   *scm.SCM
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
		count, err = c.gitspaceConfigStore.Count(ctx, gitspaceFilter)
		if err != nil {
			return fmt.Errorf("failed to count gitspaces in space: %w", err)
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
				gitspaceStateType, err := enum.GetGitspaceStateFromInstance(instance.State, instance.Updated)
				if err != nil {
					return err
				}
				gitspaceConfig.State = gitspaceStateType
				instance.SpacePath = gitspaceConfig.SpacePath
			} else {
				gitspaceConfig.State = enum.GitspaceStateUninitialized
			}
			gitspaceConfig.BranchURL = c.GetBranchURL(ctx, gitspaceConfig)
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

func (c *Service) GetBranchURL(ctx context.Context, config *types.GitspaceConfig) string {
	branchURL, err := c.scm.GetBranchURL(config.SpacePath, config.CodeRepo.Type, config.CodeRepo.URL,
		config.CodeRepo.Branch)
	if err != nil {
		log.Warn().Ctx(ctx).Err(err).Msgf("failed to get branch URL for gitspace config %s, returning repo url",
			config.Identifier)
		branchURL = config.CodeRepo.URL
	}
	return branchURL
}

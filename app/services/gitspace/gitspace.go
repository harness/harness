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
	"strconv"

	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	gitspacedeleteevents "github.com/harness/gitness/app/events/gitspacedelete"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func NewService(
	tx dbtx.Transactor,
	gitspaceStore store.GitspaceConfigStore,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	eventReporter *gitspaceevents.Reporter,
	gitspaceEventStore store.GitspaceEventStore,
	spaceFinder refcache.SpaceFinder,
	infraProviderSvc *infraprovider.Service,
	orchestrator orchestrator.Orchestrator,
	scm *scm.SCM,
	config *types.Config,
	gitspaceDeleteEventReporter *gitspacedeleteevents.Reporter,
) *Service {
	return &Service{
		tx:                          tx,
		gitspaceConfigStore:         gitspaceStore,
		gitspaceInstanceStore:       gitspaceInstanceStore,
		gitspaceEventReporter:       eventReporter,
		gitspaceEventStore:          gitspaceEventStore,
		spaceFinder:                 spaceFinder,
		infraProviderSvc:            infraProviderSvc,
		orchestrator:                orchestrator,
		scm:                         scm,
		config:                      config,
		gitspaceDeleteEventReporter: gitspaceDeleteEventReporter,
	}
}

type Service struct {
	gitspaceConfigStore         store.GitspaceConfigStore
	gitspaceInstanceStore       store.GitspaceInstanceStore
	gitspaceEventReporter       *gitspaceevents.Reporter
	gitspaceDeleteEventReporter *gitspacedeleteevents.Reporter
	gitspaceEventStore          store.GitspaceEventStore
	spaceFinder                 refcache.SpaceFinder
	tx                          dbtx.Transactor
	infraProviderSvc            *infraprovider.Service
	orchestrator                orchestrator.Orchestrator
	scm                         *scm.SCM
	config                      *types.Config
}

func (c *Service) ListGitspacesWithInstance(
	ctx context.Context,
	filter types.GitspaceFilter,
	useTransaction bool,
) ([]*types.GitspaceConfig, int64, int64, error) {
	var gitspaceConfigs []*types.GitspaceConfig
	var filterCount, allGitspacesCount int64
	var err error

	findFunc := func(ctx context.Context) (err error) {
		gitspaceConfigs, err = c.gitspaceConfigStore.ListWithLatestInstance(ctx, &filter)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}

		filterCount, err = c.gitspaceConfigStore.Count(ctx, &filter)
		if err != nil {
			return fmt.Errorf("failed to filterCount gitspaces in space: %w", err)
		}
		// Only filter from RBAC and Space is applied for this count, the user filter will be empty for admin users.
		instanceFilter := types.GitspaceInstanceFilter{
			UserIdentifier: filter.UserIdentifier,
			SpaceIDs:       filter.SpaceIDs,
		}
		allGitspacesCount, err = c.gitspaceConfigStore.Count(ctx, &types.GitspaceFilter{
			Deleted:                filter.Deleted,
			MarkedForDeletion:      filter.MarkedForDeletion,
			GitspaceInstanceFilter: instanceFilter,
		})
		if err != nil {
			return fmt.Errorf("failed to count all gitspace configs in space: %w", err)
		}

		return nil
	}

	if useTransaction {
		err = c.tx.WithTx(ctx, findFunc, dbtx.TxDefaultReadOnly)
	} else {
		err = findFunc(ctx)
	}

	if err != nil {
		return nil, 0, 0, err
	}

	for _, gitspaceConfig := range gitspaceConfigs {
		space, err := c.spaceFinder.FindByRef(ctx, strconv.FormatInt(gitspaceConfig.SpaceID, 10))
		if err != nil {
			return nil, 0, 0, err
		}
		gitspaceConfig.SpacePath = space.Path
		if gitspaceConfig.GitspaceInstance != nil {
			gitspaceConfig.GitspaceInstance.SpacePath = space.Path
		}
		gitspaceConfig.BranchURL = c.GetBranchURL(ctx, gitspaceConfig)
	}

	return gitspaceConfigs, filterCount, allGitspacesCount, nil
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

func (c *Service) Create(ctx context.Context, config *types.GitspaceConfig) error {
	return c.gitspaceConfigStore.Create(ctx, config)
}

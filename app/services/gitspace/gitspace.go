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
	config *types.Config,
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
		config:                config,
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
	config                *types.Config
}

func (c *Service) ListGitspacesForSpace(
	ctx context.Context,
	space *types.Space,
	filter types.GitspaceFilter,
) ([]*types.GitspaceConfig, int64, error) {
	var gitspaceConfigs []*types.GitspaceConfig
	var count int64
	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		gitspaceConfigs, err = c.gitspaceConfigStore.ListWithLatestInstance(ctx, &filter)
		if err != nil {
			return fmt.Errorf("failed to list gitspace configs: %w", err)
		}

		count, err = c.gitspaceConfigStore.Count(ctx, &filter)
		if err != nil {
			return fmt.Errorf("failed to count gitspaces in space: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	for _, gitspaceConfig := range gitspaceConfigs {
		gitspaceConfig.SpacePath = space.Path
		if gitspaceConfig.GitspaceInstance != nil {
			gitspaceConfig.GitspaceInstance.SpacePath = space.Path
		}

		gitspaceConfig.BranchURL = c.GetBranchURL(ctx, gitspaceConfig)
	}

	return gitspaceConfigs, count, nil
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

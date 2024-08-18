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

package instrument

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const (
	jobType = "instrumentation-total-repositories"
)

type RepositoryCount struct {
	enabled   bool
	svc       Service
	repoStore store.RepoStore
}

func NewRepositoryCount(
	ctx context.Context,
	config *types.Config,
	svc Service,
	repoStore store.RepoStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*RepositoryCount, error) {
	r := &RepositoryCount{
		enabled:   config.Instrumentation.Enable,
		svc:       svc,
		repoStore: repoStore,
	}

	err := executor.Register(jobType, r)
	if err != nil {
		return nil, fmt.Errorf("failed to register instrumentation job: %w", err)
	}

	if scheduler == nil {
		return nil, errors.New("job scheduler is nil")
	}
	err = scheduler.AddRecurring(ctx, jobType, jobType, config.Instrumentation.Cron, time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to register recurring job for instrumentation: %w", err)
	}

	return r, nil
}

func (c *RepositoryCount) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	if !c.enabled {
		return "", errors.New("instrumentation service is disabled")
	}

	if c.repoStore == nil {
		return "", errors.New("repository store not initialized")
	}
	// total repos in the system
	totalRepos, err := c.repoStore.CountByRootSpaces(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get repositories total count: %w", err)
	}

	if c.svc == nil {
		return "", errors.New("service not initialized")
	}

	systemPrincipal := bootstrap.NewSystemServiceSession().Principal

	for _, item := range totalRepos {
		err = c.svc.Track(ctx, Event{
			Type:      EventTypeRepositoryCount,
			Principal: systemPrincipal.ToPrincipalInfo(),
			GroupID:   item.SpaceUID,
			Properties: map[Property]any{
				PropertyRepositories: item.Total,
			},
		})
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for repository count operation: %s", err)
		}
	}

	return "", nil
}

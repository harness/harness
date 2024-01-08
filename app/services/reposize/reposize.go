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

package reposize

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const jobType = "repo-size-calculator"

type Calculator struct {
	enabled    bool
	cron       string
	maxDur     time.Duration
	numWorkers int
	git        git.Interface
	repoStore  store.RepoStore
	scheduler  *job.Scheduler
}

func (c *Calculator) Register(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	err := c.scheduler.AddRecurring(ctx, jobType, jobType, c.cron, c.maxDur)
	if err != nil {
		return fmt.Errorf("failed to register recurring job for calculator: %w", err)
	}

	return nil
}

func (c *Calculator) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	if !c.enabled {
		return "", nil
	}

	sizeInfos, err := c.repoStore.ListSizeInfos(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get repository sizes: %w", err)
	}

	expiredBefore := time.Now().Add(c.maxDur)
	log.Ctx(ctx).Info().Msgf(
		"start repo size calculation (operation timeout: %s)",
		expiredBefore.Format(time.RFC3339Nano),
	)

	var wg sync.WaitGroup
	taskCh := make(chan *types.RepositorySizeInfo)
	for i := 0; i < c.numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, c, &wg, taskCh)
	}
	for _, sizeInfo := range sizeInfos {
		select {
		case <-ctx.Done():
			break
		case taskCh <- sizeInfo:
		}
	}
	close(taskCh)
	wg.Wait()

	return "", nil
}

func worker(ctx context.Context, c *Calculator, wg *sync.WaitGroup, taskCh <-chan *types.RepositorySizeInfo) {
	defer wg.Done()

	for sizeInfo := range taskCh {
		log := log.Ctx(ctx).With().Str("repo_git_uid", sizeInfo.GitUID).Int64("repo_id", sizeInfo.ID).Logger()

		log.Debug().Msgf("previous repo size: %d", sizeInfo.Size)

		sizeOut, err := c.git.GetRepositorySize(
			ctx,
			&git.GetRepositorySizeParams{ReadParams: git.ReadParams{RepoUID: sizeInfo.GitUID}})
		if err != nil {
			log.Error().Msgf("failed to get repo size: %s", err.Error())
			continue
		}
		if sizeOut.Size == sizeInfo.Size {
			log.Debug().Msg("repo size not changed")
			continue
		}

		if err := c.repoStore.UpdateSize(ctx, sizeInfo.ID, sizeOut.Size); err != nil {
			log.Error().Msgf("failed to update repo size: %s", err.Error())
			continue
		}

		log.Debug().Msgf("new repo size: %d", sizeOut.Size)
	}
}

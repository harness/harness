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

package repo

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

type SizeCalculator struct {
	enabled    bool
	cron       string
	maxDur     time.Duration
	numWorkers int
	git        git.Interface
	repoStore  store.RepoStore
	scheduler  *job.Scheduler
	lfsStore   store.LFSObjectStore
}

func (s *SizeCalculator) Register(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	err := s.scheduler.AddRecurring(ctx, jobType, jobType, s.cron, s.maxDur)
	if err != nil {
		return fmt.Errorf("failed to register recurring job for calculator: %w", err)
	}

	return nil
}

func (s *SizeCalculator) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	if !s.enabled {
		return "", nil
	}

	sizeInfos, err := s.repoStore.ListSizeInfos(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get repository sizes: %w", err)
	}

	expiredBefore := time.Now().Add(s.maxDur)
	log.Ctx(ctx).Info().Msgf(
		"start repo size calculation (operation timeout: %s)",
		expiredBefore.Format(time.RFC3339Nano),
	)

	var wg sync.WaitGroup
	taskCh := make(chan *types.RepositorySizeInfo)
	for i := 0; i < s.numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, s, &wg, taskCh)
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

func worker(ctx context.Context, s *SizeCalculator, wg *sync.WaitGroup, taskCh <-chan *types.RepositorySizeInfo) {
	defer wg.Done()

	for sizeInfo := range taskCh {
		log := log.Ctx(ctx).With().Str("repo_git_uid", sizeInfo.GitUID).Int64("repo_id", sizeInfo.ID).Logger()

		log.Debug().Msgf("previous repo size: %d KiB", sizeInfo.Size)

		gitSizeOut, err := s.git.GetRepositorySize(
			ctx,
			&git.GetRepositorySizeParams{ReadParams: git.ReadParams{RepoUID: sizeInfo.GitUID}})
		if err != nil {
			log.Error().Msgf("failed to get repo size: %s", err.Error())
			continue
		}

		lfsSize, err := s.lfsStore.GetSizeInKBByRepoID(ctx, sizeInfo.ID)
		if err != nil {
			log.Error().Msgf("failed to get repo lfs objects size: %s", err.Error())
			continue
		}

		repoSize := gitSizeOut.Size + lfsSize
		if repoSize == sizeInfo.Size {
			log.Debug().Msg("repo size not changed")
			continue
		}

		if err := s.repoStore.UpdateSize(ctx, sizeInfo.ID, repoSize); err != nil {
			log.Error().Msgf("failed to update repo size: %s", err.Error())
			continue
		}

		log.Debug().Msgf("new repo size: %d KiB (git: %d KiB, lfs: %d KiB)", repoSize, gitSizeOut.Size, lfsSize)
	}
}

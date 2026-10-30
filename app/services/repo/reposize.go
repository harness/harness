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
	"strings"
	"sync"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog/log"
)

// Listener is notified as the calculator refreshes repository sizes.
type Listener interface {
	// AfterSizeUpdate is called for each repository whose size has just been written, inside the
	// transaction that wrote it: whatever the listener records and the size it records it from
	// commit together or not at all. The new size is not passed because the listener reads through
	// its own stores, and reading in this transaction already sees it.
	//
	// Returning an error rolls the update back, which leaves the repository due for measurement
	// again on the next run.
	AfterSizeUpdate(ctx context.Context, repoID, measuredAt int64) error

	// AfterSizeCalculation is called once a run has finished, with the same measurement time the
	// run's AfterSizeUpdate calls carried. It is a trigger only: implementations read whatever they
	// need through their own stores. Unlike AfterSizeUpdate it covers every repository, including
	// the ones this run did not measure because nothing was pushed to them.
	AfterSizeCalculation(ctx context.Context, measuredAt int64)
}

// noopListener is used when nothing is listening.
type noopListener struct{}

func (noopListener) AfterSizeUpdate(context.Context, int64, int64) error { return nil }
func (noopListener) AfterSizeCalculation(context.Context, int64)         {}

const jobType = "repo-size-calculator"

type SizeCalculator struct {
	enabled    bool
	cron       string
	maxDur     time.Duration
	numWorkers int
	git        git.Interface
	scheduler  *job.Scheduler
	lfsStore   store.LFSObjectStore
	tx         dbtx.Transactor

	repoStore        store.RepoStore
	spaceStore       store.SpaceStore
	usageMetricStore store.UsageMetricStore
	listener         Listener
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
	defer func() {
		if sendErr := s.sendMetric(ctx); sendErr != nil {
			log.Ctx(ctx).Error().Err(sendErr).Msgf("failed to send metric")
		}
	}()

	if !s.enabled {
		return "", nil
	}

	// One measurement time for the whole run, so every notification it produces reports the same
	// instant however long the run takes -- a listener that buckets by day cannot end up with one
	// repository on either side of midnight.
	measuredAt := time.Now().UnixMilli()

	// Deferred so a run that fails partway still notifies: the listener reads the sizes that are in
	// the database, and what it makes of earlier runs does not depend on this one.
	defer func() {
		s.listener.AfterSizeCalculation(ctx, measuredAt)
	}()

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
		go worker(ctx, s, &wg, taskCh, measuredAt)
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

func worker(
	ctx context.Context,
	s *SizeCalculator,
	wg *sync.WaitGroup,
	taskCh <-chan *types.RepositorySizeInfo,
	measuredAt int64,
) {
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

		if gitSizeOut.Size == sizeInfo.Size && lfsSize == sizeInfo.LFSSize {
			log.Debug().Msg("repo size not changed")
			continue
		}

		// The size and whatever the listener records from it are written together, so nothing can
		// observe a new size the listener was never told about. A listener failure rolls the size
		// back and the repository is measured again on the next run.
		err = s.tx.WithTx(ctx, func(ctx context.Context) error {
			if err := s.repoStore.UpdateSize(ctx, sizeInfo.ID, gitSizeOut.Size, lfsSize); err != nil {
				return fmt.Errorf("failed to update repo size: %w", err)
			}

			return s.listener.AfterSizeUpdate(ctx, sizeInfo.ID, measuredAt)
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to record new repo size")
			continue
		}

		totalSize := humanize.Bytes(uint64((gitSizeOut.Size + lfsSize) * 1024)) //nolint:gosec
		repoSize := humanize.Bytes(uint64(gitSizeOut.Size * 1024))              //nolint:gosec
		repoLFSSize := humanize.Bytes(uint64(lfsSize * 1024))                   //nolint:gosec

		log.Debug().Msgf("new repo size: %s (git: %s, lfs: %s)", totalSize, repoSize, repoLFSSize)
	}
}

func (s *SizeCalculator) sendMetric(
	ctx context.Context,
) error {
	date := time.Now()
	if strings.HasPrefix(s.cron, "0 0") {
		// if cron job runs at midnight store calculated size for prev day
		date = date.Add(-24 * time.Hour)
	}

	spaces, err := s.spaceStore.GetRootSpacesSize(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch root spaces size: %w", err)
	}

	metrics := make([]*types.UsageMetric, len(spaces))
	for i, rootSpace := range spaces {
		metrics[i] = &types.UsageMetric{
			Date:                date,
			RootSpaceID:         rootSpace.ID,
			RootSpaceIdentifier: rootSpace.Identifier,
			StorageTotal:        rootSpace.Size,
			LFSStorageTotal:     rootSpace.LFSSize,
		}
	}

	err = s.usageMetricStore.UpsertStorage(ctx, metrics)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).
			Msg("failed to send usage storage metrics")
	}

	return nil
}

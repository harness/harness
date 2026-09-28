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

package mergequeue

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	jobTypeOverdueChecks = "gitness:merge-queue:overdue-checks"
	jobOverdueCron       = "*/3 * * * *" // every 3 minutes
	jobOverdueTimeout    = 2 * time.Minute
)

type jobOverdueChecks struct {
	mergeQueueEntryStore store.MergeQueueEntryStore
	service              *Service
}

func (j *jobOverdueChecks) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	now := time.Now().UnixMilli()

	entries, err := j.mergeQueueEntryStore.ListOverdueChecks(ctx, now)
	if err != nil {
		return "", fmt.Errorf("failed to list overdue merge queue entries: %w", err)
	}

	if len(entries) == 0 {
		return "", nil
	}

	var repoCache repoCache

	for _, entry := range entries {
		var startedAt int64
		if entry.ChecksStarted != nil {
			startedAt = *entry.ChecksStarted
		}

		var deadlineAt int64
		if entry.ChecksDeadline != nil {
			deadlineAt = *entry.ChecksDeadline
		}

		logEntry := log.Ctx(ctx).With().
			Int64("merge_queue_id", entry.MergeQueueID).
			Str("commit_sha", entry.ChecksCommitSHA.String()).
			Int64("started_at", startedAt).
			Int64("deadline_at", deadlineAt).
			Int64("pullreq_id", entry.PullReqID).
			Logger()

		incompleteCheckIdent, incompleteCheckLink := j.getCheckInfo(ctx, entry, &repoCache, logEntry)

		err = j.service.Remove(ctx, entry.PullReqID, types.PullRequestActivityPayloadMergeQueueRemove{
			Reason:          enum.MergeQueueRemovalReasonCheckTimeout,
			CheckLink:       incompleteCheckLink,
			MergeCommitSHA:  entry.MergeCommitSHA.String(),
			MergeQueueCheck: incompleteCheckIdent,
		})
		if err != nil {
			logEntry.Warn().Err(err).Msg("failed to remove overdue merge queue entry")
			continue
		}

		logEntry.Info().Msg("removed overdue merge queue entry")
	}

	return "", nil
}

// getCheckInfo returns an overdue check (identifier and link) for a merge queue entry.
// A merge queue entry has a single deadline for all MQ checks to complete.
// Since in the pull request activity we can report just a single check,
// we can pick any incomplete check for this (even a non-required check).
// Note: If a required check wasn't even started, we won't find it the database.
// So, it can happen that the identifier and the link are empty.
func (j *jobOverdueChecks) getCheckInfo(
	ctx context.Context,
	entry *types.MergeQueueEntry,
	repoCache *repoCache,
	logEntry zerolog.Logger,
) (string, string) {
	repoID, err := repoCache.Get(ctx, j.service, entry.MergeQueueID)
	if err != nil {
		logEntry.Error().Err(err).Msg("failed to find repo for overdue merge queue entry")
		return "", ""
	}

	var incompleteCheckLink string
	var incompleteCheckIdent string

	checks, err := j.service.ListChecks(ctx, repoID, entry.ChecksCommitSHA)
	if err != nil {
		logEntry.Warn().Err(err).Msg("failed to list checks for overdue merge queue entry")
	} else {
		for i := range checks {
			// Bypassed checks count as satisfied, same as in the reported check handler.
			if checks[i].Status.IsCompleted() || checks[i].BypassedByID != nil {
				continue
			}

			incompleteCheckIdent = checks[i].Identifier
			incompleteCheckLink = checks[i].Link

			break
		}
	}

	return incompleteCheckIdent, incompleteCheckLink
}

type repoCache struct {
	m map[int64]int64 // merge queue ID -> repo ID
}

func (c *repoCache) Get(ctx context.Context, s *Service, qID int64) (int64, error) {
	if c.m == nil {
		c.m = make(map[int64]int64)
	} else if repoID, ok := c.m[qID]; ok {
		return repoID, nil
	}

	q, err := s.mergeQueueStore.Find(ctx, qID)
	if err != nil {
		return 0, fmt.Errorf("failed to find merge queue by ID: %w", err)
	}

	c.m[qID] = q.RepoID

	return q.RepoID, nil
}

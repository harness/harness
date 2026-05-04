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
	"github.com/harness/gitness/types/enum"

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
			Str("commit_sha", entry.ChecksCommitSHA.String()).
			Int64("started_at", startedAt).
			Int64("deadline_at", deadlineAt).
			Int64("pullreq_id", entry.PullReqID).
			Logger()

		err = j.service.Remove(ctx, entry.PullReqID, enum.MergeQueueRemovalReasonCheckTimeout)
		if err != nil {
			logEntry.Warn().Err(err).Msg("failed to remove overdue merge queue entry")
			continue
		}

		logEntry.Info().Msg("removed overdue merge queue entry")
	}

	return "", nil
}

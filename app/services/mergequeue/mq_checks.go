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

	"github.com/harness/gitness/app/bootstrap"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// updateChecks applies chain-based state transitions to ChecksPending entries.
// Consecutive pending entries are grouped into chains of at most groupSize.
// Within each chain the last entry transitions to ChecksInProgress and the rest to MergeGroup.
// All entries in a chain share the same ChecksCommitSHA (the MergeCommitSHA of the last entry)
// and have ChecksStarted set to now.
// maxInProgress limits the total number of ChecksInProgress entries (existing + newly created).
// A value <= 0 means no limit.
// maxCheckDurationSeconds is used to compute ChecksDeadline from now.
func (s *Service) updateChecks(
	entries []*types.MergeQueueEntry,
	groupSize int,
	maxInProgress int,
	maxCheckDurationSeconds int,
	now int64,
) (updated []*types.MergeQueueEntry, toStore []*types.MergeQueueEntry) {
	inProgressCount := 0
	for _, entry := range entries {
		if entry.State == enum.MergeQueueEntryStateChecksInProgress {
			inProgressCount++
		}
	}

	chainStart := -1

	for i := 0; i <= len(entries); i++ {
		isPending := i < len(entries) && entries[i].State == enum.MergeQueueEntryStateChecksPending

		if isPending && chainStart == -1 {
			chainStart = i
		}

		chainLen := 0
		if chainStart != -1 {
			chainLen = i - chainStart
		}

		if chainStart == -1 || (isPending && chainLen < groupSize) {
			continue
		}

		if maxInProgress > 0 && inProgressCount >= maxInProgress {
			if isPending {
				chainStart = i
			} else {
				chainStart = -1
			}
			continue
		}

		chain := entries[chainStart:i]
		checksCommitSHA := chain[len(chain)-1].MergeCommitSHA

		var deadline *int64
		if maxCheckDurationSeconds > 0 {
			d := now + int64(maxCheckDurationSeconds)*1000
			deadline = &d
		}

		for j, entry := range chain {
			entry.State = enum.MergeQueueEntryStateMergeGroup
			if j == len(chain)-1 {
				entry.State = enum.MergeQueueEntryStateChecksInProgress
			}
			entry.ChecksCommitSHA = checksCommitSHA
			entry.ChecksStarted = &now
			entry.ChecksDeadline = deadline
			toStore = append(toStore, entry)
		}

		inProgressCount++

		if isPending {
			chainStart = i // current entry starts the next chain
		} else {
			chainStart = -1
		}
	}

	return entries, toStore
}

func (s *Service) startChecks(ctx context.Context, q *types.MergeQueue, commitSHA sha.SHA) {
	if commitSHA.IsEmpty() || commitSHA.IsNil() {
		log.Ctx(ctx).Warn().
			Int64("repo_id", q.RepoID).
			Str("branch", q.Branch).
			Str("commit_sha", commitSHA.String()).
			Msg("startChecks called with invalid sha")
		return
	}

	s.mergeQueueEventReporter.ChecksRequested(ctx, &mergequeueevents.ChecksRequestedPayload{
		Base: mergequeueevents.Base{
			RepoID: q.RepoID,
			Branch: q.Branch,
		},
		PrincipalID: bootstrap.NewSystemServiceSession().Principal.ID,
		CommitSHA:   commitSHA.String(),
	})
}

func (s *Service) stopChecks(ctx context.Context, q *types.MergeQueue, commitSHA sha.SHA) {
	if commitSHA.IsEmpty() || commitSHA.IsNil() {
		log.Ctx(ctx).Warn().
			Int64("repo_id", q.RepoID).
			Str("branch", q.Branch).
			Str("commit_sha", commitSHA.String()).
			Msg("stopChecks called with invalid sha")
		return
	}

	s.mergeQueueEventReporter.ChecksCanceled(ctx, &mergequeueevents.ChecksCanceledPayload{
		Base: mergequeueevents.Base{
			RepoID: q.RepoID,
			Branch: q.Branch,
		},
		PrincipalID: bootstrap.NewSystemServiceSession().Principal.ID,
		CommitSHA:   commitSHA.String(),
	})
}

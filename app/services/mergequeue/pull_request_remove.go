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

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/bootstrap"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var ErrNotInQueue = errors.New("not in queue")

// Remove removes a pull request from a merge queue.
// Intended to be called from external packages, such as a pull request controller.
// Triggers merge queue Updated event. Must not be called inside a transaction.
func (s *Service) Remove(
	ctx context.Context,
	pullReqID int64,
	reason enum.MergeQueueRemovalReason,
) error {
	entry, err := s.mergeQueueEntryStore.Find(ctx, pullReqID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return ErrNotInQueue
	}
	if err != nil {
		return fmt.Errorf("failed to find merge queue entry: %w", err)
	}

	q, err := s.mergeQueueStore.Find(ctx, entry.MergeQueueID)
	if err != nil {
		return fmt.Errorf("failed to find merge queue: %w", err)
	}

	err = s.remove(ctx, pullReqID, reason)
	if err != nil {
		return fmt.Errorf("failed to remove merge queue entry: %w", err)
	}

	s.mergeQueueEventReporter.Updated(ctx, &mergequeueevents.UpdatedPayload{
		Base: mergequeueevents.Base{
			RepoID: q.RepoID,
			Branch: q.Branch,
		},
	})

	return nil
}

// removeAll removes all entries from a merge queue, deletes the merge queue row,
// and removes the merge queue git reference. Must not be called inside a transaction.
func (s *Service) removeAll(
	ctx context.Context,
	repo *types.RepositoryCore,
	q *types.MergeQueue,
) error {
	checksToAbort := make(map[sha.SHA]struct{})
	var pullReqs []*types.PullReq

	err := controller.TxOptLock(ctx, s.tx, func(ctx context.Context) error {
		entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
		if err != nil {
			return fmt.Errorf("failed to list merge queue entries: %w", err)
		}

		for _, entry := range entries {
			if entry.State == enum.MergeQueueEntryStateChecksInProgress &&
				!entry.ChecksCommitSHA.IsEmpty() && !entry.ChecksCommitSHA.IsNil() {
				checksToAbort[entry.ChecksCommitSHA] = struct{}{}
			}
		}

		pullReqs = make([]*types.PullReq, 0, len(entries))

		for _, entry := range entries {
			pullReqID := entry.PullReqID

			pr, prErr := s.pullreqStore.Find(ctx, pullReqID)
			if prErr != nil {
				return fmt.Errorf("failed to find pull request %d: %w", pullReqID, prErr)
			}

			if pr.SubState != enum.PullReqSubStateMergeQueue {
				continue
			}

			pr.SubState = enum.PullReqSubStateNone
			pr.ActivitySeq++

			if prErr = s.pullreqStore.Update(ctx, pr); prErr != nil {
				return fmt.Errorf("failed to update pull request %d: %w", pullReqID, prErr)
			}

			pullReqs = append(pullReqs, pr)
		}

		if err := s.mergeQueueEntryStore.DeleteAllForMergeQueue(ctx, q.ID); err != nil {
			return fmt.Errorf("failed to delete all merge queue entries: %w", err)
		}

		if err := s.mergeQueueStore.Delete(ctx, q.ID); err != nil {
			return fmt.Errorf("failed to delete merge queue: %w", err)
		}

		return nil
	}, dbtx.TxRepeatableRead)
	if err != nil {
		return fmt.Errorf("failed to remove all entries from merge queue: %w", err)
	}

	session := bootstrap.NewSystemServiceSession()

	for _, pr := range pullReqs {
		payload := &types.PullRequestActivityPayloadMergeQueueRemove{
			Reason: enum.MergeQueueRemovalReasonNoQueue,
		}

		_, prErr := s.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload, nil)
		if prErr != nil {
			log.Ctx(ctx).Warn().Err(prErr).Int64("pullreq_id", pr.ID).
				Msg("failed to create merge queue removal activity")
		}
	}

	for commitSHA := range checksToAbort {
		s.stopChecks(ctx, q, commitSHA)
	}

	s.deleteReference(ctx, repo, q.Branch)

	return nil
}

// remove removes a pull request from a merge queue.
// Does not trigger events. Must not be called inside a transaction.
func (s *Service) remove(
	ctx context.Context,
	pullReqID int64,
	reason enum.MergeQueueRemovalReason,
) error {
	var (
		err         error
		pr          *types.PullReq
		q           *types.MergeQueue
		entries     []*types.MergeQueueEntry
		addActivity bool
	)

	checksToAbort := make(map[sha.SHA]struct{})

	err = controller.TxOptLock(ctx, s.tx, func(ctx context.Context) error {
		pr, err = s.pullreqStore.Find(ctx, pullReqID)
		if err != nil {
			return fmt.Errorf("failed to find pull request: %w", err)
		}

		q, err = s.mergeQueueStore.FindByRepoAndBranch(ctx, pr.TargetRepoID, pr.TargetBranch)
		if errors.Is(err, store.ErrResourceNotFound) {
			return ErrNotInQueue
		}
		if err != nil {
			return fmt.Errorf("failed to find merge queue: %w", err)
		}

		entries, err = s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
		if err != nil {
			return fmt.Errorf("failed to list merge queue entries: %w", err)
		}

		index := -1
		for i, entry := range entries {
			if entry.PullReqID == pullReqID {
				index = i
				break
			}
		}
		if index == -1 {
			return ErrNotInQueue
		}

		removedEntry := entries[index]

		err = s.mergeQueueEntryStore.Delete(ctx, pullReqID)
		if err != nil {
			return fmt.Errorf("failed to delete merge queue entry: %w", err)
		}

		// Abort checks for the removed entry itself if it was the chain leader.
		if removedEntry.State == enum.MergeQueueEntryStateChecksInProgress {
			checksToAbort[removedEntry.ChecksCommitSHA] = struct{}{}
		}

		// Re-queue entries before the removed one that belong to the same chain.
		// MergeGroup entries come before their ChecksInProgress leader, so removing
		// the leader orphans them. Their merge commits are still valid - only the check
		// grouping is broken - so we move them back to ChecksPending rather than
		// fully resetting them. Identify chain members by matching ChecksCommitSHA.
		if !removedEntry.ChecksCommitSHA.IsEmpty() && !removedEntry.ChecksCommitSHA.IsNil() {
			for i := 0; i < index; i++ {
				e := entries[i]

				if e.State != enum.MergeQueueEntryStateMergeGroup {
					continue
				}

				if !e.ChecksCommitSHA.Equal(removedEntry.ChecksCommitSHA) {
					continue
				}

				e.State = enum.MergeQueueEntryStateChecksPending
				e.ChecksCommitSHA = sha.None
				e.ChecksStarted = nil
				e.ChecksDeadline = nil

				err = s.mergeQueueEntryStore.Update(ctx, e)
				if err != nil {
					return fmt.Errorf("failed to update merge queue entry: %w", err)
				}
			}
		}

		// Reset entries after the removed one whose merge commit chain is now broken.
		for i := index + 1; i < len(entries); i++ {
			entry := entries[i]

			if entry.State == enum.MergeQueueEntryStateMergePending {
				continue
			}

			if entry.State == enum.MergeQueueEntryStateChecksInProgress {
				checksToAbort[entry.MergeCommitSHA] = struct{}{}
			}

			entry.CancelMerge()

			err = s.mergeQueueEntryStore.Update(ctx, entry)
			if err != nil {
				return fmt.Errorf("failed to update merge queue entry: %w", err)
			}
		}

		if pr.State == enum.PullReqStateMerged || pr.State == enum.PullReqStateClosed ||
			pr.SubState != enum.PullReqSubStateMergeQueue {
			return nil
		}

		pr.SubState = enum.PullReqSubStateNone
		pr.ActivitySeq++

		addActivity = true

		err = s.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		return nil
	}, dbtx.TxRepeatableRead)
	if err != nil {
		return fmt.Errorf("failed to remove pull request from merge queue: %w", err)
	}

	if !addActivity {
		return nil
	}

	session := bootstrap.NewSystemServiceSession()

	payload := &types.PullRequestActivityPayloadMergeQueueRemove{
		Reason: reason,
	}

	_, err = s.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload, nil)
	if err != nil {
		// Non-critical failure
		log.Ctx(ctx).Warn().Err(err).Msg("failed to create merge queue pull request activity")
	}

	for mergeSHA := range checksToAbort {
		s.stopChecks(ctx, q, mergeSHA)
	}

	return nil
}

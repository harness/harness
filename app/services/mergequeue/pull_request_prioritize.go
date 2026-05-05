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
	"slices"

	"github.com/harness/gitness/app/api/controller"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Prioritize moves the given pull request to the front of its merge queue.
// All entries are reset to MergePending state because the reordering invalidates
// existing merge commits and checks. Must not be called inside a transaction.
func (s *Service) Prioritize(
	ctx context.Context,
	pullReqID int64,
) error {
	priorityEntry, err := s.mergeQueueEntryStore.Find(ctx, pullReqID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return ErrNotInQueue
	}
	if err != nil {
		return fmt.Errorf("failed to find merge queue entry: %w", err)
	}

	var checksToAbort map[sha.SHA]struct{}

	var q *types.MergeQueue

	err = controller.TxOptLock(ctx, s.tx, func(ctx context.Context) error {
		checksToAbort = make(map[sha.SHA]struct{})

		q, err = s.mergeQueueStore.Find(ctx, priorityEntry.MergeQueueID)
		if err != nil {
			return fmt.Errorf("failed to find merge queue: %w", err)
		}

		entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
		if err != nil {
			return fmt.Errorf("failed to list merge queue entries: %w", err)
		}

		if len(entries) == 0 {
			return ErrNotInQueue
		}

		// If already first, nothing to do.
		if entries[0].PullReqID == pullReqID {
			return ErrAlreadyHead
		}

		priorityIndex := slices.IndexFunc(entries, func(entry *types.MergeQueueEntry) bool {
			return entry.PullReqID == pullReqID
		})
		if priorityIndex == -1 {
			return ErrNotInQueue
		}

		// Sequence number for the new order.
		seq := q.OrderSequence + 1

		// Reserve new sequence numbers for all entries.
		count := int64(len(entries))
		q.OrderSequence += count

		err = s.mergeQueueStore.Update(ctx, q)
		if err != nil {
			return fmt.Errorf("failed to reserve sequence numbers: %w", err)
		}

		priorityEntry = entries[priorityIndex]

		copy(entries[1:priorityIndex+1], entries[0:priorityIndex])
		entries[0] = priorityEntry

		// Assign new order: target entry gets the first sequence number,
		// remaining entries follow in their original relative order.
		for _, entry := range entries {
			entry.OrderIndex = seq
			seq++

			// Collect checks to abort.
			if entry.State == enum.MergeQueueEntryStateChecksInProgress &&
				!entry.ChecksCommitSHA.IsEmpty() && !entry.ChecksCommitSHA.IsNil() {
				checksToAbort[entry.ChecksCommitSHA] = struct{}{}
			}

			entry.CancelMerge()
			if err := s.mergeQueueEntryStore.Update(ctx, entry); err != nil {
				return fmt.Errorf("failed to update prioritized entry: %w", err)
			}
		}

		return nil
	}, dbtx.TxRepeatableRead)
	if err != nil {
		return fmt.Errorf("failed to prioritize merge queue entry: %w", err)
	}

	for commitSHA := range checksToAbort {
		s.stopChecks(ctx, q, commitSHA)
	}

	s.mergeQueueEventReporter.Updated(ctx, &mergequeueevents.UpdatedPayload{
		Base: mergequeueevents.Base{
			RepoID: q.RepoID,
			Branch: q.Branch,
		},
	})

	return nil
}

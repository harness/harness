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
	"encoding/json"
	"fmt"
	"sort"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// BuildMergeQueueInfo composes public-facing info (state, position, checks)
// for a pull request's merge queue entry. Intended to be called from controllers.
func (s *Service) BuildMergeQueueInfo(
	ctx context.Context,
	targetRepo *types.RepositoryCore,
	entry *types.MergeQueueEntry,
	setup protection.MergeQueueSetup,
) (*types.MergeQueueInfo, error) {
	entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, entry.MergeQueueID)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge queue entries: %w", err)
	}

	prsAhead := -1
	for i, e := range entries {
		if e.PullReqID == entry.PullReqID {
			prsAhead = i
			break
		}
	}
	if prsAhead < 0 {
		return nil, usererror.BadRequest("Pull request is not in merge queue.")
	}

	return s.buildEntryInfo(ctx, targetRepo.ID, entry, prsAhead, setup.RequiredChecks)
}

// ListMergeQueue returns the entire merge queue for the given repo and branch
// in queue order, pairing each entry with its pull request and merge queue info.
// Returns an empty slice if the merge queue does not exist or has no entries.
func (s *Service) ListMergeQueue(
	ctx context.Context,
	targetRepo *types.RepositoryCore,
	branch string,
	setup protection.MergeQueueSetup,
) ([]types.MergeQueueListItem, error) {
	q, err := s.mergeQueueStore.FindByRepoAndBranch(ctx, targetRepo.ID, branch)
	if errors.Is(err, store.ErrResourceNotFound) {
		return []types.MergeQueueListItem{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find merge queue: %w", err)
	}

	entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge queue entries: %w", err)
	}

	items := make([]types.MergeQueueListItem, len(entries))
	for i, entry := range entries {
		info, err := s.buildEntryInfo(ctx, targetRepo.ID, entry, i, setup.RequiredChecks)
		if err != nil {
			return nil, fmt.Errorf("failed to build merge queue info for pull request %d: %w", entry.PullReqID, err)
		}

		pr, err := s.pullreqStore.Find(ctx, entry.PullReqID)
		if err != nil {
			return nil, fmt.Errorf("failed to find pull request %d for merge queue entry: %w", entry.PullReqID, err)
		}

		items[i] = types.MergeQueueListItem{
			MergeQueueInfo: info,
			PullReq:        pr,
		}
	}

	return items, nil
}

// buildEntryInfo composes the public-facing merge queue info for a single entry,
// given its zero-based position in the queue and the queue's required checks.
func (s *Service) buildEntryInfo(
	ctx context.Context,
	repoID int64,
	entry *types.MergeQueueEntry,
	prsAhead int,
	requiredChecks []string,
) (*types.MergeQueueInfo, error) {
	requiredIDs := make(map[string]struct{}, len(requiredChecks))
	for _, id := range requiredChecks {
		requiredIDs[id] = struct{}{}
	}

	prChecks := make([]types.PullReqCheck, 0, len(requiredIDs))

	if !entry.ChecksCommitSHA.IsEmpty() && !entry.ChecksCommitSHA.IsNil() {
		checks, err := s.ListChecks(ctx, repoID, entry.ChecksCommitSHA)
		if err != nil {
			return nil, fmt.Errorf("failed to list checks: %w", err)
		}

		for _, check := range checks {
			_, required := requiredIDs[check.Identifier]
			if required {
				delete(requiredIDs, check.Identifier)
			}

			prChecks = append(prChecks, types.PullReqCheck{
				Required:   required,
				Bypassable: false,
				Check:      check,
			})
		}
	}

	for requiredID := range requiredIDs {
		prChecks = append(prChecks, types.PullReqCheck{
			Required:   true,
			Bypassable: false,
			Check: types.Check{
				RepoID:     repoID,
				CommitSHA:  entry.ChecksCommitSHA.String(),
				Identifier: requiredID,
				Status:     enum.CheckStatusPending,
				Metadata:   json.RawMessage("{}"),
			},
		})
	}

	sort.Slice(prChecks, func(i, j int) bool {
		return prChecks[i].Check.Identifier < prChecks[j].Check.Identifier
	})

	return &types.MergeQueueInfo{
		State:             entry.State,
		MergeCommitSHA:    entry.MergeCommitSHA,
		ChecksCommitSHA:   entry.ChecksCommitSHA,
		Checks:            prChecks,
		PullRequestsAhead: prsAhead,
	}, nil
}

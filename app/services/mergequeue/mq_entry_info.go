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

	var prsAhead int
	var found bool
	for _, e := range entries {
		if e.PullReqID == entry.PullReqID {
			found = true
			break
		}
		prsAhead++
	}
	if !found {
		return nil, usererror.BadRequest("Pull request is not in merge queue.")
	}

	requiredIDs := make(map[string]struct{}, len(setup.RequiredChecks))
	for _, id := range setup.RequiredChecks {
		requiredIDs[id] = struct{}{}
	}

	prChecks := make([]types.PullReqCheck, 0, len(requiredIDs))

	if !entry.ChecksCommitSHA.IsEmpty() && !entry.ChecksCommitSHA.IsNil() {
		checks, err := s.ListChecks(ctx, targetRepo.ID, entry.ChecksCommitSHA)
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
				RepoID:     targetRepo.ID,
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

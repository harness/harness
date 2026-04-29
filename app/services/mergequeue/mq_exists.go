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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RuleIdentifier is identifier of a dummy rule for which a violation would be
// reported to block the branch update.
const RuleIdentifier = "merge_queue"

// Violation returns a dummy rule violation that would be used to block a branch change.
func Violation(branch string) types.RuleViolations {
	return types.RuleViolations{
		Rule: types.RuleInfo{
			Identifier: RuleIdentifier,
			State:      enum.RuleStateActive,
		},
		Violations: []types.Violation{
			{
				Code: RuleIdentifier,
				Message: fmt.Sprintf(
					"Cannot modify branch %q because it has a pull request in the merge queue.", branch),
			},
		},
	}
}

// IsBranchInQueue returns if the provided branch from the provided repository
// has a pull request in the merge queue.
func (s *Service) IsBranchInQueue(
	ctx context.Context,
	repoID int64,
	branch string,
) (bool, error) {
	branchesWithPR, err := s.mergeQueueEntryStore.BranchesWithPullReqInQueue(ctx, repoID, []string{branch})
	if err != nil {
		return false, fmt.Errorf("failed to get branches with a PR in merge queue: %w", err)
	}

	return len(branchesWithPR) > 0, nil
}

// BranchInQueueViolations returns []types.RuleViolations
// if the provided branch from the provided repository
// has a pull request in the merge queue.
func (s *Service) BranchInQueueViolations(
	ctx context.Context,
	repoID int64,
	branch string,
) ([]types.RuleViolations, error) {
	inQueue, err := s.IsBranchInQueue(ctx, repoID, branch)
	if err != nil {
		return nil, err
	}

	if inQueue {
		return []types.RuleViolations{Violation(branch)}, nil
	}

	return nil, nil
}

// BranchesWithPullReqInQueue returns a set of which branches from the provided list of branches
// have a pull request in the merge queue. The function is intended to be used from
// PreReceive controller.
func (s *Service) BranchesWithPullReqInQueue(
	ctx context.Context,
	repoID int64,
	branches []string,
) (map[string]struct{}, error) {
	branchesWithPR, err := s.mergeQueueEntryStore.BranchesWithPullReqInQueue(ctx, repoID, branches)
	if err != nil {
		return nil, fmt.Errorf("failed to get branches with a PR in merge queue: %w", err)
	}

	return branchesWithPR, nil
}

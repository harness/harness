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

package pullreq

import (
	"context"
	"fmt"
	"sort"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ReviewerList returns reviewer list for the pull request.
func (c *Controller) ReviewerList(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) ([]*types.PullReqReviewer, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	reviewers, err := c.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return nil, err
	}

	sortReviewersByDecision(reviewers)

	return reviewers, nil
}

// reviewDecisionRank ranks a review decision for sorting; lower sorts first
// (changereq > approved > reviewed > pending), matching getHighestOrderDecision.
func reviewDecisionRank(decision enum.PullReqReviewDecision) int {
	switch decision {
	case enum.PullReqReviewDecisionChangeReq:
		return 0
	case enum.PullReqReviewDecisionApproved:
		return 1
	case enum.PullReqReviewDecisionReviewed:
		return 2
	case enum.PullReqReviewDecisionPending:
		return 3
	default:
		return 4
	}
}

// sortReviewersByDecision stably orders reviewers by decision priority, keeping
// creation order among equal decisions.
func sortReviewersByDecision(reviewers []*types.PullReqReviewer) {
	sort.SliceStable(reviewers, func(i, j int) bool {
		return reviewDecisionRank(reviewers[i].ReviewDecision) < reviewDecisionRank(reviewers[j].ReviewDecision)
	})
}

// sortUserGroupReviewersByDecision stably orders user group reviewers by their
// compound decision priority.
func sortUserGroupReviewersByDecision(reviewers []*types.UserGroupReviewer) {
	sort.SliceStable(reviewers, func(i, j int) bool {
		return reviewDecisionRank(reviewers[i].Decision) < reviewDecisionRank(reviewers[j].Decision)
	})
}

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
	"slices"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListSuggestedReviewers returns reviewer suggestions for a pull request.
func (c *Controller) ListSuggestedReviewers(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	pagination types.Pagination,
) (*types.ListReviewerSuggestionsOutput, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pull request: %w", err)
	}

	var count int64
	var suggestions []*types.PullReqReviewerSuggestion

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		suggestions, err = c.reviewerSuggestionStore.List(ctx, pr.ID, pagination)
		if err != nil {
			return fmt.Errorf("failed to list reviewer suggestions: %w", err)
		}

		if pagination.Page == 1 && len(suggestions) < pagination.Size {
			count = int64(len(suggestions))
			return nil
		}

		count, err = c.reviewerSuggestionStore.Count(ctx, pr.ID)
		if err != nil {
			return fmt.Errorf("failed to count reviewer suggestions: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	if len(suggestions) == 0 {
		return &types.ListReviewerSuggestionsOutput{
			Suggestions: []*types.PullReqReviewerSuggestionInfo{},
		}, count, nil
	}

	// Collect all principal IDs, then dedup.
	ids := make([]int64, 0, len(suggestions)*2)
	for _, s := range suggestions {
		ids = append(ids, s.PrincipalID, s.CreatedBy)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)

	principalInfos, err := c.principalInfoCache.Map(ctx, ids)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch principal infos: %w", err)
	}

	items := make([]*types.PullReqReviewerSuggestionInfo, 0, len(suggestions))
	for _, s := range suggestions {
		reviewer, ok := principalInfos[s.PrincipalID]
		if !ok {
			continue
		}
		suggestedBy, ok := principalInfos[s.CreatedBy]
		if !ok {
			continue
		}
		items = append(items, &types.PullReqReviewerSuggestionInfo{
			Reviewer:    *reviewer,
			SuggestedBy: *suggestedBy,
			SuggestedAt: s.Created,
		})
	}

	return &types.ListReviewerSuggestionsOutput{Suggestions: items}, count, nil
}

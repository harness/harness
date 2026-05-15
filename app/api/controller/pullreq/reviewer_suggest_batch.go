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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	pullreqservice "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ReviewerSuggestBatch stores reviewer suggestions for a pull request.
func (c *Controller) ReviewerSuggestBatch(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *pullreqservice.ReviewerSuggestBatchInput,
) error {
	if err := in.Sanitize(); err != nil {
		return usererror.BadRequestf("Invalid reviewer IDs: %v", err)
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	reviewers, err := c.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return fmt.Errorf("failed to list pull request reviewers: %w", err)
	}

	existingReviewerIDs := make(map[int64]struct{}, len(reviewers))
	for _, reviewer := range reviewers {
		existingReviewerIDs[reviewer.PrincipalID] = struct{}{}
	}

	suggestions := make([]*types.PullReqReviewerSuggestion, 0, len(in.ReviewerIDs))
	for _, reviewerID := range in.ReviewerIDs {
		if _, ok := existingReviewerIDs[reviewerID]; ok {
			continue
		}

		if reviewerID == pr.CreatedBy {
			return usererror.BadRequestf("Pull request author with ID %d can't be suggested as a reviewer.", reviewerID)
		}

		reviewerPrincipal, err := c.principalStore.Find(ctx, reviewerID)
		if errors.Is(err, store.ErrResourceNotFound) {
			return usererror.NotFoundf("Suggested reviewer with ID %d could not be found.", reviewerID)
		}
		if err != nil {
			return fmt.Errorf("failed to find reviewer principal: %w", err)
		}

		err = apiauth.CheckRepo(ctx, c.authorizer, &auth.Session{
			Principal: *reviewerPrincipal,
			Metadata:  nil,
		}, repo, enum.PermissionRepoReview)
		if err != nil {
			if !apiauth.IsNoAccess(err) {
				return fmt.Errorf("failed to check suggested reviewer access: %w", err)
			}
			return usererror.BadRequestf(
				"Suggested reviewer with ID %d does not have permission to review this repository.",
				reviewerID,
			)
		}

		suggestions = append(suggestions, &types.PullReqReviewerSuggestion{
			PullReqID:   pr.ID,
			CreatedBy:   session.Principal.ID,
			PrincipalID: reviewerID,
		})
	}

	err = c.reviewerSuggestionStore.CreateMany(ctx, suggestions)
	if err != nil {
		return fmt.Errorf("failed to create reviewer suggestions: %w", err)
	}

	return nil
}

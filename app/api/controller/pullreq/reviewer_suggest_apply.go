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
	"errors"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ReviewerSuggestApply applies one reviewer suggestion to the pull request.
func (c *Controller) ReviewerSuggestApply(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	reviewerID int64,
) (*types.PullReqReviewer, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	suggestion, err := c.reviewerSuggestionStore.Find(ctx, pr.ID, reviewerID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return nil, usererror.NotFoundf("Suggested reviewer with ID %d could not be found.", reviewerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find reviewer suggestion: %w", err)
	}

	reviewer, added, err := c.pullreqService.AddReviewer(ctx, &session.Principal, repo, pr, suggestion.PrincipalID)
	if err != nil {
		return nil, err
	}

	if added {
		c.reportReviewerAddition(ctx, session, pr, reviewer)
		c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypePullReqReviewerAdded, pr)
	}

	return reviewer, nil
}

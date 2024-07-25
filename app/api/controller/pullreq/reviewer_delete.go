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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// ReviewerDelete deletes reviewer from the reviewer list for the given PR.
func (c *Controller) ReviewerDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	reviewerID int64,
) error {
	repo, err := c.getRepo(ctx, repoRef)
	if err != nil {
		return fmt.Errorf("failed to find repository: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request: %w", err)
	}

	reviewer, err := c.reviewerStore.Find(ctx, pr.ID, reviewerID)
	if err != nil {
		return fmt.Errorf("failed to find reviewer: %w", err)
	}

	var reqPermission enum.Permission
	switch {
	case session.Principal.ID == reviewer.PrincipalID:
		reqPermission = enum.PermissionRepoView // Anybody should be allowed to remove their own reviews.
	case reviewer.ReviewDecision == enum.PullReqReviewDecisionPending:
		reqPermission = enum.PermissionRepoPush // The reviewer was asked for a review but didn't submit it yet.
	default:
		reqPermission = enum.PermissionRepoEdit // RepoEdit permission is required to remove a submitted review.
	}

	// Make sure the caller has the right permission even if the PR is merged, so that we can return the correct error.
	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}

	if pr.Merged != nil {
		return usererror.BadRequest("Pull request is already merged")
	}

	err = c.reviewerStore.Delete(ctx, pr.ID, reviewerID)
	if err != nil {
		return fmt.Errorf("failed to delete reviewer: %w", err)
	}

	if reviewer.ReviewDecision == enum.PullReqReviewDecisionPending {
		// We create a pull request activity entry only if a review has actually been submitted.
		return nil
	}

	activityPayload := &types.PullRequestActivityPayloadReviewerDelete{
		CommitSHA:   reviewer.SHA,
		Decision:    reviewer.ReviewDecision,
		PrincipalID: reviewer.PrincipalID,
	}

	metadata := &types.PullReqActivityMetadata{
		Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{reviewer.PrincipalID}},
	}

	err = func() error {
		if pr, err = c.pullreqStore.UpdateActivitySeq(ctx, pr); err != nil {
			return fmt.Errorf("failed to increment pull request activity sequence: %w", err)
		}

		_, err = c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, activityPayload, metadata)
		if err != nil {
			return fmt.Errorf("failed to create pull request activity: %w", err)
		}

		return nil
	}()
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity after reviewer removal")
	}

	return nil
}

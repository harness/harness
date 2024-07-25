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
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ReviewerAddInput struct {
	ReviewerID int64 `json:"reviewer_id"`
}

// ReviewerAdd adds a new reviewer to the pull request.
func (c *Controller) ReviewerAdd(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *ReviewerAddInput,
) (*types.PullReqReviewer, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if pr.Merged != nil {
		return nil, usererror.BadRequest("Can't request review for merged pull request")
	}

	if in.ReviewerID == 0 {
		return nil, usererror.BadRequest("Must specify reviewer ID.")
	}

	if in.ReviewerID == pr.CreatedBy {
		return nil, usererror.BadRequest("Pull request author can't be added as a reviewer.")
	}

	addedByInfo := session.Principal.ToPrincipalInfo()

	var reviewerType enum.PullReqReviewerType
	switch session.Principal.ID {
	case pr.CreatedBy:
		reviewerType = enum.PullReqReviewerTypeRequested
	case in.ReviewerID:
		reviewerType = enum.PullReqReviewerTypeSelfAssigned
	default:
		reviewerType = enum.PullReqReviewerTypeAssigned
	}

	reviewerInfo := addedByInfo
	if reviewerType != enum.PullReqReviewerTypeSelfAssigned {
		var reviewerPrincipal *types.Principal
		reviewerPrincipal, err = c.principalStore.Find(ctx, in.ReviewerID)
		if err != nil {
			return nil, err
		}

		reviewerInfo = reviewerPrincipal.ToPrincipalInfo()

		// TODO: To check the reviewer's access to the repo we create a dummy session object. Fix it.
		if err = apiauth.CheckRepo(ctx, c.authorizer, &auth.Session{
			Principal: *reviewerPrincipal,
			Metadata:  nil,
		}, repo, enum.PermissionRepoView); err != nil {
			log.Ctx(ctx).Info().Msgf("Reviewer principal: %s access error: %s", reviewerInfo.UID, err)
			return nil, usererror.BadRequest("The reviewer doesn't have enough permissions for the repository.")
		}
	}

	var reviewer *types.PullReqReviewer

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		reviewer, err = c.reviewerStore.Find(ctx, pr.ID, in.ReviewerID)
		if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
			return err
		}

		if reviewer != nil {
			return nil
		}

		reviewer = newPullReqReviewer(session, pr, repo, reviewerInfo, addedByInfo, reviewerType, in)

		return c.reviewerStore.Create(ctx, reviewer)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request reviewer: %w", err)
	}

	c.reportReviewerAddition(ctx, session, pr, reviewer)
	return reviewer, err
}

func (c *Controller) reportReviewerAddition(
	ctx context.Context,
	session *auth.Session,
	pr *types.PullReq,
	reviewer *types.PullReqReviewer,
) {
	c.eventReporter.ReviewerAdded(ctx, &events.ReviewerAddedPayload{
		Base:       eventBase(pr, &session.Principal),
		ReviewerID: reviewer.PrincipalID,
	})
}

// newPullReqReviewer creates new pull request reviewer object.
func newPullReqReviewer(session *auth.Session, pullReq *types.PullReq,
	repo *types.Repository, reviewerInfo, addedByInfo *types.PrincipalInfo,
	reviewerType enum.PullReqReviewerType, in *ReviewerAddInput) *types.PullReqReviewer {
	now := time.Now().UnixMilli()
	return &types.PullReqReviewer{
		PullReqID:      pullReq.ID,
		PrincipalID:    in.ReviewerID,
		CreatedBy:      session.Principal.ID,
		Created:        now,
		Updated:        now,
		RepoID:         repo.ID,
		Type:           reviewerType,
		LatestReviewID: nil,
		ReviewDecision: enum.PullReqReviewDecisionPending,
		SHA:            "",
		Reviewer:       *reviewerInfo,
		AddedBy:        *addedByInfo,
	}
}

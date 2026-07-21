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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ReviewSubmitInput struct {
	CommitSHA string                     `json:"commit_sha"`
	Decision  enum.PullReqReviewDecision `json:"decision"`
}

func (in *ReviewSubmitInput) Validate() error {
	if in.CommitSHA == "" {
		return usererror.BadRequest("CommitSHA is a mandatory field")
	}

	decision, ok := in.Decision.Sanitize()
	if !ok || decision == enum.PullReqReviewDecisionPending {
		msg := fmt.Sprintf("Decision must be: %q, %q or %q.",
			enum.PullReqReviewDecisionApproved,
			enum.PullReqReviewDecisionChangeReq,
			enum.PullReqReviewDecisionReviewed)
		return usererror.BadRequest(msg)
	}
	in.Decision = decision

	return nil
}

// ReviewSubmit creates a new pull request review.
func (c *Controller) ReviewSubmit(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *ReviewSubmitInput,
) error {
	if err := in.Validate(); err != nil {
		return err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if pr.Merged != nil {
		return usererror.BadRequest("Can't submit a review for merged pull requests")
	}

	if pr.CreatedBy == session.Principal.ID {
		return usererror.BadRequest("Can't submit review to own pull requests.")
	}

	commit, err := c.git.GetCommit(ctx, &git.GetCommitParams{
		ReadParams: git.ReadParams{RepoUID: repo.GitUID},
		Revision:   in.CommitSHA,
	})
	if err != nil {
		return fmt.Errorf("failed to get git branch sha: %w", err)
	}

	commitSHA := commit.Commit.SHA.String()
	now := time.Now().UnixMilli()
	review := &types.PullReqReview{
		ID:        0,
		CreatedBy: session.Principal.ID,
		Created:   now,
		Updated:   now,
		PullReqID: pr.ID,
		Decision:  in.Decision,
		SHA:       commitSHA,
	}

	reviewer, err := c.reviewerStore.Find(ctx, pr.ID, session.Principal.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return fmt.Errorf("failed to fetch reviewer: %w", err)
	}

	if reviewer != nil && reviewer.ReviewDecision == review.Decision && reviewer.SHA == commitSHA {
		return nil
	}

	var reviewerAdded bool

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.reviewStore.Create(ctx, review); err != nil {
			return fmt.Errorf("failed to create review: %w", err)
		}

		if reviewer == nil {
			reviewer = &types.PullReqReviewer{
				PullReqID:      pr.ID,
				PrincipalID:    session.Principal.ID,
				CreatedBy:      session.Principal.ID,
				Created:        now,
				Updated:        now,
				RepoID:         pr.TargetRepoID,
				Type:           enum.PullReqReviewerTypeSelfAssigned,
				LatestReviewID: &review.ID,
				ReviewDecision: review.Decision,
				SHA:            commitSHA,
				Reviewer:       types.PrincipalInfo{},
				AddedBy:        types.PrincipalInfo{},
			}

			if err := c.reviewerStore.Create(ctx, reviewer); err != nil {
				return fmt.Errorf("failed to create reviewer: %w", err)
			}

			reviewerAdded = true
		} else {
			reviewer.LatestReviewID = &review.ID
			reviewer.ReviewDecision = review.Decision
			reviewer.SHA = commitSHA

			if err := c.reviewerStore.Update(ctx, reviewer); err != nil {
				return fmt.Errorf("failed to update reviewer: %w", err)
			}
		}

		deleteErr := c.reviewerSuggestionStore.Delete(ctx, pr.ID, session.Principal.ID)
		if deleteErr != nil && !errors.Is(deleteErr, store.ErrResourceNotFound) {
			return fmt.Errorf("failed to delete reviewer suggestion: %w", deleteErr)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to persist review: %w", err)
	}

	err = func(pr *types.PullReq) error {
		if pr, err = c.pullreqStore.UpdateActivitySeq(ctx, pr); err != nil {
			return fmt.Errorf("failed to increment pull request activity sequence: %w", err)
		}

		payload := &types.PullRequestActivityPayloadReviewSubmit{
			CommitSHA: commitSHA,
			Decision:  in.Decision,
		}

		_, err = c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload, nil)
		if err != nil {
			return fmt.Errorf("failed to create pull request activity: %w", err)
		}

		return nil
	}(pr)
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msg("failed to write pull request activity after review submit")
	}

	if reviewerAdded {
		c.reportReviewerAddition(ctx, session, pr, reviewer)
	}

	c.eventReporter.ReviewSubmitted(ctx, &events.ReviewSubmittedPayload{
		Base:       eventBase(pr, &session.Principal),
		Decision:   review.Decision,
		ReviewerID: review.CreatedBy,
	})

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeReviewPullRequest,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      repo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   repo.ID,
			instrument.PropertyRepositoryName: repo.Identifier,
			instrument.PropertyPullRequestID:  pr.Number,
			instrument.PropertyDecision:       in.Decision,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).
			Msg("failed to insert instrumentation record for review pull request operation")
	}

	return nil
}

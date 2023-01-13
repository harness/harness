// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ReviewSubmitInput struct {
	Decision enum.PullReqReviewDecision `json:"decision"`
	Message  string                     `json:"message"`
}

func (in *ReviewSubmitInput) Validate() error {
	decision, ok := in.Decision.Sanitize()
	if !ok || decision == enum.PullReqReviewDecisionPending {
		msg := fmt.Sprintf("Decision must be: %q, %q or %q.",
			enum.PullReqReviewDecisionApproved,
			enum.PullReqReviewDecisionChangeReq,
			enum.PullReqReviewDecisionReviewed)
		return usererror.BadRequest(msg)
	}

	in.Decision = decision
	in.Message = strings.TrimSpace(in.Message)

	return nil
}

// ReviewSubmit creates a new pull request review.
func (c *Controller) ReviewSubmit(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *ReviewSubmitInput,
) (*types.PullReqReview, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if pr.CreatedBy == session.Principal.ID {
		return nil, usererror.BadRequest("Can't submit review to own pull requests.")
	}

	ref, err := c.gitRPCClient.GetRef(ctx, &gitrpc.GetRefParams{
		ReadParams: gitrpc.ReadParams{RepoUID: repo.GitUID},
		Name:       pr.TargetBranch,
		Type:       gitrpc.RefTypeBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get git branch sha: %w", err)
	}

	if ref == nil || ref.SHA == "" {
		return nil, usererror.BadRequest("Failed to get branch SHA. Does the branch still exist?")
	}

	var review *types.PullReqReview

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		now := time.Now().UnixMilli()
		review = &types.PullReqReview{
			ID:        0,
			CreatedBy: session.Principal.ID,
			Created:   now,
			Updated:   now,
			PullReqID: pr.ID,
			Decision:  in.Decision,
			SHA:       ref.SHA,
		}

		err = c.reviewStore.Create(ctx, review)
		if err != nil {
			return err
		}

		_, err = c.updateReviewer(ctx, session, pr, review, ref.SHA)
		return err
	})
	if err != nil {
		return nil, err
	}

	act := getReviewSubmitActivity(session, pr, in)
	err = c.writeActivity(ctx, pr, act)
	if err != nil {
		// non-critical error
		log.Err(err).Msg("failed to generate pull request activity entry for submitting review")
	}

	return review, err
}

// updateReviewer updates pull request reviewer object.
func (c *Controller) updateReviewer(ctx context.Context, session *auth.Session,
	pr *types.PullReq, review *types.PullReqReview, sha string) (*types.PullReqReviewer, error) {
	reviewer, err := c.reviewerStore.Find(ctx, pr.ID, session.Principal.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, err
	}

	if reviewer != nil {
		reviewer.LatestReviewID = &review.ID
		reviewer.ReviewDecision = review.Decision
		reviewer.SHA = sha
		err = c.reviewerStore.Update(ctx, reviewer)
	} else {
		now := time.Now().UnixMilli()
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
			SHA:            sha,
			Reviewer:       types.PrincipalInfo{},
			AddedBy:        types.PrincipalInfo{},
		}
		err = c.reviewerStore.Create(ctx, reviewer)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create/update reviewer")
	}

	return reviewer, nil
}

func getReviewSubmitActivity(session *auth.Session, pr *types.PullReq, in *ReviewSubmitInput) *types.PullReqActivity {
	now := time.Now().UnixMilli()
	act := &types.PullReqActivity{
		ID:        0, // Will be populated in the data layer
		Version:   0,
		CreatedBy: session.Principal.ID,
		Created:   now,
		Updated:   now,
		Edited:    now,
		Deleted:   nil,
		ParentID:  nil, // Will be filled in CommentCreate
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     0, // Will be filled in writeActivity/writeReplyActivity
		SubOrder:  0, // Will be filled in writeReplyActivity
		ReplySeq:  0,
		Type:      enum.PullReqActivityTypeTitleChange,
		Kind:      enum.PullReqActivityKindSystem,
		Text:      "",
		Payload: map[string]interface{}{
			"Message":  in.Message,
			"Decision": in.Decision,
		},
		Metadata:   nil,
		ResolvedBy: nil,
		Resolved:   nil,
		Author:     *session.Principal.ToPrincipalInfo(),
	}

	return act
}

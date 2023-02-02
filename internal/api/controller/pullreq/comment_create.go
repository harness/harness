// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CommentCreateInput struct {
	ParentID int64                                    `json:"parent_id"`
	Text     string                                   `json:"text"`
	Payload  *types.PullRequestActivityPayloadComment `json:"payload"`
}

// CommentCreate creates a new pull request comment (pull request activity, type=comment).
func (c *Controller) CommentCreate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *CommentCreateInput,
) (*types.PullReqActivity, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	act := getCommentActivity(session, pr, in)

	if in.ParentID != 0 {
		var parentAct *types.PullReqActivity
		parentAct, err = c.checkIsReplyable(ctx, pr, in.ParentID)
		if err != nil {
			return nil, err
		}
		act.ParentID = &parentAct.ID
		err = c.writeReplyActivity(ctx, parentAct, act)
	} else {
		err = c.writeActivity(ctx, pr, act)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	_, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.CommentCount++
		return nil
	})
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msgf("failed to increment pull request comment counter")
	}

	return act, nil
}

func (c *Controller) checkIsReplyable(ctx context.Context,
	pr *types.PullReq, parentID int64) (*types.PullReqActivity, error) {
	// make sure the parent comment exists, belongs to the same PR and isn't itself a reply
	parentAct, err := c.activityStore.Find(ctx, parentID)
	if errors.Is(err, store.ErrResourceNotFound) || parentAct == nil {
		return nil, usererror.BadRequest("Parent pull request activity not found.")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find parent pull request activity: %w", err)
	}

	if parentAct.PullReqID != pr.ID || parentAct.RepoID != pr.TargetRepoID {
		return nil, usererror.BadRequest("Parent pull request activity doesn't belong to the same pull request.")
	}

	if !parentAct.IsReplyable() {
		return nil, usererror.BadRequest("Can't create a reply to the specified entry.")
	}

	return parentAct, nil
}

// writeActivity updates the PR's activity sequence number (using the optimistic locking mechanism),
// sets the correct Order value and writes the activity to the database.
// Even if the writing fails, the updating of the sequence number can succeed.
func (c *Controller) writeActivity(ctx context.Context, pr *types.PullReq, act *types.PullReqActivity) error {
	prUpd, err := c.pullreqStore.UpdateActivitySeq(ctx, pr)
	if err != nil {
		return fmt.Errorf("failed to get pull request activity number: %w", err)
	}

	*pr = *prUpd // update the pull request object

	act.Order = prUpd.ActivitySeq

	err = c.activityStore.Create(ctx, act)
	if err != nil {
		return fmt.Errorf("failed to create pull request activity: %w", err)
	}

	return nil
}

// writeReplyActivity updates the parent activity's reply sequence number (using the optimistic locking mechanism),
// sets the correct Order and SubOrder values and writes the activity to the database.
// Even if the writing fails, the updating of the sequence number can succeed.
func (c *Controller) writeReplyActivity(ctx context.Context, parent, act *types.PullReqActivity) error {
	parentUpd, err := c.activityStore.UpdateReplySeq(ctx, parent)
	if err != nil {
		return fmt.Errorf("failed to get pull request activity number: %w", err)
	}

	*parent = *parentUpd // update the parent pull request activity object

	act.Order = parentUpd.Order
	act.SubOrder = parentUpd.ReplySeq

	err = c.activityStore.Create(ctx, act)
	if err != nil {
		return fmt.Errorf("failed to create pull request activity: %w", err)
	}

	return nil
}

func getCommentActivity(session *auth.Session, pr *types.PullReq, in *CommentCreateInput) *types.PullReqActivity {
	now := time.Now().UnixMilli()
	act := &types.PullReqActivity{
		ID:         0, // Will be populated in the data layer
		Version:    0,
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
		Edited:     now,
		Deleted:    nil,
		ParentID:   nil, // Will be filled in CommentCreate
		RepoID:     pr.TargetRepoID,
		PullReqID:  pr.ID,
		Order:      0, // Will be filled in writeActivity/writeReplyActivity
		SubOrder:   0, // Will be filled in writeReplyActivity
		ReplySeq:   0,
		Type:       enum.PullReqActivityTypeComment,
		Kind:       enum.PullReqActivityKindComment,
		Text:       in.Text,
		Metadata:   nil,
		ResolvedBy: nil,
		Resolved:   nil,
		Author:     *session.Principal.ToPrincipalInfo(),
	}

	_ = act.SetPayload(in.Payload)

	return act
}

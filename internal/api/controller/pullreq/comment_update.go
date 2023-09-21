// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CommentUpdateInput struct {
	Text string `json:"text"`
}

func (in *CommentUpdateInput) Validate() error {
	// TODO: Check Text length
	return nil
}

func (in *CommentUpdateInput) hasChanges(act *types.PullReqActivity) bool {
	return in.Text != act.Text
}

// CommentUpdate updates a pull request comment.
func (c *Controller) CommentUpdate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	commentID int64,
	in *CommentUpdateInput,
) (*types.PullReqActivity, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if errValidate := in.Validate(); errValidate != nil {
		return nil, errValidate
	}

	act, err := c.getCommentCheckEditAccess(ctx, session, pr, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	if !in.hasChanges(act) {
		return act, nil
	}

	act, err = c.activityStore.UpdateOptLock(ctx, act, func(act *types.PullReqActivity) error {
		now := time.Now().UnixMilli()
		act.Edited = now
		act.Text = in.Text
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	if err = c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypePullrequesUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Msg("failed to publish PR changed event")
	}

	return act, nil
}

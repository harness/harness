// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CommentUpdateInput struct {
	Text    *string                                  `json:"text"`
	Payload *types.PullRequestActivityPayloadComment `json:"payload"`
}

func (in *CommentUpdateInput) hasChanges(act *types.PullReqActivity) (bool, error) {
	if in.Text != nil && *in.Text != act.Text {
		return true, nil
	}

	if in.Payload != nil {
		oldPayload, err := act.GetPayload()
		if errors.Is(err, types.ErrNoPayload) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("failed to get old payload: %w", err)
		}

		if !reflect.DeepEqual(oldPayload, in.Payload) {
			return true, nil
		}
	}

	return false, nil
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
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	act, err := c.getCommentCheckEditAccess(ctx, session, pr, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	hasChanges, err := in.hasChanges(act)
	if err != nil {
		return nil, fmt.Errorf("failed to verify if input has changes: %w", err)
	}
	if !hasChanges {
		return act, nil
	}

	now := time.Now().UnixMilli()
	act.Edited = now

	if in.Text != nil {
		act.Text = *in.Text
	}
	if in.Payload != nil {
		_ = act.SetPayload(in.Payload)
	}

	err = c.activityStore.Update(ctx, act)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return act, nil
}

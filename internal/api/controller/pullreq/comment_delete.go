// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// CommentDelete deletes a pull request comment.
func (c *Controller) CommentDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	commentID int64,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	act, err := c.getCommentCheckEditAccess(ctx, session, pr, commentID)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	if act.Deleted != nil {
		return nil
	}

	now := time.Now().UnixMilli()
	act.Deleted = &now

	err = c.pullreqActivityStore.Update(ctx, act)
	if err != nil {
		return fmt.Errorf("failed to mark comment as deleted: %w", err)
	}

	return nil
}

// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/api/controller"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CommentStatusInput struct {
	Status enum.PullReqCommentStatus `json:"status"`
}

func (in *CommentStatusInput) Validate() error {
	_, ok := in.Status.Sanitize()
	if !ok {
		return usererror.BadRequest("Invalid value provided for comment status")
	}

	return nil
}

func (in *CommentStatusInput) hasChanges(act *types.PullReqActivity, userID int64) bool {
	// clearing resolved
	if in.Status == enum.PullReqCommentStatusActive {
		return act.Resolved != nil
	}
	// setting resolved
	return act.Resolved == nil || act.ResolvedBy == nil || *act.ResolvedBy != userID
}

// CommentStatus updates a pull request comment status.
func (c *Controller) CommentStatus(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	commentID int64,
	in *CommentStatusInput,
) (*types.PullReqActivity, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	var act *types.PullReqActivity

	err = controller.TxOptLock(ctx, c.db, func(ctx context.Context) error {
		pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
		if err != nil {
			return fmt.Errorf("failed to find pull request by number: %w", err)
		}

		if errValidate := in.Validate(); errValidate != nil {
			return errValidate
		}

		act, err = c.getCommentCheckChangeStatusAccess(ctx, session, pr, commentID)
		if err != nil {
			return fmt.Errorf("failed to get comment: %w", err)
		}

		if !in.hasChanges(act, session.Principal.ID) {
			return nil
		}

		now := time.Now().UnixMilli()
		act.Edited = now

		act.Resolved = nil
		act.ResolvedBy = nil

		if in.Status != enum.PullReqCommentStatusActive {
			// In the future if we add more comment resolved statuses
			// we'll add the ResolvedReason field and put the reason there.
			// For now, the nullable timestamp field/db-column "Resolved" tells the status (active/resolved).
			act.Resolved = &now
			act.ResolvedBy = &session.Principal.ID
		}

		err = c.activityStore.Update(ctx, act)
		if err != nil {
			return fmt.Errorf("failed to update status of pull request activity: %w", err)
		}

		// Here we deliberately use the transaction and counting the unresolved comments,
		// rather than optimistic locking and incrementing/decrementing the counter.
		// The idea is that if the counter ever goes out of sync, this would be the place where we get it back in sync.
		unresolvedCount, err := c.activityStore.CountUnresolved(ctx, pr.ID)
		if err != nil {
			return fmt.Errorf("failed to count unresolved comments: %w", err)
		}

		pr.UnresolvedCount = unresolvedCount

		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request's unresolved comment count: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return act, nil
}

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
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// CommentDelete deletes a pull request comment.
func (c *Controller) CommentDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	commentID int64,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
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

	isBlocking := act.IsBlocking()

	_, err = c.activityStore.UpdateOptLock(ctx, act, func(act *types.PullReqActivity) error {
		now := time.Now().UnixMilli()
		act.Deleted = &now
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to mark comment as deleted: %w", err)
	}

	_, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.CommentCount--
		if isBlocking {
			pr.UnresolvedCount--
		}
		return nil
	})
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msgf("failed to decrement pull request comment counters")
	}

	if err = c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
	}

	return nil
}

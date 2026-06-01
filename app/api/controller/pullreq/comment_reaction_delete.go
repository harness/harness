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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CommentReactionDelete removes the current user's emoji reaction from a PR comment.
func (c *Controller) CommentReactionDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	commentID int64,
	emoji enum.PullReqCommentReactionEmoji,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	comment, err := c.activityStore.Find(ctx, commentID)
	if err != nil {
		return fmt.Errorf("failed to find comment: %w", err)
	}
	if comment == nil || comment.RepoID != pr.TargetRepoID || comment.PullReqID != pr.ID {
		return usererror.ErrNotFound
	}
	if err = validateCommentSupportsReactions(comment); err != nil {
		return err
	}

	principalID := session.Principal.ID
	emojiKey := string(emoji)

	_, err = c.activityStore.UpdateOptLock(ctx, comment, func(act *types.PullReqActivity) error {
		if act.Metadata == nil || act.Metadata.Reactions.IsEmpty() {
			return nil // idempotent: nothing to remove
		}
		ids := act.Metadata.Reactions.Counts[emojiKey]
		for i, id := range ids {
			if id != principalID {
				continue
			}
			act.Metadata.Reactions.Counts[emojiKey] = append(ids[:i], ids[i+1:]...)
			if len(act.Metadata.Reactions.Counts[emojiKey]) == 0 {
				delete(act.Metadata.Reactions.Counts, emojiKey)
			}
			if act.Metadata.Reactions.IsEmpty() {
				act.Metadata.Reactions = nil
			}
			return nil
		}
		return nil // idempotent: user hadn't reacted
	})
	if err != nil {
		return fmt.Errorf("failed to update comment reaction: %w", err)
	}

	return nil
}

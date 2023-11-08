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
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type CommentCreateInput struct {
	// ParentID is set only for replies
	ParentID int64 `json:"parent_id"`
	// Text is comment text
	Text string `json:"text"`
	// Used only for code comments
	TargetCommitSHA string `json:"target_commit_sha"`
	SourceCommitSHA string `json:"source_commit_sha"`
	Path            string `json:"path"`
	LineStart       int    `json:"line_start"`
	LineStartNew    bool   `json:"line_start_new"`
	LineEnd         int    `json:"line_end"`
	LineEndNew      bool   `json:"line_end_new"`
}

func (in *CommentCreateInput) IsReply() bool {
	return in.ParentID != 0
}

func (in *CommentCreateInput) IsCodeComment() bool {
	return in.SourceCommitSHA != ""
}

func (in *CommentCreateInput) Validate() error {
	// TODO: Validate Text size.

	if in.SourceCommitSHA == "" && in.TargetCommitSHA == "" {
		return nil // not a code comment
	}

	if in.SourceCommitSHA == "" || in.TargetCommitSHA == "" {
		return usererror.BadRequest("for code comments source commit SHA and target commit SHA must be provided")
	}

	if in.ParentID != 0 {
		return usererror.BadRequest("can't create a reply that is a code comment")
	}

	if in.Path == "" {
		return usererror.BadRequest("code comment requires file path")
	}

	if in.LineStart <= 0 || in.LineEnd <= 0 {
		return usererror.BadRequest("code comments require line numbers")
	}

	return nil
}

// CommentCreate creates a new pull request comment (pull request activity, type=comment/code-comment).
//
//nolint:gocognit,funlen // refactor if needed
func (c *Controller) CommentCreate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *CommentCreateInput,
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

	act := getCommentActivity(session, pr, in)

	switch {
	case in.IsCodeComment():
		var cut gitrpc.DiffCutOutput

		cut, err = c.gitRPCClient.DiffCut(ctx, &gitrpc.DiffCutParams{
			ReadParams:      gitrpc.ReadParams{RepoUID: repo.GitUID},
			SourceCommitSHA: in.SourceCommitSHA,
			SourceBranch:    pr.SourceBranch,
			TargetCommitSHA: in.TargetCommitSHA,
			TargetBranch:    pr.TargetBranch,
			Path:            in.Path,
			LineStart:       in.LineStart,
			LineStartNew:    in.LineStartNew,
			LineEnd:         in.LineEnd,
			LineEndNew:      in.LineEndNew,
		})
		if gitrpc.ErrorStatus(err) == gitrpc.StatusNotFound || gitrpc.ErrorStatus(err) == gitrpc.StatusPathNotFound {
			return nil, usererror.BadRequest(gitrpc.ErrorMessage(err))
		}
		if err != nil {
			return nil, err
		}

		setAsCodeComment(act, cut, in.Path, in.SourceCommitSHA)
		_ = act.SetPayload(&types.PullRequestActivityPayloadCodeComment{
			Title:        cut.LinesHeader,
			Lines:        cut.Lines,
			LineStartNew: in.LineStartNew,
			LineEndNew:   in.LineEndNew,
		})

		err = c.writeActivity(ctx, pr, act)

		// Migrate the comment if necessary... Note: we still need to return the code comment as is.
		needsNewLineMigrate := in.SourceCommitSHA != cut.LatestSourceSHA
		needsOldLineMigrate := pr.MergeBaseSHA != cut.MergeBaseSHA
		if err == nil && (needsNewLineMigrate || needsOldLineMigrate) {
			comments := []*types.CodeComment{act.AsCodeComment()}

			if needsNewLineMigrate {
				c.codeCommentMigrator.MigrateNew(ctx, repo.GitUID, cut.LatestSourceSHA, comments)
			}
			if needsOldLineMigrate {
				c.codeCommentMigrator.MigrateOld(ctx, repo.GitUID, cut.MergeBaseSHA, comments)
			}

			if errMigrateUpdate := c.codeCommentView.UpdateAll(ctx, comments); errMigrateUpdate != nil {
				// non-critical error
				log.Ctx(ctx).Err(errMigrateUpdate).
					Msgf("failed to migrate code comment to the latest source/merge-base commit SHA")
			}
		}
	case in.ParentID != 0:
		var parentAct *types.PullReqActivity
		parentAct, err = c.checkIsReplyable(ctx, pr, in.ParentID)
		if err != nil {
			return nil, err
		}

		act.ParentID = &parentAct.ID
		act.Kind = parentAct.Kind
		_ = act.SetPayload(types.PullRequestActivityPayloadComment{})

		err = c.writeReplyActivity(ctx, parentAct, act)
	default:
		_ = act.SetPayload(types.PullRequestActivityPayloadComment{})
		err = c.writeActivity(ctx, pr, act)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.CommentCount++
		if act.IsBlocking() {
			pr.UnresolvedCount++
		}
		return nil
	})
	if err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msgf("failed to increment pull request comment counters")
	}

	if err = c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypePullrequesUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Msg("failed to publish PR changed event")
	}
	// if it's a regular comment publish a comment create event
	if !act.IsReply() && act.Type == enum.PullReqActivityTypeComment && act.Kind == enum.PullReqActivityKindComment {
		c.eventReporter.CommentCreated(ctx, &events.CommentCreatedPayload{
			Base: events.Base{
				PullReqID:    pr.ID,
				SourceRepoID: pr.SourceRepoID,
				TargetRepoID: pr.TargetRepoID,
				PrincipalID:  session.Principal.ID,
				Number:       pr.Number,
			},
			ActivityID: act.ID,
			SourceSHA:  pr.SourceSHA,
		})
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
	parentUpd, err := c.activityStore.UpdateOptLock(ctx, parent, func(act *types.PullReqActivity) error {
		act.ReplySeq++
		return nil
	})
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

	return act
}

func setAsCodeComment(a *types.PullReqActivity, cut gitrpc.DiffCutOutput, path, sourceCommitSHA string) {
	var falseBool bool
	a.Type = enum.PullReqActivityTypeCodeComment
	a.Kind = enum.PullReqActivityKindChangeComment
	a.CodeComment = &types.CodeCommentFields{
		Outdated:     falseBool,
		MergeBaseSHA: cut.MergeBaseSHA,
		SourceSHA:    sourceCommitSHA,
		Path:         path,
		LineNew:      cut.Header.NewLine,
		SpanNew:      cut.Header.NewSpan,
		LineOld:      cut.Header.OldLine,
		SpanOld:      cut.Header.OldSpan,
	}
}

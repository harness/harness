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
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
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

func (in *CommentCreateInput) Sanitize() error {
	in.Text = strings.TrimSpace(in.Text)

	if err := validateComment(in.Text); err != nil {
		return err
	}

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

	if in.LineStartNew && !in.LineEndNew || !in.LineStartNew && in.LineEndNew {
		return usererror.BadRequest("code block must start and end on the same side")
	}

	return nil
}

// CommentCreate creates a new pull request comment (pull request activity, type=comment/code-comment).
//
//nolint:gocognit // refactor if needed
func (c *Controller) CommentCreate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *CommentCreateInput,
) (*types.PullReqActivity, error) {
	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	var pr *types.PullReq

	pr, err = c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	var parentAct *types.PullReqActivity
	if in.IsReply() {
		parentAct, err = c.checkIsReplyable(ctx, pr, in.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to verify reply: %w", err)
		}
	}

	// fetch code snippet from git for code comments
	var cut git.DiffCutOutput
	if in.IsCodeComment() {
		cut, err = c.fetchDiffCut(ctx, repo, in)
		if err != nil {
			return nil, err
		}
	}

	// generate all metadata updates
	var metadataUpdates []types.PullReqActivityMetadataUpdate

	metadataUpdates, principalInfos, err := c.appendMetadataUpdateForMentions(
		ctx, metadataUpdates, in.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to update metadata for mentions: %w", err)
	}

	// suggestion metadata in case of code comments or code comment replies (don't restrict to either side for now).
	if in.IsCodeComment() || (in.IsReply() && parentAct.IsValidCodeComment()) {
		metadataUpdates = appendMetadataUpdateForSuggestions(metadataUpdates, in.Text)
	}

	var act *types.PullReqActivity
	err = controller.TxOptLock(ctx, c.tx, func(ctx context.Context) error {
		var err error

		if pr == nil {
			// the pull request was fetched before the transaction, we re-fetch it in case of the version conflict error
			pr, err = c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
			if err != nil {
				return fmt.Errorf("failed to find pull request by number: %w", err)
			}
		}

		act = getCommentActivity(session, pr, in, metadataUpdates)

		// In the switch the pull request activity (the code comment)
		// is written to the DB (as code comment, a reply, or ordinary comment).
		switch {
		case in.IsCodeComment():
			setAsCodeComment(act, cut, in.Path, in.SourceCommitSHA)
			_ = act.SetPayload(&types.PullRequestActivityPayloadCodeComment{
				Title:        cut.LinesHeader,
				Lines:        cut.Lines,
				LineStartNew: in.LineStartNew,
				LineEndNew:   in.LineEndNew,
			})

			err = c.writeActivity(ctx, pr, act)

		case in.IsReply():
			act.ParentID = &parentAct.ID
			act.Kind = parentAct.Kind
			_ = act.SetPayload(types.PullRequestActivityPayloadComment{})

			err = c.writeReplyActivity(ctx, parentAct, act)
		default: // top level comment
			_ = act.SetPayload(types.PullRequestActivityPayloadComment{})
			err = c.writeActivity(ctx, pr, act)
		}
		if err != nil {
			return fmt.Errorf("failed to write pull request comment: %w", err)
		}

		pr.CommentCount++
		if act.IsBlocking() {
			pr.UnresolvedCount++
		}

		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to increment pull request comment counters: %w", err)
		}

		return nil
	}, controller.TxOptionResetFunc(func() {
		pr = nil // on the version conflict error force re-fetch of the pull request
	}))
	if err != nil {
		return nil, err
	}

	// Populate activity mentions (used only for response purposes).
	act.Mentions = principalInfos

	if in.IsCodeComment() {
		// Migrate the comment if necessary... Note: we still need to return the code comment as is.
		c.migrateCodeComment(ctx, repo, pr, in, act.AsCodeComment(), cut)
	}

	if err = c.sseStreamer.Publish(ctx, repo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
	}

	// if it's a regular comment publish a comment create event
	if act.Type == enum.PullReqActivityTypeComment && act.Kind == enum.PullReqActivityKindComment {
		c.reportCommentCreated(ctx, pr, session.Principal.ID, act.ID, act.IsReply())
	}

	return act, nil
}

func (c *Controller) checkIsReplyable(
	ctx context.Context,
	pr *types.PullReq,
	parentID int64,
) (*types.PullReqActivity, error) {
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

func getCommentActivity(
	session *auth.Session,
	pr *types.PullReq,
	in *CommentCreateInput,
	metadataUpdates []types.PullReqActivityMetadataUpdate,
) *types.PullReqActivity {
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

	act.UpdateMetadata(metadataUpdates...)

	return act
}

func setAsCodeComment(a *types.PullReqActivity, cut git.DiffCutOutput, path, sourceCommitSHA string) {
	var falseBool bool
	a.Type = enum.PullReqActivityTypeCodeComment
	a.Kind = enum.PullReqActivityKindChangeComment
	a.CodeComment = &types.CodeCommentFields{
		Outdated:     falseBool,
		MergeBaseSHA: cut.MergeBaseSHA.String(),
		SourceSHA:    sourceCommitSHA,
		Path:         path,
		LineNew:      cut.Header.NewLine,
		SpanNew:      cut.Header.NewSpan,
		LineOld:      cut.Header.OldLine,
		SpanOld:      cut.Header.OldSpan,
	}
}

func (c *Controller) fetchDiffCut(
	ctx context.Context,
	repo *types.Repository,
	in *CommentCreateInput,
) (git.DiffCutOutput, error) {
	// maxDiffLineCount restricts the total length of a code comment diff to 1000 lines.
	// TODO: This can still lead to wrong code comments in cases like a large file being replaced by one line.
	const maxDiffLineCount = 1000

	cut, err := c.git.DiffCut(ctx, &git.DiffCutParams{
		ReadParams:      git.ReadParams{RepoUID: repo.GitUID},
		SourceCommitSHA: in.SourceCommitSHA,
		TargetCommitSHA: in.TargetCommitSHA,
		Path:            in.Path,
		LineStart:       in.LineStart,
		LineStartNew:    in.LineStartNew,
		LineEnd:         in.LineEnd,
		LineEndNew:      in.LineEndNew,
		LineLimit:       maxDiffLineCount,
	})
	if errors.AsStatus(err) == errors.StatusNotFound {
		return git.DiffCutOutput{}, usererror.BadRequest(errors.Message(err))
	}
	if err != nil {
		return git.DiffCutOutput{}, fmt.Errorf("failed to fetch git diff cut: %w", err)
	}

	return cut, nil
}

func (c *Controller) migrateCodeComment(
	ctx context.Context,
	repo *types.Repository,
	pr *types.PullReq,
	in *CommentCreateInput,
	cc *types.CodeComment,
	cut git.DiffCutOutput,
) {
	needsNewLineMigrate := in.SourceCommitSHA != pr.SourceSHA
	needsOldLineMigrate := cut.MergeBaseSHA.String() != pr.MergeBaseSHA
	if !needsNewLineMigrate && !needsOldLineMigrate {
		return
	}

	comments := []*types.CodeComment{cc}

	if needsNewLineMigrate {
		c.codeCommentMigrator.MigrateNew(ctx, repo.GitUID, pr.SourceSHA, comments)
	}
	if needsOldLineMigrate {
		c.codeCommentMigrator.MigrateOld(ctx, repo.GitUID, pr.MergeBaseSHA, comments)
	}

	if errMigrateUpdate := c.codeCommentView.UpdateAll(ctx, comments); errMigrateUpdate != nil {
		// non-critical error
		log.Ctx(ctx).Err(errMigrateUpdate).
			Msgf("failed to migrate code comment to the latest source/merge-base commit SHA")
	}
}

func (c *Controller) reportCommentCreated(
	ctx context.Context,
	pr *types.PullReq,
	principalID int64,
	actID int64,
	isReply bool,
) {
	c.eventReporter.CommentCreated(ctx, &events.CommentCreatedPayload{
		Base: events.Base{
			PullReqID:    pr.ID,
			SourceRepoID: pr.SourceRepoID,
			TargetRepoID: pr.TargetRepoID,
			PrincipalID:  principalID,
			Number:       pr.Number,
		},
		ActivityID: actID,
		SourceSHA:  pr.SourceSHA,
		IsReply:    isReply,
	})
}

func appendMetadataUpdateForSuggestions(
	updates []types.PullReqActivityMetadataUpdate,
	comment string,
) []types.PullReqActivityMetadataUpdate {
	suggestions := parseSuggestions(comment)
	return append(
		updates,
		types.WithPullReqActivitySuggestionsMetadataUpdate(
			func(m *types.PullReqActivitySuggestionsMetadata) {
				m.CheckSums = make([]string, len(suggestions))
				for i := range suggestions {
					m.CheckSums[i] = suggestions[i].checkSum
				}
			}),
	)
}

func (c *Controller) appendMetadataUpdateForMentions(
	ctx context.Context,
	updates []types.PullReqActivityMetadataUpdate,
	comment string,
) ([]types.PullReqActivityMetadataUpdate, map[int64]*types.PrincipalInfo, error) {
	principalInfos, err := c.processMentions(ctx, comment)
	if err != nil {
		return nil, map[int64]*types.PrincipalInfo{}, err
	}

	ids := make([]int64, len(principalInfos))
	i := 0
	for id := range principalInfos {
		ids[i] = id
		i++
	}

	return append(
		updates,
		types.WithPullReqActivityMentionsMetadataUpdate(
			func(m *types.PullReqActivityMentionsMetadata) {
				m.IDs = ids
			}),
	), principalInfos, nil
}

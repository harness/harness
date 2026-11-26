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
	"net/http"
	"strings"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const cprCommitA = "1111111111111111111111111111111111111111"
const cprCommitB = "3333333333333333333333333333333333333333"

func TestCprCommentCreateInput_Sanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      CommentCreateInput
		wantErr bool
	}{
		{name: "plain comment", in: CommentCreateInput{Text: "  hello  "}},
		{name: "too long", in: CommentCreateInput{Text: strings.Repeat("a", 16<<10+1)}, wantErr: true},
		{name: "only source sha", in: CommentCreateInput{Text: "x", SourceCommitSHA: cprCommitA}, wantErr: true},
		{name: "only target sha", in: CommentCreateInput{Text: "x", TargetCommitSHA: cprCommitA}, wantErr: true},
		{
			name: "code reply",
			in: CommentCreateInput{
				Text: "x", ParentID: 1, SourceCommitSHA: cprCommitA, TargetCommitSHA: cprCommitB,
			},
			wantErr: true,
		},
		{
			name: "code comment without path",
			in: CommentCreateInput{
				Text: "x", SourceCommitSHA: cprCommitA, TargetCommitSHA: cprCommitB, LineStart: 1, LineEnd: 1,
			},
			wantErr: true,
		},
		{
			name: "code comment without lines",
			in: CommentCreateInput{
				Text: "x", SourceCommitSHA: cprCommitA, TargetCommitSHA: cprCommitB, Path: "a.go",
			},
			wantErr: true,
		},
		{
			name: "code comment mixed sides",
			in: CommentCreateInput{
				Text: "x", SourceCommitSHA: cprCommitA, TargetCommitSHA: cprCommitB, Path: "a.go",
				LineStart: 1, LineEnd: 2, LineStartNew: true,
			},
			wantErr: true,
		},
		{
			name: "valid code comment",
			in: CommentCreateInput{
				Text: "x", SourceCommitSHA: cprCommitA, TargetCommitSHA: cprCommitB, Path: "a.go",
				LineStart: 1, LineEnd: 2, LineStartNew: true, LineEndNew: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.in.Sanitize()
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, http.StatusBadRequest, cprStatus(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCprCommentCreateInput_Kinds(t *testing.T) {
	t.Parallel()

	require.False(t, (&CommentCreateInput{}).IsReply())
	require.True(t, (&CommentCreateInput{ParentID: 3}).IsReply())
	require.False(t, (&CommentCreateInput{}).IsCodeComment())
	require.True(t, (&CommentCreateInput{SourceCommitSHA: cprCommitA}).IsCodeComment())
}

func TestCprCommentCreate(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)

	t.Run("top level comment with mention", func(t *testing.T) {
		act, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "hi @[1] there"}, nil)
		require.NoError(t, err)
		require.NotZero(t, act.ID)
		require.Equal(t, enum.PullReqActivityTypeComment, act.Type)
		require.Equal(t, e.reviewer.ID, act.CreatedBy)
		require.Contains(t, act.Mentions, e.author.ID)
		require.NotEmpty(t, e.streamer.published)

		stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Equal(t, 1, stored.CommentCount)
		require.Equal(t, 1, stored.UnresolvedCount)
	})

	t.Run("reply", func(t *testing.T) {
		parent := e.createComment(pr, e.author, "parent")
		reply, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "reply", ParentID: parent.ID}, nil)
		require.NoError(t, err)
		require.NotNil(t, reply.ParentID)
		require.Equal(t, parent.ID, *reply.ParentID)
		require.EqualValues(t, 1, reply.SubOrder)
	})

	t.Run("mutate function error", func(t *testing.T) {
		wantErr := errors.New("mutate failed")
		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "x"},
			func(*types.PullReqActivity) error { return wantErr })
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("mutate function applied", func(t *testing.T) {
		act, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "x"},
			func(a *types.PullReqActivity) error { a.Text = "mutated"; return nil })
		require.NoError(t, err)
		require.Equal(t, "mutated", act.Text)
	})

	t.Run("reply to unknown parent", func(t *testing.T) {
		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "x", ParentID: 99999}, nil)
		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reply to parent of another pull request", func(t *testing.T) {
		other := e.createPR()
		parent := e.createComment(other, e.author, "foreign")
		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "x", ParentID: parent.ID}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reply to reply", func(t *testing.T) {
		parent := e.createComment(pr, e.author, "p")
		child := e.createComment(pr, e.author, "c", func(a *types.PullReqActivity) {
			a.ParentID = &parent.ID
			a.SubOrder = 1
		})
		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: "x", ParentID: child.ID}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), 9999,
			&CommentCreateInput{Text: "x"}, nil)
		require.Error(t, err)
	})

	t.Run("invalid input", func(t *testing.T) {
		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number,
			&CommentCreateInput{Text: strings.Repeat("a", 16<<10+1)}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("empty repo ref", func(t *testing.T) {
		_, err := e.ctrl.CommentCreate(e.ctx, session, "", pr.Number, &CommentCreateInput{Text: "x"}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

func TestCprCommentCreate_PermissionDenied(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	e.ctrl.authorizer = cprDenyAuthorizer()

	_, err := e.ctrl.CommentCreate(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
		&CommentCreateInput{Text: "x"}, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, apiauth.ErrForbidden)
}

func TestCprCommentCreate_CodeComment(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)

	in := func() *CommentCreateInput {
		return &CommentCreateInput{
			Text:            "```suggestion\nfoo\n```",
			SourceCommitSHA: pr.SourceSHA,
			TargetCommitSHA: pr.MergeBaseSHA,
			Path:            "a.go",
			LineStart:       1,
			LineEnd:         1,
			LineStartNew:    true,
			LineEndNew:      true,
		}
	}

	t.Run("success", func(t *testing.T) {
		e.git.On("DiffCut", mock.Anything, mock.Anything).Return(git.DiffCutOutput{
			Header:       git.HunkHeader{NewLine: 1, NewSpan: 1},
			LinesHeader:  "@@ -1 +1 @@",
			Lines:        []string{"+foo"},
			MergeBaseSHA: sha.Must(pr.MergeBaseSHA),
		}, nil).Once()

		act, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number, in(), nil)
		require.NoError(t, err)
		require.Equal(t, enum.PullReqActivityTypeCodeComment, act.Type)
		require.Equal(t, enum.PullReqActivityKindChangeComment, act.Kind)
		require.NotNil(t, act.CodeComment)
		require.Equal(t, "a.go", act.CodeComment.Path)
	})

	t.Run("diff cut not found", func(t *testing.T) {
		e.git.On("DiffCut", mock.Anything, mock.Anything).
			Return(git.DiffCutOutput{}, errors.NotFound("no such path")).Once()

		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number, in(), nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("diff cut internal error", func(t *testing.T) {
		e.git.On("DiffCut", mock.Anything, mock.Anything).
			Return(git.DiffCutOutput{}, errors.New("boom")).Once()

		_, err := e.ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number, in(), nil)
		require.Error(t, err)
		require.Zero(t, cprStatus(err))
	})
}

func TestCprCommentUpdate(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)
	comment := e.createComment(pr, e.reviewer, "original")

	t.Run("success", func(t *testing.T) {
		act, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentUpdateInput{Text: "  updated @[1] "}, nil)
		require.NoError(t, err)
		require.Equal(t, "updated @[1]", act.Text)
		require.Contains(t, act.Mentions, e.author.ID)
	})

	t.Run("no changes", func(t *testing.T) {
		act, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentUpdateInput{Text: "updated @[1]"}, nil)
		require.NoError(t, err)
		require.Equal(t, "updated @[1]", act.Text)
	})

	t.Run("mutate function error", func(t *testing.T) {
		wantErr := errors.New("mutate failed")
		_, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentUpdateInput{Text: "again"}, func(*types.PullReqActivity) error { return wantErr })
		require.ErrorIs(t, err, wantErr)
	})

	t.Run("reply update", func(t *testing.T) {
		reply := e.createComment(pr, e.reviewer, "reply", func(a *types.PullReqActivity) {
			a.ParentID = &comment.ID
			a.SubOrder = 1
		})
		act, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, reply.ID,
			&CommentUpdateInput{Text: "edited reply"}, nil)
		require.NoError(t, err)
		require.Equal(t, "edited reply", act.Text)
	})

	t.Run("not own comment", func(t *testing.T) {
		_, err := e.ctrl.CommentUpdate(e.ctx, e.session(e.other), e.repoRef(), pr.Number, comment.ID,
			&CommentUpdateInput{Text: "nope"}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("invalid comment id", func(t *testing.T) {
		_, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, 0,
			&CommentUpdateInput{Text: "x"}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("too long", func(t *testing.T) {
		_, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentUpdateInput{Text: strings.Repeat("a", 16<<10+1)}, nil)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		_, err := e.ctrl.CommentUpdate(e.ctx, session, e.repoRef(), 4242, comment.ID,
			&CommentUpdateInput{Text: "x"}, nil)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.CommentUpdate(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentUpdateInput{Text: "x"}, nil)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprCommentUpdateInput_HasChanges(t *testing.T) {
	t.Parallel()

	in := &CommentUpdateInput{Text: "a"}
	require.False(t, in.hasChanges(&types.PullReqActivity{Text: "a"}))
	require.True(t, in.hasChanges(&types.PullReqActivity{Text: "b"}))
}

func TestCprCommentDelete(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)

	t.Run("deletes and decrements counters", func(t *testing.T) {
		comment := e.createComment(pr, e.reviewer, "to delete")
		stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
		require.NoError(t, err)
		stored.CommentCount = 1
		stored.UnresolvedCount = 1
		require.NoError(t, e.pullreqStore.Update(e.ctx, stored))

		require.NoError(t, e.ctrl.CommentDelete(e.ctx, session, e.repoRef(), pr.Number, comment.ID))

		got, err := e.activityStore.Find(e.ctx, comment.ID)
		require.NoError(t, err)
		require.NotNil(t, got.Deleted)

		stored, err = e.pullreqStore.Find(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Equal(t, 0, stored.CommentCount)
		require.Equal(t, 0, stored.UnresolvedCount)

		// Deleted comments can't be fetched for edit again.
		err = e.ctrl.CommentDelete(e.ctx, session, e.repoRef(), pr.Number, comment.ID)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("not own comment", func(t *testing.T) {
		comment := e.createComment(pr, e.author, "foreign")
		err := e.ctrl.CommentDelete(e.ctx, session, e.repoRef(), pr.Number, comment.ID)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		require.Error(t, e.ctrl.CommentDelete(e.ctx, session, e.repoRef(), 777, 1))
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		err := ctrl.CommentDelete(e.ctx, session, e.repoRef(), pr.Number, 1)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprCommentStatusInput_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&CommentStatusInput{Status: enum.PullReqCommentStatusResolved}).Validate())
	require.NoError(t, (&CommentStatusInput{Status: enum.PullReqCommentStatusActive}).Validate())
	err := (&CommentStatusInput{Status: "bogus"}).Validate()
	require.Equal(t, http.StatusBadRequest, cprStatus(err))
}

func TestCprCommentStatusInput_HasChanges(t *testing.T) {
	t.Parallel()

	now := int64(10)
	user := int64(5)
	otherUser := int64(6)

	tests := []struct {
		name   string
		status enum.PullReqCommentStatus
		act    types.PullReqActivity
		want   bool
	}{
		{"activate unresolved", enum.PullReqCommentStatusActive, types.PullReqActivity{}, false},
		{"activate resolved", enum.PullReqCommentStatusActive, types.PullReqActivity{Resolved: &now}, true},
		{"resolve unresolved", enum.PullReqCommentStatusResolved, types.PullReqActivity{}, true},
		{
			"resolve resolved by same user", enum.PullReqCommentStatusResolved,
			types.PullReqActivity{Resolved: &now, ResolvedBy: &user}, false,
		},
		{
			"resolve resolved by other user", enum.PullReqCommentStatusResolved,
			types.PullReqActivity{Resolved: &now, ResolvedBy: &otherUser}, true,
		},
		{
			"resolve resolved without resolver", enum.PullReqCommentStatusResolved,
			types.PullReqActivity{Resolved: &now}, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &CommentStatusInput{Status: tt.status}
			require.Equal(t, tt.want, in.hasChanges(&tt.act, user))
		})
	}
}

func TestCprCommentStatus(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)
	comment := e.createComment(pr, e.author, "needs resolving")

	t.Run("resolve then reactivate", func(t *testing.T) {
		act, err := e.ctrl.CommentStatus(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentStatusInput{Status: enum.PullReqCommentStatusResolved})
		require.NoError(t, err)
		require.NotNil(t, act.Resolved)
		require.NotNil(t, act.ResolvedBy)
		require.Equal(t, e.reviewer.ID, *act.ResolvedBy)

		// Resolving again is a no-op.
		_, err = e.ctrl.CommentStatus(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentStatusInput{Status: enum.PullReqCommentStatusResolved})
		require.NoError(t, err)

		act, err = e.ctrl.CommentStatus(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentStatusInput{Status: enum.PullReqCommentStatusActive})
		require.NoError(t, err)
		require.Nil(t, act.Resolved)
		require.Nil(t, act.ResolvedBy)

		stored, err := e.pullreqStore.Find(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Equal(t, 1, stored.UnresolvedCount)
	})

	t.Run("invalid status", func(t *testing.T) {
		_, err := e.ctrl.CommentStatus(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentStatusInput{Status: "bogus"})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("status of a reply is rejected", func(t *testing.T) {
		reply := e.createComment(pr, e.author, "reply", func(a *types.PullReqActivity) {
			a.ParentID = &comment.ID
			a.SubOrder = 1
		})
		_, err := e.ctrl.CommentStatus(e.ctx, session, e.repoRef(), pr.Number, reply.ID,
			&CommentStatusInput{Status: enum.PullReqCommentStatusResolved})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, err := e.ctrl.CommentStatus(e.ctx, session, e.repoRef(), 5555, comment.ID,
			&CommentStatusInput{Status: enum.PullReqCommentStatusResolved})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.CommentStatus(e.ctx, session, e.repoRef(), pr.Number, comment.ID,
			&CommentStatusInput{Status: enum.PullReqCommentStatusResolved})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprCommentReactions(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	comment := e.createComment(pr, e.author, "react to me")
	emoji := enum.PullReqCommentReactionEmojiHeart

	t.Run("create is idempotent and delete removes", func(t *testing.T) {
		for range 2 {
			reaction, err := e.ctrl.CommentReactionCreate(
				e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, comment.ID, emoji)
			require.NoError(t, err)
			require.Equal(t, string(emoji), reaction.Emoji)
			require.Equal(t, e.reviewer.ID, reaction.Author.ID)
		}

		got, err := e.activityStore.Find(e.ctx, comment.ID)
		require.NoError(t, err)
		require.NotNil(t, got.Metadata)
		require.Equal(t, []int64{e.reviewer.ID}, got.Metadata.Reactions.Counts[string(emoji)])

		// Another user reacting keeps both entries.
		_, err = e.ctrl.CommentReactionCreate(
			e.ctx, e.session(e.other), e.repoRef(), pr.Number, comment.ID, emoji)
		require.NoError(t, err)

		require.NoError(t, e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, comment.ID, emoji))
		got, err = e.activityStore.Find(e.ctx, comment.ID)
		require.NoError(t, err)
		require.Equal(t, []int64{e.other.ID}, got.Metadata.Reactions.Counts[string(emoji)])

		// Removing a reaction that does not exist is a no-op.
		require.NoError(t, e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, comment.ID, emoji))

		require.NoError(t, e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.other), e.repoRef(), pr.Number, comment.ID, emoji))
		got, err = e.activityStore.Find(e.ctx, comment.ID)
		require.NoError(t, err)
		require.True(t, got.Metadata == nil || got.Metadata.Reactions.IsEmpty())

		// Nothing left to remove at all.
		require.NoError(t, e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.other), e.repoRef(), pr.Number, comment.ID, emoji))
	})

	t.Run("system activity does not support reactions", func(t *testing.T) {
		sys := e.createComment(pr, e.author, "sys", func(a *types.PullReqActivity) {
			a.Kind = enum.PullReqActivityKindSystem
			a.Type = enum.PullReqActivityTypeBranchUpdate
		})
		_, err := e.ctrl.CommentReactionCreate(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, sys.ID, emoji)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
		err = e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, sys.ID, emoji)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("comment of another pull request", func(t *testing.T) {
		other := e.createPR()
		foreign := e.createComment(other, e.author, "foreign")
		_, err := e.ctrl.CommentReactionCreate(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, foreign.ID, emoji)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
		err = e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, foreign.ID, emoji)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("deleted comment cannot be reacted to", func(t *testing.T) {
		deleted := e.createComment(pr, e.author, "gone")
		now := int64(1)
		deleted.Deleted = &now
		require.NoError(t, e.activityStore.Update(e.ctx, deleted))
		_, err := e.ctrl.CommentReactionCreate(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, deleted.ID, emoji)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("unknown comment and pull request", func(t *testing.T) {
		_, err := e.ctrl.CommentReactionCreate(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, 987654, emoji)
		require.Error(t, err)
		err = e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, 987654, emoji)
		require.Error(t, err)
		_, err = e.ctrl.CommentReactionCreate(
			e.ctx, e.session(e.reviewer), e.repoRef(), 8888, comment.ID, emoji)
		require.Error(t, err)
		err = e.ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), 8888, comment.ID, emoji)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.CommentReactionCreate(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, comment.ID, emoji)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		err = ctrl.CommentReactionDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, comment.ID, emoji)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

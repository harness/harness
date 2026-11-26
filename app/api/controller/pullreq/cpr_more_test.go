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
	"io"
	"net/http"
	"strings"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// cprResolver resolves user groups from a fixed map.
type cprResolver struct {
	groups map[string]*types.UserGroup
}

func (r cprResolver) Resolve(_ context.Context, identifier string) (*types.UserGroup, error) {
	group, ok := r.groups[identifier]
	if !ok {
		return nil, usergroup.ErrNotFound
	}
	return group, nil
}

// cprAnything returns n mock.Anything matchers, used for variadic mock calls.
func cprAnything(n int) []interface{} {
	out := make([]interface{}, n)
	for i := range out {
		out[i] = mock.Anything
	}
	return out
}

func TestCprGetHighestOrderDecision(t *testing.T) {
	t.Parallel()

	const (
		pending = enum.PullReqReviewDecisionPending
		review  = enum.PullReqReviewDecisionReviewed
		approve = enum.PullReqReviewDecisionApproved
		changes = enum.PullReqReviewDecisionChangeReq
	)

	tests := []struct {
		d1, d2, want enum.PullReqReviewDecision
	}{
		{pending, pending, pending},
		{pending, review, review},
		{review, pending, review},
		{review, approve, approve},
		{approve, review, approve},
		{approve, changes, changes},
		{changes, approve, changes},
		{pending, changes, changes},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, getHighestOrderDecision(tt.d1, tt.d2), "%s vs %s", tt.d1, tt.d2)
	}
}

func TestCprGetCommentForPR(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	other := e.createPR()

	valid := e.createComment(pr, e.author, "valid")

	t.Run("valid", func(t *testing.T) {
		got, err := e.ctrl.getCommentForPR(e.ctx, pr, valid.ID)
		require.NoError(t, err)
		require.Equal(t, valid.ID, got.ID)
	})

	t.Run("invalid ids", func(t *testing.T) {
		for _, id := range []int64{0, -1} {
			_, err := e.ctrl.getCommentForPR(e.ctx, pr, id)
			require.Equal(t, http.StatusBadRequest, cprStatus(err))
		}
		_, err := e.ctrl.getCommentForPR(e.ctx, pr, 987654)
		require.Error(t, err)
	})

	t.Run("comment of another pull request", func(t *testing.T) {
		foreign := e.createComment(other, e.author, "foreign")
		_, err := e.ctrl.getCommentForPR(e.ctx, pr, foreign.ID)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("deleted comment", func(t *testing.T) {
		deleted := e.createComment(pr, e.author, "deleted")
		now := int64(1)
		deleted.Deleted = &now
		require.NoError(t, e.activityStore.Update(e.ctx, deleted))
		_, err := e.ctrl.getCommentForPR(e.ctx, pr, deleted.ID)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("system comment", func(t *testing.T) {
		sys := e.createComment(pr, e.author, "sys", func(a *types.PullReqActivity) {
			a.Kind = enum.PullReqActivityKindSystem
		})
		_, err := e.ctrl.getCommentForPR(e.ctx, pr, sys.ID)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("not a comment type", func(t *testing.T) {
		act := e.createComment(pr, e.author, "act", func(a *types.PullReqActivity) {
			a.Type = enum.PullReqActivityTypeTitleChange
		})
		_, err := e.ctrl.getCommentForPR(e.ctx, pr, act.ID)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

func TestCprReviewersListCombined(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionApproved)
	group := e.createUserGroup("cpr_combined")
	_, err := e.ctrl.UserGroupReviewerAdd(e.ctx, session, e.repoRef(), pr.Number,
		&UserGroupReviewerAddInput{UserGroupID: group.ID})
	require.NoError(t, err)

	got, err := e.ctrl.ReviewersListCombined(e.ctx, session, e.repoRef(), pr.Number)
	require.NoError(t, err)
	require.Len(t, got.Reviewers, 1)
	require.Len(t, got.UserGroupReviewers, 1)
	require.Equal(t, group.ID, got.UserGroupReviewers[0].UserGroupID)

	t.Run("errors", func(t *testing.T) {
		_, err := e.ctrl.ReviewersListCombined(e.ctx, session, e.repoRef(), 4242)
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.ReviewersListCombined(e.ctx, session, e.repoRef(), pr.Number)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprPullReqViewCreate(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)

	mockDiff := func(files ...string) {
		ch := make(chan *git.FileDiff, len(files))
		for _, f := range files {
			ch <- &git.FileDiff{Path: f, OldSHA: "old-" + f, SHA: "new-" + f}
		}
		close(ch)
		e.git.On("Diff", cprAnything(8)...).
			Return((<-chan *git.FileDiff)(ch), (<-chan error)(make(chan error))).Once()
	}

	in := func() *PullReqViewCreateInput {
		return &PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{
			{
				Title: "Backend", Description: "api", Tags: map[string]string{"risk": "high"},
				Files: []string{"a.go", "b.go", "a.go"},
			},
			{Title: "Docs", Files: []string{"README.md"}},
		}}
	}

	t.Run("success replaces all groups", func(t *testing.T) {
		mockDiff("a.go", "b.go", "README.md")
		require.NoError(t, e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, in()))

		groups, err := e.ctrl.fileGroupStore.List(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Len(t, groups, 2)

		mockDiff("a.go")
		single := &PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{{Title: "Only", Files: []string{"a.go"}}}}
		require.NoError(t, e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, single))
		groups, err = e.ctrl.fileGroupStore.List(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Len(t, groups, 1)
		require.Len(t, groups[0].Files, 1)
		require.Equal(t, "new-a.go", groups[0].Files[0].NewSHA)
	})

	t.Run("explicit SHAs skip PR lookups", func(t *testing.T) {
		mockDiff("a.go")
		explicit := &PullReqViewCreateInput{
			SourceSHA:    sha.Must(cprSourceSHA),
			MergeBaseSHA: sha.Must(cprBaseSHA),
			Groups:       []PullReqViewCreateInputGroup{{Title: "Only", Files: []string{"a.go"}}},
		}
		require.NoError(t, e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, explicit))
	})

	t.Run("file outside of the diff", func(t *testing.T) {
		mockDiff("a.go")
		err := e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, &PullReqViewCreateInput{
			Groups: []PullReqViewCreateInputGroup{{Title: "Missing", Files: []string{"a.go", "zzz.go"}}},
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("empty file path", func(t *testing.T) {
		err := e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, &PullReqViewCreateInput{
			Groups: []PullReqViewCreateInputGroup{{Title: "Empty", Files: []string{""}}},
		})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("diff stream error", func(t *testing.T) {
		ch := make(chan *git.FileDiff)
		errCh := make(chan error, 1)
		errCh <- io.ErrUnexpectedEOF
		close(ch)
		e.git.On("Diff", cprAnything(8)...).
			Return((<-chan *git.FileDiff)(ch), (<-chan error)(errCh)).Once()
		err := e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, &PullReqViewCreateInput{
			Groups: []PullReqViewCreateInputGroup{{Title: "Broken", Files: []string{"a.go"}}},
		})
		require.Error(t, err)
	})

	t.Run("no groups clears everything", func(t *testing.T) {
		require.NoError(t, e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, &PullReqViewCreateInput{}))
		groups, err := e.ctrl.fileGroupStore.List(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Empty(t, groups)
	})

	t.Run("SHAs resolved from git when the pull request has none", func(t *testing.T) {
		empty := e.createPR(func(p *types.PullReq) { p.SourceSHA = ""; p.MergeBaseSHA = "" })
		cprMockBranches(e)
		e.git.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
			return p.Type == gitenum.RefTypePullReqHead
		})).Return(git.GetRefResponse{SHA: sha.Must(cprSourceSHA)}, nil).Maybe()
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(cprBaseSHA)}, nil).Once()
		mockDiff("a.go")
		require.NoError(t, e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), empty.Number,
			&PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{{Title: "G", Files: []string{"a.go"}}}}))
	})

	t.Run("errors", func(t *testing.T) {
		err := e.ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), 4242, in())
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		err = ctrl.PullReqViewCreate(e.ctx, session, e.repoRef(), pr.Number, in())
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprCommentApplySuggestions_Validation(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusBadRequest, cprStatus((&SuggestionReference{CheckSum: "x"}).sanitize()))
	require.Equal(t, http.StatusBadRequest, cprStatus((&SuggestionReference{CommentID: 1, CheckSum: "  "}).sanitize()))

	ref := &SuggestionReference{CommentID: 1, CheckSum: " abc "}
	require.NoError(t, ref.sanitize())
	require.Equal(t, "abc", ref.CheckSum)

	require.Equal(t, http.StatusBadRequest, cprStatus((&CommentApplySuggestionsInput{}).sanitize()))
	require.Equal(t, http.StatusBadRequest, cprStatus((&CommentApplySuggestionsInput{
		Suggestions: []SuggestionReference{{CommentID: 0, CheckSum: "x"}},
	}).sanitize()))

	in := &CommentApplySuggestionsInput{
		Suggestions: []SuggestionReference{{CommentID: 1, CheckSum: "x"}},
		Title:       "  t  ",
		Message:     " m ",
	}
	require.NoError(t, in.sanitize())
	require.Equal(t, "t", in.Title)
	require.Equal(t, "m", in.Message)
}

func TestCprCommentApplySuggestions(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	text := "please change\n```suggestion\nfoo()\n```\n"
	checkSum := parseSuggestions(text)[0].checkSum

	codeComment := func(mutate ...func(a *types.PullReqActivity)) *types.PullReqActivity {
		return e.createComment(pr, e.reviewer, text, append([]func(*types.PullReqActivity){
			func(a *types.PullReqActivity) {
				a.Type = enum.PullReqActivityTypeCodeComment
				a.Kind = enum.PullReqActivityKindChangeComment
				a.CodeComment = &types.CodeCommentFields{
					MergeBaseSHA: pr.MergeBaseSHA, SourceSHA: pr.SourceSHA, Path: "a.go", LineNew: 3, SpanNew: 1,
				}
				require.NoError(t, a.SetPayload(&types.PullRequestActivityPayloadCodeComment{
					LineStartNew: true, LineEndNew: true,
				}))
			},
		}, mutate...)...)
	}
	input := func(c *types.PullReqActivity) *CommentApplySuggestionsInput {
		return &CommentApplySuggestionsInput{
			Suggestions: []SuggestionReference{{CommentID: c.ID, CheckSum: checkSum}},
		}
	}

	t.Run("dry run rules", func(t *testing.T) {
		in := input(codeComment())
		in.DryRunRules = true
		out, violations, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, in)
		require.NoError(t, err)
		require.Nil(t, violations)
		require.True(t, out.DryRunRules)
		require.Empty(t, out.CommitID)
	})

	t.Run("blocked by rules", func(t *testing.T) {
		rules := cprSetup(t)
		rules.createBranchRule("cpr-lifecycle", `{"lifecycle":{"update_forbidden":true}}`)
		rulesPR := rules.createPR()
		c := rules.createComment(rulesPR, rules.reviewer, text, func(a *types.PullReqActivity) {
			a.Type = enum.PullReqActivityTypeCodeComment
			a.Kind = enum.PullReqActivityKindChangeComment
		})
		_, violations, err := rules.ctrl.CommentApplySuggestions(rules.ctx, rules.session(rules.author),
			rules.repoRef(), rulesPR.Number, input(c))
		require.NoError(t, err)
		require.NotEmpty(t, violations)
	})

	t.Run("no suggestions", func(t *testing.T) {
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number,
			&CommentApplySuggestionsInput{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("enqueued pull request", func(t *testing.T) {
		queued := e.createPR(func(p *types.PullReq) { p.SubState = enum.PullReqSubStateMergeQueue })
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), queued.Number,
			&CommentApplySuggestionsInput{Suggestions: []SuggestionReference{{CommentID: 1, CheckSum: "x"}}})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("plain comment is not a code comment", func(t *testing.T) {
		plain := e.createComment(pr, e.reviewer, text)
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, input(plain))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("outdated code comment", func(t *testing.T) {
		c := codeComment(func(a *types.PullReqActivity) { a.CodeComment.Outdated = true })
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, input(c))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("comment on the old side", func(t *testing.T) {
		c := codeComment(func(a *types.PullReqActivity) {
			require.NoError(t, a.SetPayload(&types.PullRequestActivityPayloadCodeComment{
				LineStartNew: false, LineEndNew: false,
			}))
		})
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, input(c))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("checksum does not match", func(t *testing.T) {
		c := codeComment()
		in := input(c)
		in.Suggestions[0].CheckSum = "does-not-match"
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, in)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("same code comment twice", func(t *testing.T) {
		c := codeComment()
		in := input(c)
		in.Suggestions = append(in.Suggestions, in.Suggestions[0])
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: git.TreeNode{Type: git.TreeNodeTypeBlob, SHA: cprBaseSHA},
		}, nil).Once()
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, in)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reply to a plain comment", func(t *testing.T) {
		parent := e.createComment(pr, e.reviewer, "parent")
		reply := e.createComment(pr, e.reviewer, text, func(a *types.PullReqActivity) {
			a.ParentID = &parent.ID
			a.Order = parent.Order
			a.SubOrder = 1
		})
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, input(reply))
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("tree node lookup fails", func(t *testing.T) {
		c := codeComment()
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).
			Return((*git.GetTreeNodeOutput)(nil), errors.NotFound("tree node")).Once()
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number, input(c))
		require.ErrorContains(t, err, "tree node")
	})

	t.Run("unknown comment", func(t *testing.T) {
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number,
			&CommentApplySuggestionsInput{Suggestions: []SuggestionReference{{CommentID: 999999, CheckSum: "x"}}})
		require.Error(t, err)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), 4242,
			&CommentApplySuggestionsInput{Suggestions: []SuggestionReference{{CommentID: 1, CheckSum: "x"}}})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.CommentApplySuggestions(e.ctx, session, e.repoRef(), pr.Number,
			&CommentApplySuggestionsInput{Suggestions: []SuggestionReference{{CommentID: 1, CheckSum: "x"}}})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprCreate_ReviewersLabelsAndCodeOwners(t *testing.T) {
	e := cprSetup(t)
	cprMockCreateGit(e)

	group := e.createUserGroup("cpr_team")
	teamMember := e.reviewer.UID
	e.ctrl.userGroupResolver = cprResolver{groups: map[string]*types.UserGroup{
		group.Identifier: {
			ID: group.ID, Identifier: group.Identifier, Name: group.Name, SpaceID: e.space.ID,
			Users: []string{teamMember, e.author.UID},
		},
	}}

	e.ctrl.codeOwners = codeowners.New(
		e.ctrl.repoStore, e.git, codeowners.Config{FilePaths: []string{"CODEOWNERS"}},
		e.ctrl.principalStore, e.ctrl.userGroupResolver)
	e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
		Node: git.TreeNode{Type: git.TreeNodeTypeBlob, Mode: git.TreeNodeModeFile, SHA: cprBaseSHA},
	}, nil).Maybe()
	e.git.On("GetBlob", mock.Anything, mock.Anything).Return(func(
		context.Context, *git.GetBlobParams,
	) *git.GetBlobOutput {
		content := "* " + e.other.Email + " @" + group.Identifier + " " + e.author.Email + "\n"
		return &git.GetBlobOutput{
			SHA: sha.Must(cprBaseSHA), Size: int64(len(content)), ContentSize: int64(len(content)),
			Content: io.NopCloser(strings.NewReader(content)),
		}
	}, nil).Maybe()
	e.git.On("DiffFileNames", mock.Anything, mock.Anything).
		Return(git.DiffFileNamesOutput{Files: []string{"a.go"}}, nil).Maybe()

	e.createBranchRule("cpr-reviewers",
		`{"pullreq":{"reviewers":{"request_code_owners":true,"default_reviewer_ids":[2],`+
			`"default_user_group_reviewer_ids":[1]}}}`)

	bug, err := e.ctrl.labelSvc.Define(e.ctx, e.author.ID, nil, &e.repo.ID, &types.DefineLabelInput{
		Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
	})
	require.NoError(t, err)

	pr, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
		Title:                "with reviewers",
		SourceBranch:         cprFeature,
		TargetBranch:         "main",
		UserGroupReviewerIDs: []int64{group.ID},
		Labels:               []*types.PullReqLabelAssignInput{{LabelID: bug.ID}},
	})
	require.NoError(t, err)
	require.Len(t, pr.Labels, 1)

	reviewers, err := e.reviewerStore.List(e.ctx, pr.ID)
	require.NoError(t, err)
	ids := make([]int64, len(reviewers))
	for i, r := range reviewers {
		ids[i] = r.PrincipalID
	}
	// Default reviewer and code owner; the author is never added.
	require.ElementsMatch(t, []int64{e.reviewer.ID, e.other.ID}, ids)

	groups, err := e.ctrl.userGroupReviewerStore.List(e.ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
	require.NoError(t, err)
	types2 := map[enum.PullReqActivityType]int{}
	for _, a := range acts {
		types2[a.Type]++
	}
	require.Positive(t, types2[enum.PullReqActivityTypeReviewerAdd])
	require.Positive(t, types2[enum.PullReqActivityTypeUserGroupReviewerAdd])
	require.Positive(t, types2[enum.PullReqActivityTypeLabelModify])
}

func TestCprCreate_ReviewerLimit(t *testing.T) {
	e := cprSetup(t)
	cprMockCreateGit(e)

	ids := make([]int64, 0, types.MaxPullReqReviewers+1)
	for i := range types.MaxPullReqReviewers + 1 {
		user := e.createUser(int64(100+i), "cpr_bulk_"+strings.Repeat("x", 1)+string(rune('a'+i%26))+string(rune('a'+i/26)))
		ids = append(ids, user.ID)
	}

	_, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
		Title: "too many", SourceBranch: cprFeature, TargetBranch: "main", ReviewerIDs: ids,
	})
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, cprStatus(err))
}

func TestCprCreate_UserGroupLookupFailure(t *testing.T) {
	e := cprSetup(t)
	cprMockCreateGit(e)

	// Unknown user group IDs are simply not part of the lookup result.
	pr, err := e.ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), &CreateInput{
		Title: "group", SourceBranch: cprFeature, TargetBranch: "main", UserGroupReviewerIDs: []int64{4242},
	})
	require.NoError(t, err)
	require.NotNil(t, pr)
}

func TestCprAuthorizerError(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	wantErr := errors.New("authorizer down")
	ctrl := *e.ctrl
	ctrl.authorizer = cprErrAuthorizer{err: wantErr}

	_, err := ctrl.Find(e.ctx, session, e.repoRef(), pr.Number, types.PullReqMetadataOptions{})
	require.ErrorIs(t, err, wantErr)

	_, _, err = ctrl.List(e.ctx, session, e.repoRef(), &types.PullReqFilter{})
	require.ErrorIs(t, err, wantErr)

	_, err = ctrl.CommentCreate(e.ctx, session, e.repoRef(), pr.Number, &CommentCreateInput{Text: "x"}, nil)
	require.ErrorIs(t, err, wantErr)

	err = ctrl.ReviewerDelete(e.ctx, session, e.repoRef(), pr.Number, e.reviewer.ID)
	require.Error(t, err)

	_, _, err = ctrl.Merge(e.ctx, session, e.repoRef(), pr.Number,
		&MergeInput{Method: enum.MergeMethodMerge, SourceSHA: pr.SourceSHA})
	require.ErrorIs(t, err, wantErr)
}

func TestCprCreate_ReviewerAccessErrors(t *testing.T) {
	e := cprSetup(t)
	cprMockCreateGit(e)
	in := func() *CreateInput {
		return &CreateInput{
			Title: "t", SourceBranch: cprFeature, TargetBranch: "main", ReviewerIDs: []int64{e.reviewer.ID},
		}
	}

	t.Run("reviewer is forbidden", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprSelectiveAuthorizer{denied: map[int64]bool{e.reviewer.ID: true}}
		_, err := ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), in())
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("reviewer access check fails", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprSelectiveAuthorizer{failing: map[int64]error{e.reviewer.ID: errors.New("boom")}}
		_, err := ctrl.Create(e.ctx, e.session(e.author), e.repoRef(), in())
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

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
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCprWatchers(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()

	t.Run("watch, list and unwatch", func(t *testing.T) {
		require.NoError(t, e.ctrl.Watch(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number))
		require.NoError(t, e.ctrl.Watch(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number)) // idempotent
		require.NoError(t, e.ctrl.Watch(e.ctx, e.session(e.other), e.repoRef(), pr.Number))

		list, err := e.ctrl.WatcherList(e.ctx, e.session(e.author), e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Len(t, list, 2)

		require.NoError(t, e.ctrl.Unwatch(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number))
		require.NoError(t, e.ctrl.Unwatch(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number)) // idempotent

		list, err = e.ctrl.WatcherList(e.ctx, e.session(e.author), e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, e.other.ID, list[0].Watcher.ID)
	})

	t.Run("anonymous callers are unauthorized", func(t *testing.T) {
		anonymous := &auth.Session{Principal: auth.AnonymousPrincipal}
		require.ErrorIs(t, e.ctrl.Watch(e.ctx, anonymous, e.repoRef(), pr.Number), usererror.ErrUnauthorized)
		require.ErrorIs(t, e.ctrl.Unwatch(e.ctx, anonymous, e.repoRef(), pr.Number), usererror.ErrUnauthorized)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		require.Error(t, e.ctrl.Watch(e.ctx, e.session(e.reviewer), e.repoRef(), 4242))
		require.Error(t, e.ctrl.Unwatch(e.ctx, e.session(e.reviewer), e.repoRef(), 4242))
		_, err := e.ctrl.WatcherList(e.ctx, e.session(e.reviewer), e.repoRef(), 4242)
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		require.ErrorIs(t, ctrl.Watch(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number), apiauth.ErrForbidden)
		_, err := ctrl.WatcherList(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprFileViewInput_Validate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&FileViewAddInput{Path: "a.go", CommitSHA: cprSourceSHA}).Validate())
	require.Equal(t, http.StatusBadRequest, cprStatus((&FileViewAddInput{CommitSHA: cprSourceSHA}).Validate()))
	require.Equal(t, http.StatusBadRequest, cprStatus((&FileViewAddInput{Path: "a.go", CommitSHA: "xyz"}).Validate()))
}

func cprTreeNodeMatcher(gitRef string) interface{} {
	return mock.MatchedBy(func(p *git.GetTreeNodeParams) bool { return p.GitREF == gitRef })
}

func TestCprFileViews(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.reviewer)
	in := func() *FileViewAddInput { return &FileViewAddInput{Path: "a.go", CommitSHA: cprSourceSHA} }

	blob := func(nodeSHA string) *git.GetTreeNodeOutput {
		return &git.GetTreeNodeOutput{Node: git.TreeNode{Type: git.TreeNodeTypeBlob, SHA: nodeSHA}}
	}

	t.Run("add, list and delete", func(t *testing.T) {
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(cprSourceSHA)).
			Return(blob("aaaa"), nil).Once()
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(pr.MergeBaseSHA)).
			Return(blob("bbbb"), nil).Once()

		view, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.NoError(t, err)
		require.Equal(t, "a.go", view.Path)
		require.Equal(t, "aaaa", view.SHA)
		require.False(t, view.Obsolete)

		list, err := e.ctrl.FileViewList(e.ctx, session, e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Len(t, list, 1)

		other, err := e.ctrl.FileViewList(e.ctx, e.session(e.other), e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Empty(t, other)

		require.NoError(t, e.ctrl.FileViewDelete(e.ctx, session, e.repoRef(), pr.Number, "a.go"))
		list, err = e.ctrl.FileViewList(e.ctx, session, e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Empty(t, list)
	})

	t.Run("deleted file is stored with the nil sha", func(t *testing.T) {
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(cprSourceSHA)).
			Return((*git.GetTreeNodeOutput)(nil), errors.NotFound("gone")).Once()
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(pr.MergeBaseSHA)).
			Return(blob("bbbb"), nil).Once()

		view, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.NoError(t, err)
		require.Equal(t, types.NilSHA, view.SHA)
	})

	t.Run("file missing on both sides", func(t *testing.T) {
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).
			Return((*git.GetTreeNodeOutput)(nil), errors.NotFound("gone")).Twice()
		_, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("file unchanged", func(t *testing.T) {
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(blob("same"), nil).Twice()
		_, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("path is a directory", func(t *testing.T) {
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(cprSourceSHA)).
			Return(&git.GetTreeNodeOutput{Node: git.TreeNode{Type: git.TreeNodeTypeTree, SHA: "dir"}}, nil).Once()
		_, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("git errors", func(t *testing.T) {
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(cprSourceSHA)).
			Return((*git.GetTreeNodeOutput)(nil), errors.New("boom")).Once()
		_, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.ErrorContains(t, err, "provided sha")

		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(cprSourceSHA)).
			Return(blob("aaaa"), nil).Once()
		e.git.On("GetTreeNode", mock.Anything, cprTreeNodeMatcher(pr.MergeBaseSHA)).
			Return((*git.GetTreeNodeOutput)(nil), errors.New("boom")).Once()
		_, err = e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.ErrorContains(t, err, "MergeBaseSHA")
	})

	t.Run("validation and lookups", func(t *testing.T) {
		_, err := e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, &FileViewAddInput{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
		_, err = e.ctrl.FileViewAdd(e.ctx, session, e.repoRef(), 4242, in())
		require.Error(t, err)
		_, err = e.ctrl.FileViewList(e.ctx, session, e.repoRef(), 4242)
		require.Error(t, err)
		require.Error(t, e.ctrl.FileViewDelete(e.ctx, session, e.repoRef(), 4242, "a.go"))
		err = e.ctrl.FileViewDelete(e.ctx, session, e.repoRef(), pr.Number, "")
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.FileViewAdd(e.ctx, session, e.repoRef(), pr.Number, in())
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		_, err = ctrl.FileViewList(e.ctx, session, e.repoRef(), pr.Number)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		require.ErrorIs(t, ctrl.FileViewDelete(e.ctx, session, e.repoRef(), pr.Number, "a.go"), apiauth.ErrForbidden)
	})
}

func TestCprActivityList(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	group := e.createUserGroup("cpr_mentioned")
	live := e.createComment(pr, e.author, "hello @[2]", func(a *types.PullReqActivity) {
		a.Metadata = &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{
				IDs: []int64{e.reviewer.ID}, UserGroupIDs: []int64{group.ID},
			},
			Reactions: &types.PullReqActivityReactionsMetadata{
				Counts: map[string][]int64{"heart": {e.other.ID, 4242}},
			},
		}
	})

	deletedThread := e.createComment(pr, e.author, "secret")
	now := int64(10)
	deletedThread.Deleted = &now
	require.NoError(t, e.activityStore.Update(e.ctx, deletedThread))

	partial := e.createComment(pr, e.author, "root")
	partial.Deleted = &now
	require.NoError(t, e.activityStore.Update(e.ctx, partial))
	e.createComment(pr, e.reviewer, "visible reply", func(a *types.PullReqActivity) {
		a.ParentID = &partial.ID
		a.Order = partial.Order
		a.SubOrder = 1
	})

	list, err := e.ctrl.ActivityList(e.ctx, session, e.repoRef(), pr.Number, &types.PullReqActivityFilter{})
	require.NoError(t, err)

	byID := map[int64]*types.PullReqActivity{}
	for _, a := range list {
		byID[a.ID] = a
	}

	require.Contains(t, byID, live.ID)
	require.Contains(t, byID[live.ID].Mentions, e.reviewer.ID)
	require.Contains(t, byID[live.ID].GroupMentions, group.ID)
	require.Len(t, byID[live.ID].Reactions, 1)
	require.Equal(t, "heart", byID[live.ID].Reactions[0].Emoji)
	require.Equal(t, e.other.ID, byID[live.ID].Reactions[0].Author.ID)

	require.NotContains(t, byID, deletedThread.ID, "fully deleted threads are dropped")
	require.Contains(t, byID, partial.ID, "threads with a live reply are kept")
	require.Empty(t, byID[partial.ID].Text, "deleted comments lose their text")

	t.Run("unknown pull request", func(t *testing.T) {
		_, err := e.ctrl.ActivityList(e.ctx, session, e.repoRef(), 4242, &types.PullReqActivityFilter{})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.ActivityList(e.ctx, session, e.repoRef(), pr.Number, &types.PullReqActivityFilter{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprRemoveDeletedComments(t *testing.T) {
	t.Parallel()

	now := int64(1)
	acts := []*types.PullReqActivity{
		{ID: 1, Kind: enum.PullReqActivityKindSystem, Order: 1},
		{ID: 2, Kind: enum.PullReqActivityKindComment, Order: 2, Deleted: &now, Text: "x"},
		{ID: 3, Kind: enum.PullReqActivityKindComment, Order: 2, SubOrder: 1, Deleted: &now},
		{ID: 4, Kind: enum.PullReqActivityKindChangeComment, Order: 3},
		{ID: 5, Kind: enum.PullReqActivityKindComment, Order: 4, Deleted: &now},
		{ID: 6, Kind: enum.PullReqActivityKindSystem, Order: 5},
	}

	got := removeDeletedComments(acts)

	ids := make([]int64, len(got))
	for i, a := range got {
		ids[i] = a.ID
	}
	require.Equal(t, []int64{1, 4, 6}, ids)
	require.True(t, allCommentsDeleted([]*types.PullReqActivity{{Deleted: &now}}))
	require.False(t, allCommentsDeleted([]*types.PullReqActivity{{Deleted: &now}, {}}))
}

func TestCprParseMentions(t *testing.T) {
	t.Parallel()

	require.Equal(t, []int64{1, 22}, parseMentions(t.Context(), "hi @[1] and @[22], @[x] @[]"))
	require.Empty(t, parseMentions(t.Context(), "no mentions"))
}

func TestCprProcessMentions(t *testing.T) {
	e := cprSetup(t)

	got, err := e.ctrl.processMentions(e.ctx, "nobody")
	require.NoError(t, err)
	require.Empty(t, got)

	got, err = e.ctrl.processMentions(e.ctx, "@[1] and @[2]")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestCprLabels(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	static, err := e.ctrl.labelSvc.Define(e.ctx, e.author.ID, nil, &e.repo.ID, &types.DefineLabelInput{
		Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
	})
	require.NoError(t, err)
	dynamic, err := e.ctrl.labelSvc.Define(e.ctx, e.author.ID, nil, &e.repo.ID, &types.DefineLabelInput{
		Key: "priority", Type: enum.LabelTypeDynamic, Color: enum.LabelColorBlue,
	})
	require.NoError(t, err)

	t.Run("assign static and dynamic labels", func(t *testing.T) {
		got, err := e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: static.ID})
		require.NoError(t, err)
		require.Equal(t, static.ID, got.LabelID)

		// Assigning it again is a no-op.
		_, err = e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: static.ID})
		require.NoError(t, err)

		high, err := e.ctrl.labelSvc.DefineValue(e.ctx, e.author.ID, dynamic.ID,
			&types.DefineValueInput{Value: "high", Color: enum.LabelColorRed})
		require.NoError(t, err)
		low, err := e.ctrl.labelSvc.DefineValue(e.ctx, e.author.ID, dynamic.ID,
			&types.DefineValueInput{Value: "low", Color: enum.LabelColorGreen})
		require.NoError(t, err)

		got, err = e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: dynamic.ID, ValueID: &high.ID})
		require.NoError(t, err)
		require.NotNil(t, got.ValueID)
		require.Equal(t, high.ID, *got.ValueID)

		// Re-assigning a different value reports a re-assign.
		got, err = e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: dynamic.ID, ValueID: &low.ID})
		require.NoError(t, err)
		require.NotNil(t, got.ValueID)
		require.Equal(t, low.ID, *got.ValueID)

		acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
		require.NoError(t, err)
		var labelActs int
		for _, a := range acts {
			if a.Type == enum.PullReqActivityTypeLabelModify {
				labelActs++
			}
		}
		require.Equal(t, 3, labelActs)
	})

	t.Run("assign errors", func(t *testing.T) {
		_, err := e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: dynamic.ID, Value: "x", ValueID: ptr.Int64(5)})
		require.Error(t, err)
		_, err = e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: 4242})
		require.Error(t, err)
		_, err = e.ctrl.AssignLabel(e.ctx, session, e.repoRef(), 4242,
			&types.PullReqLabelAssignInput{LabelID: static.ID})
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.AssignLabel(e.ctx, session, e.repoRef(), pr.Number,
			&types.PullReqLabelAssignInput{LabelID: static.ID})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("unassign", func(t *testing.T) {
		require.NoError(t, e.ctrl.UnassignLabel(e.ctx, session, e.repoRef(), pr.Number, dynamic.ID))
		// Unassigning a label which isn't assigned anymore is a no-op or an error, but never a panic.
		_ = e.ctrl.UnassignLabel(e.ctx, session, e.repoRef(), pr.Number, dynamic.ID)

		require.Error(t, e.ctrl.UnassignLabel(e.ctx, session, e.repoRef(), 4242, static.ID))
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		require.ErrorIs(t, ctrl.UnassignLabel(e.ctx, session, e.repoRef(), pr.Number, static.ID), apiauth.ErrForbidden)
	})
}

func TestCprLabelSuggestions(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	bug, err := e.ctrl.labelSvc.Define(e.ctx, e.author.ID, nil, &e.repo.ID, &types.DefineLabelInput{
		Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
	})
	require.NoError(t, err)
	docs, err := e.ctrl.labelSvc.Define(e.ctx, e.author.ID, nil, &e.repo.ID, &types.DefineLabelInput{
		Key: "docs", Type: enum.LabelTypeStatic, Color: enum.LabelColorBlue,
	})
	require.NoError(t, err)

	t.Run("invalid requests", func(t *testing.T) {
		err := e.ctrl.SuggestLabels(e.ctx, session, e.repoRef(), pr.Number, label.CreatePullReqLabelSuggestionsRequest{})
		require.Error(t, err)

		err = e.ctrl.SuggestLabels(e.ctx, session, e.repoRef(), pr.Number, label.CreatePullReqLabelSuggestionsRequest{
			Labels: []*label.CreatePullReqLabelSuggestionInput{{LabelID: bug.ID}, {LabelID: bug.ID}},
		})
		require.Error(t, err)
	})

	t.Run("suggest, list, apply and remove", func(t *testing.T) {
		err := e.ctrl.SuggestLabels(e.ctx, session, e.repoRef(), pr.Number, label.CreatePullReqLabelSuggestionsRequest{
			Labels: []*label.CreatePullReqLabelSuggestionInput{{LabelID: bug.ID}, {LabelID: docs.ID}},
		})
		require.NoError(t, err)

		list, count, err := e.ctrl.ListSuggestedLabels(e.ctx, session, e.repoRef(), pr.Number,
			types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}})
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
		require.Len(t, list, 2)

		applied, err := e.ctrl.ApplySuggestedLabel(e.ctx, session, e.repoRef(), pr.Number, bug.ID)
		require.NoError(t, err)
		require.Equal(t, bug.ID, applied.LabelID)

		require.NoError(t, e.ctrl.RemoveSuggestedLabel(e.ctx, session, e.repoRef(), pr.Number, docs.ID))
		err = e.ctrl.RemoveSuggestedLabel(e.ctx, session, e.repoRef(), pr.Number, docs.ID)
		require.Equal(t, http.StatusNotFound, cprStatus(err))

		_, err = e.ctrl.ApplySuggestedLabel(e.ctx, session, e.repoRef(), pr.Number, docs.ID)
		require.Error(t, err)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.ListSuggestedLabels(e.ctx, session, e.repoRef(), 4242, types.ListQueryFilter{})
		require.Error(t, err)
		_, err = e.ctrl.ApplySuggestedLabel(e.ctx, session, e.repoRef(), 4242, bug.ID)
		require.Error(t, err)
		require.Error(t, e.ctrl.RemoveSuggestedLabel(e.ctx, session, e.repoRef(), 4242, bug.ID))
		err = e.ctrl.SuggestLabels(e.ctx, session, e.repoRef(), 4242, label.CreatePullReqLabelSuggestionsRequest{
			Labels: []*label.CreatePullReqLabelSuggestionInput{},
		})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.ListSuggestedLabels(e.ctx, session, e.repoRef(), pr.Number, types.ListQueryFilter{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		_, err = ctrl.ApplySuggestedLabel(e.ctx, session, e.repoRef(), pr.Number, bug.ID)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		require.ErrorIs(t, ctrl.RemoveSuggestedLabel(e.ctx, session, e.repoRef(), pr.Number, bug.ID), apiauth.ErrForbidden)
		err = ctrl.SuggestLabels(e.ctx, session, e.repoRef(), pr.Number, label.CreatePullReqLabelSuggestionsRequest{
			Labels: []*label.CreatePullReqLabelSuggestionInput{},
		})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprListLabels(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()

	_, err := e.ctrl.labelSvc.Define(e.ctx, e.author.ID, nil, &e.repo.ID, &types.DefineLabelInput{
		Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
	})
	require.NoError(t, err)

	t.Run("list", func(t *testing.T) {
		got, total, err := e.ctrl.ListLabels(
			e.ctx, e.session(e.author), e.repoRef(), pr.Number, &types.AssignableLabelFilter{})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.GreaterOrEqual(t, total, int64(0))
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.ListLabels(e.ctx, e.session(e.author), e.repoRef(), pr.Number, &types.AssignableLabelFilter{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, _, err := e.ctrl.ListLabels(e.ctx, e.session(e.author), e.repoRef(), 4242, &types.AssignableLabelFilter{})
		require.Error(t, err)
	})
}

func TestCprCodeOwners(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	t.Run("no CODEOWNERS file", func(t *testing.T) {
		_, err := e.ctrl.CodeOwners(e.ctx, session, e.repoRef(), pr.Number)
		require.ErrorIs(t, err, codeowners.ErrNotFound)
	})

	e.ctrl.codeOwners = codeowners.New(
		e.ctrl.repoStore, e.git, codeowners.Config{FilePaths: []string{"CODEOWNERS"}},
		e.ctrl.principalStore, e.ctrl.userGroupResolver)

	e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
		Node: git.TreeNode{Type: git.TreeNodeTypeBlob, Mode: git.TreeNodeModeFile, SHA: cprBaseSHA},
	}, nil).Maybe()
	e.git.On("GetBlob", mock.Anything, mock.Anything).Return(func(
		context.Context, *git.GetBlobParams,
	) *git.GetBlobOutput {
		content := "* " + cprUserReviewer + "@example.com\n"
		return &git.GetBlobOutput{
			SHA: sha.Must(cprBaseSHA), Size: int64(len(content)), ContentSize: int64(len(content)),
			Content: io.NopCloser(strings.NewReader(content)),
		}
	}, nil).Maybe()
	e.git.On("DiffFileNames", mock.Anything, mock.Anything).
		Return(git.DiffFileNamesOutput{Files: []string{"a.go"}}, nil).Maybe()

	e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionApproved)

	t.Run("evaluation", func(t *testing.T) {
		got, err := e.ctrl.CodeOwners(e.ctx, session, e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Equal(t, cprBaseSHA, got.FileSha)
		require.Len(t, got.EvaluationEntries, 1)
		require.Equal(t, "*", got.EvaluationEntries[0].Pattern)
		require.Len(t, got.EvaluationEntries[0].OwnerEvaluations, 1)
		require.Equal(t, e.reviewer.ID, got.EvaluationEntries[0].OwnerEvaluations[0].Owner.ID)
		require.Equal(t, enum.PullReqReviewDecisionApproved, got.EvaluationEntries[0].OwnerEvaluations[0].ReviewDecision)
	})

	t.Run("errors", func(t *testing.T) {
		_, err := e.ctrl.CodeOwners(e.ctx, session, e.repoRef(), 4242)
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.CodeOwners(e.ctx, session, e.repoRef(), pr.Number)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprMapCodeOwnerEvaluation(t *testing.T) {
	t.Parallel()

	owner := codeowners.UserEvaluation{
		Owner:          types.PrincipalInfo{ID: 7},
		ReviewDecision: enum.PullReqReviewDecisionChangeReq,
		ReviewSHA:      cprBaseSHA,
	}
	in := &codeowners.Evaluation{
		EvaluationEntries: []codeowners.EvaluationEntry{{
			LineNumber:      3,
			Pattern:         "*.go",
			UserEvaluations: []codeowners.UserEvaluation{owner},
			UserGroupEvaluations: []codeowners.UserGroupEvaluation{{
				Identifier:  "grp",
				Name:        "Group",
				Evaluations: []codeowners.UserEvaluation{owner},
			}},
		}},
	}

	got := mapCodeOwnerEvaluation(in)
	require.Len(t, got, 1)
	require.EqualValues(t, 3, got[0].LineNumber)
	require.Equal(t, "*.go", got[0].Pattern)
	require.Equal(t, int64(7), got[0].OwnerEvaluations[0].Owner.ID)
	require.Equal(t, enum.PullReqReviewDecisionChangeReq, got[0].OwnerEvaluations[0].ReviewDecision)
	require.Equal(t, "grp", got[0].UserGroupOwnerEvaluations[0].ID)
	require.Equal(t, "Group", got[0].UserGroupOwnerEvaluations[0].Name)
	require.Len(t, got[0].UserGroupOwnerEvaluations[0].Evaluations, 1)
}

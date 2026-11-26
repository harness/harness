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

package database_test

import (
	"errors"
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// prqNewActivity builds a comment activity for the pull request with the given order and sub order.
func prqNewActivity(
	pr *types.PullReq,
	order, subOrder int64,
	mutators ...func(a *types.PullReqActivity),
) *types.PullReqActivity {
	act := &types.PullReqActivity{
		CreatedBy: prqUserA,
		Created:   prqBaseTime + order*10 + subOrder,
		Updated:   prqBaseTime + order*10 + subOrder,
		Edited:    prqBaseTime + order*10 + subOrder,
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     order,
		SubOrder:  subOrder,
		Type:      enum.PullReqActivityTypeComment,
		Kind:      enum.PullReqActivityKindComment,
		Text:      "comment text",
		Metadata:  &types.PullReqActivityMetadata{},
	}
	_ = act.SetPayload(&types.PullRequestActivityPayloadComment{})
	for _, mutate := range mutators {
		mutate(act)
	}
	return act
}

// prqNewCodeComment builds a code comment activity at the given path and line.
func prqNewCodeComment(
	pr *types.PullReq,
	order int64,
	kind enum.PullReqActivityKind,
	path string,
	line int,
	mutators ...func(a *types.PullReqActivity),
) *types.PullReqActivity {
	return prqNewActivity(pr, order, 0, append([]func(a *types.PullReqActivity){
		func(a *types.PullReqActivity) {
			a.Type = enum.PullReqActivityTypeCodeComment
			a.Kind = kind
			a.CodeComment = &types.CodeCommentFields{
				MergeBaseSHA: "base-1",
				SourceSHA:    "source-1",
				Path:         path,
				LineNew:      line,
				SpanNew:      1,
				LineOld:      line - 1,
				SpanOld:      2,
			}
			_ = a.SetPayload(&types.PullRequestActivityPayloadCodeComment{Title: "title"})
		},
	}, mutators...)...)
}

func TestPrqPullReqActivityStore_CreateFindUpdate(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	resolvedBy := prqUserB
	resolved := prqBaseTime + 500
	in := prqNewActivity(pr, 1, 0, func(a *types.PullReqActivity) {
		a.Metadata = &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{prqUserC}},
		}
		a.ResolvedBy = &resolvedBy
		a.Resolved = &resolved
	})
	require.NoError(t, acts.Create(f.ctx, in))
	require.NotZero(t, in.ID)

	got, err := acts.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, in.ID, got.ID)
	require.Equal(t, "comment text", got.Text)
	require.Equal(t, enum.PullReqActivityKindComment, got.Kind)
	require.Equal(t, pr.ID, got.PullReqID)
	require.Equal(t, prqUserA, got.Author.ID)
	require.NotNil(t, got.Resolver)
	require.Equal(t, prqUserB, got.Resolver.ID)
	require.Equal(t, &resolved, got.Resolved)
	require.Nil(t, got.CodeComment)
	require.NotNil(t, got.Metadata)
	require.NotNil(t, got.Metadata.Mentions)
	require.Equal(t, []int64{prqUserC}, got.Metadata.Mentions.IDs)

	// update bumps the version and reflects the new data.
	got.Text = "edited text"
	got.ReplySeq = 3
	deleted := prqBaseTime + 900
	got.Deleted = &deleted
	require.NoError(t, acts.Update(f.ctx, got))
	require.Equal(t, int64(1), got.Version)
	require.Equal(t, prqUserA, got.Author.ID)

	reloaded, err := acts.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, "edited text", reloaded.Text)
	require.Equal(t, int64(3), reloaded.ReplySeq)
	require.Equal(t, &deleted, reloaded.Deleted)
	require.Equal(t, int64(1), reloaded.Version)
}

func TestPrqPullReqActivityStore_FindAndCreateErrors(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	_, err := acts.Find(f.ctx, 404)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 1, 0)))
	err = acts.Create(f.ctx, prqNewActivity(pr, 1, 0))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = acts.Create(f.ctx, prqNewActivity(&types.PullReq{ID: 9999, TargetRepoID: prqRepoMain}, 1, 0))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqPullReqActivityStore_UpdateVersionConflict(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	act := prqNewActivity(pr, 1, 0)
	require.NoError(t, acts.Create(f.ctx, act))

	stale := *act
	require.NoError(t, acts.Update(f.ctx, act))

	stale.Text = "stale"
	err := acts.Update(f.ctx, &stale)
	require.ErrorIs(t, err, gitness_store.ErrVersionConflict)
}

func TestPrqPullReqActivityStore_UpdateOptLock(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	act := prqNewActivity(pr, 1, 0)
	require.NoError(t, acts.Create(f.ctx, act))

	concurrent := *act
	require.NoError(t, acts.Update(f.ctx, &concurrent))

	calls := 0
	updated, err := acts.UpdateOptLock(f.ctx, act, func(a *types.PullReqActivity) error {
		calls++
		a.Text = "optimistic"
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "optimistic", updated.Text)
	require.Equal(t, int64(2), updated.Version)

	stored, err := acts.Find(f.ctx, act.ID)
	require.NoError(t, err)
	require.Equal(t, "optimistic", stored.Text)

	errBoomPrq := errors.New("mutate failed")
	_, err = acts.UpdateOptLock(f.ctx, stored, func(*types.PullReqActivity) error { return errBoomPrq })
	require.ErrorIs(t, err, errBoomPrq)

	// stale copy of a deleted activity can't be reloaded.
	stale := *stored
	require.NoError(t, acts.Update(f.ctx, stored))
	require.NoError(t, f.pullreqs.Delete(f.ctx, pr.ID))
	_, err = acts.UpdateOptLock(f.ctx, &stale, func(*types.PullReqActivity) error { return nil })
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqActivityStore_CreateWithPayload(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1, func(p *types.PullReq) { p.ActivitySeq = 4 })
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	metadata := &types.PullReqActivityMetadata{
		Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{prqUserB}},
	}
	payload := &types.PullRequestActivityPayloadBranchDelete{SHA: "deleted-sha"}
	act, err := acts.CreateWithPayload(f.ctx, pr, prqUserB, payload, metadata)
	require.NoError(t, err)
	require.NotZero(t, act.ID)
	require.Equal(t, enum.PullReqActivityKindSystem, act.Kind)
	require.Equal(t, enum.PullReqActivityTypeBranchDelete, act.Type)
	require.Equal(t, int64(4), act.Order)

	got, err := acts.Find(f.ctx, act.ID)
	require.NoError(t, err)
	require.Equal(t, prqUserB, got.CreatedBy)
	require.Equal(t, prqUserB, got.Author.ID)
	require.Equal(t, prqRepoMain, got.RepoID)
	require.JSONEq(t, `{"sha":"deleted-sha"}`, string(got.PayloadRaw))
	require.Equal(t, []int64{prqUserB}, got.Metadata.Mentions.IDs)

	// the same order can't be used twice.
	_, err = acts.CreateWithPayload(f.ctx, pr, prqUserB, payload, nil)
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)
}

func TestPrqPullReqActivityStore_ListAndCountFilters(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	system := func(a *types.PullReqActivity) {
		a.Type = enum.PullReqActivityTypeStateChange
		a.Kind = enum.PullReqActivityKindSystem
		_ = a.SetPayload(&types.PullRequestActivityPayloadStateChange{})
	}

	byKey := map[string]*types.PullReqActivity{
		"comment": prqNewActivity(pr, 1, 0),
		"reply":   prqNewActivity(pr, 1, 1),
		"system":  prqNewActivity(pr, 2, 0, system),
		"change": prqNewActivity(pr, 3, 0, func(a *types.PullReqActivity) {
			a.Kind = enum.PullReqActivityKindChangeComment
		}),
		"code":  prqNewCodeComment(pr, 4, enum.PullReqActivityKindReviewComment, "a.go", 10),
		"other": prqNewActivity(other, 1, 0),
	}
	for _, key := range []string{"comment", "reply", "system", "change", "code", "other"} {
		require.NoError(t, acts.Create(f.ctx, byKey[key]), key)
	}

	tests := []struct {
		name   string
		filter types.PullReqActivityFilter
		want   []string
	}{
		{name: "all", filter: types.PullReqActivityFilter{}, want: []string{"comment", "reply", "system", "change", "code"}},
		{
			name:   "single type",
			filter: types.PullReqActivityFilter{Types: []enum.PullReqActivityType{enum.PullReqActivityTypeStateChange}},
			want:   []string{"system"},
		},
		{
			name: "multiple types",
			filter: types.PullReqActivityFilter{Types: []enum.PullReqActivityType{
				enum.PullReqActivityTypeStateChange, enum.PullReqActivityTypeCodeComment,
			}},
			want: []string{"system", "code"},
		},
		{
			name:   "single kind",
			filter: types.PullReqActivityFilter{Kinds: []enum.PullReqActivityKind{enum.PullReqActivityKindSystem}},
			want:   []string{"system"},
		},
		{
			name: "multiple kinds",
			filter: types.PullReqActivityFilter{Kinds: []enum.PullReqActivityKind{
				enum.PullReqActivityKindChangeComment, enum.PullReqActivityKindReviewComment,
			}},
			want: []string{"change", "code"},
		},
		{
			name:   "after",
			filter: types.PullReqActivityFilter{After: byKey["system"].Created},
			want:   []string{"change", "code"},
		},
		{
			name:   "before",
			filter: types.PullReqActivityFilter{Before: byKey["system"].Created},
			want:   []string{"comment", "reply"},
		},
		{
			name:   "limit",
			filter: types.PullReqActivityFilter{Limit: 2},
			want:   []string{"comment", "reply"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := acts.List(f.ctx, pr.ID, &tt.filter)
			require.NoError(t, err)

			got := make([]int64, len(list))
			for i, a := range list {
				got[i] = a.ID
			}
			want := make([]int64, len(tt.want))
			for i, key := range tt.want {
				want[i] = byKey[key].ID
			}
			require.Equal(t, want, got)

			// count has no limit support.
			if tt.filter.Limit == 0 {
				count, err := acts.Count(f.ctx, pr.ID, &tt.filter)
				require.NoError(t, err)
				require.Equal(t, int64(len(want)), count)
			}
		})
	}

	// the code comment fields are populated for code comments only.
	list, err := acts.List(f.ctx, pr.ID, &types.PullReqActivityFilter{
		Types: []enum.PullReqActivityType{enum.PullReqActivityTypeCodeComment},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].CodeComment)
	require.Equal(t, "a.go", list[0].CodeComment.Path)
	require.Equal(t, 10, list[0].CodeComment.LineNew)
	require.Equal(t, 9, list[0].CodeComment.LineOld)
	require.Equal(t, 2, list[0].CodeComment.SpanOld)
	require.False(t, list[0].CodeComment.Outdated)
}

func TestPrqPullReqActivityStore_ListAuthorIDs(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	by := func(principal int64) func(a *types.PullReqActivity) {
		return func(a *types.PullReqActivity) { a.CreatedBy = principal }
	}
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 1, 0, by(prqUserA))))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 1, 1, by(prqUserB))))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 1, 2, by(prqUserB))))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 2, 0, by(prqUserC))))

	ids, err := acts.ListAuthorIDs(f.ctx, pr.ID, 1)
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{prqUserA, prqUserB}, ids)

	ids, err = acts.ListAuthorIDs(f.ctx, pr.ID, 2)
	require.NoError(t, err)
	require.Equal(t, []int64{prqUserC}, ids)

	ids, err = acts.ListAuthorIDs(f.ctx, pr.ID, 3)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestPrqPullReqActivityStore_CountUnresolved(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)

	count, err := acts.CountUnresolved(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	resolved := prqBaseTime + 1
	deleted := prqBaseTime + 2

	// counted: unresolved top level comment and change comment.
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 1, 0)))
	require.NoError(t, acts.Create(f.ctx, prqNewCodeComment(pr, 2, enum.PullReqActivityKindChangeComment, "a.go", 1)))
	// not counted: reply, resolved, deleted, review comment and system activity.
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 1, 1)))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 3, 0, func(a *types.PullReqActivity) {
		a.Resolved = &resolved
	})))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 4, 0, func(a *types.PullReqActivity) {
		a.Deleted = &deleted
	})))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 5, 0, func(a *types.PullReqActivity) {
		a.Kind = enum.PullReqActivityKindReviewComment
	})))
	require.NoError(t, acts.Create(f.ctx, prqNewActivity(pr, 6, 0, func(a *types.PullReqActivity) {
		a.Type = enum.PullReqActivityTypeStateChange
		a.Kind = enum.PullReqActivityKindSystem
		_ = a.SetPayload(&types.PullRequestActivityPayloadStateChange{})
	})))

	count, err = acts.CountUnresolved(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func prqCodeCommentIDs(list []*types.CodeComment) []int64 {
	ids := make([]int64, len(list))
	for i, cc := range list {
		ids[i] = cc.ID
	}
	return ids
}

func TestPrqCodeCommentView_List(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)
	view := database.NewCodeCommentView(f.db)

	empty, err := view.ListNotAtSourceSHA(f.ctx, pr.ID, "source-1")
	require.NoError(t, err)
	require.NotNil(t, empty)
	require.Empty(t, empty)

	withSHAs := func(base, source string) func(a *types.PullReqActivity) {
		return func(a *types.PullReqActivity) {
			a.CodeComment.MergeBaseSHA = base
			a.CodeComment.SourceSHA = source
		}
	}
	deleted := prqBaseTime

	byKey := map[string]*types.PullReqActivity{
		// ordered by path and then by new line.
		"b-line5":  prqNewCodeComment(pr, 1, enum.PullReqActivityKindChangeComment, "b.go", 5),
		"a-line20": prqNewCodeComment(pr, 2, enum.PullReqActivityKindReviewComment, "a.go", 20),
		"a-line3":  prqNewCodeComment(pr, 3, enum.PullReqActivityKindChangeComment, "a.go", 3),
		// current: matches both shas.
		"current": prqNewCodeComment(pr, 4, enum.PullReqActivityKindChangeComment, "c.go", 1, withSHAs("base-2", "source-2")),
		// excluded: outdated, deleted, a reply, a non code comment kind and another pull request.
		"outdated": prqNewCodeComment(pr, 5, enum.PullReqActivityKindChangeComment, "d.go", 1,
			func(a *types.PullReqActivity) { a.CodeComment.Outdated = true }),
		"deleted": prqNewCodeComment(pr, 6, enum.PullReqActivityKindChangeComment, "e.go", 1,
			func(a *types.PullReqActivity) { a.Deleted = &deleted }),
		"plain":    prqNewActivity(pr, 8, 0),
		"otherpr":  prqNewCodeComment(other, 1, enum.PullReqActivityKindChangeComment, "a.go", 1),
		"reply-to": prqNewCodeComment(pr, 9, enum.PullReqActivityKindChangeComment, "f.go", 1),
	}
	for _, key := range []string{
		"b-line5", "a-line20", "a-line3", "current", "outdated", "deleted", "plain", "otherpr", "reply-to",
	} {
		require.NoError(t, acts.Create(f.ctx, byKey[key]), key)
	}
	// a reply (has a parent) must not be listed.
	reply := prqNewCodeComment(pr, 9, enum.PullReqActivityKindChangeComment, "f.go", 2, func(a *types.PullReqActivity) {
		a.SubOrder = 1
		a.ParentID = &byKey["reply-to"].ID
	})
	require.NoError(t, acts.Create(f.ctx, reply))
	// the parent itself is a regular top level comment and is listed after the others (path f.go).

	list, err := view.ListNotAtSourceSHA(f.ctx, pr.ID, "source-2")
	require.NoError(t, err)
	require.Equal(t, []int64{
		byKey["a-line3"].ID, byKey["a-line20"].ID, byKey["b-line5"].ID, byKey["reply-to"].ID,
	}, prqCodeCommentIDs(list))
	require.Equal(t, "a.go", list[0].Path)
	require.Equal(t, 3, list[0].LineNew)
	require.Equal(t, 1, list[0].SpanNew)
	require.Equal(t, 2, list[0].LineOld)
	require.Equal(t, 2, list[0].SpanOld)
	require.Equal(t, "base-1", list[0].MergeBaseSHA)
	require.Equal(t, "source-1", list[0].SourceSHA)
	require.False(t, list[0].Outdated)

	list, err = view.ListNotAtMergeBaseSHA(f.ctx, pr.ID, "base-2")
	require.NoError(t, err)
	require.Equal(t, []int64{
		byKey["a-line3"].ID, byKey["a-line20"].ID, byKey["b-line5"].ID, byKey["reply-to"].ID,
	}, prqCodeCommentIDs(list))

	list, err = view.ListNotAtMergeBaseSHA(f.ctx, pr.ID, "base-1")
	require.NoError(t, err)
	require.Equal(t, []int64{byKey["current"].ID}, prqCodeCommentIDs(list))
}

func TestPrqCodeCommentView_UpdateAll(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	acts := database.NewPullReqActivityStore(f.db, f.pCache)
	view := database.NewCodeCommentView(f.db)

	// nothing to update.
	require.NoError(t, view.UpdateAll(f.ctx, nil))

	first := prqNewCodeComment(pr, 1, enum.PullReqActivityKindChangeComment, "a.go", 10)
	second := prqNewCodeComment(pr, 2, enum.PullReqActivityKindReviewComment, "b.go", 20)
	third := prqNewCodeComment(pr, 3, enum.PullReqActivityKindChangeComment, "c.go", 30)
	for _, act := range []*types.PullReqActivity{first, second, third} {
		require.NoError(t, acts.Create(f.ctx, act))
	}

	list, err := view.ListNotAtSourceSHA(f.ctx, pr.ID, "source-2")
	require.NoError(t, err)
	require.Len(t, list, 3)

	byID := map[int64]*types.CodeComment{}
	for _, cc := range list {
		byID[cc.ID] = cc
	}

	// move the first comment, and mark the second one as outdated.
	moved := byID[first.ID]
	moved.SourceSHA = "source-2"
	moved.MergeBaseSHA = "base-2"
	moved.LineNew = 12
	moved.SpanNew = 3
	moved.LineOld = 11
	moved.SpanOld = 4
	outdated := byID[second.ID]
	outdated.Outdated = true

	// the third update is based on a stale version and is skipped without an error.
	stale := byID[third.ID]
	stale.Version = 99
	stale.Path = "stale.go"

	require.NoError(t, view.UpdateAll(f.ctx, []*types.CodeComment{moved, outdated, stale}))
	require.Equal(t, int64(1), moved.Version)
	require.NotZero(t, moved.Updated)

	got, err := acts.Find(f.ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Version)
	require.Equal(t, "source-2", got.CodeComment.SourceSHA)
	require.Equal(t, "base-2", got.CodeComment.MergeBaseSHA)
	require.Equal(t, 12, got.CodeComment.LineNew)
	require.Equal(t, 3, got.CodeComment.SpanNew)
	require.Equal(t, 11, got.CodeComment.LineOld)
	require.Equal(t, 4, got.CodeComment.SpanOld)

	got, err = acts.Find(f.ctx, second.ID)
	require.NoError(t, err)
	require.True(t, got.CodeComment.Outdated)

	got, err = acts.Find(f.ctx, third.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), got.Version)
	require.Equal(t, "c.go", got.CodeComment.Path)

	// outdated comments are no longer listed, moved ones are no longer behind the new source sha.
	list, err = view.ListNotAtSourceSHA(f.ctx, pr.ID, "source-2")
	require.NoError(t, err)
	require.Equal(t, []int64{third.ID}, prqCodeCommentIDs(list))
}

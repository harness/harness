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
	"context"
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	prqBaseTime int64 = 1_000_000

	prqSHA2 = "2222222222222222222222222222222222222222"
)

// prqFilterData is the dataset used by the list and count filter tests.
type prqFilterData struct {
	*prqFixture

	// prs maps a short key (e.g. "r1-3") to the stored pull request.
	prs map[string]*types.PullReq

	labelBug  int64
	labelArea int64
	valueHigh int64
}

func (d *prqFilterData) ids(keys ...string) []int64 {
	out := make([]int64, 0, len(keys))
	for _, key := range keys {
		out = append(out, d.prs[key].ID)
	}
	return out
}

// prqNewFilterData builds the following pull requests, created in this order (created = base + 100 * index):
//
//	r1-1 open    by A "Fix login bug"       source feature-1, mergeable, labels: bug
//	r1-2 open    by B "Add search"          draft, rebase conflict, source sha 2222.., labels: bug=high, area
//	r1-3 merged  by A "Refactor store"      merged by B, labels: area
//	r1-4 closed  by C "Docs update"         target branch release
//	r1-5 open    by B "Auto merge me"       substate auto_merge
//	r2-1 open    by A "Other repo change"   target and source repo 2.
func prqNewFilterData(t *testing.T) *prqFilterData {
	t.Helper()

	f := prqNewFixture(t)
	d := &prqFilterData{prqFixture: f, prs: map[string]*types.PullReq{}}

	setTimes := func(index int64) func(pr *types.PullReq) {
		return func(pr *types.PullReq) {
			pr.Created = prqBaseTime + index*100
			pr.Updated = pr.Created + 1
			pr.Edited = pr.Created + 2
		}
	}

	d.prs["r1-1"] = f.prqCreatePullReq(t, prqRepoMain, 1, setTimes(1), func(pr *types.PullReq) {
		pr.Title = "Fix login bug"
		pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
	})
	d.prs["r1-2"] = f.prqCreatePullReq(t, prqRepoMain, 2, setTimes(2), func(pr *types.PullReq) {
		pr.CreatedBy = prqUserB
		pr.Title = "Add search"
		pr.IsDraft = true
		pr.RebaseCheckStatus = enum.MergeCheckStatusConflict
		pr.SourceSHA = prqSHA2
	})
	merger := prqUserB
	d.prs["r1-3"] = f.prqCreatePullReq(t, prqRepoMain, 3, setTimes(3), func(pr *types.PullReq) {
		pr.Title = "Refactor store"
		pr.State = enum.PullReqStateMerged
		pr.MergedBy = &merger
	})
	d.prs["r1-4"] = f.prqCreatePullReq(t, prqRepoMain, 4, setTimes(4), func(pr *types.PullReq) {
		pr.CreatedBy = prqUserC
		pr.Title = "Docs update"
		pr.State = enum.PullReqStateClosed
		pr.TargetBranch = "release"
	})
	d.prs["r1-5"] = f.prqCreatePullReq(t, prqRepoMain, 5, setTimes(5), func(pr *types.PullReq) {
		pr.CreatedBy = prqUserB
		pr.Title = "Auto merge me"
		pr.SubState = enum.PullReqSubStateAutoMerge
	})
	d.prs["r2-1"] = f.prqCreatePullReq(t, prqRepoOther, 1, setTimes(6), func(pr *types.PullReq) {
		pr.Title = "Other repo change"
		pr.SourceBranch = "other-branch"
	})

	// auto merge row so that the auto_merge substate filter (which joins auto_merges) matches.
	require.NoError(t, database.NewAutoMergeStore(f.db).Upsert(f.ctx, &types.AutoMerge{
		PullReqID:   d.prs["r1-5"].ID,
		Requested:   prqBaseTime,
		RequestedBy: prqUserA,
		MergeMethod: enum.MergeMethodMerge,
		Title:       "t",
		Message:     "m",
	}))

	d.seedActivities(t)
	d.seedReviewers(t)
	d.seedLabels(t)

	return d
}

func (d *prqFilterData) seedActivities(t *testing.T) {
	t.Helper()

	activities := database.NewPullReqActivityStore(d.db, d.pCache)
	create := func(prKey string, kind enum.PullReqActivityKind, by int64, mutate func(a *types.PullReqActivity)) {
		pr := d.prs[prKey]
		act := &types.PullReqActivity{
			CreatedBy: by,
			Created:   prqBaseTime,
			Updated:   prqBaseTime,
			Edited:    prqBaseTime,
			RepoID:    pr.TargetRepoID,
			PullReqID: pr.ID,
			Order:     1,
			Type:      enum.PullReqActivityTypeComment,
			Kind:      kind,
			Text:      "text",
			Metadata:  &types.PullReqActivityMetadata{},
		}
		require.NoError(t, act.SetPayload(&types.PullRequestActivityPayloadComment{}))
		if mutate != nil {
			mutate(act)
		}
		require.NoError(t, activities.Create(d.ctx, act))
	}

	// user B commented on r1-1 (counts) and mentions A and C.
	create("r1-1", enum.PullReqActivityKindComment, prqUserB, func(a *types.PullReqActivity) {
		a.Metadata = &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{prqUserA, prqUserC}},
		}
	})
	// a deleted comment of B on r1-2 doesn't count, but is by C otherwise to separate commenters.
	deleted := prqBaseTime
	create("r1-2", enum.PullReqActivityKindComment, prqUserB, func(a *types.PullReqActivity) {
		a.Deleted = &deleted
		a.Metadata = &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{prqUserC}},
		}
	})
	// system activities don't count as comments.
	create("r1-3", enum.PullReqActivityKindSystem, prqUserB, nil)
	// review comment of B on r1-4 counts, mentions C.
	create("r1-4", enum.PullReqActivityKindReviewComment, prqUserB, func(a *types.PullReqActivity) {
		a.Metadata = &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{IDs: []int64{prqUserC}},
		}
	})
	// user C commented on r1-5.
	create("r1-5", enum.PullReqActivityKindChangeComment, prqUserC, nil)
}

func (d *prqFilterData) seedReviewers(t *testing.T) {
	t.Helper()

	reviewers := database.NewPullReqReviewerStore(d.db, d.pCache)
	add := func(prKey string, principal int64, decision enum.PullReqReviewDecision) {
		pr := d.prs[prKey]
		require.NoError(t, reviewers.Create(d.ctx, &types.PullReqReviewer{
			PullReqID:      pr.ID,
			PrincipalID:    principal,
			CreatedBy:      prqUserA,
			Created:        prqBaseTime,
			Updated:        prqBaseTime,
			RepoID:         pr.TargetRepoID,
			Type:           enum.PullReqReviewerTypeRequested,
			ReviewDecision: decision,
			SHA:            "reviewed-sha",
		}))
	}

	add("r1-1", prqUserB, enum.PullReqReviewDecisionApproved)
	add("r1-2", prqUserB, enum.PullReqReviewDecisionPending)
	add("r1-3", prqUserC, enum.PullReqReviewDecisionChangeReq)
}

func (d *prqFilterData) seedLabels(t *testing.T) {
	t.Helper()

	d.labelBug = prqDefineLabel(d.ctx, t, d.db, "bug")
	d.labelArea = prqDefineLabel(d.ctx, t, d.db, "area")
	d.valueHigh = prqDefineLabelValue(d.ctx, t, d.db, d.labelBug, "high")

	labels := database.NewPullReqLabelStore(d.db)
	assign := func(prKey string, labelID int64, valueID *int64) {
		require.NoError(t, labels.Assign(d.ctx, &types.PullReqLabel{
			PullReqID: d.prs[prKey].ID,
			LabelID:   labelID,
			ValueID:   valueID,
			Created:   prqBaseTime,
			Updated:   prqBaseTime,
			CreatedBy: prqUserA,
			UpdatedBy: prqUserA,
		}))
	}

	assign("r1-1", d.labelBug, nil)
	assign("r1-2", d.labelBug, &d.valueHigh)
	assign("r1-2", d.labelArea, nil)
	assign("r1-3", d.labelArea, nil)
}

// prqDefineLabel creates a repository scoped label and returns its ID.
func prqDefineLabel(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	key string,
) int64 {
	repoID := prqRepoMain

	t.Helper()

	lbl := &types.Label{
		RepoID:    &repoID,
		Key:       key,
		Type:      enum.LabelTypeStatic,
		Color:     enum.LabelColorBlue,
		Created:   prqBaseTime,
		Updated:   prqBaseTime,
		CreatedBy: prqUserA,
		UpdatedBy: prqUserA,
	}
	require.NoError(t, database.NewLabelStore(db).Define(ctx, lbl))
	require.NotZero(t, lbl.ID)

	return lbl.ID
}

// prqDefineLabelValue creates a label value and returns its ID.
func prqDefineLabelValue(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	labelID int64,
	value string,
) int64 {
	t.Helper()

	val := &types.LabelValue{
		LabelID:   labelID,
		Value:     value,
		Color:     enum.LabelColorRed,
		Created:   prqBaseTime,
		Updated:   prqBaseTime,
		CreatedBy: prqUserA,
		UpdatedBy: prqUserA,
	}
	require.NoError(t, database.NewLabelValueStore(db).Define(ctx, val))
	require.NotZero(t, val.ID)

	return val.ID
}

func TestPrqPullReqStore_ListAndCountFilters(t *testing.T) {
	d := prqNewFilterData(t)

	boolTrue := true
	mergeable := enum.MergeCheckStatusMergeable
	conflict := enum.MergeCheckStatusConflict

	tests := []struct {
		name   string
		filter func() *types.PullReqFilter
		want   []string // keys in created order
	}{
		{
			name:   "no filter",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{} },
			want:   []string{"r1-1", "r1-2", "r1-3", "r1-4", "r1-5", "r2-1"},
		},
		{
			name: "single state",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{States: []enum.PullReqState{enum.PullReqStateOpen}}
			},
			want: []string{"r1-1", "r1-2", "r1-5", "r2-1"},
		},
		{
			name: "multiple states",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{States: []enum.PullReqState{enum.PullReqStateMerged, enum.PullReqStateClosed}}
			},
			want: []string{"r1-3", "r1-4"},
		},
		{
			name: "auto merge substate joins auto merges",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{SubStates: []enum.PullReqSubState{enum.PullReqSubStateAutoMerge}}
			},
			want: []string{"r1-5"},
		},
		{
			name: "multiple substates",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{
					SubStates: []enum.PullReqSubState{enum.PullReqSubStateNone, enum.PullReqSubStateMergeQueue},
				}
			},
			want: []string{"r1-1", "r1-2", "r1-3", "r1-4", "r2-1"},
		},
		{
			name:   "draft",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{IsDraft: &boolTrue} },
			want:   []string{"r1-2"},
		},
		{
			name:   "merge check status",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{MergeCheckStatus: &mergeable} },
			want:   []string{"r1-1"},
		},
		{
			name:   "rebase check status",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{RebaseCheckStatus: &conflict} },
			want:   []string{"r1-2"},
		},
		{
			name:   "source repo",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{SourceRepoID: prqRepoOther} },
			want:   []string{"r2-1"},
		},
		{
			name:   "source branch",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{SourceBranch: "other-branch"} },
			want:   []string{"r2-1"},
		},
		{
			name:   "source sha",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{SourceSHA: sha.Must(prqSHA2)} },
			want:   []string{"r1-2"},
		},
		{
			name:   "target repo",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{TargetRepoID: prqRepoOther} },
			want:   []string{"r2-1"},
		},
		{
			name:   "target branch",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{TargetBranch: "release"} },
			want:   []string{"r1-4"},
		},
		{
			name:   "query is a case insensitive partial match",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{Query: "LOGIN"} },
			want:   []string{"r1-1"},
		},
		{
			name:   "query escapes wildcards",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{Query: "%"} },
			want:   nil,
		},
		{
			name: "created range",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{
					CreatedFilter: types.CreatedFilter{CreatedGt: prqBaseTime + 100, CreatedLt: prqBaseTime + 400},
				}
			},
			want: []string{"r1-2", "r1-3"},
		},
		{
			name: "updated range",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{
					UpdatedFilter: types.UpdatedFilter{UpdatedGt: prqBaseTime + 401, UpdatedLt: prqBaseTime + 601},
				}
			},
			want: []string{"r1-5"},
		},
		{
			name: "edited range",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{
					EditedFilter: types.EditedFilter{EditedGt: prqBaseTime + 502, EditedLt: prqBaseTime + 700},
				}
			},
			want: []string{"r2-1"},
		},
		{
			name:   "single space",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{SpaceIDs: []int64{prqSpaceID}} },
			want:   []string{"r1-1", "r1-2", "r1-3", "r1-4", "r1-5", "r2-1"},
		},
		{
			name:   "multiple spaces",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{SpaceIDs: []int64{prqSpaceID, 99}} },
			want:   []string{"r1-1", "r1-2", "r1-3", "r1-4", "r1-5", "r2-1"},
		},
		{
			name:   "unknown space",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{SpaceIDs: []int64{99}} },
			want:   nil,
		},
		{
			name:   "single repo id",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{RepoIDs: []int64{prqRepoOther}} },
			want:   []string{"r2-1"},
		},
		{
			name:   "multiple repo ids",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{RepoIDs: []int64{prqRepoMain, prqRepoOther}} },
			want:   []string{"r1-1", "r1-2", "r1-3", "r1-4", "r1-5", "r2-1"},
		},
		{
			name:   "single repo blacklist",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{RepoIDBlacklist: []int64{prqRepoOther}} },
			want:   []string{"r1-1", "r1-2", "r1-3", "r1-4", "r1-5"},
		},
		{
			name: "multiple repo blacklist",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{RepoIDBlacklist: []int64{prqRepoMain, prqRepoOther}}
			},
			want: nil,
		},
		{
			name:   "single creator",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{CreatedBy: []int64{prqUserA}} },
			want:   []string{"r1-1", "r1-3", "r2-1"},
		},
		{
			name:   "multiple creators",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{CreatedBy: []int64{prqUserB, prqUserC}} },
			want:   []string{"r1-2", "r1-4", "r1-5"},
		},
		{
			name:   "commenter",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{CommenterID: prqUserB} },
			want:   []string{"r1-1", "r1-4"},
		},
		{
			name:   "mentioned",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{MentionedID: prqUserC} },
			want:   []string{"r1-1", "r1-4"},
		},
		{
			name:   "mentioned user without mentions",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{MentionedID: prqUserB} },
			want:   nil,
		},
		{
			name:   "reviewer",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{ReviewerID: prqUserB} },
			want:   []string{"r1-1", "r1-2"},
		},
		{
			name: "reviewer with decision",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{
					ReviewerID:      prqUserB,
					ReviewDecisions: []enum.PullReqReviewDecision{enum.PullReqReviewDecisionApproved},
				}
			},
			want: []string{"r1-1"},
		},
		{
			name:   "single label",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{LabelID: []int64{d.labelBug}} },
			want:   []string{"r1-1", "r1-2"},
		},
		{
			name:   "all of multiple labels",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{LabelID: []int64{d.labelBug, d.labelArea}} },
			want:   []string{"r1-2"},
		},
		{
			name:   "label value",
			filter: func() *types.PullReqFilter { return &types.PullReqFilter{ValueID: []int64{d.valueHigh}} },
			want:   []string{"r1-2"},
		},
		{
			name: "label and value",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{LabelID: []int64{d.labelArea}, ValueID: []int64{d.valueHigh}}
			},
			want: []string{"r1-2"},
		},
		{
			name: "combined filters",
			filter: func() *types.PullReqFilter {
				return &types.PullReqFilter{
					States:    []enum.PullReqState{enum.PullReqStateOpen},
					CreatedBy: []int64{prqUserB},
					RepoIDs:   []int64{prqRepoMain},
				}
			},
			want: []string{"r1-2", "r1-5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := tt.filter()
			filter.Sort = enum.PullReqSortCreated
			filter.Order = enum.OrderAsc

			list, err := d.pullreqs.List(d.ctx, filter)
			require.NoError(t, err)

			gotIDs := make([]int64, len(list))
			for i, pr := range list {
				gotIDs[i] = pr.ID
			}
			require.Equal(t, d.ids(tt.want...), gotIDs)

			count, err := d.pullreqs.Count(d.ctx, tt.filter())
			require.NoError(t, err)
			require.Equal(t, int64(len(tt.want)), count)
		})
	}
}

func TestPrqPullReqStore_ListSortingAndPagination(t *testing.T) {
	d := prqNewFilterData(t)

	list := func(filter *types.PullReqFilter) []int64 {
		t.Helper()
		prs, err := d.pullreqs.List(d.ctx, filter)
		require.NoError(t, err)
		out := make([]int64, len(prs))
		for i, pr := range prs {
			out[i] = pr.ID
		}
		return out
	}

	base := func() *types.PullReqFilter {
		return &types.PullReqFilter{TargetRepoID: prqRepoMain}
	}

	t.Run("default sort is number descending", func(t *testing.T) {
		require.Equal(t, d.ids("r1-5", "r1-4", "r1-3", "r1-2", "r1-1"), list(base()))
	})

	t.Run("invalid sort falls back to number", func(t *testing.T) {
		f := base()
		f.Sort = "bogus"
		f.Order = enum.OrderAsc
		require.Equal(t, d.ids("r1-1", "r1-2", "r1-3", "r1-4", "r1-5"), list(f))
	})

	t.Run("sort by updated descending", func(t *testing.T) {
		f := base()
		f.Sort = enum.PullReqSortUpdated
		f.Order = enum.OrderDesc
		require.Equal(t, d.ids("r1-5", "r1-4", "r1-3", "r1-2", "r1-1"), list(f))
	})

	t.Run("pagination", func(t *testing.T) {
		pageOf := func(page int) []int64 {
			f := base()
			f.Sort = enum.PullReqSortCreated
			f.Order = enum.OrderAsc
			f.Page = page
			f.Size = 2
			return list(f)
		}
		require.Equal(t, d.ids("r1-1", "r1-2"), pageOf(1))
		require.Equal(t, d.ids("r1-3", "r1-4"), pageOf(2))
		require.Equal(t, d.ids("r1-5"), pageOf(3))
		require.Empty(t, pageOf(4))
	})

	t.Run("exclude description", func(t *testing.T) {
		f := base()
		f.ExcludeDescription = true
		prs, err := d.pullreqs.List(d.ctx, f)
		require.NoError(t, err)
		require.Len(t, prs, 5)
		for _, pr := range prs {
			require.Empty(t, pr.Description)
			require.NotEmpty(t, pr.Title)
		}

		f.ExcludeDescription = false
		prs, err = d.pullreqs.List(d.ctx, f)
		require.NoError(t, err)
		for _, pr := range prs {
			require.NotEmpty(t, pr.Description)
		}
	})

	t.Run("author and merger are attached", func(t *testing.T) {
		f := base()
		f.States = []enum.PullReqState{enum.PullReqStateMerged}
		prs, err := d.pullreqs.List(d.ctx, f)
		require.NoError(t, err)
		require.Len(t, prs, 1)
		require.Equal(t, prqUserA, prs[0].Author.ID)
		require.NotNil(t, prs[0].Merger)
		require.Equal(t, prqUserB, prs[0].Merger.ID)
	})
}

func TestPrqPullReqStore_ListRecentAuthorIDs(t *testing.T) {
	f := prqNewFixture(t)

	// service principals are not users and must not be reported as authors.
	svc := &types.Service{UID: "prq_service", Email: "prq-service@example.com", Blocked: false}
	require.NoError(t, f.principals.CreateService(f.ctx, svc))

	authors := []int64{prqUserA, prqUserB, prqUserB, prqUserC, prqUserA, svc.ID}
	for i, author := range authors {
		f.prqCreatePullReq(t, prqRepoMain, int64(i+1), func(pr *types.PullReq) { pr.CreatedBy = author })
	}
	// pull request in another repo must be ignored.
	f.prqCreatePullReq(t, prqRepoOther, 1, func(pr *types.PullReq) { pr.CreatedBy = prqUserC })

	tests := []struct {
		name  string
		limit int
		want  []int64
	}{
		{name: "distinct users, newest first", limit: 10, want: []int64{prqUserA, prqUserC, prqUserB}},
		{name: "limit applied", limit: 2, want: []int64{prqUserA, prqUserC}},
		{name: "limit of one", limit: 1, want: []int64{prqUserA}},
		{name: "zero limit uses default", limit: 0, want: []int64{prqUserA, prqUserC, prqUserB}},
		{name: "huge limit is capped", limit: 100000, want: []int64{prqUserA, prqUserC, prqUserB}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.pullreqs.ListRecentAuthorIDs(f.ctx, prqRepoMain, tt.limit)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	got, err := f.pullreqs.ListRecentAuthorIDs(f.ctx, 9999, 5)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestPrqPullReqStore_Stream(t *testing.T) {
	d := prqNewFilterData(t)

	chPRs, chErr := d.pullreqs.Stream(d.ctx, &types.PullReqFilter{TargetRepoID: prqRepoMain})

	var got []int64
	for pr := range chPRs {
		require.NotZero(t, pr.Author.ID)
		got = append(got, pr.ID)
	}
	require.NoError(t, <-chErr)

	// streaming is ordered by the last update, newest first.
	require.Equal(t, d.ids("r1-5", "r1-4", "r1-3", "r1-2", "r1-1"), got)

	chPRs, chErr = d.pullreqs.Stream(d.ctx, &types.PullReqFilter{
		States: []enum.PullReqState{enum.PullReqStateMerged},
	})
	var merged []*types.PullReq
	for pr := range chPRs {
		merged = append(merged, pr)
	}
	require.NoError(t, <-chErr)
	require.Len(t, merged, 1)
	require.Equal(t, d.prs["r1-3"].ID, merged[0].ID)
	require.NotNil(t, merged[0].Merger)
}

func TestPrqPullReqStore_StreamError(t *testing.T) {
	d := prqNewFilterData(t)

	ctx, cancel := context.WithCancel(d.ctx)
	cancel()

	chPRs, chErr := d.pullreqs.Stream(ctx, &types.PullReqFilter{})
	for range chPRs {
		t.Fatal("no pull requests expected on a canceled context")
	}
	require.Error(t, <-chErr)
}

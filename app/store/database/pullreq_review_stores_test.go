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
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func prqNewReviewer(pr *types.PullReq, principalID int64, created int64) *types.PullReqReviewer {
	return &types.PullReqReviewer{
		PullReqID:      pr.ID,
		PrincipalID:    principalID,
		CreatedBy:      prqUserA,
		Created:        created,
		Updated:        created,
		RepoID:         pr.TargetRepoID,
		Type:           enum.PullReqReviewerTypeRequested,
		ReviewDecision: enum.PullReqReviewDecisionPending,
		SHA:            "reviewer-sha",
	}
}

func TestPrqPullReqReviewStore_CreateAndFind(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	reviews := database.NewPullReqReviewStore(f.db)

	in := &types.PullReqReview{
		CreatedBy: prqUserB,
		Created:   prqBaseTime,
		Updated:   prqBaseTime + 1,
		PullReqID: pr.ID,
		Decision:  enum.PullReqReviewDecisionApproved,
		SHA:       "review-sha",
	}
	require.NoError(t, reviews.Create(f.ctx, in))
	require.NotZero(t, in.ID)

	second := *in
	second.ID = 0
	second.Decision = enum.PullReqReviewDecisionChangeReq
	require.NoError(t, reviews.Create(f.ctx, &second))
	require.NotEqual(t, in.ID, second.ID)

	got, err := reviews.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, in, got)

	got, err = reviews.Find(f.ctx, second.ID)
	require.NoError(t, err)
	require.Equal(t, enum.PullReqReviewDecisionChangeReq, got.Decision)
}

func TestPrqPullReqReviewStore_Errors(t *testing.T) {
	f := prqNewFixture(t)
	reviews := database.NewPullReqReviewStore(f.db)

	_, err := reviews.Find(f.ctx, 404)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	err = reviews.Create(f.ctx, &types.PullReqReview{
		CreatedBy: prqUserA,
		PullReqID: 404,
		Decision:  enum.PullReqReviewDecisionApproved,
		SHA:       "sha",
	})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqPullReqReviewerStore_CRUD(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	reviewers := database.NewPullReqReviewerStore(f.db, f.pCache)
	reviews := database.NewPullReqReviewStore(f.db)

	require.NoError(t, reviewers.Create(f.ctx, prqNewReviewer(pr, prqUserB, prqBaseTime)))

	got, err := reviewers.Find(f.ctx, pr.ID, prqUserB)
	require.NoError(t, err)
	require.Equal(t, prqUserB, got.PrincipalID)
	require.Equal(t, enum.PullReqReviewerTypeRequested, got.Type)
	require.Equal(t, enum.PullReqReviewDecisionPending, got.ReviewDecision)
	require.Nil(t, got.LatestReviewID)
	require.Equal(t, prqUserB, got.Reviewer.ID)
	require.Equal(t, prqUserA, got.AddedBy.ID)

	review := &types.PullReqReview{
		CreatedBy: prqUserB,
		Created:   prqBaseTime,
		Updated:   prqBaseTime,
		PullReqID: pr.ID,
		Decision:  enum.PullReqReviewDecisionApproved,
		SHA:       "approved-sha",
	}
	require.NoError(t, reviews.Create(f.ctx, review))

	got.LatestReviewID = &review.ID
	got.ReviewDecision = enum.PullReqReviewDecisionApproved
	got.SHA = "approved-sha"
	got.Updated = 0
	require.NoError(t, reviewers.Update(f.ctx, got))
	require.NotZero(t, got.Updated)

	updated, err := reviewers.Find(f.ctx, pr.ID, prqUserB)
	require.NoError(t, err)
	require.NotNil(t, updated.LatestReviewID)
	require.Equal(t, review.ID, *updated.LatestReviewID)
	require.Equal(t, enum.PullReqReviewDecisionApproved, updated.ReviewDecision)
	require.Equal(t, "approved-sha", updated.SHA)
	require.Equal(t, got.Updated, updated.Updated)

	require.NoError(t, reviewers.Delete(f.ctx, pr.ID, prqUserB))
	_, err = reviewers.Find(f.ctx, pr.ID, prqUserB)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// deleting again is not an error.
	require.NoError(t, reviewers.Delete(f.ctx, pr.ID, prqUserB))
}

func TestPrqPullReqReviewerStore_ListAndCount(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	reviewers := database.NewPullReqReviewerStore(f.db, f.pCache)

	count, err := reviewers.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	list, err := reviewers.List(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Empty(t, list)

	// created out of order to verify ascending order by creation time.
	require.NoError(t, reviewers.Create(f.ctx, prqNewReviewer(pr, prqUserC, prqBaseTime+20)))
	require.NoError(t, reviewers.Create(f.ctx, prqNewReviewer(pr, prqUserB, prqBaseTime+10)))
	require.NoError(t, reviewers.Create(f.ctx, prqNewReviewer(other, prqUserB, prqBaseTime)))

	count, err = reviewers.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	list, err = reviewers.List(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, prqUserB, list[0].PrincipalID)
	require.Equal(t, prqUserC, list[1].PrincipalID)
	for _, r := range list {
		require.Equal(t, r.PrincipalID, r.Reviewer.ID)
		require.Equal(t, prqUserA, r.AddedBy.ID)
	}
}

func TestPrqPullReqReviewerStore_Errors(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	reviewers := database.NewPullReqReviewerStore(f.db, f.pCache)

	_, err := reviewers.Find(f.ctx, pr.ID, prqUserB)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.NoError(t, reviewers.Create(f.ctx, prqNewReviewer(pr, prqUserB, prqBaseTime)))
	err = reviewers.Create(f.ctx, prqNewReviewer(pr, prqUserB, prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = reviewers.Create(f.ctx, prqNewReviewer(pr, 9999, prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqUsergroupReviewerStore_CRUD(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)

	groups := database.NewUserGroupStore(f.db)
	groupA := &types.UserGroup{Identifier: "prq-group-a", Name: "Group A", Created: prqBaseTime, Updated: prqBaseTime}
	groupB := &types.UserGroup{Identifier: "prq-group-b", Name: "Group B", Created: prqBaseTime, Updated: prqBaseTime}
	require.NoError(t, groups.Create(f.ctx, prqSpaceID, groupA))
	require.NoError(t, groups.Create(f.ctx, prqSpaceID, groupB))

	store := database.NewUsergroupReviewerStore(f.db, f.pCache, groups)

	newReviewer := func(target *types.PullReq, groupID int64) *types.UserGroupReviewer {
		return &types.UserGroupReviewer{
			PullReqID:   target.ID,
			UserGroupID: groupID,
			CreatedBy:   prqUserA,
			Created:     prqBaseTime,
			Updated:     prqBaseTime + 1,
			RepoID:      target.TargetRepoID,
		}
	}

	count, err := store.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	require.NoError(t, store.Create(f.ctx, newReviewer(pr, groupA.ID)))
	require.NoError(t, store.Create(f.ctx, newReviewer(pr, groupB.ID)))
	require.NoError(t, store.Create(f.ctx, newReviewer(other, groupA.ID)))

	err = store.Create(f.ctx, newReviewer(pr, groupA.ID))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = store.Create(f.ctx, newReviewer(pr, 9999))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)

	count, err = store.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	got, err := store.Find(f.ctx, pr.ID, groupA.ID)
	require.NoError(t, err)
	require.Equal(t, groupA.ID, got.UserGroupID)
	require.Equal(t, pr.ID, got.PullReqID)
	require.Equal(t, prqUserA, got.CreatedBy)
	require.Equal(t, prqBaseTime, got.Created)
	require.Equal(t, prqBaseTime+1, got.Updated)
	require.Equal(t, prqRepoMain, got.RepoID)

	list, err := store.List(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	require.NoError(t, store.Delete(f.ctx, pr.ID, groupA.ID))
	_, err = store.Find(f.ctx, pr.ID, groupA.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// the same group on another pull request is unaffected.
	_, err = store.Find(f.ctx, other.ID, groupA.ID)
	require.NoError(t, err)

	list, err = store.List(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, groupB.ID, list[0].UserGroupID)

	// deleting twice is fine.
	require.NoError(t, store.Delete(f.ctx, pr.ID, groupA.ID))
}

func TestPrqPullReqFileViewStore_UpsertAndList(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	views := database.NewPullReqFileViewStore(f.db)

	newView := func(principal int64, path, sha string) *types.PullReqFileView {
		return &types.PullReqFileView{
			PullReqID:   pr.ID,
			PrincipalID: principal,
			Path:        path,
			SHA:         sha,
			Created:     prqBaseTime,
			Updated:     prqBaseTime,
		}
	}

	first := newView(prqUserA, "a.go", "sha-one")
	require.NoError(t, views.Upsert(f.ctx, first))
	require.Equal(t, prqBaseTime, first.Created)
	require.NoError(t, views.Upsert(f.ctx, newView(prqUserA, "b.go", "sha-two")))
	require.NoError(t, views.Upsert(f.ctx, newView(prqUserB, "a.go", "sha-three")))

	// a second upsert replaces sha and obsolete flag but keeps the original creation time.
	again := newView(prqUserA, "a.go", "sha-new")
	again.Created = prqBaseTime + 500
	again.Updated = prqBaseTime + 600
	require.NoError(t, views.Upsert(f.ctx, again))
	require.Equal(t, prqBaseTime, again.Created)

	list, err := views.List(f.ctx, pr.ID, prqUserA)
	require.NoError(t, err)
	require.Len(t, list, 2)

	byPath := map[string]*types.PullReqFileView{}
	for _, v := range list {
		byPath[v.Path] = v
	}
	require.Equal(t, "sha-new", byPath["a.go"].SHA)
	require.Equal(t, prqBaseTime+600, byPath["a.go"].Updated)
	require.Equal(t, prqBaseTime, byPath["a.go"].Created)
	require.False(t, byPath["a.go"].Obsolete)
	require.Equal(t, "sha-two", byPath["b.go"].SHA)

	list, err = views.List(f.ctx, pr.ID, prqUserC)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestPrqPullReqFileViewStore_MarkObsoleteAndDelete(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	views := database.NewPullReqFileViewStore(f.db)

	for _, target := range []*types.PullReq{pr, other} {
		for _, principal := range []int64{prqUserA, prqUserB} {
			for _, path := range []string{"a.go", "b.go", "c.go"} {
				require.NoError(t, views.Upsert(f.ctx, &types.PullReqFileView{
					PullReqID:   target.ID,
					PrincipalID: principal,
					Path:        path,
					SHA:         "sha",
					Created:     prqBaseTime,
					Updated:     prqBaseTime,
				}))
			}
		}
	}

	require.NoError(t, views.MarkObsolete(f.ctx, pr.ID, []string{"a.go", "c.go"}))

	obsolete := func(target *types.PullReq, principal int64) map[string]bool {
		t.Helper()
		list, err := views.List(f.ctx, target.ID, principal)
		require.NoError(t, err)
		out := map[string]bool{}
		for _, v := range list {
			out[v.Path] = v.Obsolete
		}
		return out
	}

	want := map[string]bool{"a.go": true, "b.go": false, "c.go": true}
	require.Equal(t, want, obsolete(pr, prqUserA))
	require.Equal(t, want, obsolete(pr, prqUserB))
	require.Equal(t, map[string]bool{"a.go": false, "b.go": false, "c.go": false}, obsolete(other, prqUserA))

	// viewing the file again resets the obsolete flag.
	require.NoError(t, views.Upsert(f.ctx, &types.PullReqFileView{
		PullReqID:   pr.ID,
		PrincipalID: prqUserA,
		Path:        "a.go",
		SHA:         "sha-latest",
		Created:     prqBaseTime,
		Updated:     prqBaseTime,
	}))
	require.Equal(t, map[string]bool{"a.go": false, "b.go": false, "c.go": true}, obsolete(pr, prqUserA))
}

func prqFileGroupStoreWithPR(
	t *testing.T,
) (context.Context, *database.PullReqFileGroupStore, *types.PullReq, *types.PullReq) {
	t.Helper()

	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)

	return f.ctx, database.NewPullReqFileGroupStore(f.db), pr, other
}

func TestPrqPullReqFileGroupStore_CreateListDelete(t *testing.T) {
	ctx, groups, pr, other := prqFileGroupStoreWithPR(t)

	newGroup := func(
		target *types.PullReq,
		title string,
		files ...*types.PullReqFileGroupFile,
	) *types.PullReqFileGroupWithFiles {
		return &types.PullReqFileGroupWithFiles{
			PullReqFileGroup: types.PullReqFileGroup{
				PullReqID:   target.ID,
				Title:       title,
				Description: title + " description",
				Tags:        map[string]string{"kind": "prq"},
				Created:     prqBaseTime,
				Updated:     prqBaseTime + 1,
				CreatedBy:   prqUserA,
				UpdatedBy:   prqUserB,
			},
			Files: files,
		}
	}

	in := []*types.PullReqFileGroupWithFiles{
		newGroup(pr, "Backend",
			&types.PullReqFileGroupFile{Path: "z.go", OldSHA: "old-z", NewSHA: "new-z"},
			&types.PullReqFileGroupFile{Path: "a.go", OldSHA: types.NilSHA, NewSHA: "new-a"},
			&types.PullReqFileGroupFile{Path: "m.go"},
		),
		newGroup(pr, "Empty group"),
		newGroup(other, "Other PR", &types.PullReqFileGroupFile{Path: "o.go", NewSHA: "new-o"}),
	}
	require.NoError(t, groups.CreateMany(ctx, in))
	require.NotZero(t, in[0].ID)
	require.NotEqual(t, in[0].ID, in[1].ID)

	list, err := groups.List(ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)

	backend := list[0]
	require.Equal(t, in[0].ID, backend.ID)
	require.Equal(t, "Backend", backend.Title)
	require.Equal(t, "Backend description", backend.Description)
	require.Equal(t, map[string]string{"kind": "prq"}, backend.Tags)
	require.Equal(t, prqBaseTime, backend.Created)
	require.Equal(t, prqBaseTime+1, backend.Updated)
	require.Equal(t, prqUserA, backend.CreatedBy)
	require.Equal(t, prqUserB, backend.UpdatedBy)

	// files are ordered by path; empty and nil SHAs are normalized to empty values.
	require.Len(t, backend.Files, 3)
	require.Equal(t, "a.go", backend.Files[0].Path)
	require.Equal(t, "", backend.Files[0].OldSHA)
	require.Equal(t, "new-a", backend.Files[0].NewSHA)
	require.Equal(t, "m.go", backend.Files[1].Path)
	require.Equal(t, "", backend.Files[1].NewSHA)
	require.Equal(t, "z.go", backend.Files[2].Path)
	require.Equal(t, "old-z", backend.Files[2].OldSHA)
	require.Equal(t, backend.ID, backend.Files[2].PullReqFileGroupID)

	require.Equal(t, "Empty group", list[1].Title)
	require.NotNil(t, list[1].Files)
	require.Empty(t, list[1].Files)

	require.NoError(t, groups.DeleteByPrID(ctx, pr.ID))

	list, err = groups.List(ctx, pr.ID)
	require.NoError(t, err)
	require.Empty(t, list)

	// groups of other pull requests remain.
	list, err = groups.List(ctx, other.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].Files, 1)

	// deleting again is not an error.
	require.NoError(t, groups.DeleteByPrID(ctx, pr.ID))
}

func TestPrqPullReqFileGroupStore_Errors(t *testing.T) {
	ctx, groups, pr, _ := prqFileGroupStoreWithPR(t)

	group := func(title string) *types.PullReqFileGroupWithFiles {
		return &types.PullReqFileGroupWithFiles{
			PullReqFileGroup: types.PullReqFileGroup{
				PullReqID: pr.ID, Title: title, CreatedBy: prqUserA, UpdatedBy: prqUserA,
			},
		}
	}

	require.NoError(t, groups.CreateMany(ctx, []*types.PullReqFileGroupWithFiles{group("same")}))

	err := groups.CreateMany(ctx, []*types.PullReqFileGroupWithFiles{group("same")})
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	// the same file path can't be part of two groups of the same pull request.
	files := func(title string) *types.PullReqFileGroupWithFiles {
		g := group(title)
		g.Files = []*types.PullReqFileGroupFile{{Path: "dup.go"}}
		return g
	}
	require.NoError(t, groups.CreateMany(ctx, []*types.PullReqFileGroupWithFiles{files("first")}))
	err = groups.CreateMany(ctx, []*types.PullReqFileGroupWithFiles{files("second")})
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	// nothing to do.
	require.NoError(t, groups.CreateMany(ctx, nil))
}

func TestPrqPullReqFileViewStore_DeleteByFileForPrincipal(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	views := database.NewPullReqFileViewStore(f.db)

	for _, principal := range []int64{prqUserA, prqUserB} {
		for _, path := range []string{"a.go", "b.go"} {
			require.NoError(t, views.Upsert(f.ctx, &types.PullReqFileView{
				PullReqID:   pr.ID,
				PrincipalID: principal,
				Path:        path,
				SHA:         "sha",
				Created:     prqBaseTime,
				Updated:     prqBaseTime,
			}))
		}
	}

	require.NoError(t, views.DeleteByFileForPrincipal(f.ctx, pr.ID, prqUserA, "a.go"))
	// deleting a non existing entry is fine.
	require.NoError(t, views.DeleteByFileForPrincipal(f.ctx, pr.ID, prqUserA, "a.go"))

	paths := func(principal int64) []string {
		t.Helper()
		list, err := views.List(f.ctx, pr.ID, principal)
		require.NoError(t, err)
		out := make([]string, len(list))
		for i, v := range list {
			out[i] = v.Path
		}
		return out
	}

	require.Equal(t, []string{"b.go"}, paths(prqUserA))
	require.ElementsMatch(t, []string{"a.go", "b.go"}, paths(prqUserB))
}

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
	"encoding/json"
	"testing"
	"time"

	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func rstRepoIDs(repos []*types.Repository) []int64 {
	ids := make([]int64, len(repos))
	for i, r := range repos {
		ids[i] = r.ID
	}

	return ids
}

func rstRepoIdents(repos []*types.Repository) []string {
	idents := make([]string, len(repos))
	for i, r := range repos {
		idents[i] = r.Identifier
	}

	return idents
}

func TestRstRepoStore_FindByUID(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)
	repo := f.rstMkRepo(ctx, t, tree["beta"].ID, "MyRepo")
	require.Equal(t, "alpha/beta/MyRepo", repo.Path)

	t.Run("active by uid is case insensitive", func(t *testing.T) {
		got, err := f.repos.FindActiveByUID(ctx, tree["beta"].ID, "myrepo")
		require.NoError(t, err)
		require.Equal(t, repo.ID, got.ID)
		require.Equal(t, enum.RepoTypeNormal, got.Type)
		require.Equal(t, "alpha/beta/MyRepo", got.Path)
	})

	t.Run("active by uid wrong parent", func(t *testing.T) {
		_, err := f.repos.FindActiveByUID(ctx, tree["alpha"].ID, "myrepo")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("deleted by uid", func(t *testing.T) {
		deletedAt := time.Now().UnixMilli()
		require.NoError(t, f.repos.SoftDelete(ctx, repo, deletedAt))

		_, err := f.repos.FindActiveByUID(ctx, tree["beta"].ID, "myrepo")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		got, err := f.repos.FindDeletedByUID(ctx, tree["beta"].ID, "MYREPO", deletedAt)
		require.NoError(t, err)
		require.Equal(t, repo.ID, got.ID)
		require.NotNil(t, got.Deleted)

		_, err = f.repos.FindDeletedByUID(ctx, tree["beta"].ID, "myrepo", deletedAt+1)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		got, err = f.repos.FindDeleted(ctx, repo.ID, &deletedAt)
		require.NoError(t, err)
		require.Equal(t, repo.ID, got.ID)
	})
}

func TestRstRepoStore_Lifecycle(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)
	beta := tree["beta"].ID

	repo := f.rstMkRepo(ctx, t, beta, "lifecycle")

	t.Run("create duplicate identifier", func(t *testing.T) {
		err := f.repos.Create(ctx, &types.Repository{
			Identifier: "lifecycle", GitUID: "other-git-uid", ParentID: beta, CreatedBy: userID,
			Tags: json.RawMessage(`{}`),
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("update persists fields and detects conflicts", func(t *testing.T) {
		stale := *repo

		repo.Description = "updated description"
		repo.DefaultBranch = "develop"
		repo.State = enum.RepoStateArchived
		repo.IsEmpty = true
		repo.Language = "go"
		repo.Tags = json.RawMessage(`{"team":"core"}`)
		require.NoError(t, f.repos.Update(ctx, repo))
		require.Equal(t, int64(1), repo.Version)

		got, err := f.repos.Find(ctx, repo.ID)
		require.NoError(t, err)
		require.Equal(t, "updated description", got.Description)
		require.Equal(t, "develop", got.DefaultBranch)
		require.Equal(t, enum.RepoStateArchived, got.State)
		require.True(t, got.IsEmpty)
		require.Equal(t, "go", got.Language)
		require.JSONEq(t, `{"team":"core"}`, string(got.Tags))

		stale.Description = rstValStale
		require.ErrorIs(t, f.repos.Update(ctx, &stale), gitness_store.ErrVersionConflict)
	})

	t.Run("update opt lock retries after conflict", func(t *testing.T) {
		current, err := f.repos.Find(ctx, repo.ID)
		require.NoError(t, err)

		concurrent := *current
		concurrent.Description = "concurrent"
		require.NoError(t, f.repos.Update(ctx, &concurrent))

		calls := 0
		updated, err := f.repos.UpdateOptLock(ctx, current, func(r *types.Repository) error {
			calls++
			r.NumPulls = 7
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, 7, updated.NumPulls)
		// the concurrent change is preserved by the retry.
		require.Equal(t, "concurrent", updated.Description)
	})

	t.Run("update size and lfs size", func(t *testing.T) {
		require.NoError(t, f.repos.UpdateSize(ctx, repo.ID, 120, 30))

		size, err := f.repos.GetSize(ctx, repo.ID)
		require.NoError(t, err)
		require.Equal(t, int64(120), size)

		lfs, err := f.repos.GetLFSSize(ctx, repo.ID)
		require.NoError(t, err)
		require.Equal(t, int64(30), lfs)

		require.ErrorIs(t, f.repos.UpdateSize(ctx, 9999, 1, 1), gitness_store.ErrResourceNotFound)

		_, err = f.repos.GetLFSSize(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("fork counters and fork id", func(t *testing.T) {
		upstream := f.rstMkRepo(ctx, t, beta, "upstream")
		fork := &types.Repository{
			Identifier: "fork", GitUID: "uid-fork", ParentID: beta, CreatedBy: userID, ForkID: upstream.ID,
			Tags: json.RawMessage(`{}`),
		}
		require.NoError(t, f.repos.Create(ctx, fork))

		require.NoError(t, f.repos.UpdateNumForks(ctx, upstream.ID, 2))
		require.NoError(t, f.repos.UpdateNumForks(ctx, upstream.ID, -1))
		got, err := f.repos.Find(ctx, upstream.ID)
		require.NoError(t, err)
		require.Equal(t, 1, got.NumForks)

		gotFork, err := f.repos.Find(ctx, fork.ID)
		require.NoError(t, err)
		require.Equal(t, upstream.ID, gotFork.ForkID)

		require.NoError(t, f.repos.ClearForkID(ctx, upstream.ID))
		gotFork, err = f.repos.Find(ctx, fork.ID)
		require.NoError(t, err)
		require.Zero(t, gotFork.ForkID)
	})

	t.Run("update root space", func(t *testing.T) {
		require.NoError(t, f.repos.UpdateRootSpace(ctx, nil, 1, "alpha"))
		require.NoError(t, f.repos.UpdateRootSpace(ctx, []int64{repo.ID}, tree["alpha"].ID, "alpha"))

		got, err := f.repos.Find(ctx, repo.ID)
		require.NoError(t, err)
		require.Equal(t, tree["alpha"].ID, got.RootSpaceID)
		require.Equal(t, "alpha", got.RootSpaceIdentifier)
	})
}

func TestRstRepoStore_SoftDeleteRestorePurge(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)
	beta, epsilon := tree["beta"].ID, tree["epsilon"].ID

	repo := f.rstMkRepo(ctx, t, beta, "doomed")
	deletedAt := time.Now().UnixMilli()

	require.NoError(t, f.repos.SoftDelete(ctx, repo, deletedAt))

	_, err := f.repos.Find(ctx, repo.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	deleted, err := f.repos.FindDeleted(ctx, repo.ID, &deletedAt)
	require.NoError(t, err)

	t.Run("opt lock refuses to modify deleted repo", func(t *testing.T) {
		_, err := f.repos.UpdateOptLock(ctx, deleted, func(*types.Repository) error { return nil })
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("restore refuses active repo", func(t *testing.T) {
		active := f.rstMkRepo(ctx, t, beta, "active")
		_, err := f.repos.Restore(ctx, active, nil, nil)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("restore into other space with new identifier", func(t *testing.T) {
		ident := "revived"
		restored, err := f.repos.Restore(ctx, deleted, &ident, &epsilon)
		require.NoError(t, err)
		require.Nil(t, restored.Deleted)
		require.Equal(t, "alpha/epsilon/revived", restored.Path)

		got, err := f.repos.Find(ctx, repo.ID)
		require.NoError(t, err)
		require.Equal(t, epsilon, got.ParentID)
		require.Equal(t, ident, got.Identifier)
	})

	t.Run("purge", func(t *testing.T) {
		// a deleted timestamp that does not match leaves the active repo untouched.
		require.NoError(t, f.repos.Purge(ctx, repo.ID, &deletedAt))
		_, err := f.repos.Find(ctx, repo.ID)
		require.NoError(t, err)

		require.NoError(t, f.repos.Purge(ctx, repo.ID, nil))
		_, err = f.repos.Find(ctx, repo.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("purge soft deleted", func(t *testing.T) {
		other := f.rstMkRepo(ctx, t, beta, "other")
		require.NoError(t, f.repos.SoftDelete(ctx, other, deletedAt))
		require.NoError(t, f.repos.Purge(ctx, other.ID, &deletedAt))
		_, err := f.repos.FindDeleted(ctx, other.ID, &deletedAt)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func TestRstRepoStore_ListAndCount(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)
	alpha, beta, gamma := tree["alpha"].ID, tree["beta"].ID, tree["gamma"].ID

	// created in a non-alphabetical order so that the identifier sort is distinguishable.
	rBravo := f.rstMkRepo(ctx, t, beta, "bravo")
	rAlpha := f.rstMkRepo(ctx, t, beta, "alpha-repo")
	rCharlie := f.rstMkRepo(ctx, t, gamma, "charlie")
	rDelta := f.rstMkRepo(ctx, t, alpha, "delta-repo")

	// distinct sort keys
	for i, r := range []*types.Repository{rBravo, rAlpha, rCharlie, rDelta} {
		current, err := f.repos.Find(ctx, r.ID)
		require.NoError(t, err)
		_, err = f.repos.UpdateOptLock(ctx, current, func(r *types.Repository) error {
			r.Language = "lang"
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, f.repos.UpdateLastGITPush(ctx, r.ID, int64(100-i)))
	}

	// tags
	tagged, err := f.repos.Find(ctx, rCharlie.ID)
	require.NoError(t, err)
	_, err = f.repos.UpdateOptLock(ctx, tagged, func(r *types.Repository) error {
		r.Tags = json.RawMessage(`{"team":"core","tier":"gold"}`)
		return nil
	})
	require.NoError(t, err)
	tagged, err = f.repos.Find(ctx, rBravo.ID)
	require.NoError(t, err)
	_, err = f.repos.UpdateOptLock(ctx, tagged, func(r *types.Repository) error {
		r.Tags = json.RawMessage(`{"team":"infra"}`)
		return nil
	})
	require.NoError(t, err)

	// a favourite for the user
	_, err = f.db.ExecContext(ctx,
		`INSERT INTO favorite_repos (favorite_principal_id, favorite_repo_id, favorite_created) VALUES (?, ?, ?)`,
		userID, rAlpha.ID, 1)
	require.NoError(t, err)

	asc := enum.OrderAsc
	favUser := userID

	tests := []struct {
		name      string
		spaceID   int64
		filter    types.RepoFilter
		wantIdent []string
	}{
		{"direct", beta, types.RepoFilter{Order: asc}, []string{"alpha-repo", "bravo"}},
		{"direct desc", beta, types.RepoFilter{Order: enum.OrderDesc, Sort: enum.RepoAttrIdentifier},
			[]string{"bravo", "alpha-repo"}},
		{"recursive", alpha, types.RepoFilter{Order: asc, Recursive: true},
			[]string{"alpha-repo", "bravo", "charlie", "delta-repo"}},
		{"query", alpha, types.RepoFilter{Order: asc, Recursive: true, Query: "ar"}, []string{"charlie"}},
		{"identifiers", alpha, types.RepoFilter{Order: asc, Recursive: true, Identifiers: []string{"BRAVO", "delta-repo"}},
			[]string{"bravo", "delta-repo"}},
		{"paging", alpha, types.RepoFilter{Order: asc, Recursive: true, Page: 2, Size: 2},
			[]string{"charlie", "delta-repo"}},
		{"sort created", beta, types.RepoFilter{Order: asc, Sort: enum.RepoAttrCreated},
			[]string{"bravo", "alpha-repo"}},
		{"sort updated", beta, types.RepoFilter{Order: asc, Sort: enum.RepoAttrUpdated},
			[]string{"bravo", "alpha-repo"}},
		{"sort deleted", beta, types.RepoFilter{Order: asc, Sort: enum.RepoAttrDeleted},
			[]string{"bravo", "alpha-repo"}},
		{"sort last git push", beta, types.RepoFilter{Order: asc, Sort: enum.RepoAttrLastGITPush},
			[]string{"alpha-repo", "bravo"}},
		{"tag key only", alpha, types.RepoFilter{Order: asc, Recursive: true, Tags: map[string][]string{"tier": nil}},
			[]string{"charlie"}},
		{"tag key value", alpha,
			types.RepoFilter{Order: asc, Recursive: true, Tags: map[string][]string{"team": {"core", "infra"}}},
			[]string{"bravo", "charlie"}},
		{"tag empty key", alpha,
			types.RepoFilter{Order: asc, Recursive: true, Tags: map[string][]string{"": {"x"}}}, []string{}},
		{"favorites", alpha, types.RepoFilter{Order: asc, Recursive: true, OnlyFavoritesFor: &favUser},
			[]string{"alpha-repo"}},
		{"normal type", alpha, types.RepoFilter{Order: asc, Recursive: true, Types: []enum.RepoType{enum.RepoTypeNormal}},
			[]string{"alpha-repo", "bravo", "charlie", "delta-repo"}},
		{"linked type", alpha, types.RepoFilter{Order: asc, Recursive: true, Types: []enum.RepoType{enum.RepoTypeLinked}},
			[]string{}},
	}

	// rows with identical sort keys have no defined order.
	anyOrder := map[string]bool{"sort created": true, "sort updated": true, "sort deleted": true}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.repos.List(ctx, tt.spaceID, &tt.filter)
			require.NoError(t, err)
			if anyOrder[tt.name] {
				require.ElementsMatch(t, tt.wantIdent, rstRepoIdents(got))
			} else {
				require.Equal(t, tt.wantIdent, rstRepoIdents(got))
			}

			if tt.filter.Size == 0 {
				count, err := f.repos.Count(ctx, tt.spaceID, &tt.filter)
				require.NoError(t, err)
				require.Equal(t, int64(len(tt.wantIdent)), count)
			}
		})
	}

	t.Run("list all and count all", func(t *testing.T) {
		all, err := f.repos.ListAll(ctx, &types.RepoFilter{Order: asc})
		require.NoError(t, err)
		require.Equal(t, []string{"alpha-repo", "bravo", "charlie", "delta-repo"}, rstRepoIdents(all))

		all, err = f.repos.ListAll(ctx, &types.RepoFilter{Order: asc, Query: "brav"})
		require.NoError(t, err)
		require.Equal(t, []string{"bravo"}, rstRepoIdents(all))

		count, err := f.repos.Count(ctx, 0, &types.RepoFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(4), count)
	})

	t.Run("count by root spaces", func(t *testing.T) {
		counts, err := f.repos.CountByRootSpaces(ctx)
		require.NoError(t, err)
		require.Len(t, counts, 1)
		require.Equal(t, alpha, counts[0].SpaceID)
		require.Equal(t, 4, counts[0].Total)
	})

	t.Run("deleted filters", func(t *testing.T) {
		current, err := f.repos.Find(ctx, rDelta.ID)
		require.NoError(t, err)
		deletedAt := time.Now().UnixMilli()
		require.NoError(t, f.repos.SoftDelete(ctx, current, deletedAt))

		got, err := f.repos.List(ctx, alpha, &types.RepoFilter{DeletedAt: &deletedAt})
		require.NoError(t, err)
		require.Equal(t, []int64{rDelta.ID}, rstRepoIDs(got))

		got, err = f.repos.List(ctx, alpha, &types.RepoFilter{DeletedBeforeOrAt: &deletedAt})
		require.NoError(t, err)
		require.Equal(t, []int64{rDelta.ID}, rstRepoIDs(got))
	})

	t.Run("ids by parent spaces", func(t *testing.T) {
		ids, err := f.repos.ListIDsByParentSpaceIDs(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, ids)

		ids, err = f.repos.ListIDsByParentSpaceIDs(ctx, []int64{beta, gamma})
		require.NoError(t, err)
		require.ElementsMatch(t, []int64{rBravo.ID, rAlpha.ID, rCharlie.ID}, ids)
	})

	t.Run("map of all repos", func(t *testing.T) {
		got, err := f.repos.MapOfAllRepos(ctx, beta, false)
		require.NoError(t, err)
		require.Equal(t, map[int64]string{
			rBravo.ID: "alpha/beta/bravo",
			rAlpha.ID: "alpha/beta/alpha-repo",
		}, got)

		got, err = f.repos.MapOfAllRepos(ctx, alpha, true)
		require.NoError(t, err)
		require.Equal(t, map[int64]string{
			rBravo.ID:   "alpha/beta/bravo",
			rAlpha.ID:   "alpha/beta/alpha-repo",
			rCharlie.ID: "alpha/beta/gamma/charlie",
		}, got)
	})

	t.Run("update parent", func(t *testing.T) {
		moved, err := f.repos.UpdateParent(ctx, gamma, alpha)
		require.NoError(t, err)
		require.Equal(t, int64(1), moved)

		got, err := f.repos.Find(ctx, rCharlie.ID)
		require.NoError(t, err)
		require.Equal(t, alpha, got.ParentID)

		moved, err = f.repos.UpdateParent(ctx, gamma, alpha)
		require.NoError(t, err)
		require.Zero(t, moved)
	})
}

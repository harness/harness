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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/cache"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	prqUserA int64 = 1
	prqUserB int64 = 2
	prqUserC int64 = 3

	prqRepoMain  int64 = 1
	prqRepoOther int64 = 2

	prqSpaceID int64 = 1

	prqBranchRelease = "release"
	prqBranchAlpha   = "alpha"
)

// prqFixture bundles a migrated SQLite database with the stores needed by the pull request store tests.
type prqFixture struct {
	ctx        context.Context
	db         *sqlx.DB
	principals *database.PrincipalStore
	spaces     *database.SpaceStore
	repos      *database.RepoStore
	pullreqs   *database.PullReqStore
	pCache     store.PrincipalInfoCache
}

// prqNewFixture creates a database containing three users, one space and two repositories.
func prqNewFixture(t *testing.T) *prqFixture {
	t.Helper()

	db, teardown := setupDB(t)
	t.Cleanup(teardown)

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	insertPrincipalRaw(ctx, t, principalStore, prqUserB)
	insertPrincipalRaw(ctx, t, principalStore, prqUserC)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, prqSpaceID, 0)
	createRepo(ctx, t, repoStore, prqRepoMain, prqSpaceID, 0)
	createRepo(ctx, t, repoStore, prqRepoOther, prqSpaceID, 0)

	pCache := cache.NewExtended[int64, *types.PrincipalInfo](database.NewPrincipalInfoView(db), time.Minute)

	return &prqFixture{
		ctx:        ctx,
		db:         db,
		principals: principalStore,
		spaces:     spaceStore,
		repos:      repoStore,
		pullreqs:   database.NewPullReqStore(db, pCache),
		pCache:     pCache,
	}
}

// prqNewPullReq builds a valid open pull request for the given target repository.
func prqNewPullReq(repoID, number int64) *types.PullReq {
	now := time.Now().UnixMilli()
	return &types.PullReq{
		Number:              number,
		RootSpaceID:         prqSpaceID,
		RootSpaceIdentifier: "root",
		CreatedBy:           prqUserA,
		Created:             now,
		Updated:             now,
		Edited:              now,
		State:               enum.PullReqStateOpen,
		Title:               fmt.Sprintf("Title %d", number),
		Description:         fmt.Sprintf("Description %d", number),
		SourceRepoID:        &repoID,
		SourceBranch:        fmt.Sprintf("feature-%d", number),
		SourceSHA:           fmt.Sprintf("source-sha-%d", number),
		TargetRepoID:        repoID,
		TargetBranch:        "main",
		MergeBaseSHA:        "merge-base-sha",
		MergeCheckStatus:    enum.MergeCheckStatusUnchecked,
		RebaseCheckStatus:   enum.MergeCheckStatusUnchecked,
	}
}

// prqCreatePullReq creates a pull request and applies the optional mutators before persisting it.
func (f *prqFixture) prqCreatePullReq(
	t *testing.T,
	repoID, number int64,
	mutators ...func(pr *types.PullReq),
) *types.PullReq {
	t.Helper()

	pr := prqNewPullReq(repoID, number)
	for _, mutate := range mutators {
		mutate(pr)
	}
	require.NoError(t, f.pullreqs.Create(f.ctx, pr))
	require.NotZero(t, pr.ID)

	return pr
}

func TestPrqPullReqStore_CreateAndFind(t *testing.T) {
	f := prqNewFixture(t)

	closed := int64(1700000000100)
	merged := int64(1700000000200)
	mergedBy := prqUserB
	method := enum.MergeMethodSquash
	mergeSHA := "merge-sha"
	targetSHA := "target-sha"
	bypassed := true
	commits, files, adds, dels := int64(3), int64(4), int64(50), int64(6)
	prType := enum.PullReqTypeLinked

	in := prqNewPullReq(prqRepoMain, 1)
	in.State = enum.PullReqStateMerged
	in.SubState = enum.PullReqSubStateAutoMerge
	in.IsDraft = true
	in.Closed = &closed
	in.Merged = &merged
	in.MergedBy = &mergedBy
	in.MergeMethod = &method
	in.MergeSHA = &mergeSHA
	in.MergeTargetSHA = &targetSHA
	in.MergeViolationsBypassed = &bypassed
	in.MergeConflicts = []string{"a.go", "b.go"}
	in.RebaseConflicts = []string{"c.go"}
	in.CommentCount = 5
	in.UnresolvedCount = 2
	in.Stats.Commits = &commits
	in.Stats.FilesChanged = &files
	in.Stats.Additions = &adds
	in.Stats.Deletions = &dels
	in.Type = &prType

	require.NoError(t, f.pullreqs.Create(f.ctx, in))
	require.NotZero(t, in.ID)

	byID, err := f.pullreqs.Find(f.ctx, in.ID)
	require.NoError(t, err)
	byNumber, err := f.pullreqs.FindByNumber(f.ctx, prqRepoMain, 1)
	require.NoError(t, err)
	require.Equal(t, byID, byNumber)

	require.Equal(t, in.ID, byID.ID)
	require.Equal(t, int64(1), byID.Number)
	require.Equal(t, enum.PullReqStateMerged, byID.State)
	require.Equal(t, enum.PullReqSubStateAutoMerge, byID.SubState)
	require.True(t, byID.IsDraft)
	require.Equal(t, &closed, byID.Closed)
	require.Equal(t, &merged, byID.Merged)
	require.Equal(t, &mergedBy, byID.MergedBy)
	require.Equal(t, &method, byID.MergeMethod)
	require.Equal(t, &mergeSHA, byID.MergeSHA)
	require.Equal(t, &targetSHA, byID.MergeTargetSHA)
	require.Equal(t, &bypassed, byID.MergeViolationsBypassed)
	require.Equal(t, []string{"a.go", "b.go"}, byID.MergeConflicts)
	require.Equal(t, []string{"c.go"}, byID.RebaseConflicts)
	require.Equal(t, 5, byID.CommentCount)
	require.Equal(t, 2, byID.UnresolvedCount)
	require.Equal(t, &commits, byID.Stats.Commits)
	require.Equal(t, &files, byID.Stats.FilesChanged)
	require.Equal(t, &adds, byID.Stats.Additions)
	require.Equal(t, &dels, byID.Stats.Deletions)
	require.Equal(t, 5, byID.Stats.Conversations)
	require.NotNil(t, byID.Type)
	require.Equal(t, enum.PullReqTypeLinked, *byID.Type)
	require.Equal(t, "Description 1", byID.Description)
	require.Equal(t, prqSpaceID, byID.RootSpaceID)
	require.Equal(t, "root", byID.RootSpaceIdentifier)

	// the author and merger principal infos are resolved through the cache.
	require.Equal(t, prqUserA, byID.Author.ID)
	require.NotNil(t, byID.Merger)
	require.Equal(t, prqUserB, byID.Merger.ID)
}

func TestPrqPullReqStore_CreateWithoutOptionalFields(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	got, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Nil(t, got.Closed)
	require.Nil(t, got.Merged)
	require.Nil(t, got.MergedBy)
	require.Nil(t, got.Merger)
	require.Nil(t, got.MergeMethod)
	require.Empty(t, got.MergeConflicts)
	require.Empty(t, got.RebaseConflicts)
	require.Nil(t, got.Stats.Commits)
	require.Equal(t, enum.PullReqStateOpen, got.State)
}

func TestPrqPullReqStore_FindNotFound(t *testing.T) {
	f := prqNewFixture(t)

	_, err := f.pullreqs.Find(f.ctx, 12345)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	_, err = f.pullreqs.FindWithLock(f.ctx, 12345)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	_, err = f.pullreqs.FindByNumber(f.ctx, prqRepoMain, 99)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	_, err = f.pullreqs.FindByNumberWithLock(f.ctx, prqRepoMain, 99)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqStore_FindWithLock(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 7)

	byID, err := f.pullreqs.FindWithLock(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, pr.ID, byID.ID)

	byNumber, err := f.pullreqs.FindByNumberWithLock(f.ctx, prqRepoMain, 7)
	require.NoError(t, err)
	require.Equal(t, pr.ID, byNumber.ID)

	// the same number in a different target repository is a different pull request.
	_, err = f.pullreqs.FindByNumberWithLock(f.ctx, prqRepoOther, 7)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqStore_CreateErrors(t *testing.T) {
	f := prqNewFixture(t)

	f.prqCreatePullReq(t, prqRepoMain, 1)

	tests := []struct {
		name   string
		mutate func(pr *types.PullReq)
		number int64
		want   error
	}{
		{
			name:   "duplicate number in same repo",
			number: 1,
			want:   gitness_store.ErrDuplicate,
		},
		{
			name:   "unknown target repo",
			number: 2,
			mutate: func(pr *types.PullReq) { pr.TargetRepoID = 9999 },
			want:   gitness_store.ErrForeignKeyViolation,
		},
		{
			name:   "unknown author",
			number: 3,
			mutate: func(pr *types.PullReq) { pr.CreatedBy = 9999 },
			want:   gitness_store.ErrForeignKeyViolation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := prqNewPullReq(prqRepoMain, tt.number)
			if tt.mutate != nil {
				tt.mutate(pr)
			}
			err := f.pullreqs.Create(f.ctx, pr)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestPrqPullReqStore_Update(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	closed := time.Now().UnixMilli()
	pr.State = enum.PullReqStateClosed
	pr.Closed = &closed
	pr.Title = "Updated title"
	pr.Description = "Updated description"
	pr.IsDraft = true
	pr.SourceSHA = "new-source-sha"
	pr.TargetBranch = prqBranchRelease
	pr.MergeConflicts = []string{"x.go"}
	pr.MergeCheckStatus = enum.MergeCheckStatusConflict
	pr.CommentCount = 4
	pr.UnresolvedCount = 1
	files := int64(9)
	pr.Stats.FilesChanged = &files

	oldUpdated := pr.Updated
	require.NoError(t, f.pullreqs.Update(f.ctx, pr))
	require.Equal(t, int64(1), pr.Version)
	require.GreaterOrEqual(t, pr.Updated, oldUpdated)
	require.Equal(t, pr.Updated, pr.Edited)
	require.Equal(t, prqUserA, pr.Author.ID)

	got, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Version)
	require.Equal(t, enum.PullReqStateClosed, got.State)
	require.Equal(t, &closed, got.Closed)
	require.Equal(t, "Updated title", got.Title)
	require.Equal(t, "Updated description", got.Description)
	require.True(t, got.IsDraft)
	require.Equal(t, "new-source-sha", got.SourceSHA)
	require.Equal(t, prqBranchRelease, got.TargetBranch)
	require.Equal(t, []string{"x.go"}, got.MergeConflicts)
	require.Equal(t, enum.MergeCheckStatusConflict, got.MergeCheckStatus)
	require.Equal(t, 4, got.CommentCount)
	require.Equal(t, 1, got.UnresolvedCount)
	require.Equal(t, &files, got.Stats.FilesChanged)
}

func TestPrqPullReqStore_UpdateVersionConflict(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	stale := *pr
	require.NoError(t, f.pullreqs.Update(f.ctx, pr))

	stale.Title = "stale write"
	err := f.pullreqs.Update(f.ctx, &stale)
	require.ErrorIs(t, err, gitness_store.ErrVersionConflict)

	got, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, pr.Title, got.Title)
}

func TestPrqPullReqStore_UpdateOptLock(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	// make the in-memory copy stale so the first attempt conflicts and the store has to reload.
	concurrent := *pr
	require.NoError(t, f.pullreqs.Update(f.ctx, &concurrent))

	calls := 0
	updated, err := f.pullreqs.UpdateOptLock(f.ctx, pr, func(p *types.PullReq) error {
		calls++
		p.Title = "optimistic title"
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "optimistic title", updated.Title)
	require.Equal(t, int64(2), updated.Version)

	got, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, "optimistic title", got.Title)
	require.Equal(t, int64(2), got.Version)
}

func TestPrqPullReqStore_UpdateOptLockMutateError(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	errBoomPrq := errors.New("mutate failed")
	updated, err := f.pullreqs.UpdateOptLock(f.ctx, pr, func(*types.PullReq) error { return errBoomPrq })
	require.ErrorIs(t, err, errBoomPrq)
	require.Nil(t, updated)
}

func TestPrqPullReqStore_UpdateOptLockReloadFails(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	// a stale copy of a pull request that has been deleted must surface the not found error.
	stale := *pr
	require.NoError(t, f.pullreqs.Update(f.ctx, pr))
	require.NoError(t, f.pullreqs.Delete(f.ctx, pr.ID))

	_, err := f.pullreqs.UpdateOptLock(f.ctx, &stale, func(*types.PullReq) error { return nil })
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqStore_UpdateActivitySeq(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	require.Equal(t, int64(0), pr.ActivitySeq)

	for want := int64(1); want <= 3; want++ {
		updated, err := f.pullreqs.UpdateActivitySeq(f.ctx, pr)
		require.NoError(t, err)
		require.Equal(t, want, updated.ActivitySeq)
		pr = updated
	}

	got, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), got.ActivitySeq)
}

func TestPrqPullReqStore_UpdateMergeCheckMetadataOptLock(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	// stale copy forces one retry.
	concurrent := *pr
	_, err := f.pullreqs.UpdateMergeCheckMetadataOptLock(f.ctx, &concurrent, func(p *types.PullReq) error {
		p.MergeCheckStatus = enum.MergeCheckStatusMergeable
		return nil
	})
	require.NoError(t, err)

	calls := 0
	targetSHA := "meta-target-sha"
	mergeSHA := "meta-merge-sha"
	commits, files, adds, dels := int64(1), int64(2), int64(3), int64(4)
	updated, err := f.pullreqs.UpdateMergeCheckMetadataOptLock(f.ctx, pr, func(p *types.PullReq) error {
		calls++
		p.MergeCheckStatus = enum.MergeCheckStatusConflict
		p.MergeConflicts = []string{"conflict.go"}
		p.RebaseCheckStatus = enum.MergeCheckStatusMergeable
		p.MergeTargetSHA = &targetSHA
		p.MergeSHA = &mergeSHA
		p.MergeBaseSHA = "meta-base-sha"
		p.Stats.Commits = &commits
		p.Stats.FilesChanged = &files
		p.Stats.Additions = &adds
		p.Stats.Deletions = &dels
		// this field is not part of the merge check metadata and must not be persisted.
		p.Title = "ignored title"
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, int64(2), updated.Version)

	got, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, enum.MergeCheckStatusConflict, got.MergeCheckStatus)
	require.Equal(t, []string{"conflict.go"}, got.MergeConflicts)
	require.Equal(t, enum.MergeCheckStatusMergeable, got.RebaseCheckStatus)
	require.Equal(t, &targetSHA, got.MergeTargetSHA)
	require.Equal(t, &mergeSHA, got.MergeSHA)
	require.Equal(t, "meta-base-sha", got.MergeBaseSHA)
	require.Equal(t, &commits, got.Stats.Commits)
	require.Equal(t, &dels, got.Stats.Deletions)
	require.Equal(t, pr.Title, got.Title)
}

func TestPrqPullReqStore_UpdateMergeCheckMetadataOptLockErrors(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	errBoomPrq := errors.New("mutate failed")
	_, err := f.pullreqs.UpdateMergeCheckMetadataOptLock(f.ctx, pr, func(*types.PullReq) error { return errBoomPrq })
	require.ErrorIs(t, err, errBoomPrq)

	stale := *pr
	require.NoError(t, f.pullreqs.Update(f.ctx, pr))
	require.NoError(t, f.pullreqs.Delete(f.ctx, pr.ID))

	_, err = f.pullreqs.UpdateMergeCheckMetadataOptLock(f.ctx, &stale, func(*types.PullReq) error { return nil })
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqStore_UpdateRootSpace(t *testing.T) {
	f := prqNewFixture(t)

	prMain := f.prqCreatePullReq(t, prqRepoMain, 1)
	prOther := f.prqCreatePullReq(t, prqRepoOther, 1)

	// no repos is a no-op.
	require.NoError(t, f.pullreqs.UpdateRootSpace(f.ctx, nil, 42, "nobody"))
	got, err := f.pullreqs.Find(f.ctx, prMain.ID)
	require.NoError(t, err)
	require.Equal(t, prqSpaceID, got.RootSpaceID)

	require.NoError(t, f.pullreqs.UpdateRootSpace(f.ctx, []int64{prqRepoMain}, 42, "moved"))

	got, err = f.pullreqs.Find(f.ctx, prMain.ID)
	require.NoError(t, err)
	require.Equal(t, int64(42), got.RootSpaceID)
	require.Equal(t, "moved", got.RootSpaceIdentifier)

	got, err = f.pullreqs.Find(f.ctx, prOther.ID)
	require.NoError(t, err)
	require.Equal(t, prqSpaceID, got.RootSpaceID)
	require.Equal(t, "root", got.RootSpaceIdentifier)
}

func TestPrqPullReqStore_ResetMergeCheckStatus(t *testing.T) {
	f := prqNewFixture(t)

	targetSHA := "old-target-sha"
	mergeSHA := "old-merge-sha"
	commits := int64(2)
	dirty := func(pr *types.PullReq) {
		pr.MergeCheckStatus = enum.MergeCheckStatusMergeable
		pr.RebaseCheckStatus = enum.MergeCheckStatusConflict
		pr.MergeTargetSHA = &targetSHA
		pr.MergeSHA = &mergeSHA
		pr.MergeConflicts = []string{"z.go"}
		pr.Stats.Commits = &commits
	}

	open := f.prqCreatePullReq(t, prqRepoMain, 1, dirty)
	merged := f.prqCreatePullReq(t, prqRepoMain, 2, dirty, func(pr *types.PullReq) {
		pr.State = enum.PullReqStateMerged
	})
	closed := f.prqCreatePullReq(t, prqRepoMain, 3, dirty, func(pr *types.PullReq) {
		pr.State = enum.PullReqStateClosed
	})
	otherBranch := f.prqCreatePullReq(t, prqRepoMain, 4, dirty, func(pr *types.PullReq) {
		pr.TargetBranch = prqBranchRelease
	})
	otherRepo := f.prqCreatePullReq(t, prqRepoOther, 1, dirty)

	require.NoError(t, f.pullreqs.ResetMergeCheckStatus(f.ctx, prqRepoMain, "main"))

	got, err := f.pullreqs.Find(f.ctx, open.ID)
	require.NoError(t, err)
	require.Equal(t, enum.MergeCheckStatusUnchecked, got.MergeCheckStatus)
	require.Equal(t, enum.MergeCheckStatusUnchecked, got.RebaseCheckStatus)
	require.Nil(t, got.MergeTargetSHA)
	require.Nil(t, got.MergeSHA)
	require.Empty(t, got.MergeConflicts)
	require.Nil(t, got.Stats.Commits)
	require.Equal(t, int64(1), got.Version)

	for _, untouched := range []*types.PullReq{merged, closed, otherBranch, otherRepo} {
		got, err = f.pullreqs.Find(f.ctx, untouched.ID)
		require.NoError(t, err)
		require.Equal(t, enum.MergeCheckStatusMergeable, got.MergeCheckStatus, "pr %d", untouched.ID)
		require.Equal(t, int64(0), got.Version, "pr %d", untouched.ID)
	}
}

func TestPrqPullReqStore_Delete(t *testing.T) {
	f := prqNewFixture(t)

	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	keep := f.prqCreatePullReq(t, prqRepoMain, 2)

	require.NoError(t, f.pullreqs.Delete(f.ctx, pr.ID))

	_, err := f.pullreqs.Find(f.ctx, pr.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	_, err = f.pullreqs.Find(f.ctx, keep.ID)
	require.NoError(t, err)

	// deleting a missing pull request is not an error.
	require.NoError(t, f.pullreqs.Delete(f.ctx, pr.ID))
}

func TestPrqPullReqStore_ListOpenByBranchName(t *testing.T) {
	f := prqNewFixture(t)

	first := f.prqCreatePullReq(t, prqRepoMain, 1, func(pr *types.PullReq) {
		pr.SourceBranch = prqBranchAlpha
		pr.TargetBranch = "main"
	})
	second := f.prqCreatePullReq(t, prqRepoMain, 2, func(pr *types.PullReq) {
		pr.SourceBranch = prqBranchAlpha
		pr.TargetBranch = prqBranchRelease
		pr.Updated += 1000
	})
	f.prqCreatePullReq(t, prqRepoMain, 3, func(pr *types.PullReq) {
		pr.SourceBranch = "beta"
	})
	f.prqCreatePullReq(t, prqRepoMain, 4, func(pr *types.PullReq) {
		pr.SourceBranch = prqBranchAlpha
		pr.State = enum.PullReqStateClosed
	})
	f.prqCreatePullReq(t, prqRepoOther, 1, func(pr *types.PullReq) {
		pr.SourceBranch = prqBranchAlpha
	})

	res, err := f.pullreqs.ListOpenByBranchName(f.ctx, prqRepoMain, []string{prqBranchAlpha, "gamma"})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[prqBranchAlpha], 2)

	// most recently updated first; the description is not loaded.
	require.Equal(t, second.ID, res[prqBranchAlpha][0].ID)
	require.Equal(t, first.ID, res[prqBranchAlpha][1].ID)
	require.Empty(t, res[prqBranchAlpha][0].Description)
	require.Equal(t, prqUserA, res[prqBranchAlpha][0].Author.ID)

	res, err = f.pullreqs.ListOpenByBranchName(f.ctx, prqRepoMain, []string{"nope"})
	require.NoError(t, err)
	require.Empty(t, res)
}

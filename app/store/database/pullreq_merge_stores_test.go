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
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	prqSHA1 = "1111111111111111111111111111111111111111"
	prqSHA3 = "3333333333333333333333333333333333333333"
	prqSHA4 = "4444444444444444444444444444444444444444"
	prqSHA5 = "5555555555555555555555555555555555555555"
	prqSHA6 = "6666666666666666666666666666666666666666"
)

func TestPrqAutoMergeStore_UpsertFindDelete(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	store := database.NewAutoMergeStore(f.db)

	_, err := store.Find(f.ctx, pr.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	in := &types.AutoMerge{
		PullReqID:    pr.ID,
		Requested:    prqBaseTime,
		RequestedBy:  prqUserB,
		MergeMethod:  enum.MergeMethodSquash,
		Title:        "squash title",
		Message:      "squash message",
		DeleteBranch: true,
	}
	require.NoError(t, store.Upsert(f.ctx, in))

	got, err := store.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, in, got)

	// upsert replaces every field of the existing auto merge request.
	changed := &types.AutoMerge{
		PullReqID:    pr.ID,
		Requested:    prqBaseTime + 10,
		RequestedBy:  prqUserC,
		MergeMethod:  enum.MergeMethodRebase,
		Title:        "rebase title",
		Message:      "rebase message",
		DeleteBranch: false,
	}
	require.NoError(t, store.Upsert(f.ctx, changed))

	got, err = store.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, changed, got)

	deleted, err := store.Delete(f.ctx, pr.ID)
	require.NoError(t, err)
	require.True(t, deleted)

	deleted, err = store.Delete(f.ctx, pr.ID)
	require.NoError(t, err)
	require.False(t, deleted)

	_, err = store.Find(f.ctx, pr.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqAutoMergeStore_UpsertErrors(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewAutoMergeStore(f.db)

	err := store.Upsert(f.ctx, &types.AutoMerge{
		PullReqID:   9999,
		RequestedBy: prqUserA,
		MergeMethod: enum.MergeMethodMerge,
	})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func prqNewBranch(name, branchSHA string, by, at int64) *types.BranchTable {
	return &types.BranchTable{
		Name:      name,
		SHA:       sha.Must(branchSHA),
		CreatedBy: by,
		Created:   at,
		UpdatedBy: by,
		Updated:   at,
	}
}

func TestPrqBranchStore_UpsertFindDelete(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	store := database.NewBranchStore(f.db)

	_, err := store.Find(f.ctx, prqRepoMain, "feature")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	in := prqNewBranch("feature", prqSHA1, prqUserA, prqBaseTime)
	in.LastCreatedPullReqID = &pr.ID
	require.NoError(t, store.Upsert(f.ctx, prqRepoMain, in))

	got, err := store.Find(f.ctx, prqRepoMain, "feature")
	require.NoError(t, err)
	require.Equal(t, "feature", got.Name)
	require.Equal(t, prqSHA1, got.SHA.String())
	require.Equal(t, prqUserA, got.CreatedBy)
	require.Equal(t, prqBaseTime, got.Created)
	require.Equal(t, prqUserA, got.UpdatedBy)
	require.Equal(t, prqBaseTime, got.Updated)
	require.NotNil(t, got.LastCreatedPullReqID)
	require.Equal(t, pr.ID, *got.LastCreatedPullReqID)

	// upsert only updates sha, updater and update time and keeps the last pull request if none is provided.
	update := prqNewBranch("feature", prqSHA3, prqUserB, prqBaseTime+100)
	update.CreatedBy = prqUserC
	update.Created = prqBaseTime + 100
	require.NoError(t, store.Upsert(f.ctx, prqRepoMain, update))

	got, err = store.Find(f.ctx, prqRepoMain, "feature")
	require.NoError(t, err)
	require.Equal(t, prqSHA3, got.SHA.String())
	require.Equal(t, prqUserB, got.UpdatedBy)
	require.Equal(t, prqBaseTime+100, got.Updated)
	require.Equal(t, prqUserA, got.CreatedBy)
	require.Equal(t, prqBaseTime, got.Created)
	require.NotNil(t, got.LastCreatedPullReqID)
	require.Equal(t, pr.ID, *got.LastCreatedPullReqID)

	// a branch with the same name in another repo is a different branch.
	_, err = store.Find(f.ctx, prqRepoOther, "feature")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	require.NoError(t, store.Delete(f.ctx, prqRepoMain, "feature"))
	_, err = store.Find(f.ctx, prqRepoMain, "feature")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// deleting twice is fine.
	require.NoError(t, store.Delete(f.ctx, prqRepoMain, "feature"))
}

func TestPrqBranchStore_UpsertErrors(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewBranchStore(f.db)

	err := store.Upsert(f.ctx, 9999, prqNewBranch("feature", prqSHA1, prqUserA, prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqBranchStore_UpdateLastPR(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	store := database.NewBranchStore(f.db)

	require.NoError(t, store.Upsert(f.ctx, prqRepoMain, prqNewBranch("feature", prqSHA1, prqUserA, prqBaseTime)))

	got, err := store.Find(f.ctx, prqRepoMain, "feature")
	require.NoError(t, err)
	require.Nil(t, got.LastCreatedPullReqID)

	require.NoError(t, store.UpdateLastPR(f.ctx, prqRepoMain, "feature", &pr.ID))
	got, err = store.Find(f.ctx, prqRepoMain, "feature")
	require.NoError(t, err)
	require.NotNil(t, got.LastCreatedPullReqID)
	require.Equal(t, pr.ID, *got.LastCreatedPullReqID)

	require.NoError(t, store.UpdateLastPR(f.ctx, prqRepoMain, "feature", nil))
	got, err = store.Find(f.ctx, prqRepoMain, "feature")
	require.NoError(t, err)
	require.Nil(t, got.LastCreatedPullReqID)

	// updating a missing branch is not an error.
	require.NoError(t, store.UpdateLastPR(f.ctx, prqRepoMain, "missing", &pr.ID))
}

func TestPrqBranchStore_FindBranchesWithoutOpenPRs(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewBranchStore(f.db)

	cutOff := prqBaseTime

	openPR := f.prqCreatePullReq(t, prqRepoMain, 1, func(pr *types.PullReq) {
		pr.SourceBranch = "has-open-pr"
		pr.SourceSHA = prqSHA4
	})
	closedPR := f.prqCreatePullReq(t, prqRepoMain, 2, func(pr *types.PullReq) {
		pr.SourceBranch = "has-closed-pr"
		pr.SourceSHA = prqSHA1
		pr.State = enum.PullReqStateClosed
	})
	mergedSameSHA := f.prqCreatePullReq(t, prqRepoMain, 3, func(pr *types.PullReq) {
		pr.SourceBranch = "merged-same-sha"
		pr.SourceSHA = prqSHA5
		pr.State = enum.PullReqStateMerged
	})

	branch := func(name, branchSHA string, by, at int64, prID *int64) {
		b := prqNewBranch(name, branchSHA, by, at)
		b.LastCreatedPullReqID = prID
		require.NoError(t, store.Upsert(f.ctx, prqRepoMain, b), name)
	}

	// candidates, newest first.
	branch("no-pr-newest", prqSHA3, prqUserA, cutOff+500, nil)
	branch("closed-pr-moved", prqSHA3, prqUserA, cutOff+400, &closedPR.ID)
	branch("no-pr-older", prqSHA5, prqUserA, cutOff+300, nil)
	// excluded branches.
	branch("has-open-pr", prqSHA4, prqUserA, cutOff+450, &openPR.ID)
	branch("merged-same-sha", prqSHA5, prqUserA, cutOff+350, &mergedSameSHA.ID)
	branch("too-old", prqSHA3, prqUserA, cutOff-1, nil)
	branch("other-author", prqSHA3, prqUserB, cutOff+500, nil)
	branch("excluded-sha", prqSHA6, prqUserA, cutOff+500, nil)

	res, err := store.FindBranchesWithoutOpenPRs(f.ctx, prqRepoMain, prqUserA, cutOff, 10, prqSHA6)
	require.NoError(t, err)

	names := make([]string, len(res))
	for i, b := range res {
		names[i] = b.Name
	}
	require.Equal(t, []string{"no-pr-newest", "closed-pr-moved", "no-pr-older"}, names)

	res, err = store.FindBranchesWithoutOpenPRs(f.ctx, prqRepoMain, prqUserA, cutOff, 2, prqSHA6)
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, "no-pr-newest", res[0].Name)

	res, err = store.FindBranchesWithoutOpenPRs(f.ctx, prqRepoOther, prqUserA, cutOff, 10, prqSHA6)
	require.NoError(t, err)
	require.Empty(t, res)
}

func prqNewMergeQueue(repoID int64, branch string) *types.MergeQueue {
	return &types.MergeQueue{
		RepoID:        repoID,
		Branch:        branch,
		Created:       prqBaseTime,
		Updated:       prqBaseTime,
		OrderSequence: 1,
	}
}

func TestPrqMergeQueueStore_CRUD(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewMergeQueueStore(f.db)

	in := prqNewMergeQueue(prqRepoMain, "main")
	require.NoError(t, store.Create(f.ctx, in))
	require.NotZero(t, in.ID)

	byID, err := store.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, in, byID)

	byBranch, err := store.FindByRepoAndBranch(f.ctx, prqRepoMain, "main")
	require.NoError(t, err)
	require.Equal(t, in, byBranch)

	_, err = store.FindByRepoAndBranch(f.ctx, prqRepoMain, prqBranchRelease)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = store.FindByRepoAndBranch(f.ctx, prqRepoOther, "main")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = store.Find(f.ctx, 9999)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	byID.OrderSequence = 7
	require.NoError(t, store.Update(f.ctx, byID))
	require.Equal(t, int64(1), byID.Version)
	require.Equal(t, int64(7), byID.OrderSequence)

	got, err := store.Find(f.ctx, in.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Version)
	require.Equal(t, int64(7), got.OrderSequence)
	require.Equal(t, byID.Updated, got.Updated)

	require.NoError(t, store.Delete(f.ctx, in.ID))
	_, err = store.Find(f.ctx, in.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	require.NoError(t, store.Delete(f.ctx, in.ID))
}

func TestPrqMergeQueueStore_CreateErrors(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewMergeQueueStore(f.db)

	require.NoError(t, store.Create(f.ctx, prqNewMergeQueue(prqRepoMain, "main")))

	err := store.Create(f.ctx, prqNewMergeQueue(prqRepoMain, "main"))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = store.Create(f.ctx, prqNewMergeQueue(9999, "main"))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)

	// same branch name in a different repo is fine.
	require.NoError(t, store.Create(f.ctx, prqNewMergeQueue(prqRepoOther, "main")))
}

func TestPrqMergeQueueStore_UpdateVersionConflictAndOptLock(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewMergeQueueStore(f.db)

	q := prqNewMergeQueue(prqRepoMain, "main")
	require.NoError(t, store.Create(f.ctx, q))

	stale := *q
	concurrent := *q
	require.NoError(t, store.Update(f.ctx, &concurrent))

	err := store.Update(f.ctx, &stale)
	require.ErrorIs(t, err, gitness_store.ErrVersionConflict)

	calls := 0
	updated, err := store.UpdateOptLock(f.ctx, q, func(mq *types.MergeQueue) error {
		calls++
		mq.OrderSequence += 10
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, int64(2), updated.Version)
	require.Equal(t, int64(11), updated.OrderSequence)

	errBoomPrq := errors.New("mutate failed")
	_, err = store.UpdateOptLock(f.ctx, updated, func(*types.MergeQueue) error { return errBoomPrq })
	require.ErrorIs(t, err, errBoomPrq)

	// reloading a deleted queue fails with not found.
	staleDeleted := *updated
	require.NoError(t, store.Update(f.ctx, updated))
	require.NoError(t, store.Delete(f.ctx, updated.ID))
	_, err = store.UpdateOptLock(f.ctx, &staleDeleted, func(*types.MergeQueue) error { return nil })
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func prqNewQueueEntry(pr *types.PullReq, queueID, index int64) *types.MergeQueueEntry {
	return &types.MergeQueueEntry{
		PullReqID:     pr.ID,
		MergeQueueID:  queueID,
		CreatedBy:     prqUserA,
		Created:       prqBaseTime,
		Updated:       prqBaseTime,
		OrderIndex:    index,
		State:         enum.MergeQueueEntryStateMergePending,
		MergeMethod:   enum.MergeMethodMerge,
		CommitTitle:   "merge title",
		CommitMessage: "merge message",
	}
}

type prqQueueFixture struct {
	*prqFixture
	queues  *database.MergeQueueStore
	entries *database.MergeQueueEntryStore
	queue   *types.MergeQueue
}

func prqNewQueueFixture(t *testing.T) *prqQueueFixture {
	t.Helper()

	f := prqNewFixture(t)
	qf := &prqQueueFixture{
		prqFixture: f,
		queues:     database.NewMergeQueueStore(f.db),
		entries:    database.NewMergeQueueEntryStore(f.db),
		queue:      prqNewMergeQueue(prqRepoMain, "main"),
	}
	require.NoError(t, qf.queues.Create(f.ctx, qf.queue))

	return qf
}

func TestPrqMergeQueueEntryStore_CreateFindUpdate(t *testing.T) {
	f := prqNewQueueFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	started := prqBaseTime + 10
	deadline := prqBaseTime + 20
	in := prqNewQueueEntry(pr, f.queue.ID, 1)
	in.State = enum.MergeQueueEntryStateChecksInProgress
	in.BaseCommitSHA = sha.Must(prqSHA1)
	in.HeadCommitSHA = sha.Must(prqSHA3)
	in.MergeCommitSHA = sha.Must(prqSHA4)
	in.MergeBaseSHA = sha.Must(prqSHA5)
	in.ChecksCommitSHA = sha.Must(prqSHA6)
	in.ChecksStarted = &started
	in.ChecksDeadline = &deadline
	in.CommitCount = 2
	in.ChangedFileCount = 3
	in.Additions = 40
	in.Deletions = 5
	in.MergeMethod = enum.MergeMethodSquash
	in.DeleteSourceBranch = true
	require.NoError(t, f.entries.Create(f.ctx, in))

	got, err := f.entries.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, in, got)

	byCommit, err := f.entries.FindByMergeCommit(f.ctx, sha.Must(prqSHA4))
	require.NoError(t, err)
	require.Equal(t, pr.ID, byCommit.PullReqID)

	_, err = f.entries.FindByMergeCommit(f.ctx, sha.Must(prqSHA6))
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = f.entries.Find(f.ctx, 9999)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// optional fields are stored as null when empty.
	bare := f.prqCreatePullReq(t, prqRepoMain, 2)
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(bare, f.queue.ID, 2)))
	bareGot, err := f.entries.Find(f.ctx, bare.ID)
	require.NoError(t, err)
	require.True(t, bareGot.BaseCommitSHA.IsEmpty())
	require.True(t, bareGot.MergeCommitSHA.IsEmpty())
	require.Nil(t, bareGot.ChecksStarted)
	require.Nil(t, bareGot.ChecksDeadline)

	// update
	got.State = enum.MergeQueueEntryStateMergeGroup
	got.OrderIndex = 5
	got.MergeCommitSHA = sha.Must(prqSHA6)
	got.ChecksStarted = nil
	got.CommitTitle = "new title"
	got.CommitMessage = "new message"
	got.DeleteSourceBranch = false
	require.NoError(t, f.entries.Update(f.ctx, got))
	require.Equal(t, int64(1), got.Version)

	updated, err := f.entries.Find(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, got, updated)
	require.Equal(t, enum.MergeQueueEntryStateMergeGroup, updated.State)
	require.Equal(t, int64(5), updated.OrderIndex)
	require.Nil(t, updated.ChecksStarted)
	require.False(t, updated.DeleteSourceBranch)
}

func TestPrqMergeQueueEntryStore_CreateErrors(t *testing.T) {
	f := prqNewQueueFixture(t)
	first := f.prqCreatePullReq(t, prqRepoMain, 1)
	second := f.prqCreatePullReq(t, prqRepoMain, 2)

	entry := prqNewQueueEntry(first, f.queue.ID, 1)
	entry.MergeCommitSHA = sha.Must(prqSHA4)
	require.NoError(t, f.entries.Create(f.ctx, entry))

	// same pull request twice.
	err := f.entries.Create(f.ctx, prqNewQueueEntry(first, f.queue.ID, 2))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	// same order index in the same queue.
	err = f.entries.Create(f.ctx, prqNewQueueEntry(second, f.queue.ID, 1))
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	// same merge commit.
	clash := prqNewQueueEntry(second, f.queue.ID, 2)
	clash.MergeCommitSHA = sha.Must(prqSHA4)
	err = f.entries.Create(f.ctx, clash)
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	err = f.entries.Create(f.ctx, prqNewQueueEntry(second, 9999, 2))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqMergeQueueEntryStore_InvalidStoredData(t *testing.T) {
	f := prqNewQueueFixture(t)
	badState := f.prqCreatePullReq(t, prqRepoMain, 1)
	badSHA := f.prqCreatePullReq(t, prqRepoMain, 2)

	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(badState, f.queue.ID, 1)))
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(badSHA, f.queue.ID, 2)))

	_, err := f.db.ExecContext(f.ctx,
		`UPDATE merge_queue_entries SET merge_queue_entry_state = 'bogus' WHERE merge_queue_entry_pullreq_id = ?`,
		badState.ID)
	require.NoError(t, err)
	_, err = f.db.ExecContext(f.ctx,
		`UPDATE merge_queue_entries SET merge_queue_entry_head_commit_sha = 'not-a-sha'
		 WHERE merge_queue_entry_pullreq_id = ?`, badSHA.ID)
	require.NoError(t, err)

	_, err = f.entries.Find(f.ctx, badState.ID)
	require.ErrorContains(t, err, "invalid state")

	_, err = f.entries.Find(f.ctx, badSHA.ID)
	require.ErrorContains(t, err, "head commit SHA")

	_, err = f.entries.ListForMergeQueue(f.ctx, f.queue.ID)
	require.Error(t, err)
}

func TestPrqMergeQueueEntryStore_UpdateConflictAndOptLock(t *testing.T) {
	f := prqNewQueueFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)

	entry := prqNewQueueEntry(pr, f.queue.ID, 1)
	require.NoError(t, f.entries.Create(f.ctx, entry))

	stale := *entry
	concurrent := *entry
	require.NoError(t, f.entries.Update(f.ctx, &concurrent))

	err := f.entries.Update(f.ctx, &stale)
	require.ErrorIs(t, err, gitness_store.ErrVersionConflict)

	calls := 0
	updated, err := f.entries.UpdateOptLock(f.ctx, entry, func(e *types.MergeQueueEntry) error {
		calls++
		e.State = enum.MergeQueueEntryStateChecksPending
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, int64(2), updated.Version)
	require.Equal(t, enum.MergeQueueEntryStateChecksPending, updated.State)

	errBoomPrq := errors.New("mutate failed")
	_, err = f.entries.UpdateOptLock(f.ctx, updated, func(*types.MergeQueueEntry) error { return errBoomPrq })
	require.ErrorIs(t, err, errBoomPrq)

	staleDeleted := *updated
	require.NoError(t, f.entries.Update(f.ctx, updated))
	require.NoError(t, f.entries.Delete(f.ctx, pr.ID))
	_, err = f.entries.UpdateOptLock(f.ctx, &staleDeleted, func(*types.MergeQueueEntry) error { return nil })
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqMergeQueueEntryStore_ListCountDelete(t *testing.T) {
	f := prqNewQueueFixture(t)

	otherQueue := prqNewMergeQueue(prqRepoMain, prqBranchRelease)
	require.NoError(t, f.queues.Create(f.ctx, otherQueue))

	prs := make([]*types.PullReq, 0, 4)
	for number := int64(1); number <= 4; number++ {
		prs = append(prs, f.prqCreatePullReq(t, prqRepoMain, number))
	}

	// inserted out of order to verify ordering by the order index.
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(prs[0], f.queue.ID, 3)))
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(prs[1], f.queue.ID, 1)))
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(prs[2], f.queue.ID, 2)))
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(prs[3], otherQueue.ID, 1)))

	list, err := f.entries.ListForMergeQueue(f.ctx, f.queue.ID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, []int64{prs[1].ID, prs[2].ID, prs[0].ID}, []int64{
		list[0].PullReqID, list[1].PullReqID, list[2].PullReqID,
	})

	empty, err := f.entries.ListForMergeQueue(f.ctx, 9999)
	require.NoError(t, err)
	require.Empty(t, empty)

	count, err := f.entries.CountForRepoAndBranch(f.ctx, prqRepoMain, "main")
	require.NoError(t, err)
	require.Equal(t, int64(3), count)

	count, err = f.entries.CountForRepoAndBranch(f.ctx, prqRepoMain, prqBranchRelease)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	count, err = f.entries.CountForRepoAndBranch(f.ctx, prqRepoOther, "main")
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	require.NoError(t, f.entries.Delete(f.ctx, prs[0].ID))
	count, err = f.entries.CountForRepoAndBranch(f.ctx, prqRepoMain, "main")
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	require.NoError(t, f.entries.DeleteAllForMergeQueue(f.ctx, f.queue.ID))
	count, err = f.entries.CountForRepoAndBranch(f.ctx, prqRepoMain, "main")
	require.NoError(t, err)
	require.Equal(t, int64(0), count)

	// entries of other queues are not removed.
	count, err = f.entries.CountForRepoAndBranch(f.ctx, prqRepoMain, prqBranchRelease)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestPrqMergeQueueEntryStore_BranchesWithPullReqInQueue(t *testing.T) {
	f := prqNewQueueFixture(t)

	for i, branch := range []string{prqBranchAlpha, "beta", "gamma"} {
		pr := f.prqCreatePullReq(t, prqRepoMain, int64(i+1), func(p *types.PullReq) { p.SourceBranch = branch })
		require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(pr, f.queue.ID, int64(i+1))))
	}
	// a second pull request from the same branch must not duplicate the result.
	dup := f.prqCreatePullReq(t, prqRepoMain, 4, func(p *types.PullReq) {
		p.SourceBranch = prqBranchAlpha
		p.TargetBranch = prqBranchRelease
	})
	require.NoError(t, f.entries.Create(f.ctx, prqNewQueueEntry(dup, f.queue.ID, 4)))
	// a pull request that is not in the queue.
	f.prqCreatePullReq(t, prqRepoMain, 5, func(p *types.PullReq) { p.SourceBranch = "delta" })

	tests := []struct {
		name     string
		repoID   int64
		branches []string
		want     map[string]struct{}
	}{
		{
			name:   "all branches",
			repoID: prqRepoMain,
			want:   map[string]struct{}{prqBranchAlpha: {}, "beta": {}, "gamma": {}},
		},
		{
			name:     "single branch",
			repoID:   prqRepoMain,
			branches: []string{"beta"},
			want:     map[string]struct{}{"beta": {}},
		},
		{
			name:     "multiple branches",
			repoID:   prqRepoMain,
			branches: []string{prqBranchAlpha, "gamma", "delta"},
			want:     map[string]struct{}{prqBranchAlpha: {}, "gamma": {}},
		},
		{
			name:     "branch not in queue",
			repoID:   prqRepoMain,
			branches: []string{"delta"},
			want:     map[string]struct{}{},
		},
		{
			name:   "other repository",
			repoID: prqRepoOther,
			want:   map[string]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.entries.BranchesWithPullReqInQueue(f.ctx, tt.repoID, tt.branches)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPrqMergeQueueEntryStore_ListOverdueChecks(t *testing.T) {
	f := prqNewQueueFixture(t)

	newEntry := func(number, index int64, state enum.MergeQueueEntryState, deadline *int64) *types.MergeQueueEntry {
		pr := f.prqCreatePullReq(t, prqRepoMain, number)
		entry := prqNewQueueEntry(pr, f.queue.ID, index)
		entry.State = state
		entry.ChecksDeadline = deadline
		require.NoError(t, f.entries.Create(f.ctx, entry))
		return entry
	}

	early, late, future := prqBaseTime+10, prqBaseTime+50, prqBaseTime+500
	lateEntry := newEntry(1, 1, enum.MergeQueueEntryStateChecksInProgress, &late)
	earlyEntry := newEntry(2, 2, enum.MergeQueueEntryStateChecksInProgress, &early)
	// not overdue yet, no deadline, or not running checks.
	newEntry(3, 3, enum.MergeQueueEntryStateChecksInProgress, &future)
	newEntry(4, 4, enum.MergeQueueEntryStateChecksInProgress, nil)
	newEntry(5, 5, enum.MergeQueueEntryStateChecksPending, &early)

	list, err := f.entries.ListOverdueChecks(f.ctx, prqBaseTime+100)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// the oldest deadline comes first.
	require.Equal(t, earlyEntry.PullReqID, list[0].PullReqID)
	require.Equal(t, lateEntry.PullReqID, list[1].PullReqID)

	list, err = f.entries.ListOverdueChecks(f.ctx, prqBaseTime)
	require.NoError(t, err)
	require.Empty(t, list)

	// the boundary is exclusive.
	list, err = f.entries.ListOverdueChecks(f.ctx, early)
	require.NoError(t, err)
	require.Empty(t, list)
}

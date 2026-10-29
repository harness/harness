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
	"encoding/json"
	"errors"
	"strconv"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

const (
	numTestRepos = 10
	repoSize     = int64(100)
	// rootSpaceID is the space the size-info tests create their repos under.
	rootSpaceID = int64(1)
)

func TestDatabase_GetSize(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	repoID := int64(1)
	createRepo(ctx, t, repoStore, repoID, 1, repoSize)

	tests := []struct {
		name       string
		Size       int64
		areSizesEq bool
	}{
		{
			name:       "size equal to repo size",
			Size:       repoSize,
			areSizesEq: true,
		},
		{
			name:       "size less than repo size",
			Size:       repoSize / 2,
			areSizesEq: false,
		},
		{
			name:       "size greater than repo size",
			Size:       repoSize * 2,
			areSizesEq: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := repoStore.GetSize(ctx, repoID)
			if err != nil {
				t.Errorf("GetSize() error = %v, want error = %v", err, nil)
			}
			areSizesEq := size == tt.Size
			if areSizesEq != tt.areSizesEq {
				t.Errorf("size == tt.Size = %v, want %v", areSizesEq, tt.areSizesEq)
			}
		})
	}
}

func TestDatabase_UpdateLastGITPush(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	repoID := int64(1)
	createRepo(ctx, t, repoStore, repoID, 1, repoSize)

	before, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}

	if err = repoStore.UpdateLastGITPush(ctx, repoID, 2_000); err != nil {
		t.Fatalf("UpdateLastGITPush() error = %v", err)
	}
	if err = repoStore.UpdateLastGITPush(ctx, repoID, 1_000); err != nil {
		t.Fatalf("UpdateLastGITPush() older timestamp error = %v", err)
	}

	after, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if after.LastGITPush != 2_000 {
		t.Errorf("LastGITPush = %d, want 2000", after.LastGITPush)
	}
	if after.Version != before.Version || after.Updated != before.Updated || after.PullReqSeq != before.PullReqSeq {
		t.Errorf("repo version/updated/seq changed: before=%+v after=%+v", before, after)
	}

	if err = repoStore.UpdateLastGITPush(ctx, 999, 3_000); err != nil {
		t.Errorf("UpdateLastGITPush() missing repo error = %v, want nil", err)
	}

	stale := *after
	stale.LastGITPush = 0
	stale.Description = "touched"
	if err = repoStore.Update(ctx, &stale); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	kept, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if kept.LastGITPush != 2_000 {
		t.Errorf("Update() LastGITPush = %d, want 2000", kept.LastGITPush)
	}

	deletedAt := int64(50)
	if err = repoStore.SoftDelete(ctx, kept, deletedAt); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}
	if err = repoStore.UpdateLastGITPush(ctx, repoID, 4_000); err != nil {
		t.Fatalf("UpdateLastGITPush() deleted repo error = %v", err)
	}
	deleted, err := repoStore.FindDeleted(ctx, repoID, &deletedAt)
	if err != nil {
		t.Fatalf("FindDeleted() error = %v", err)
	}
	if deleted.LastGITPush != 2_000 {
		t.Errorf("deleted LastGITPush = %d, want 2000", deleted.LastGITPush)
	}
}

func TestDatabase_Count(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	numRepos := createRepos(ctx, t, repoStore, 0, numTestRepos, 1)

	count, err := repoStore.Count(ctx, 1, &types.RepoFilter{})
	if err != nil {
		t.Fatalf("failed to count repos %v", err)
	}
	if count != numRepos {
		t.Errorf("count = %v, want %v", count, numRepos)
	}
}

func TestDatabase_CountAll(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	numSpaces := createNestedSpaces(ctx, t, spaceStore, spacePathStore)
	var numRepos int64
	for i := 1; i <= numSpaces; i++ {
		numRepos += createRepos(ctx, t, repoStore, numRepos, numTestRepos/2, int64(i))
	}

	count, err := repoStore.Count(ctx, 1, &types.RepoFilter{Recursive: true})
	if err != nil {
		t.Fatalf("failed to count repos %v", err)
	}
	if count != numRepos {
		t.Errorf("count = %v, want %v", count, numRepos)
	}
}

func TestDatabase_List(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	numRepos := createRepos(ctx, t, repoStore, 0, numTestRepos, 1)

	repos, err := repoStore.List(ctx, 1, &types.RepoFilter{})
	if err != nil {
		t.Fatalf("failed to count repos %v", err)
	}

	lenRepos := int64(len(repos))
	if lenRepos != numRepos {
		t.Errorf("count = %v, want %v", lenRepos, numRepos)
	}
}

func TestDatabase_ListAll(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	numSpaces := createNestedSpaces(ctx, t, spaceStore, spacePathStore)
	var numRepos int64
	for i := 1; i <= numSpaces; i++ {
		numRepos += createRepos(ctx, t, repoStore, numRepos, numTestRepos/2, int64(i))
	}

	repos, err := repoStore.List(ctx, 1,
		&types.RepoFilter{Size: numSpaces * numTestRepos, Recursive: true})
	if err != nil {
		t.Fatalf("failed to count repos %v", err)
	}
	lenRepos := int64(len(repos))
	if lenRepos != numRepos {
		t.Errorf("count = %v, want %v", lenRepos, numRepos)
	}
}

func TestDatabase_ListSizeInfos(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	// States whose repos must be reported by ListSizeInfos. Note that
	// RepoStateMigrateDataImport is intentionally included - only git-import,
	// migrate-git-push and import-failed repos are excluded.
	includedStates := []enum.RepoState{
		enum.RepoStateActive,
		enum.RepoStateMigrateDataImport,
		enum.RepoStateArchived,
	}
	// States whose repos hold no meaningful size and must be excluded.
	excludedStates := []enum.RepoState{
		enum.RepoStateGitImport,
		enum.RepoStateMigrateGitPush,
		enum.RepoStateImportFailed,
	}

	var id int64
	wantIDs := map[int64]bool{}
	for _, state := range includedStates {
		id++
		wantIDs[createRepoWithState(ctx, t, repoStore, id, state)] = true
	}
	for _, state := range excludedStates {
		id++
		createRepoWithState(ctx, t, repoStore, id, state)
	}

	infos, err := repoStore.ListSizeInfos(ctx)
	if err != nil {
		t.Fatalf("ListSizeInfos() error = %v", err)
	}

	gotIDs := map[int64]bool{}
	for _, info := range infos {
		gotIDs[info.ID] = true
	}

	if len(gotIDs) != len(wantIDs) {
		t.Errorf("ListSizeInfos() returned %d repos, want %d", len(gotIDs), len(wantIDs))
	}
	for wantID := range wantIDs {
		if !gotIDs[wantID] {
			t.Errorf("repo %d is expected in size infos but is missing", wantID)
		}
	}
	for gotID := range gotIDs {
		if !wantIDs[gotID] {
			t.Errorf("repo %d must be excluded from size infos but is present", gotID)
		}
	}
}

func createRepo(
	ctx context.Context,
	t *testing.T,
	repoStore *database.RepoStore,
	id int64,
	spaceID int64,
	size int64,
) {
	t.Helper()

	identifier := "repo_" + strconv.FormatInt(id, 10)
	repo := types.Repository{
		Identifier: identifier, ID: id, GitUID: identifier,
		ParentID: spaceID,
		Size:     size,
		Tags:     json.RawMessage{},
	}
	if err := repoStore.Create(ctx, &repo); err != nil {
		t.Fatalf("failed to create repo %v", err)
	}
}

// createRepoWithState creates a repo in the given state under the root space every caller sets up,
// and returns the ID assigned by the store.
func createRepoWithState(
	ctx context.Context,
	t *testing.T,
	repoStore *database.RepoStore,
	id int64,
	state enum.RepoState,
) int64 {
	t.Helper()

	identifier := "repo_" + strconv.FormatInt(id, 10)
	repo := types.Repository{
		Identifier: identifier, GitUID: identifier,
		ParentID: rootSpaceID,
		State:    state,
		Size:     repoSize,
		Tags:     json.RawMessage{},
	}
	if err := repoStore.Create(ctx, &repo); err != nil {
		t.Fatalf("failed to create repo %v", err)
	}

	return repo.ID
}

func createRepos(
	ctx context.Context,
	t *testing.T,
	repoStore *database.RepoStore,
	numCreatedRepos int64,
	numReposToCreate int64,
	spaceID int64,
) int64 {
	t.Helper()

	var numRepos int64
	for j := 0; j < int(numReposToCreate); j++ {
		// numCreatedRepos+numRepos ensures the uniqueness of the repo id
		createRepo(ctx, t, repoStore, numCreatedRepos+numRepos, spaceID, 0)
		numRepos++
	}
	return numRepos
}

func createNestedSpaces(
	ctx context.Context,
	t *testing.T,
	spaceStore *database.SpaceStore,
	spacePathStore store.SpacePathStore,
) int {
	t.Helper()

	spaceTree, numSpaces := createSpaceTree()
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	for i := 1; i < numSpaces; i++ {
		parentID := int64(i)
		for _, spaceID := range spaceTree[parentID] {
			createSpace(ctx, t, spaceStore, spacePathStore, userID, spaceID, parentID)
		}
	}
	return numSpaces
}

func TestRepoStore_ListAllSizeInfosIter(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	// State filtering must match ListSizeInfos, even though the push-time filter is weaker.
	includedStates := []enum.RepoState{
		enum.RepoStateActive,
		enum.RepoStateMigrateDataImport,
		enum.RepoStateArchived,
	}
	excludedStates := []enum.RepoState{
		enum.RepoStateGitImport,
		enum.RepoStateMigrateGitPush,
		enum.RepoStateImportFailed,
	}

	var id int64
	wantAllSizeInfosIDs := map[int64]bool{}
	for _, state := range includedStates {
		id++
		wantAllSizeInfosIDs[createRepoWithState(ctx, t, repoStore, id, state)] = true
	}
	for _, state := range excludedStates {
		id++
		createRepoWithState(ctx, t, repoStore, id, state)
	}

	// A repo with repo_last_git_push < repo_size_updated: not pushed to since its size was last
	// updated, so ListSizeInfos filters it out and ListAllSizeInfosIter must not.
	id++
	notPushedID := createRepoWithState(ctx, t, repoStore, id, enum.RepoStateActive)
	currentTime := int64(1000000000)
	pastTime := int64(999999999)
	if _, err := db.Exec(
		"UPDATE repositories SET repo_size_updated = ?, repo_last_git_push = ? WHERE repo_id = ?",
		currentTime, pastTime, notPushedID,
	); err != nil {
		t.Fatalf("failed to update repo timestamps: %v", err)
	}
	wantAllSizeInfosIDs[notPushedID] = true

	// A soft-deleted repo, excluded from both listings.
	id++
	deletedID := createRepoWithState(ctx, t, repoStore, id, enum.RepoStateActive)
	if _, err := db.Exec(
		"UPDATE repositories SET repo_deleted = ? WHERE repo_id = ?",
		currentTime, deletedID,
	); err != nil {
		t.Fatalf("failed to soft-delete repo: %v", err)
	}
	// deletedID is deliberately left out of wantAllSizeInfosIDs.

	allSizeInfos, err := repoStore.ListAllSizeInfosIter(ctx)
	if err != nil {
		t.Fatalf("ListAllSizeInfosIter failed: %v", err)
	}

	gotIDs := map[int64]bool{}
	for info, err := range allSizeInfos {
		if err != nil {
			t.Fatalf("ListAllSizeInfosIter failed mid-iteration: %v", err)
		}
		gotIDs[info.ID] = true
	}

	if len(gotIDs) != len(wantAllSizeInfosIDs) {
		t.Errorf("ListAllSizeInfosIter returned %d repos, want %d", len(gotIDs), len(wantAllSizeInfosIDs))
	}
	for wantID := range wantAllSizeInfosIDs {
		if !gotIDs[wantID] {
			t.Errorf("repo %d is expected in size infos but is missing", wantID)
		}
	}
	for gotID := range gotIDs {
		if !wantAllSizeInfosIDs[gotID] {
			t.Errorf("repo %d must be excluded from size infos but is present", gotID)
		}
	}

	// The difference this method exists for: ListSizeInfos drops the unpushed repo, the iterator
	// keeps it.
	sizeInfos, err := repoStore.ListSizeInfos(ctx)
	if err != nil {
		t.Fatalf("ListSizeInfos failed: %v", err)
	}

	gotSizeInfosIDs := map[int64]bool{}
	for _, info := range sizeInfos {
		gotSizeInfosIDs[info.ID] = true
	}

	if gotSizeInfosIDs[notPushedID] {
		t.Errorf("repo %d should be filtered out by ListSizeInfos (unpushed), but it was returned", notPushedID)
	}
	if !gotIDs[notPushedID] {
		t.Errorf("repo %d should be returned by ListAllSizeInfosIter even though unpushed, but it was missing",
			notPushedID)
	}
}

// Snapshots record LFS storage from what these listings return, so a mapping that drops
// repo_lfs_size reports every repository as having no LFS data at all.
func TestRepoStore_SizeInfosCarryLFSSize(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	repoID := createRepoWithState(ctx, t, repoStore, 1, enum.RepoStateActive)

	const wantLFSSize = int64(4096)
	if _, err := db.Exec(
		"UPDATE repositories SET repo_lfs_size = ? WHERE repo_id = ?", wantLFSSize, repoID,
	); err != nil {
		t.Fatalf("failed to set lfs size: %v", err)
	}

	sizeInfos, err := repoStore.ListSizeInfos(ctx)
	if err != nil {
		t.Fatalf("ListSizeInfos failed: %v", err)
	}
	if len(sizeInfos) != 1 {
		t.Fatalf("ListSizeInfos returned %d repos, want 1", len(sizeInfos))
	}
	if sizeInfos[0].LFSSize != wantLFSSize {
		t.Errorf("ListSizeInfos LFSSize = %d, want %d", sizeInfos[0].LFSSize, wantLFSSize)
	}

	iterInfos, err := repoStore.ListAllSizeInfosIter(ctx)
	if err != nil {
		t.Fatalf("ListAllSizeInfosIter failed: %v", err)
	}

	var yielded int
	for info, err := range iterInfos {
		if err != nil {
			t.Fatalf("ListAllSizeInfosIter failed mid-iteration: %v", err)
		}
		yielded++
		if info.LFSSize != wantLFSSize {
			t.Errorf("ListAllSizeInfosIter LFSSize = %d, want %d", info.LFSSize, wantLFSSize)
		}
	}
	if yielded != 1 {
		t.Errorf("ListAllSizeInfosIter yielded %d repos, want 1", yielded)
	}
}

// seedSizeInfoRepos inserts count active repos in one statement; the store's create path is far
// too slow to cross several 1000-row pages.
func seedSizeInfoRepos(ctx context.Context, t *testing.T, db *sqlx.DB, count int) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		WITH RECURSIVE i(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM i WHERE i < $1)
		INSERT INTO repositories (
			 repo_version, repo_parent_id, repo_uid, repo_created_by, repo_created, repo_updated
			,repo_git_uid, repo_default_branch, repo_pullreq_seq, repo_num_forks, repo_num_pulls
			,repo_num_closed_pulls, repo_num_open_pulls, repo_num_merged_pulls
			,repo_size, repo_size_updated, repo_is_empty, repo_state, repo_last_git_push
			,repo_lfs_size, repo_tags, repo_root_space_id, repo_root_space_identifier
		)
		SELECT
			 0, $2, 'repo_' || i, 1, 1, 1
			,'repo_' || i, 'main', 0, 0, 0
			,0, 0, 0
			,i, 1, false, 0, 1
			,0, '{}', $2, 'root'
		FROM i`,
		count, rootSpaceID,
	); err != nil {
		t.Fatalf("failed to seed %d repos: %v", count, err)
	}
}

func newSizeInfoStore(t *testing.T, count int) (*database.RepoStore, *sqlx.DB) {
	t.Helper()

	ctx := context.Background()

	db, teardown := setupDB(t)
	t.Cleanup(teardown)

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	seedSizeInfoRepos(ctx, t, db, count)

	return repoStore, db
}

// The page size is 1000, so these counts cover a partial last page (2500) and an exact multiple,
// where the last page is full and termination depends on one more, empty, read (2000).
func TestRepoStore_ListAllSizeInfosIter_PagesThroughEveryRepo(t *testing.T) {
	for _, count := range []int{1000, 2000, 2500} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			ctx := context.Background()
			repoStore, _ := newSizeInfoStore(t, count)

			seq, err := repoStore.ListAllSizeInfosIter(ctx)
			if err != nil {
				t.Fatalf("ListAllSizeInfosIter failed: %v", err)
			}

			var prev int64
			var yielded int
			for info, err := range seq {
				if err != nil {
					t.Fatalf("failed mid-iteration: %v", err)
				}
				if info.ID <= prev {
					t.Fatalf("repo %d yielded after %d, want ascending IDs without repeats", info.ID, prev)
				}
				prev = info.ID
				yielded++
			}
			if yielded != count {
				t.Errorf("yielded %d repos, want %d", yielded, count)
			}
		})
	}
}

func TestRepoStore_ListAllSizeInfosIter_StopsWhenConsumerBreaks(t *testing.T) {
	ctx := context.Background()
	repoStore, _ := newSizeInfoStore(t, 2500)

	seq, err := repoStore.ListAllSizeInfosIter(ctx)
	if err != nil {
		t.Fatalf("ListAllSizeInfosIter failed: %v", err)
	}

	// 1500 is past the first page, so the break lands after a page continuation.
	const stopAfter = 1500
	var yielded int
	for _, err := range seq {
		if err != nil {
			t.Fatalf("failed mid-iteration: %v", err)
		}
		yielded++
		if yielded == stopAfter {
			break
		}
	}
	if yielded != stopAfter {
		t.Errorf("yielded %d repos, want %d", yielded, stopAfter)
	}
}

// A failure reading a later page must reach the consumer once and end the sequence, rather than
// returning the pages read so far as if they were everything.
func TestRepoStore_ListAllSizeInfosIter_ReportsNextPageFailure(t *testing.T) {
	ctx := context.Background()
	repoStore, db := newSizeInfoStore(t, 2500)

	seq, err := repoStore.ListAllSizeInfosIter(ctx)
	if err != nil {
		t.Fatalf("ListAllSizeInfosIter failed: %v", err)
	}

	var yielded, failures int
	for _, err := range seq {
		if err != nil {
			failures++
			continue
		}
		yielded++
		if yielded == 1000 {
			// The first page is fully consumed; the next read has no table to hit.
			if _, execErr := db.ExecContext(ctx, "ALTER TABLE repositories RENAME TO repositories_gone"); execErr != nil {
				t.Fatalf("failed to break the next page: %v", execErr)
			}
		}
	}
	if yielded != 1000 {
		t.Errorf("yielded %d repos, want only the first page's 1000", yielded)
	}
	if failures != 1 {
		t.Errorf("got %d errors, want exactly 1", failures)
	}
}

func TestRepoStore_ListAllSizeInfosIter_ReportsCancelledContextBetweenPages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repoStore, _ := newSizeInfoStore(t, 2500)

	seq, err := repoStore.ListAllSizeInfosIter(ctx)
	if err != nil {
		t.Fatalf("ListAllSizeInfosIter failed: %v", err)
	}

	var yielded int
	var gotErr error
	for _, err := range seq {
		if err != nil {
			gotErr = err
			continue
		}
		yielded++
		if yielded == 1 {
			cancel()
		}
	}
	if yielded != 1000 {
		t.Errorf("yielded %d repos, want the first page's 1000 and nothing after", yielded)
	}
	if !errors.Is(gotErr, context.Canceled) {
		t.Errorf("got error %v, want context.Canceled", gotErr)
	}
}

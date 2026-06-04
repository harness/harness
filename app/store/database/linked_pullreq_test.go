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
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

const (
	// testPullReqRepoID is the fixed repoID used by all rows that
	// insertPullReqRaw creates; the tests only exercise a single repo.
	testPullReqRepoID int64 = 1

	// testProviderTypeGitHub is the provider-type column value used
	// by linked-repo / linked-pullreq fixtures.
	testProviderTypeGitHub = "github"

	// testProviderRepoID is the upstream repo SCM id used by linked-pullreq fixtures.
	testProviderRepoID = "280125018"

	// testLinkedAccountID is the account scope used by linked-repo fixtures.
	testLinkedAccountID = "acct-1"

	// testConnectorIdentGitHub is the connector identifier used by linked-repo fixtures.
	testConnectorIdentGitHub = "ghc"
)

// insertPullReqRaw inserts a minimal pullreqs row directly via SQL. We bypass
// PullReqStore here so the tests don't drag in a PrincipalInfoCache.
func insertPullReqRaw(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	pullReqID int64,
	number int64,
	prType *enum.PullReqType,
) {
	t.Helper()
	const repoID = testPullReqRepoID

	now := time.Now().UnixMilli()
	const q = `
	INSERT INTO pullreqs (
		 pullreq_id
		,pullreq_version
		,pullreq_created_by
		,pullreq_created
		,pullreq_updated
		,pullreq_edited
		,pullreq_number
		,pullreq_state
		,pullreq_title
		,pullreq_description
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_source_sha
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_merge_check_status
		,pullreq_merge_base_sha
		,pullreq_type
	) VALUES (
		?, 0, ?, ?, ?, ?, ?, 'open', ?, '',
		?, ?, ?, ?, ?,
		'unchecked', '', ?
	)`

	var typeVal interface{}
	if prType != nil {
		typeVal = string(*prType)
	}

	sourceBranch := fmt.Sprintf("feat/x-%d", pullReqID)
	sourceSHA := fmt.Sprintf("sha-%016d", pullReqID)
	targetBranch := fmt.Sprintf("main-%d", pullReqID)

	if _, err := db.ExecContext(ctx, q,
		pullReqID, userID, now, now, now,
		number, fmt.Sprintf("Test PR %d", pullReqID),
		repoID, sourceBranch, sourceSHA, repoID, targetBranch, typeVal,
	); err != nil {
		t.Fatalf("insertPullReqRaw: %v", err)
	}
}

// newLinkedPR builds a linked-pullreq fixture; upstream PR number lives on the parent pullreqs row.
func newLinkedPR(pullReqID int64) *types.LinkedPullReq {
	return &types.LinkedPullReq{
		PullReqID:               pullReqID,
		ProviderType:            testProviderTypeGitHub,
		ProviderRepoID:          testProviderRepoID,
		ProviderURL:             fmt.Sprintf("https://github.com/owner/%s/pull/%d", testProviderRepoID, pullReqID),
		ProviderAuthorLogin:     "octocat",
		ProviderAuthorAvatarURL: "https://avatars.githubusercontent.com/u/583231",
		ProviderAuthorURL:       "https://github.com/octocat",
	}
}

func TestLinkedPullReqStore_CreateAndFind(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)

	prType := enum.PullReqTypeLinked
	insertPullReqRaw(ctx, t, db, 100, 1, &prType)

	store := database.NewLinkedPullReqStore(db)

	in := newLinkedPR(100)
	if err := store.Create(ctx, in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if in.LastSyncedAt == 0 {
		t.Errorf("Create did not stamp LastSyncedAt")
	}

	got, err := store.Find(ctx, 100)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.PullReqID != 100 || got.ProviderRepoID != testProviderRepoID {
		t.Errorf("Find returned wrong row: %+v", got)
	}
	if got.ProviderAuthorLogin != "octocat" {
		t.Errorf("Find: ProviderAuthorLogin = %q, want octocat", got.ProviderAuthorLogin)
	}
}

// FindByLinkedRepoAndProviderPR resolves (linkedRepo, providerRepoID, prNumber)
// by joining linked_pullreqs to the parent pullreqs row and matching on
// pullreq_number. A single harness target repo always mirrors a single
// upstream, so the prNumber predicate goes through the parent.
func TestLinkedPullReqStore_FindByLinkedRepoAndProviderPR(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)

	prType := enum.PullReqTypeLinked
	insertPullReqRaw(ctx, t, db, 100, 1, &prType)
	insertPullReqRaw(ctx, t, db, 101, 2, &prType)

	store := database.NewLinkedPullReqStore(db)

	if err := store.Create(ctx, newLinkedPR(100)); err != nil {
		t.Fatalf("Create #1: %v", err)
	}
	if err := store.Create(ctx, newLinkedPR(101)); err != nil {
		t.Fatalf("Create #2: %v", err)
	}

	// Resolve by (linkedRepo, providerRepoID, parent.pullreq_number).
	got, err := store.FindByLinkedRepoAndProviderPR(ctx, testPullReqRepoID, testProviderTypeGitHub, testProviderRepoID, 2)
	if err != nil {
		t.Fatalf("FindByLinkedRepoAndProviderPR: %v", err)
	}
	if got.PullReqID != 101 {
		t.Errorf("expected PR 101 for (280125018,#2), got %d", got.PullReqID)
	}

	got2, err := store.FindByLinkedRepoAndProviderPR(ctx, testPullReqRepoID, testProviderTypeGitHub, testProviderRepoID, 1)
	if err != nil {
		t.Fatalf("FindByLinkedRepoAndProviderPR (#1): %v", err)
	}
	if got2.PullReqID != 100 {
		t.Errorf("expected PR 100 for (280125018,#1), got %d", got2.PullReqID)
	}

	// Different providerRepoID is filtered out by the linked_pullreqs predicate.
	if _, err := store.FindByLinkedRepoAndProviderPR(
		ctx, testPullReqRepoID, testProviderTypeGitHub, "nonexistent", 1,
	); err == nil {
		t.Errorf("expected error for missing provider id, got nil")
	}
	// Missing PR number is filtered out by the parent join.
	if _, err := store.FindByLinkedRepoAndProviderPR(
		ctx, testPullReqRepoID, testProviderTypeGitHub, testProviderRepoID, 999,
	); err == nil {
		t.Errorf("expected error for missing PR number, got nil")
	}
	// Wrong linkedRepoID -> no match (parent target_repo_id filter).
	if _, err := store.FindByLinkedRepoAndProviderPR(
		ctx, testPullReqRepoID+999, testProviderTypeGitHub, testProviderRepoID, 1,
	); err == nil {
		t.Errorf("expected error when linkedRepoID doesn't match, got nil")
	}
}

func TestLinkedPullReqStore_Update(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)

	prType := enum.PullReqTypeLinked
	insertPullReqRaw(ctx, t, db, 100, 1, &prType)

	store := database.NewLinkedPullReqStore(db)
	in := newLinkedPR(100)
	if err := store.Create(ctx, in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	firstSync := in.LastSyncedAt
	time.Sleep(2 * time.Millisecond) // ensure timestamp moves forward

	in.ProviderURL = "https://github.com/owner/repo/pull/7"
	in.ProviderAuthorAvatarURL = "https://new-avatar.example.com/x.png"
	in.MergerLogin = "merger-bot"
	if err := store.Update(ctx, in); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if in.LastSyncedAt <= firstSync {
		t.Errorf("Update did not advance LastSyncedAt: before=%d after=%d", firstSync, in.LastSyncedAt)
	}

	got, err := store.Find(ctx, 100)
	if err != nil {
		t.Fatalf("Find after Update: %v", err)
	}
	if got.ProviderURL != in.ProviderURL || got.ProviderAuthorAvatarURL != in.ProviderAuthorAvatarURL {
		t.Errorf("Update did not persist provider fields: %+v", got)
	}
	if got.MergerLogin != "merger-bot" {
		t.Errorf("Update did not persist MergerLogin: got %q", got.MergerLogin)
	}
}

func TestLinkedPullReqStore_FKCascade(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)

	prType := enum.PullReqTypeLinked
	insertPullReqRaw(ctx, t, db, 100, 1, &prType)

	store := database.NewLinkedPullReqStore(db)
	if err := store.Create(ctx, newLinkedPR(100)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Deleting the parent pullreqs row must cascade delete the linked_pullreqs row.
	if _, err := db.ExecContext(ctx, "DELETE FROM pullreqs WHERE pullreq_id = ?", 100); err != nil {
		t.Fatalf("delete parent: %v", err)
	}

	if _, err := store.Find(ctx, 100); err == nil {
		t.Errorf("expected linked row to be cascade-deleted, but Find returned no error")
	}
}

func TestLinkedRepoStore_ListByProviderID(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	createRepo(ctx, t, repoStore, 2, 1, 0)

	store := database.NewLinkedRepoStore(db)
	// repo 1 -> acme/widget on github (node R_widget) in acct-1
	if err := store.Create(ctx, &types.LinkedRepo{
		RepoID:  1,
		Created: 1, Updated: 1, LastFullSync: 1,
		ConnectorPath:       testLinkedAccountID,
		ConnectorIdentifier: testConnectorIdentGitHub,
		ProviderType:        testProviderTypeGitHub,
		ProviderRepoID:      "R_widget",
		ConnectorRepo:       "acme/widget",
	}); err != nil {
		t.Fatalf("Create #1: %v", err)
	}
	// repo 2 -> different upstream (R_other) in same account
	if err := store.Create(ctx, &types.LinkedRepo{
		RepoID:  2,
		Created: 1, Updated: 1, LastFullSync: 1,
		ConnectorPath:       testLinkedAccountID,
		ConnectorIdentifier: testConnectorIdentGitHub,
		ProviderType:        testProviderTypeGitHub,
		ProviderRepoID:      "R_other",
		ConnectorRepo:       "acme/other",
	}); err != nil {
		t.Fatalf("Create #2: %v", err)
	}

	pageAll := types.Pagination{Page: 1, Size: 100}
	got, err := store.ListByProviderID(
		ctx, testLinkedAccountID, testProviderTypeGitHub, "R_widget", pageAll,
	)
	if err != nil {
		t.Fatalf("ListByProviderID: %v", err)
	}
	if len(got) != 1 || got[0].RepoID != 1 {
		t.Fatalf("expected RepoID 1 only, got %+v", got)
	}

	// Different account scope -> no match even though provider id matches.
	got2, err := store.ListByProviderID(
		ctx, "other-acct", testProviderTypeGitHub, "R_widget", pageAll,
	)
	if err != nil {
		t.Fatalf("ListByProviderID (other acct): %v", err)
	}
	if len(got2) != 0 {
		t.Errorf("expected 0 matches across accounts, got %d", len(got2))
	}

	// Unknown provider -> no match.
	got3, err := store.ListByProviderID(
		ctx, testLinkedAccountID, "gitlab", "R_widget", pageAll,
	)
	if err != nil {
		t.Fatalf("ListByProviderID (gitlab): %v", err)
	}
	if len(got3) != 0 {
		t.Errorf("expected 0 matches for different provider, got %d", len(got3))
	}
}

func TestLinkedRepoStore_ListByProviderID_Pagination(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	for repoID := int64(1); repoID <= 3; repoID++ {
		createRepo(ctx, t, repoStore, repoID, 1, 0)
	}

	store := database.NewLinkedRepoStore(db)
	for repoID := int64(1); repoID <= 3; repoID++ {
		if err := store.Create(ctx, &types.LinkedRepo{
			RepoID:  repoID,
			Created: 1, Updated: 1, LastFullSync: 1,
			ConnectorPath:       testLinkedAccountID,
			ConnectorIdentifier: testConnectorIdentGitHub,
			ProviderType:        testProviderTypeGitHub,
			ProviderRepoID:      "R_shared",
			ConnectorRepo:       fmt.Sprintf("acme/repo-%d", repoID),
		}); err != nil {
			t.Fatalf("Create repo %d: %v", repoID, err)
		}
	}

	page1, err := store.ListByProviderID(
		ctx, testLinkedAccountID, testProviderTypeGitHub, "R_shared",
		types.Pagination{Page: 1, Size: 2},
	)
	if err != nil {
		t.Fatalf("ListByProviderID page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1: expected 2 rows, got %d", len(page1))
	}

	page2, err := store.ListByProviderID(
		ctx, testLinkedAccountID, testProviderTypeGitHub, "R_shared",
		types.Pagination{Page: 2, Size: 2},
	)
	if err != nil {
		t.Fatalf("ListByProviderID page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("page 2: expected 1 row, got %d", len(page2))
	}

	seen := map[int64]struct{}{page1[0].RepoID: {}, page1[1].RepoID: {}, page2[0].RepoID: {}}
	if len(seen) != 3 {
		t.Fatalf("expected 3 distinct repo ids across pages, got %v", seen)
	}
}

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
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

// TestDatabase_RootSpacePropagationOnCrossRootMove asserts that the store
// operations the space-move flow relies on (SpaceStore.UpdateRootSpace,
// RepoStore.ListIDsByParentSpaceIDs / UpdateRootSpace, PullReqStore.UpdateRootSpace)
// together re-stamp the root space id/identifier on a moved subtree: the moved
// spaces, every repo directly under them, and every pull request targeting those
// repos. Rows outside the moved subtree must keep their original root space.
//
// This mirrors Service.propagateRootSpace, which is otherwise not exercised by
// any test (it depends on a transactor + refcache wiring not present here).
func TestDatabase_RootSpacePropagationOnCrossRootMove(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	pullreqStore := database.NewPullReqStore(db, nil)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	// Source root space with a subtree (srcRoot -> mid -> leaf) and a separate
	// target root space we move the mid subtree into. IDs are assigned by the DB
	// (the spaces/repos inserts use RETURNING id and ignore explicit ids), so we
	// capture them rather than hard-code.
	srcRootID := createRootSpace(ctx, t, spaceStore, spacePathStore)
	srcRootIdentifier := "space_" + strconv.FormatInt(srcRootID, 10)
	midID := createChildSpace(ctx, t, spaceStore, spacePathStore, srcRootID, srcRootID, srcRootIdentifier)
	leafID := createChildSpace(ctx, t, spaceStore, spacePathStore, midID, srcRootID, srcRootIdentifier)
	tgtRootID := createRootSpace(ctx, t, spaceStore, spacePathStore)
	tgtRootIdentifier := "space_" + strconv.FormatInt(tgtRootID, 10)

	// Repos: two under the moved subtree (mid and leaf) and one under the
	// untouched source root space, all initially rooted at the source root.
	repoMidID := createRepoWithRoot(ctx, t, repoStore, midID, srcRootID, srcRootIdentifier)
	repoLeafID := createRepoWithRoot(ctx, t, repoStore, leafID, srcRootID, srcRootIdentifier)
	repoRootID := createRepoWithRoot(ctx, t, repoStore, srcRootID, srcRootID, srcRootIdentifier)

	// One PR per repo, targeting that repo, initially rooted at the source root.
	prMid := insertPullReqWithRoot(ctx, t, db, 1000, repoMidID, srcRootID, srcRootIdentifier)
	prLeaf := insertPullReqWithRoot(ctx, t, db, 1001, repoLeafID, srcRootID, srcRootIdentifier)
	prRoot := insertPullReqWithRoot(ctx, t, db, 1002, repoRootID, srcRootID, srcRootIdentifier)

	// --- Simulate moving the mid space (and its subtree) under the target root. ---

	// GetDescendantsIDs includes the anchor space itself, so this is {mid, leaf}.
	descendantSpaceIDs, err := spaceStore.GetDescendantsIDs(ctx, midID)
	if err != nil {
		t.Fatalf("GetDescendantsIDs: %v", err)
	}
	if !int64SetEqual(descendantSpaceIDs, []int64{midID, leafID}) {
		t.Fatalf("GetDescendantsIDs(%d) = %v, want {%d,%d}", midID, descendantSpaceIDs, midID, leafID)
	}

	if err := spaceStore.UpdateRootSpace(
		ctx, descendantSpaceIDs, tgtRootID, tgtRootIdentifier,
	); err != nil {
		t.Fatalf("SpaceStore.UpdateRootSpace: %v", err)
	}

	repoIDs, err := repoStore.ListIDsByParentSpaceIDs(ctx, descendantSpaceIDs)
	if err != nil {
		t.Fatalf("RepoStore.ListIDsByParentSpaceIDs: %v", err)
	}
	if !int64SetEqual(repoIDs, []int64{repoMidID, repoLeafID}) {
		t.Fatalf("ListIDsByParentSpaceIDs(%v) = %v, want {%d,%d}",
			descendantSpaceIDs, repoIDs, repoMidID, repoLeafID)
	}

	if err := repoStore.UpdateRootSpace(
		ctx, repoIDs, tgtRootID, tgtRootIdentifier,
	); err != nil {
		t.Fatalf("RepoStore.UpdateRootSpace: %v", err)
	}

	if err := pullreqStore.UpdateRootSpace(
		ctx, repoIDs, tgtRootID, tgtRootIdentifier,
	); err != nil {
		t.Fatalf("PullReqStore.UpdateRootSpace: %v", err)
	}

	// --- Assertions: moved subtree carries the new root space, the rest doesn't. ---

	// Moved spaces.
	assertSpaceRoot(ctx, t, spaceStore, midID, tgtRootID, tgtRootIdentifier)
	assertSpaceRoot(ctx, t, spaceStore, leafID, tgtRootID, tgtRootIdentifier)
	// Untouched source root space keeps rooting at itself.
	assertSpaceRoot(ctx, t, spaceStore, srcRootID, srcRootID, srcRootIdentifier)

	// Moved repos.
	assertRepoRoot(ctx, t, repoStore, repoMidID, tgtRootID, tgtRootIdentifier)
	assertRepoRoot(ctx, t, repoStore, repoLeafID, tgtRootID, tgtRootIdentifier)
	// Repo under the untouched root space is unchanged.
	assertRepoRoot(ctx, t, repoStore, repoRootID, srcRootID, srcRootIdentifier)

	// PRs targeting moved repos.
	assertPullReqRoot(ctx, t, db, prMid, tgtRootID, tgtRootIdentifier)
	assertPullReqRoot(ctx, t, db, prLeaf, tgtRootID, tgtRootIdentifier)
	// PR targeting the untouched repo is unchanged.
	assertPullReqRoot(ctx, t, db, prRoot, srcRootID, srcRootIdentifier)
}

// createRootSpace creates a root space (parent 0) that points at itself and
// returns its DB-assigned id.
func createRootSpace(
	ctx context.Context,
	t *testing.T,
	spaceStore *database.SpaceStore,
	spacePathStore store.SpacePathStore,
) int64 {
	t.Helper()

	// create first to obtain the DB-assigned id, then root it at itself.
	id := createSpaceRow(ctx, t, spaceStore, spacePathStore, 0, 0, "")
	identifier := "space_" + strconv.FormatInt(id, 10)
	if err := spaceStore.UpdateRootSpace(ctx, []int64{id}, id, identifier); err != nil {
		t.Fatalf("failed to root space %d at itself: %v", id, err)
	}
	return id
}

func createChildSpace(
	ctx context.Context,
	t *testing.T,
	spaceStore *database.SpaceStore,
	spacePathStore store.SpacePathStore,
	parentID int64,
	rootSpaceID int64,
	rootSpaceIdentifier string,
) int64 {
	t.Helper()
	return createSpaceRow(ctx, t, spaceStore, spacePathStore, parentID, rootSpaceID, rootSpaceIdentifier)
}

// createSpaceRow inserts a space and its primary path segment, returning the
// DB-assigned space id (the spaces insert ignores explicit ids).
func createSpaceRow(
	ctx context.Context,
	t *testing.T,
	spaceStore *database.SpaceStore,
	spacePathStore store.SpacePathStore,
	parentID int64,
	rootSpaceID int64,
	rootSpaceIdentifier string,
) int64 {
	t.Helper()

	space := types.Space{
		Identifier:          "tmp", // replaced below once we know the id
		CreatedBy:           userID,
		ParentID:            parentID,
		RootSpaceID:         rootSpaceID,
		RootSpaceIdentifier: rootSpaceIdentifier,
	}
	if err := spaceStore.Create(ctx, &space); err != nil {
		t.Fatalf("failed to create space: %v", err)
	}

	identifier := "space_" + strconv.FormatInt(space.ID, 10)
	// re-read so Update's optimistic lock version matches.
	created, err := spaceStore.Find(ctx, space.ID)
	if err != nil {
		t.Fatalf("failed to find created space %d: %v", space.ID, err)
	}
	created.Identifier = identifier
	if err := spaceStore.Update(ctx, created); err != nil {
		t.Fatalf("failed to set space %d identifier: %v", space.ID, err)
	}

	if err := spacePathStore.InsertSegment(ctx, &types.SpacePathSegment{
		ID: space.ID, Identifier: identifier, CreatedBy: userID, SpaceID: space.ID, IsPrimary: true,
	}); err != nil {
		t.Fatalf("failed to insert segment for space %d: %v", space.ID, err)
	}

	return space.ID
}

// createRepoWithRoot creates a repo under parentID with the given root space and
// returns its DB-assigned id.
func createRepoWithRoot(
	ctx context.Context,
	t *testing.T,
	repoStore *database.RepoStore,
	parentID int64,
	rootSpaceID int64,
	rootSpaceIdentifier string,
) int64 {
	t.Helper()

	repo := types.Repository{
		ParentID:            parentID,
		RootSpaceID:         rootSpaceID,
		RootSpaceIdentifier: rootSpaceIdentifier,
		Tags:                json.RawMessage{},
	}
	if err := repoStore.Create(ctx, &repo); err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	// give it a unique identifier/git_uid now that we know the id.
	identifier := "repo_" + strconv.FormatInt(repo.ID, 10)
	repo.Identifier = identifier
	repo.GitUID = identifier
	if err := repoStore.Update(ctx, &repo); err != nil {
		t.Fatalf("failed to set repo %d identifier: %v", repo.ID, err)
	}

	return repo.ID
}

// insertPullReqWithRoot inserts a minimal pullreqs row targeting targetRepoID and
// returns pullReqID.
func insertPullReqWithRoot(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	pullReqID int64,
	targetRepoID int64,
	rootSpaceID int64,
	rootSpaceIdentifier string,
) int64 {
	t.Helper()

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
		,pullreq_root_space_id
		,pullreq_root_space_identifier
	) VALUES (
		?, 0, ?, ?, ?, ?, ?, 'open', ?, '',
		?, ?, ?, ?, ?,
		'unchecked', '', ?, ?
	)`

	sourceBranch := fmt.Sprintf("feat/x-%d", pullReqID)
	sourceSHA := fmt.Sprintf("sha-%016d", pullReqID)
	targetBranch := fmt.Sprintf("main-%d", pullReqID)

	if _, err := db.ExecContext(ctx, q,
		pullReqID, userID, now, now, now,
		pullReqID, fmt.Sprintf("Test PR %d", pullReqID),
		targetRepoID, sourceBranch, sourceSHA, targetRepoID, targetBranch,
		rootSpaceID, rootSpaceIdentifier,
	); err != nil {
		t.Fatalf("insertPullReqWithRoot %d: %v", pullReqID, err)
	}

	return pullReqID
}

func assertSpaceRoot(
	ctx context.Context,
	t *testing.T,
	spaceStore *database.SpaceStore,
	spaceID int64,
	wantRootSpaceID int64,
	wantRootSpaceIdentifier string,
) {
	t.Helper()
	space, err := spaceStore.Find(ctx, spaceID)
	if err != nil {
		t.Fatalf("SpaceStore.Find(%d): %v", spaceID, err)
	}
	if space.RootSpaceID != wantRootSpaceID || space.RootSpaceIdentifier != wantRootSpaceIdentifier {
		t.Errorf("space %d root = (%d, %q), want (%d, %q)",
			spaceID, space.RootSpaceID, space.RootSpaceIdentifier, wantRootSpaceID, wantRootSpaceIdentifier)
	}
}

func assertRepoRoot(
	ctx context.Context,
	t *testing.T,
	repoStore *database.RepoStore,
	repoID int64,
	wantRootSpaceID int64,
	wantRootSpaceIdentifier string,
) {
	t.Helper()
	repo, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("RepoStore.Find(%d): %v", repoID, err)
	}
	if repo.RootSpaceID != wantRootSpaceID || repo.RootSpaceIdentifier != wantRootSpaceIdentifier {
		t.Errorf("repo %d root = (%d, %q), want (%d, %q)",
			repoID, repo.RootSpaceID, repo.RootSpaceIdentifier, wantRootSpaceID, wantRootSpaceIdentifier)
	}
}

func assertPullReqRoot(
	ctx context.Context,
	t *testing.T,
	db *sqlx.DB,
	pullReqID int64,
	wantRootSpaceID int64,
	wantRootSpaceIdentifier string,
) {
	t.Helper()
	var gotID int64
	var gotIdentifier string
	if err := db.QueryRowContext(ctx,
		`SELECT pullreq_root_space_id, pullreq_root_space_identifier FROM pullreqs WHERE pullreq_id = ?`,
		pullReqID,
	).Scan(&gotID, &gotIdentifier); err != nil {
		t.Fatalf("select pullreq %d root space: %v", pullReqID, err)
	}
	if gotID != wantRootSpaceID || gotIdentifier != wantRootSpaceIdentifier {
		t.Errorf("pullreq %d root = (%d, %q), want (%d, %q)",
			pullReqID, gotID, gotIdentifier, wantRootSpaceID, wantRootSpaceIdentifier)
	}
}

// int64SetEqual reports whether a and b contain the same ids, ignoring order.
func int64SetEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[int64]int, len(a))
	for _, v := range a {
		set[v]++
	}
	for _, v := range b {
		set[v]--
		if set[v] < 0 {
			return false
		}
	}
	return true
}

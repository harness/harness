// Copyright 2026 Harness, Inc.
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
	"slices"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

// suggestionTestSpaceID is the space that setupSuggestionTestWithDB creates.
const suggestionTestSpaceID = 1

func setupSuggestionTest(t *testing.T) (context.Context, store.PullReqReviewerSuggestionStore, func()) {
	t.Helper()

	ctx, _, suggestions, teardown := setupSuggestionTestWithDB(t)

	return ctx, suggestions, teardown
}

func setupSuggestionTestWithDB(
	t *testing.T,
) (context.Context, *sqlx.DB, store.PullReqReviewerSuggestionStore, func()) {
	t.Helper()

	db, teardown := setupDB(t)
	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	insertPrincipalRaw(ctx, t, principalStore, 2)
	insertPrincipalRaw(ctx, t, principalStore, 3)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	insertPullReqRaw(ctx, t, db, 100, 1, nil)

	return ctx, db, database.NewPullReqReviewerSuggestionStore(db), teardown
}

func suggestion(
	principalID int64,
	kind enum.ReviewerSuggestionKind,
	reason string,
	rank *int64,
) *types.PullReqReviewerSuggestion {
	return &types.PullReqReviewerSuggestion{
		PullReqID:   100,
		CreatedBy:   1,
		PrincipalID: principalID,
		Kind:        kind,
		Reason:      reason,
		Rank:        rank,
	}
}

func TestPullReqReviewerSuggestionStore_UpsertManyInsertsNewFields(t *testing.T) {
	ctx, suggestions, teardown := setupSuggestionTest(t)
	defer teardown()

	in := suggestion(2, enum.ReviewerSuggestionKindRequired, "owns this file", new(int64(1)))
	in.CommitSHA = "abc123"
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{in}); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	list, err := suggestions.List(ctx, 100, types.Pagination{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	got := list[0]
	if got.Kind != enum.ReviewerSuggestionKindRequired || got.Reason != "owns this file" ||
		got.Rank == nil || *got.Rank != 1 || got.CommitSHA != "abc123" || got.UserGroupID != nil {
		t.Errorf("unexpected suggestion: %+v", got)
	}

	found, err := suggestions.Find(ctx, 100, 2)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.Kind != enum.ReviewerSuggestionKindRequired || found.Reason != "owns this file" {
		t.Errorf("Find returned unexpected suggestion: %+v", found)
	}
}

func TestPullReqReviewerSuggestionStore_UpsertManyUpdatesExisting(t *testing.T) {
	ctx, suggestions, teardown := setupSuggestionTest(t)
	defer teardown()

	first := suggestion(2, enum.ReviewerSuggestionKindRequired, "first", new(int64(1)))
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{first}); err != nil {
		t.Fatalf("UpsertMany first: %v", err)
	}
	second := suggestion(2, enum.ReviewerSuggestionKindOptional, "changed", nil)
	second.CreatedBy = 3
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{second}); err != nil {
		t.Fatalf("UpsertMany second: %v", err)
	}

	count, err := suggestions.Count(ctx, 100)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}

	got, err := suggestions.Find(ctx, 100, 2)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Kind != enum.ReviewerSuggestionKindOptional || got.Reason != "changed" || got.Rank != nil {
		t.Errorf("unexpected suggestion: %+v", got)
	}
	if got.CreatedBy != 1 {
		t.Errorf("CreatedBy = %d, want 1 (must not be overwritten)", got.CreatedBy)
	}
}

func TestPullReqReviewerSuggestionStore_CreateManyStillIgnoresConflicts(t *testing.T) {
	ctx, suggestions, teardown := setupSuggestionTest(t)
	defer teardown()

	first := suggestion(2, enum.ReviewerSuggestionKindRequired, "keep me", new(int64(1)))
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{first}); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}
	if err := suggestions.CreateMany(ctx, []*types.PullReqReviewerSuggestion{
		suggestion(2, "", "", nil),
	}); err != nil {
		t.Fatalf("CreateMany: %v", err)
	}

	got, err := suggestions.Find(ctx, 100, 2)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.Kind != enum.ReviewerSuggestionKindRequired || got.Reason != "keep me" {
		t.Errorf("unexpected suggestion: %+v", got)
	}

	// A fresh row created by CreateMany defaults to optional.
	if err := suggestions.CreateMany(ctx, []*types.PullReqReviewerSuggestion{
		suggestion(3, "", "", nil),
	}); err != nil {
		t.Fatalf("CreateMany new: %v", err)
	}
	fresh, err := suggestions.Find(ctx, 100, 3)
	if err != nil {
		t.Fatalf("Find fresh: %v", err)
	}
	if fresh.Kind != enum.ReviewerSuggestionKindOptional {
		t.Errorf("Kind = %q, want optional", fresh.Kind)
	}
}

func TestPullReqReviewerSuggestionStore_ListOrdersByRankNullsLast(t *testing.T) {
	ctx, suggestions, teardown := setupSuggestionTest(t)
	defer teardown()

	opt := enum.ReviewerSuggestionKindOptional
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{
		suggestion(1, opt, "", new(int64(2))),
		suggestion(2, opt, "", nil),
		suggestion(3, opt, "", new(int64(1))),
	}); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	list, err := suggestions.List(ctx, 100, types.Pagination{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got []int64
	for _, s := range list {
		got = append(got, s.PrincipalID)
	}
	want := []int64{3, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("principals = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("principals = %v, want %v", got, want)
		}
	}
}

func TestPullReqReviewerSuggestionStore_ListBreaksTiesByPrincipalID(t *testing.T) {
	ctx, suggestions, teardown := setupSuggestionTest(t)
	defer teardown()

	opt := enum.ReviewerSuggestionKindOptional
	// One batch stamps the same created on every row, so these rows tie on rank and created.
	// Without the principal id tiebreaker the order isn't total and paginated reads could
	// repeat or drop rows.
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{
		suggestion(3, opt, "", nil),
		suggestion(1, opt, "", nil),
		suggestion(2, opt, "", nil),
	}); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	principals := func(list []*types.PullReqReviewerSuggestion) []int64 {
		ids := make([]int64, 0, len(list))
		for _, s := range list {
			ids = append(ids, s.PrincipalID)
		}
		return ids
	}

	all, err := suggestions.List(ctx, 100, types.Pagination{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := principals(all); !slices.Equal(got, []int64{1, 2, 3}) {
		t.Fatalf("principals = %v, want [1 2 3]", got)
	}

	// Paging through the tied rows must not repeat or drop any of them.
	first, err := suggestions.List(ctx, 100, types.Pagination{Page: 1, Size: 2})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	second, err := suggestions.List(ctx, 100, types.Pagination{Page: 2, Size: 2})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if got := principals(first); !slices.Equal(got, []int64{1, 2}) {
		t.Errorf("page 1 principals = %v, want [1 2]", got)
	}
	if got := principals(second); !slices.Equal(got, []int64{3}) {
		t.Errorf("page 2 principals = %v, want [3]", got)
	}
}

func TestPullReqReviewerSuggestionStore_UpsertManyEmptyIsNoop(t *testing.T) {
	ctx, suggestions, teardown := setupSuggestionTest(t)
	defer teardown()

	if err := suggestions.UpsertMany(ctx, nil); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}
	count, err := suggestions.Count(ctx, 100)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("Count = %d, want 0", count)
	}
}

func insertUserGroupRaw(ctx context.Context, t *testing.T, db *sqlx.DB, id int64) {
	t.Helper()

	identifier := fmt.Sprintf("group-%d", id)
	_, err := db.ExecContext(ctx, `
		INSERT INTO usergroups (usergroup_id, usergroup_identifier, usergroup_name, usergroup_space_id)
		VALUES (?, ?, ?, ?)`, id, identifier, identifier, suggestionTestSpaceID)
	if err != nil {
		t.Fatalf("insert user group: %v", err)
	}
}

func TestPullReqReviewerSuggestionStore_UserGroupIDRoundTrips(t *testing.T) {
	ctx, db, suggestions, teardown := setupSuggestionTestWithDB(t)
	defer teardown()

	insertUserGroupRaw(ctx, t, db, 900)

	in := suggestion(2, enum.ReviewerSuggestionKindRequired, "group owner", nil)
	in.UserGroupID = new(int64(900))
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{in}); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	list, err := suggestions.List(ctx, 100, types.Pagination{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].UserGroupID == nil || *list[0].UserGroupID != 900 {
		t.Errorf("unexpected suggestions: %+v", list)
	}
}

// The user group is part of why a person is suggested, so a refresh must move it along with the
// reason instead of leaving a stale attribution behind.
func TestPullReqReviewerSuggestionStore_UpsertManyRefreshesUserGroupID(t *testing.T) {
	ctx, db, suggestions, teardown := setupSuggestionTestWithDB(t)
	defer teardown()

	insertUserGroupRaw(ctx, t, db, 900)
	insertUserGroupRaw(ctx, t, db, 901)

	upsert := func(reason string, groupID *int64) *types.PullReqReviewerSuggestion {
		t.Helper()

		in := suggestion(2, enum.ReviewerSuggestionKindRequired, reason, nil)
		in.UserGroupID = groupID
		if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{in}); err != nil {
			t.Fatalf("UpsertMany %q: %v", reason, err)
		}
		got, err := suggestions.Find(ctx, 100, 2)
		if err != nil {
			t.Fatalf("Find %q: %v", reason, err)
		}

		return got
	}

	// groupOf reports 0 for "no group" so a failure prints a number rather than a pointer.
	groupOf := func(s *types.PullReqReviewerSuggestion) int64 {
		if s.UserGroupID == nil {
			return 0
		}

		return *s.UserGroupID
	}

	if got := groupOf(upsert("via group a", new(int64(900)))); got != 900 {
		t.Fatalf("first upsert: user group = %d, want 900", got)
	}
	if got := groupOf(upsert("via group b", new(int64(901)))); got != 901 {
		t.Errorf("moved to another group: user group = %d, want 901", got)
	}
	if got := groupOf(upsert("as an individual owner", nil)); got != 0 {
		t.Errorf("suggested without a group: user group = %d, want none", got)
	}
}

func TestPullReqReviewerSuggestionStore_DeletingUserGroupClearsUserGroupID(t *testing.T) {
	ctx, db, suggestions, teardown := setupSuggestionTestWithDB(t)
	defer teardown()

	insertUserGroupRaw(ctx, t, db, 900)

	in := suggestion(2, enum.ReviewerSuggestionKindRequired, "group owner", nil)
	in.UserGroupID = new(int64(900))
	if err := suggestions.UpsertMany(ctx, []*types.PullReqReviewerSuggestion{in}); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM usergroups WHERE usergroup_id = 900`); err != nil {
		t.Fatalf("delete user group: %v", err)
	}

	got, err := suggestions.Find(ctx, 100, 2)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got.UserGroupID != nil {
		t.Errorf("UserGroupID = %d, want nil after the user group is deleted", *got.UserGroupID)
	}
	if got.Reason != "group owner" {
		t.Errorf("suggestion must survive the user group deletion: %+v", got)
	}
}

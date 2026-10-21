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
	"reflect"
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

func TestPullReqStoreListRecentAuthorIDs(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	users := []*types.User{
		{ID: 1, UID: "alice", Email: "alice@example.com", DisplayName: "Alice"},
		{ID: 2, UID: "bob", Email: "bob@example.com", DisplayName: "Bob"},
		{ID: 3, UID: "carol", Email: "carol@example.com", DisplayName: "Carol"},
	}
	for _, user := range users {
		if err := principalStore.CreateUser(ctx, user); err != nil {
			t.Fatalf("create user %q: %v", user.UID, err)
		}
	}
	serviceAccount := &types.ServiceAccount{
		UID:         "ci-bot",
		Email:       "ci-bot@example.com",
		DisplayName: "CI Bot",
		ParentType:  enum.ParentResourceTypeSpace,
		ParentID:    1,
	}
	if err := principalStore.CreateServiceAccount(ctx, serviceAccount); err != nil {
		t.Fatalf("create service account: %v", err)
	}

	createSpace(ctx, t, spaceStore, spacePathStore, 1, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	createRepo(ctx, t, repoStore, 2, 1, 0)
	createRepo(ctx, t, repoStore, 3, 1, 0)

	insertAuthorPullReq(t, db, 1, 1, 1, 4)
	insertAuthorPullReq(t, db, 2, 1, 2, 5)
	insertAuthorPullReq(t, db, 3, 1, 2, 6)
	insertAuthorPullReq(t, db, 4, 1, 3, 3)
	insertAuthorPullReq(t, db, 5, 2, 3, 100)
	insertAuthorPullReq(t, db, 6, 1, serviceAccount.ID, 7)

	store := database.NewPullReqStore(db, nil)

	got, err := store.ListRecentAuthorIDs(ctx, 1, 3)
	if err != nil {
		t.Fatalf("ListRecentAuthorIDs: %v", err)
	}
	if want := []int64{2, 1, 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ListRecentAuthorIDs = %v, want %v", got, want)
	}

	got, err = store.ListRecentAuthorIDs(ctx, 1, 2)
	if err != nil {
		t.Fatalf("ListRecentAuthorIDs with limit: %v", err)
	}
	if want := []int64{2, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ListRecentAuthorIDs with limit = %v, want %v", got, want)
	}

	got, err = store.ListRecentAuthorIDs(ctx, 3, 3)
	if err != nil {
		t.Fatalf("ListRecentAuthorIDs for empty repository: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListRecentAuthorIDs for empty repository = %v, want empty", got)
	}
}

func insertAuthorPullReq(t *testing.T, db *sqlx.DB, id, repoID, authorID, number int64) {
	t.Helper()

	const query = `
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
		) VALUES (?, 0, ?, ?, ?, ?, ?, 'open', ?, '', ?, ?, ?, ?, 'main', 'unchecked', '', 1, 'space_1')`

	if _, err := db.Exec(
		query,
		id,
		authorID,
		number,
		number,
		number,
		number,
		fmt.Sprintf("PR %d", number),
		repoID,
		fmt.Sprintf("branch-%d", number),
		fmt.Sprintf("sha-%d", number),
		repoID,
	); err != nil {
		t.Fatalf("insert pull request %d: %v", id, err)
	}
}

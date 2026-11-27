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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

// TestPullReqStore_SourceSHAPushedBy verifies that the pusher of the source SHA
// is persisted on create, and that update can both change and clear it.
func TestPullReqStore_SourceSHAPushedBy(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)

	store := database.NewPullReqStore(db, activityPrincipalInfoCacheStub{})

	repoID := int64(1)
	pr := &types.PullReq{
		Number:              1,
		CreatedBy:           userID,
		State:               enum.PullReqStateOpen,
		Title:               "test",
		SourceRepoID:        &repoID,
		SourceBranch:        "feature",
		SourceSHA:           "1111111111111111111111111111111111111111",
		SourceSHAPushedBy:   ptr.Int64(userID),
		TargetRepoID:        repoID,
		TargetBranch:        "main",
		RootSpaceID:         1,
		RootSpaceIdentifier: "space_1",
		MergeCheckStatus:    enum.MergeCheckStatusUnchecked,
		RebaseCheckStatus:   enum.MergeCheckStatusUnchecked,
	}
	if err := store.Create(ctx, pr); err != nil {
		t.Fatalf("Create: %v", err)
	}

	assertPushedBy := func(t *testing.T, want *int64) {
		t.Helper()

		got, err := store.Find(ctx, pr.ID)
		if err != nil {
			t.Fatalf("Find: %v", err)
		}

		switch {
		case want == nil && got.SourceSHAPushedBy != nil:
			t.Fatalf("SourceSHAPushedBy = %d, want nil", *got.SourceSHAPushedBy)
		case want != nil && (got.SourceSHAPushedBy == nil || *got.SourceSHAPushedBy != *want):
			t.Fatalf("SourceSHAPushedBy = %v, want %d", got.SourceSHAPushedBy, *want)
		}
	}

	assertPushedBy(t, ptr.Int64(userID))

	pr.SourceSHA = "2222222222222222222222222222222222222222"
	pr.SourceSHAPushedBy = ptr.Int64(42)
	if err := store.Update(ctx, pr); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertPushedBy(t, ptr.Int64(42))

	pr.SourceSHAPushedBy = nil
	if err := store.Update(ctx, pr); err != nil {
		t.Fatalf("Update: %v", err)
	}
	assertPushedBy(t, nil)
}

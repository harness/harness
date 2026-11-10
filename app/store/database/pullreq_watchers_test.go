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
	"strconv"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

// principalInfoCacheStub resolves principal infos from a static map.
type principalInfoCacheStub struct {
	store.PrincipalInfoCache
}

func (principalInfoCacheStub) Map(_ context.Context, ids []int64) (map[int64]*types.PrincipalInfo, error) {
	m := make(map[int64]*types.PrincipalInfo, len(ids))
	for _, id := range ids {
		m[id] = &types.PrincipalInfo{ID: id}
	}
	return m, nil
}

// insertPrincipalRaw inserts an extra user so that multiple principals can watch the same PR.
func insertPrincipalRaw(ctx context.Context, t *testing.T, principalStore *database.PrincipalStore, id int64) {
	t.Helper()

	uid := "watcher_" + strconv.FormatInt(id, 10)
	if err := principalStore.CreateUser(ctx, &types.User{ID: id, UID: uid, Email: uid + "@example.com"}); err != nil {
		t.Fatalf("failed to create user %d: %v", id, err)
	}
}

func setupWatcherTest(t *testing.T) (context.Context, *sqlx.DB, *database.PullReqWatcherStore, func()) {
	t.Helper()

	db, teardown := setupDB(t)
	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	insertPrincipalRaw(ctx, t, principalStore, 2)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	insertPullReqRaw(ctx, t, db, 100, 1, nil)
	insertPullReqRaw(ctx, t, db, 101, 2, nil)

	return ctx, db, database.NewPullReqWatcherStore(db, principalInfoCacheStub{}), teardown
}

func TestPullReqWatcherStore_CreateIsIdempotent(t *testing.T) {
	ctx, _, watchers, teardown := setupWatcherTest(t)
	defer teardown()

	for range 2 {
		if err := watchers.Create(ctx, 100, 1); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	count, err := watchers.Count(ctx, 100)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}
}

func TestPullReqWatcherStore_ExistsAndDelete(t *testing.T) {
	ctx, _, watchers, teardown := setupWatcherTest(t)
	defer teardown()

	if ok, err := watchers.Exists(ctx, 100, 1); err != nil || ok {
		t.Fatalf("Exists before watch = %v, %v; want false, nil", ok, err)
	}

	if err := watchers.Create(ctx, 100, 1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ok, err := watchers.Exists(ctx, 100, 1); err != nil || !ok {
		t.Fatalf("Exists after watch = %v, %v; want true, nil", ok, err)
	}
	// watching one PR doesn't affect another
	if ok, err := watchers.Exists(ctx, 101, 1); err != nil || ok {
		t.Fatalf("Exists on other PR = %v, %v; want false, nil", ok, err)
	}

	// deleting twice must not fail
	for range 2 {
		if err := watchers.Delete(ctx, 100, 1); err != nil {
			t.Fatalf("Delete: %v", err)
		}
	}
	if ok, err := watchers.Exists(ctx, 100, 1); err != nil || ok {
		t.Fatalf("Exists after unwatch = %v, %v; want false, nil", ok, err)
	}
}

func TestPullReqWatcherStore_ListOrderedWithPrincipalInfo(t *testing.T) {
	ctx, _, watchers, teardown := setupWatcherTest(t)
	defer teardown()

	for _, principalID := range []int64{2, 1} {
		if err := watchers.Create(ctx, 100, principalID); err != nil {
			t.Fatalf("Create(%d): %v", principalID, err)
		}
	}
	if err := watchers.Create(ctx, 101, 1); err != nil {
		t.Fatalf("Create other PR: %v", err)
	}

	got, err := watchers.List(ctx, 100)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List returned %d watchers, want 2", len(got))
	}
	for _, w := range got {
		if w.Watcher.ID != w.PrincipalID {
			t.Errorf("watcher info not attached: %+v", w)
		}
		if w.PullReqID != 100 {
			t.Errorf("watcher of wrong PR returned: %+v", w)
		}
	}
}

func TestPullReqWatcherStore_RemovedWithPullReq(t *testing.T) {
	ctx, db, watchers, teardown := setupWatcherTest(t)
	defer teardown()

	if err := watchers.Create(ctx, 100, 1); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM pullreqs WHERE pullreq_id = 100`); err != nil {
		t.Fatalf("delete pullreq: %v", err)
	}

	count, err := watchers.Count(ctx, 100)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("Count after PR delete = %d, want 0 (cascade)", count)
	}
}

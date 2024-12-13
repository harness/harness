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
	"os"
	"strconv"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/rs/xid"
)

var userID int64 = 1

func New(dsn string) (*sqlx.DB, error) {
	if dsn == ":memory:" {
		dsn = fmt.Sprintf("file:%s.db?mode=memory&cache=shared", xid.New().String())
	}
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("foreign keys pragma: %w", err)
	}
	return db, nil
}

func setupDB(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()
	// must use file as db because in memory have only basic features
	// file is anyway removed on every test. SQLite is fast
	// so it will not affect too much performance.
	_ = os.Remove("test.db")
	db, err := New("test.db")
	if err != nil {
		t.Fatalf("Error opening db, err: %v", err)
	}
	_, _ = db.Exec("PRAGMA busy_timeout = 5000;")
	if err = migrate.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Error migrating db, err: %v", err)
	}

	return db, func() {
		db.Close()
	}
}

func setupStores(t *testing.T, db *sqlx.DB) (
	*database.PrincipalStore,
	*database.SpaceStore,
	store.SpacePathStore,
	*database.RepoStore,
) {
	t.Helper()

	principalStore := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)

	spacePathTransformation := store.ToLowerSpacePathTransformation
	spacePathStore := database.NewSpacePathStore(db, store.ToLowerSpacePathTransformation)
	spacePathCache := cache.New(spacePathStore, spacePathTransformation)

	spaceStore := database.NewSpaceStore(db, spacePathCache, spacePathStore)
	repoStore := database.NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)

	return principalStore, spaceStore, spacePathStore, repoStore
}

func createUser(
	ctx context.Context,
	t *testing.T,
	principalStore *database.PrincipalStore,
) {
	t.Helper()

	uid := "user_" + strconv.FormatInt(userID, 10)
	if err := principalStore.CreateUser(ctx,
		&types.User{ID: userID, UID: uid}); err != nil {
		t.Fatalf("failed to create user %v", err)
	}
}

func createSpace(
	ctx context.Context,
	t *testing.T,
	spaceStore *database.SpaceStore,
	spacePathStore store.SpacePathStore,
	userID int64,
	spaceID int64,
	parentID int64,
) {
	t.Helper()

	identifier := "space_" + strconv.FormatInt(spaceID, 10)

	space := types.Space{ID: spaceID, Identifier: identifier, CreatedBy: userID, ParentID: parentID}
	if err := spaceStore.Create(ctx, &space); err != nil {
		t.Fatalf("failed to create space %v", err)
	}

	if err := spacePathStore.InsertSegment(ctx, &types.SpacePathSegment{
		ID: space.ID, Identifier: identifier, CreatedBy: userID, SpaceID: spaceID, IsPrimary: true,
	}); err != nil {
		t.Fatalf("failed to insert segment %v", err)
	}
}

func createSpaceTree() (map[int64][]int64, int) {
	spaceTree := make(map[int64][]int64)
	spaceTree[1] = []int64{2, 3}
	spaceTree[2] = []int64{4, 5, 6}
	spaceTree[3] = []int64{7, 8}
	spaceTree[4] = []int64{9, 10}
	spaceTree[5] = []int64{11}
	return spaceTree, 11
}

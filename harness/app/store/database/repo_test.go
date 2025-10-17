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
	"strconv"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
)

const (
	numTestRepos = 10
	repoSize     = int64(100)
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

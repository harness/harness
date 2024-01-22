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
	"math/rand"
	"strconv"
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
)

const repoSize = int64(100)

func TestDatabase_GetSize(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(t, &ctx, principalStore, 1)
	createSpace(t, &ctx, spaceStore, spacePathStore, userID, 1, 0)

	repoID := int64(1)
	createRepo(t, &ctx, repoStore, repoID, 1, repoSize)

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

func TestDatabase_CountAll(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(t, &ctx, principalStore, 1)

	var numRepos int64
	spaceTree, numSpaces := createSpaceTree()
	createSpace(t, &ctx, spaceStore, spacePathStore, userID, 1, 0)
	for i := 1; i < numSpaces; i++ {
		parentID := int64(i)
		for _, spaceID := range spaceTree[parentID] {
			createSpace(t, &ctx, spaceStore, spacePathStore, userID, spaceID, parentID)

			for j := 0; j < rand.Intn(4); j++ {
				numRepos++
				createRepo(t, &ctx, repoStore, numRepos, spaceID, 0)
			}
		}
	}

	count, err := repoStore.CountAll(ctx, 1)
	if err != nil {
		t.Fatalf("failed to count repos %v", err)
	}
	if count != numRepos {
		t.Errorf("count = %v, want %v", count, numRepos)
	}
}

func createRepo(
	t *testing.T,
	ctx *context.Context,
	repoStore *database.RepoStore,
	id int64,
	spaceID int64,
	size int64,
) {
	t.Helper()

	uid := "repo_" + strconv.FormatInt(id, 10)
	repo := types.Repository{UID: uid, ID: id, ParentID: spaceID, GitUID: uid, Size: size}
	if err := repoStore.Create(*ctx, &repo); err != nil {
		t.Fatalf("failed to create repo %v", err)
	}
}

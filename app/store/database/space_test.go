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
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestDatabase_GetRootSpace(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	numSpaces := createNestedSpaces(ctx, t, spaceStore, spacePathStore)

	for i := 1; i <= numSpaces; i++ {
		rootSpc, err := spaceStore.GetRootSpace(ctx, int64(i))
		if err != nil {
			t.Fatalf("failed to get root space %v", err)
		}
		if rootSpc.ID != 1 {
			t.Errorf("rootSpc.ID = %v, want %v", rootSpc.ID, 1)
		}
	}
}

func TestSpaceStore_FindByIDs(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)

	_ = createNestedSpaces(ctx, t, spaceStore, spacePathStore)

	spaces, err := spaceStore.FindByIDs(ctx, 4, 5, 6)
	require.NoError(t, err)

	require.Len(t, spaces, 3)
	require.Equal(t, int64(4), spaces[0].ID)
	require.Equal(t, int64(5), spaces[1].ID)
	require.Equal(t, int64(6), spaces[2].ID)
}

func createNestedSpacesForStorageSize(
	ctx context.Context,
	t *testing.T,
	spaceStore store.SpaceStore,
) []types.Space {
	// Create root spaces
	rootSpaces := []types.Space{
		{Identifier: "root-space-1", Description: "Root Space 1"},
		{Identifier: "root-space-2", Description: "Root Space 2"},
		{Identifier: "root-space-3", Description: "Root Space 3"},
	}

	for _, rootSpace := range rootSpaces {
		err := spaceStore.Create(ctx, &rootSpace)
		require.NoError(t, err)

		// Create nested subspaces for each root space
		for j := 1; j <= 3; j++ { // Create 3 subspaces for each root space
			subSpace := types.Space{
				Identifier:  fmt.Sprintf("sub-space-%d-of-%s", j, rootSpace.Identifier),
				ParentID:    rootSpace.ID, // Set the parent ID to the root space ID
				Description: fmt.Sprintf("Sub Space %d of %s", j, rootSpace.Identifier),
			}
			err = spaceStore.Create(ctx, &subSpace)
			require.NoError(t, err)
		}
	}

	return rootSpaces
}

func createRepositoriesForSpaces(
	ctx context.Context,
	t *testing.T,
	db dbtx.Accessor,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
) (rootSpaces []types.Space, total int64) {
	rootSpaces = createNestedSpacesForStorageSize(ctx, t, spaceStore)

	type row struct {
		ID         int64
		Identifier string
		ParentID   *int64
	}

	// Directly query the database for all spaces
	var spaces []row
	query := "SELECT space_id, space_uid, space_parent_id FROM spaces"

	rows, err := db.QueryContext(ctx, query)
	require.NoError(t, err)

	defer rows.Close()

	for rows.Next() {
		var space row
		err := rows.Scan(
			&space.ID,
			&space.Identifier,
			&space.ParentID,
		)
		require.NoError(t, err)
		spaces = append(spaces, space)
	}

	require.NoError(t, rows.Err())

	defaultSize := int64(100)

	// Print out the spaces
	for i, space := range spaces {
		t.Logf("Space ID: %d, Identifier: %s, Size: %d", space.ID, space.Identifier, defaultSize)

		repo := &types.Repository{
			ParentID:   space.ID,
			GitUID:     fmt.Sprintf("repo-%d", i),
			Identifier: fmt.Sprintf("repo-%d", i),
			Tags:       json.RawMessage{},
		}
		err := repoStore.Create(ctx, repo) // Assuming CreateRepository is defined
		require.NoError(t, err)

		err = repoStore.UpdateSize(ctx, repo.ID, defaultSize, defaultSize)
		require.NoError(t, err)

		total += defaultSize
	}

	// add one deleted repo
	repo := &types.Repository{
		ParentID:   spaces[0].ID,
		GitUID:     "repo-deleted",
		Identifier: "repo-deleted",
		Tags:       json.RawMessage{},
	}
	err = repoStore.Create(ctx, repo) // Assuming CreateRepository is defined
	require.NoError(t, err)

	err = repoStore.UpdateSize(ctx, repo.ID, defaultSize, defaultSize)
	require.NoError(t, err)

	err = repoStore.SoftDelete(ctx, repo, time.Now().Unix())
	require.NoError(t, err)

	return rootSpaces, total
}

func TestSpaceStore_GetRootSpacesSize(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, _, repoStore := setupStores(t, db)

	ctx := context.Background()

	// Create a user for context
	createUser(ctx, t, principalStore)

	// Create repositories for each space
	rootSpaces, totalSize := createRepositoriesForSpaces(ctx, t, db, repoStore, spaceStore)

	// Call the GetRootSpacesSize function
	spaces, err := spaceStore.GetRootSpacesSize(ctx)
	require.NoError(t, err)

	// Verify the results
	require.NotNil(t, spaces)
	require.Greater(t, len(spaces), 0, "Expected at least one root space")

	expectedSize := totalSize / int64(len(rootSpaces))

	for i, space := range rootSpaces {
		require.Equal(t, space.Identifier, spaces[i].Identifier)
		require.Equal(t, expectedSize, spaces[i].Size)
		require.Equal(t, expectedSize, spaces[i].LFSSize)
	}
}

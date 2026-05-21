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
)

func TestLinkedRepoStore_CreateFind_RoundTripsProviderColumns(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	const repoID int64 = 1
	createRepo(ctx, t, repoStore, repoID, 1, 0)

	store := database.NewLinkedRepoStore(db)

	in := &types.LinkedRepo{
		RepoID:              repoID,
		Version:             0,
		Created:             1700000000000,
		Updated:             1700000000000,
		LastFullSync:        1700000000000,
		ConnectorPath:       "account.acme",
		ConnectorIdentifier: "githubConn",
		ConnectorRepo:       "acme/widgets",
		ProviderRepoID:      "12345",
		ProviderType:        "github",
	}

	if err := store.Create(ctx, in); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := store.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}

	if got.ProviderRepoID != in.ProviderRepoID {
		t.Errorf("ProviderRepoID = %q, want %q", got.ProviderRepoID, in.ProviderRepoID)
	}
	if got.ProviderType != in.ProviderType {
		t.Errorf("ProviderType = %q, want %q", got.ProviderType, in.ProviderType)
	}
	if got.ConnectorPath != in.ConnectorPath {
		t.Errorf("ConnectorPath = %q, want %q", got.ConnectorPath, in.ConnectorPath)
	}
	if got.ConnectorIdentifier != in.ConnectorIdentifier {
		t.Errorf("ConnectorIdentifier = %q, want %q", got.ConnectorIdentifier, in.ConnectorIdentifier)
	}
}

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

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
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

// ExistsByRootSpace is the cheap pre-filter the webhook consumer runs against the
// platform-wide firehose; it matches on the owning repo's root space identifier,
// which is the account id in Harness.
func TestLinkedRepoStore_ExistsByRootSpace(t *testing.T) {
	const rootSpaceIdentifier = "acctXYZ"

	cases := []struct {
		name          string
		repoRootSpace string
		query         string
		want          bool
	}{
		{
			name:          "repo in the root space matches",
			repoRootSpace: rootSpaceIdentifier,
			query:         rootSpaceIdentifier,
			want:          true,
		},
		{
			name:          "other root space does not match",
			repoRootSpace: rootSpaceIdentifier,
			query:         "otherAccount",
			want:          false,
		},
		{
			name:          "root space that is only a string prefix does not match",
			repoRootSpace: rootSpaceIdentifier + "extra",
			query:         rootSpaceIdentifier,
			want:          false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, teardown := setupDB(t)
			defer teardown()

			principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
			ctx := context.Background()

			createUser(ctx, t, principalStore)
			createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

			repoID := createLinkedTypeRepo(ctx, t, repoStore, tc.repoRootSpace)

			store := database.NewLinkedRepoStore(db)
			if err := store.Create(ctx, &types.LinkedRepo{
				RepoID:              repoID,
				ConnectorPath:       "irrelevant/to/root/space",
				ConnectorIdentifier: "githubConn",
				ConnectorRepo:       "acme/widgets",
				ProviderRepoID:      "12345",
				ProviderType:        "github",
			}); err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			got, err := store.ExistsByRootSpace(ctx, tc.query)
			if err != nil {
				t.Fatalf("ExistsByRootSpace() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("ExistsByRootSpace(%q) with repo root space %q = %v, want %v",
					tc.query, tc.repoRootSpace, got, tc.want)
			}
		})
	}
}

// A soft-deleted repo's linked row must not keep the root space on the hot path:
// ListByProviderID filters it out, so the pre-filter has to agree or every
// event for the root space pays a query that can only return nothing.
func TestLinkedRepoStore_ExistsByRootSpace_IgnoresDeletedRepos(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	repoID := createLinkedTypeRepo(ctx, t, repoStore, "acctXYZ")

	store := database.NewLinkedRepoStore(db)
	if err := store.Create(ctx, &types.LinkedRepo{
		RepoID:              repoID,
		ConnectorPath:       "acctXYZ",
		ConnectorIdentifier: "githubConn",
		ConnectorRepo:       "acme/widgets",
		ProviderRepoID:      "12345",
		ProviderType:        "github",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	repo, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if err := repoStore.SoftDelete(ctx, repo, 1700000000000); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	got, err := store.ExistsByRootSpace(ctx, "acctXYZ")
	if err != nil {
		t.Fatalf("ExistsByRootSpace() error = %v", err)
	}
	if got {
		t.Error("ExistsByRootSpace() = true for a soft-deleted repo, want false")
	}
}

// createLinkedTypeRepo creates a repo of type linked under space 1 with the given
// root space identifier and returns its DB-assigned id.
func createLinkedTypeRepo(
	ctx context.Context,
	t *testing.T,
	repoStore *database.RepoStore,
	rootSpaceIdentifier string,
) int64 {
	t.Helper()

	repo := types.Repository{
		ParentID:            1,
		RootSpaceID:         1,
		RootSpaceIdentifier: rootSpaceIdentifier,
		Type:                enum.RepoTypeLinked,
		Tags:                json.RawMessage{},
	}
	if err := repoStore.Create(ctx, &repo); err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	identifier := "repo_" + strconv.FormatInt(repo.ID, 10)
	repo.Identifier = identifier
	repo.GitUID = identifier
	if err := repoStore.Update(ctx, &repo); err != nil {
		t.Fatalf("failed to set repo %d identifier: %v", repo.ID, err)
	}

	return repo.ID
}

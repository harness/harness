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

	"github.com/harness/gitness/types"
)

// TestDatabase_RepoLOCRoundTrip verifies that the repository LOC (scc source
// lines of code) persists via the opt-lock update path and reads back intact.
// It also exercises the 0190 migration through the shared sqlite test harness.
func TestDatabase_RepoLOCRoundTrip(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	repoID := int64(1)
	createRepo(ctx, t, repoStore, repoID, 1, repoSize)

	repo, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}

	// Freshly created repos default to zero LOC.
	if repo.LOC != 0 {
		t.Errorf("new repo LOC = %d, want 0", repo.LOC)
	}

	const want = int64(1200)
	if _, err := repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		r.LOC = want
		return nil
	}); err != nil {
		t.Fatalf("UpdateOptLock() error = %v", err)
	}

	got, err := repoStore.Find(ctx, repoID)
	if err != nil {
		t.Fatalf("Find() after update error = %v", err)
	}
	if got.LOC != want {
		t.Errorf("persisted repo LOC = %d, want %d", got.LOC, want)
	}
}

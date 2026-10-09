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
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestCheckStore_ResultSummary_ExcludePayloadKinds(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)

	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	const repoID int64 = 1
	createRepo(ctx, t, repoStore, repoID, 1, 0)

	const (
		mixedSHA       = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		agentOnlySHA   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		agentReportKey = enum.CheckPayloadKind("agent_report")
	)

	checkStore := database.NewCheckStore(db, nil)
	for _, check := range []types.Check{
		{CommitSHA: mixedSHA, Identifier: "ci", Status: enum.CheckStatusSuccess},
		{CommitSHA: mixedSHA, Identifier: "lint", Status: enum.CheckStatusRunning},
		{CommitSHA: mixedSHA, Identifier: "review", Status: enum.CheckStatusFailure, Payload: types.CheckPayload{
			Kind: agentReportKey,
		}},
		{CommitSHA: agentOnlySHA, Identifier: "review", Status: enum.CheckStatusSuccess, Payload: types.CheckPayload{
			Kind: agentReportKey,
		}},
	} {
		check.CreatedBy = userID
		check.RepoID = repoID
		check.Metadata = json.RawMessage("{}")
		check.Payload.Data = json.RawMessage("{}")
		if err := checkStore.Upsert(ctx, &check); err != nil {
			t.Fatalf("Upsert(%s) error = %v", check.Identifier, err)
		}
	}

	commitSHAs := []string{mixedSHA, agentOnlySHA}

	all, err := checkStore.ResultSummary(ctx, repoID, commitSHAs, nil)
	if err != nil {
		t.Fatalf("ResultSummary() error = %v", err)
	}
	wantAll := map[sha.SHA]types.CheckCountSummary{
		sha.Must(mixedSHA):     {Success: 1, Running: 1, Failure: 1},
		sha.Must(agentOnlySHA): {Success: 1},
	}
	if len(all) != len(wantAll) {
		t.Fatalf("ResultSummary() = %v, want %v", all, wantAll)
	}
	for commitSHA, want := range wantAll {
		if all[commitSHA] != want {
			t.Errorf("ResultSummary()[%s] = %+v, want %+v", commitSHA, all[commitSHA], want)
		}
	}

	filtered, err := checkStore.ResultSummary(ctx, repoID, commitSHAs, []enum.CheckPayloadKind{agentReportKey})
	if err != nil {
		t.Fatalf("ResultSummary() error = %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("ResultSummary() with exclusion = %v, want only %s", filtered, mixedSHA)
	}
	if got, want := filtered[sha.Must(mixedSHA)], (types.CheckCountSummary{Success: 1, Running: 1}); got != want {
		t.Errorf("ResultSummary()[%s] = %+v, want %+v", mixedSHA, got, want)
	}
}

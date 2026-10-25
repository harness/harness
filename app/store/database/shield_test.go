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
)

type shieldTestEnv struct {
	tasks    *database.TaskStore
	scans    *database.ShieldScanStore
	findings *database.ShieldFindingStore
}

func setupShieldEnv(t *testing.T) (context.Context, *shieldTestEnv, func()) {
	t.Helper()

	db, teardown := setupDB(t)
	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	createRepo(ctx, t, repoStore, 2, 1, 0)

	return ctx, &shieldTestEnv{
		tasks:    database.NewTaskStore(db),
		scans:    database.NewShieldScanStore(db),
		findings: database.NewShieldFindingStore(db),
	}, teardown
}

func newShieldScan(t *testing.T, ctx context.Context, env *shieldTestEnv, repoID int64) *types.ShieldScan {
	t.Helper()
	scan := &types.ShieldScan{
		RepoID:  repoID,
		Trigger: enum.ShieldScanTriggerManual,
		Scanner: enum.ShieldScannerOSV,
		Status:  enum.ShieldScanStatusRunning,
		Started: 1000,
		Created: 1000,
	}
	if err := env.scans.Create(ctx, scan); err != nil {
		t.Fatalf("Create scan error = %v", err)
	}
	return scan
}

func newShieldFinding(repoID, scanID int64, fp string, risk int, now int64) *types.ShieldFinding {
	strategy := enum.ShieldFixStrategyBump
	return &types.ShieldFinding{
		RepoID:          repoID,
		Fingerprint:     fp,
		Scanner:         enum.ShieldScannerOSV,
		Kind:            enum.ShieldFindingKindDependency,
		Ecosystem:       "npm",
		Package:         "lodash",
		VulnID:          "GHSA-" + fp,
		Aliases:         []string{"CVE-2020-1"},
		FixedVersions:   []string{"4.17.21"},
		FixStrategy:     &strategy,
		MinFixedVersion: "4.17.21",
		Reachable:       enum.ShieldReachabilityUnknown,
		RiskScore:       risk,
		State:           enum.ShieldFindingStateOpen,
		FirstSeenScanID: scanID,
		LastSeenScanID:  scanID,
		FirstSeen:       now,
		LastSeen:        now,
		Created:         now,
		Updated:         now,
	}
}

func TestShieldScanStore_CreateUpdateFindList(t *testing.T) {
	ctx, env, teardown := setupShieldEnv(t)
	defer teardown()

	scan := newShieldScan(t, ctx, env, 1)
	newShieldScan(t, ctx, env, 1)
	newShieldScan(t, ctx, env, 2)

	finished := int64(2000)
	scan.Status = enum.ShieldScanStatusDone
	scan.PackagesCount = 10
	scan.FindingsCount = 3
	scan.Finished = &finished
	if err := env.scans.Update(ctx, scan); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := env.scans.Find(ctx, scan.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if got.Status != enum.ShieldScanStatusDone || got.PackagesCount != 10 || got.FindingsCount != 3 ||
		got.Finished == nil || *got.Finished != 2000 {
		t.Errorf("unexpected scan after update: %+v", got)
	}

	list, err := env.scans.ListByRepo(ctx, 1, 1, 10)
	if err != nil {
		t.Fatalf("ListByRepo() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByRepo() = %d scans, want 2", len(list))
	}
}

func TestShieldFindingStore_UpsertLifecycle(t *testing.T) {
	ctx, env, teardown := setupShieldEnv(t)
	defer teardown()

	scan1 := newShieldScan(t, ctx, env, 1)
	f := newShieldFinding(1, scan1.ID, "a", 70, 1000)
	if err := env.findings.Upsert(ctx, f); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if f.ID == 0 {
		t.Fatal("expected ID to be set")
	}

	// second scan sees the same finding with a new risk
	scan2 := newShieldScan(t, ctx, env, 1)
	again := newShieldFinding(1, scan2.ID, "a", 90, 2000)
	again.FirstSeenScanID, again.FirstSeen, again.Created = scan2.ID, 2000, 2000
	if err := env.findings.Upsert(ctx, again); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if again.ID != f.ID {
		t.Errorf("ID = %d, want %d (same finding)", again.ID, f.ID)
	}
	if again.RiskScore != 90 || again.LastSeen != 2000 || again.LastSeenScanID != scan2.ID {
		t.Errorf("last seen / risk not refreshed: %+v", again)
	}
	if again.FirstSeen != 1000 || again.FirstSeenScanID != scan1.ID {
		t.Errorf("first seen must be preserved: %+v", again)
	}
	if again.Version != 1 {
		t.Errorf("Version = %d, want 1", again.Version)
	}

	// another finding only in scan2; "a" is not seen in scan3 -> fixed
	other := newShieldFinding(1, scan2.ID, "b", 10, 2000)
	if err := env.findings.Upsert(ctx, other); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	scan3 := newShieldScan(t, ctx, env, 1)
	seen := newShieldFinding(1, scan3.ID, "b", 10, 3000)
	if err := env.findings.Upsert(ctx, seen); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	n, err := env.findings.MarkFixedNotSeenIn(ctx, 1, scan3.ID, 3000)
	if err != nil {
		t.Fatalf("MarkFixedNotSeenIn() error = %v", err)
	}
	if n != 1 {
		t.Errorf("fixed count = %d, want 1", n)
	}
	got, err := env.findings.Find(ctx, f.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if got.State != enum.ShieldFindingStateFixed || got.FixedAt == nil || *got.FixedAt != 3000 {
		t.Errorf("expected fixed with fixed_at, got %+v", got)
	}

	// reappears -> reopened
	scan4 := newShieldScan(t, ctx, env, 1)
	back := newShieldFinding(1, scan4.ID, "a", 90, 4000)
	if err = env.findings.Upsert(ctx, back); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if back.State != enum.ShieldFindingStateOpen || back.FixedAt != nil {
		t.Errorf("expected reopened finding, got state=%s fixed_at=%v", back.State, back.FixedAt)
	}
}

func TestShieldFindingStore_UpsertKeepsIgnored(t *testing.T) {
	ctx, env, teardown := setupShieldEnv(t)
	defer teardown()

	scan := newShieldScan(t, ctx, env, 1)
	f := newShieldFinding(1, scan.ID, "a", 50, 1000)
	if err := env.findings.Upsert(ctx, f); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := env.findings.UpdateState(ctx, f.ID, enum.ShieldFindingStateIgnored, "false positive", 1500); err != nil {
		t.Fatalf("UpdateState() error = %v", err)
	}

	again := newShieldFinding(1, scan.ID, "a", 55, 2000)
	if err := env.findings.Upsert(ctx, again); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if again.State != enum.ShieldFindingStateIgnored || again.StateReason != "false positive" {
		t.Errorf("ignored state must be kept, got %s / %q", again.State, again.StateReason)
	}
}

func TestShieldFindingStore_ListCountFilters(t *testing.T) {
	ctx, env, teardown := setupShieldEnv(t)
	defer teardown()

	scan := newShieldScan(t, ctx, env, 1)
	for i, risk := range []int{10, 40, 80, 95} {
		f := newShieldFinding(1, scan.ID, string(rune('a'+i)), risk, 1000)
		if i == 3 {
			f.Kind = enum.ShieldFindingKindSecret
		}
		if err := env.findings.Upsert(ctx, f); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}
	if err := env.findings.Upsert(ctx, newShieldFinding(2, scan.ID, "z", 99, 1000)); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	all, err := env.findings.List(ctx, 1, &types.ShieldFindingFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 4 || all[0].RiskScore != 95 || all[3].RiskScore != 10 {
		t.Errorf("unexpected list order/size: %d", len(all))
	}

	filter := &types.ShieldFindingFilter{
		MinRiskScore: 40,
		Kinds:        []enum.ShieldFindingKind{enum.ShieldFindingKindDependency},
		States:       []enum.ShieldFindingState{enum.ShieldFindingStateOpen},
	}
	count, err := env.findings.Count(ctx, 1, filter)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}

	page, err := env.findings.List(ctx, 1, &types.ShieldFindingFilter{Page: 2, Size: 3})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page) != 1 {
		t.Errorf("page 2 size 3 = %d findings, want 1", len(page))
	}
}

func TestShieldFindingStore_SetTask(t *testing.T) {
	ctx, env, teardown := setupShieldEnv(t)
	defer teardown()

	scan := newShieldScan(t, ctx, env, 1)
	f := newShieldFinding(1, scan.ID, "a", 50, 1000)
	if err := env.findings.Upsert(ctx, f); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	task := newTask(1, "fix vuln")
	if err := env.tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task error = %v", err)
	}

	if err := env.findings.SetTask(ctx, f.ID, &task.ID, 2000); err != nil {
		t.Fatalf("SetTask() error = %v", err)
	}
	got, err := env.findings.Find(ctx, f.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if got.TaskID == nil || *got.TaskID != task.ID {
		t.Errorf("TaskID = %v, want %d", got.TaskID, task.ID)
	}

	if err = env.findings.SetTask(ctx, f.ID, nil, 3000); err != nil {
		t.Fatalf("SetTask(nil) error = %v", err)
	}
	got, _ = env.findings.Find(ctx, f.ID)
	if got.TaskID != nil {
		t.Errorf("TaskID = %v, want nil", got.TaskID)
	}
}

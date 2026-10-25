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
	"errors"
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

type insightsTestEnv struct {
	db       *sqlx.DB
	projects *database.InsightsProjectStore
	issues   *database.InsightsIssueStore
	events   *database.InsightsEventStore
	tasks    *database.TaskStore
	repos    *database.RepoStore
}

func setupInsightsEnv(t *testing.T) (context.Context, *insightsTestEnv, func()) {
	t.Helper()

	db, teardown := setupDB(t)
	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	createRepo(ctx, t, repoStore, 2, 1, 0)

	return ctx, &insightsTestEnv{
		db:       db,
		projects: database.NewInsightsProjectStore(db),
		issues:   database.NewInsightsIssueStore(db),
		events:   database.NewInsightsEventStore(db),
		tasks:    database.NewTaskStore(db),
		repos:    repoStore,
	}, teardown
}

func newInsightsProject(repoID int64, identifier, key string) *types.InsightsProject {
	return &types.InsightsProject{
		RepoID: repoID, Identifier: identifier, PublicKey: key, Enabled: true,
		RateLimitPerMinute: 600, SampleRate: 0.25, CreatedBy: userID, Created: 1700000000000, Updated: 1700000000000,
	}
}

func newInsightsIssue(p *types.InsightsProject, fingerprint string, lastSeen int64) *types.InsightsIssue {
	return &types.InsightsIssue{
		ProjectID: p.ID, RepoID: p.RepoID, Fingerprint: fingerprint, Title: "TypeError: boom " + fingerprint,
		Level: "error", Platform: "go", FirstSeen: lastSeen, LastSeen: lastSeen, TimesSeen: 1,
		FirstRelease: "1.0.0", LastRelease: "1.0.0", Status: enum.InsightsIssueStatusUnresolved,
		Created: lastSeen, Updated: lastSeen,
	}
}

func TestInsightsProjectStore(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-api")
	if err := env.projects.Create(ctx, p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if p.ID == 0 {
		t.Fatal("ID not set")
	}

	got, err := env.projects.Find(ctx, p.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if *got != *p {
		t.Errorf("Find() = %+v, want %+v", got, p)
	}

	byIdent, err := env.projects.FindByIdentifier(ctx, 1, "api")
	if err != nil || byIdent.ID != p.ID {
		t.Errorf("FindByIdentifier() = %v, %v", byIdent, err)
	}
	if _, err = env.projects.FindByIdentifier(ctx, 2, "api"); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Errorf("FindByIdentifier() other repo error = %v, want not found", err)
	}

	// identifier unique per repo, key unique globally.
	if err = env.projects.Create(ctx, newInsightsProject(1, "api", "other-key")); !errors.Is(err, gitness_store.ErrDuplicate) {
		t.Errorf("duplicate identifier error = %v, want ErrDuplicate", err)
	}
	if err = env.projects.Create(ctx, newInsightsProject(2, "other", "key-api")); !errors.Is(err, gitness_store.ErrDuplicate) {
		t.Errorf("duplicate key error = %v, want ErrDuplicate", err)
	}
	if err = env.projects.Create(ctx, newInsightsProject(2, "api", "key-api-2")); err != nil {
		t.Errorf("same identifier in another repo: %v", err)
	}

	// sample rate range is enforced by the database.
	bad := newInsightsProject(1, "bad", "key-bad")
	bad.SampleRate = 1.5
	if err = env.projects.Create(ctx, bad); err == nil {
		t.Error("sample rate 1.5 accepted")
	}

	p.Enabled, p.RateLimitPerMinute, p.SampleRate, p.Updated = false, 5, 0.5, 1700000001000
	if err = env.projects.Update(ctx, p); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	got, _ = env.projects.Find(ctx, p.ID)
	if got.Enabled || got.RateLimitPerMinute != 5 || got.SampleRate != 0.5 {
		t.Errorf("after update = %+v", got)
	}

	if err = env.projects.Update(ctx, &types.InsightsProject{ID: 9999}); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Errorf("Update() missing error = %v", err)
	}

	list, err := env.projects.List(ctx, 1)
	if err != nil || len(list) != 1 || list[0].Identifier != "api" {
		t.Errorf("List() = %v, %v", list, err)
	}
}

func TestInsightsIssueStore(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-api")
	if err := env.projects.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	a := newInsightsIssue(p, "fp-a", 1000)
	if err := env.issues.Create(ctx, a); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := env.issues.Create(ctx, newInsightsIssue(p, "fp-a", 2000)); !errors.Is(err, gitness_store.ErrDuplicate) {
		t.Errorf("duplicate fingerprint error = %v, want ErrDuplicate", err)
	}
	b := newInsightsIssue(p, "fp-b", 3000)
	b.Status = enum.InsightsIssueStatusResolved
	b.ResolvedInRelease = "1.1.0"
	if err := env.issues.Create(ctx, b); err != nil {
		t.Fatal(err)
	}

	got, err := env.issues.FindByFingerprint(ctx, p.ID, "fp-a")
	if err != nil || got.ID != a.ID || got.TaskID != nil {
		t.Fatalf("FindByFingerprint() = %+v, %v", got, err)
	}
	if _, err = env.issues.FindByFingerprint(ctx, p.ID, "nope"); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Errorf("FindByFingerprint() missing error = %v", err)
	}

	// optimistic locking.
	got.TimesSeen = 5
	got.Status = enum.InsightsIssueStatusRegressed
	stale := *got
	if err = env.issues.Update(ctx, got); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if err = env.issues.Update(ctx, &stale); !errors.Is(err, gitness_store.ErrVersionConflict) {
		t.Errorf("stale Update() error = %v, want ErrVersionConflict", err)
	}
	reloaded, _ := env.issues.Find(ctx, a.ID)
	if reloaded.TimesSeen != 5 || reloaded.Status != enum.InsightsIssueStatusRegressed {
		t.Errorf("reloaded = %+v", reloaded)
	}

	// list: most recently seen first, filter by status and query.
	list, err := env.issues.List(ctx, 1, &types.InsightsIssueFilter{})
	if err != nil || len(list) != 2 || list[0].Fingerprint != "fp-b" {
		t.Errorf("List() = %v, %v", list, err)
	}
	list, _ = env.issues.List(ctx, 1, &types.InsightsIssueFilter{Statuses: []enum.InsightsIssueStatus{enum.InsightsIssueStatusResolved}})
	if len(list) != 1 || list[0].ResolvedInRelease != "1.1.0" {
		t.Errorf("List(resolved) = %v", list)
	}
	cnt, err := env.issues.Count(ctx, 1, &types.InsightsIssueFilter{ListQueryFilter: types.ListQueryFilter{Query: "FP-A"}})
	if err != nil || cnt != 1 {
		t.Errorf("Count(query) = %d, %v", cnt, err)
	}
	if cnt, _ = env.issues.Count(ctx, 2, nil); cnt != 0 {
		t.Errorf("Count(other repo) = %d", cnt)
	}
}

func TestInsightsIssueStore_TaskLinkSetNullOnTaskDelete(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-api")
	if err := env.projects.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	task := newTask(1, "fix issue")
	if err := env.tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}

	issue := newInsightsIssue(p, "fp", 1000)
	taskID := task.ID
	issue.TaskID = &taskID
	if err := env.issues.Create(ctx, issue); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, _ := env.issues.Find(ctx, issue.ID)
	if got.TaskID == nil || *got.TaskID != task.ID {
		t.Fatalf("TaskID = %v", got.TaskID)
	}

	if _, err := env.db.ExecContext(ctx, `DELETE FROM tasks WHERE task_id = $1`, task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = env.issues.Find(ctx, issue.ID)
	if got.TaskID != nil {
		t.Errorf("TaskID = %v, want nil after task delete", *got.TaskID)
	}
}

func TestInsightsEventStore(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-api")
	if err := env.projects.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	issue := newInsightsIssue(p, "fp", 1000)
	if err := env.issues.Create(ctx, issue); err != nil {
		t.Fatal(err)
	}

	mk := func(eventID string, ts int64) *types.InsightsEvent {
		return &types.InsightsEvent{
			IssueID: issue.ID, ProjectID: p.ID, EventID: eventID, Timestamp: ts, Release: "1.0.0",
			Environment: "prod", Level: "error", Message: "boom", Payload: `{"a":1}`, Created: ts,
		}
	}
	for i, id := range []string{"e1", "e2", "e3"} {
		if err := env.events.Create(ctx, mk(id, int64(100*(i+1)))); err != nil {
			t.Fatalf("Create(%s) error = %v", id, err)
		}
	}
	if err := env.events.Create(ctx, mk("e1", 999)); !errors.Is(err, gitness_store.ErrDuplicate) {
		t.Errorf("duplicate event id error = %v, want ErrDuplicate", err)
	}

	if ok, err := env.events.Exists(ctx, p.ID, "e2"); err != nil || !ok {
		t.Errorf("Exists(e2) = %v, %v", ok, err)
	}
	if ok, _ := env.events.Exists(ctx, p.ID, "zz"); ok {
		t.Error("Exists(zz) = true")
	}
	if ok, _ := env.events.Exists(ctx, p.ID+1, "e2"); ok {
		t.Error("Exists is not scoped to the project")
	}

	list, err := env.events.ListByIssue(ctx, issue.ID, 2)
	if err != nil || len(list) != 2 || list[0].EventID != "e3" || list[1].EventID != "e2" || list[0].Payload != `{"a":1}` {
		t.Errorf("ListByIssue() = %+v, %v", list, err)
	}
}

func TestInsightsProjectDelete_Cascades(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-api")
	if err := env.projects.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	issue := newInsightsIssue(p, "fp", 1000)
	if err := env.issues.Create(ctx, issue); err != nil {
		t.Fatal(err)
	}
	if err := env.events.Create(ctx, &types.InsightsEvent{
		IssueID: issue.ID, ProjectID: p.ID, EventID: "e1", Timestamp: 1, Payload: "{}", Created: 1,
	}); err != nil {
		t.Fatal(err)
	}

	if err := env.projects.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := env.issues.Find(ctx, issue.ID); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Errorf("issue still there: %v", err)
	}
	if ok, _ := env.events.Exists(ctx, p.ID, "e1"); ok {
		t.Error("event still there")
	}
}

func TestInsightsIssueStore_AdminFilters(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p1 := newInsightsProject(1, "api", "key-1")
	p2 := newInsightsProject(1, "web", "key-2")
	for _, p := range []*types.InsightsProject{p1, p2} {
		if err := env.projects.Create(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	a := newInsightsIssue(p1, "fp-a", 100)
	a.TimesSeen = 7
	b := newInsightsIssue(p1, "fp-b", 300)
	b.Level, b.LastRelease = "warning", "1.1.0"
	c := newInsightsIssue(p2, "fp-c", 200)
	c.TimesSeen = 3
	for _, i := range []*types.InsightsIssue{a, b, c} {
		if err := env.issues.Create(ctx, i); err != nil {
			t.Fatal(err)
		}
	}

	list := func(f types.InsightsIssueFilter) []int64 {
		t.Helper()
		f.Pagination = types.Pagination{Page: 1, Size: 10}
		got, err := env.issues.List(ctx, 1, &f)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]int64, len(got))
		for i, x := range got {
			out[i] = x.ID
		}
		return out
	}
	eq := func(got []int64, want ...*types.InsightsIssue) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("got %v, want %d issues", got, len(want))
		}
		for i := range want {
			if got[i] != want[i].ID {
				t.Fatalf("got %v, want position %d = %d", got, i, want[i].ID)
			}
		}
	}

	eq(list(types.InsightsIssueFilter{}), b, c, a)
	eq(list(types.InsightsIssueFilter{Order: enum.OrderAsc}), a, c, b)
	eq(list(types.InsightsIssueFilter{Sort: enum.InsightsIssueSortTimesSeen}), a, c, b)
	eq(list(types.InsightsIssueFilter{ProjectID: p2.ID}), c)
	eq(list(types.InsightsIssueFilter{Level: "warning"}), b)
	eq(list(types.InsightsIssueFilter{Release: "1.1.0"}), b)

	n, err := env.issues.Count(ctx, 1, &types.InsightsIssueFilter{ProjectID: p1.ID})
	if err != nil || n != 2 {
		t.Errorf("Count() = %d, %v; want 2", n, err)
	}
}

func TestInsightsEventStore_ListAndCount(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-1")
	if err := env.projects.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	issue := newInsightsIssue(p, "fp", 100)
	if err := env.issues.Create(ctx, issue); err != nil {
		t.Fatal(err)
	}
	for i := int64(1); i <= 5; i++ {
		ev := &types.InsightsEvent{
			IssueID: issue.ID, ProjectID: p.ID, EventID: string(rune('a' + i)), Timestamp: i * 10,
			Payload: `{"x":1}`, Created: i,
		}
		if err := env.events.Create(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}

	got, err := env.events.List(ctx, issue.ID, types.Pagination{Page: 2, Size: 2})
	if err != nil || len(got) != 2 || got[0].Timestamp != 30 || got[1].Timestamp != 20 || got[0].Payload == "" {
		t.Fatalf("List() page 2 = %v, %v", got, err)
	}
	n, err := env.events.Count(ctx, issue.ID)
	if err != nil || n != 5 {
		t.Errorf("Count() = %d, %v; want 5", n, err)
	}
}

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
	"errors"
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

type taskTestEnv struct {
	db    *sqlx.DB
	tasks *database.TaskStore
}

func newTask(repoID int64, title string) *types.Task {
	return &types.Task{
		RepoID:    repoID,
		Title:     title,
		Origin:    enum.TaskOriginManual,
		Priority:  enum.TaskPriorityP2,
		State:     enum.TaskStateOpen,
		CreatedBy: userID,
		Created:   1700000000000,
		Updated:   1700000000000,
	}
}

func setupTaskEnv(t *testing.T) (context.Context, *taskTestEnv, func()) {
	t.Helper()

	db, teardown := setupDB(t)
	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)
	createRepo(ctx, t, repoStore, 2, 1, 0)

	return ctx, &taskTestEnv{db: db, tasks: database.NewTaskStore(db)}, teardown
}

func TestTaskStore_CreateAllocatesNumberPerRepo(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	for i, want := range []int64{1, 2, 3} {
		tk := newTask(1, "task")
		if err := env.tasks.Create(ctx, tk); err != nil {
			t.Fatalf("Create() #%d error = %v", i, err)
		}
		if tk.Number != want {
			t.Errorf("Number = %d, want %d", tk.Number, want)
		}
	}

	other := newTask(2, "other repo")
	if err := env.tasks.Create(ctx, other); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if other.Number != 1 {
		t.Errorf("Number in other repo = %d, want 1", other.Number)
	}

	got, err := env.tasks.FindByNumber(ctx, 1, 2)
	if err != nil {
		t.Fatalf("FindByNumber() error = %v", err)
	}
	if got.RepoID != 1 || got.Number != 2 {
		t.Errorf("FindByNumber() = repo %d number %d", got.RepoID, got.Number)
	}
}

func TestTaskStore_UpdateOptimisticLock(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	tk := newTask(1, "first")
	if err := env.tasks.Create(ctx, tk); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stale, err := env.tasks.Find(ctx, tk.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}

	assignee := userID
	tk.State = enum.TaskStateInProgress
	tk.AssigneePrincipalID = &assignee
	tk.AssigneeIsAgent = true
	if err = env.tasks.Update(ctx, tk); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if tk.Version != 1 {
		t.Errorf("Version = %d, want 1", tk.Version)
	}

	got, err := env.tasks.Find(ctx, tk.ID)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if got.State != enum.TaskStateInProgress || got.AssigneePrincipalID == nil ||
		*got.AssigneePrincipalID != assignee || !got.AssigneeIsAgent || got.Version != 1 {
		t.Errorf("unexpected task after update: %+v", got)
	}

	stale.Title = "stale"
	if err = env.tasks.Update(ctx, stale); !errors.Is(err, gitness_store.ErrVersionConflict) {
		t.Errorf("Update() stale error = %v, want ErrVersionConflict", err)
	}
}

func TestTaskStore_ListAndCountFilters(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	mk := func(title string, mod func(*types.Task)) *types.Task {
		tk := newTask(1, title)
		mod(tk)
		if err := env.tasks.Create(ctx, tk); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		return tk
	}

	assignee := userID
	mk("alpha", func(tk *types.Task) {})
	mk("beta crash", func(tk *types.Task) {
		tk.Origin = enum.TaskOriginInsights
		tk.Priority = enum.TaskPriorityP0
		tk.State = enum.TaskStateTriage
	})
	labelled := mk("gamma", func(tk *types.Task) {
		tk.Origin = enum.TaskOriginShield
		tk.AssigneePrincipalID = &assignee
		tk.AssigneeIsAgent = true
	})

	labelStore := database.NewLabelStore(env.db)
	repoID := int64(1)
	lbl := &types.Label{
		RepoID: &repoID, Key: "bug", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed,
		Created: 1, Updated: 1, CreatedBy: userID, UpdatedBy: userID,
	}
	if err := labelStore.Define(ctx, lbl); err != nil {
		t.Fatalf("Define() error = %v", err)
	}
	if err := database.NewTaskLabelStore(env.db).Assign(ctx, &types.TaskLabel{
		TaskID: labelled.ID, LabelID: lbl.ID, Created: 1, Updated: 1, CreatedBy: userID, UpdatedBy: userID,
	}); err != nil {
		t.Fatalf("Assign() error = %v", err)
	}

	isAgent := true
	tests := []struct {
		name   string
		filter types.TaskFilter
		want   int
	}{
		{"all", types.TaskFilter{}, 3},
		{"state", types.TaskFilter{States: []enum.TaskState{enum.TaskStateTriage}}, 1},
		{"priority", types.TaskFilter{Priorities: []enum.TaskPriority{enum.TaskPriorityP0}}, 1},
		{"origin", types.TaskFilter{Origins: []enum.TaskOrigin{enum.TaskOriginShield, enum.TaskOriginInsights}}, 2},
		{"assignee", types.TaskFilter{AssigneeIDs: []int64{assignee}}, 1},
		{"is_agent", types.TaskFilter{IsAgent: &isAgent}, 1},
		{"label", types.TaskFilter{LabelIDs: []int64{lbl.ID}}, 1},
		{"query", types.TaskFilter{Query: "CRASH"}, 1},
		{"page size", types.TaskFilter{Page: 1, Size: 2}, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list, err := env.tasks.List(ctx, 1, &tc.filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if len(list) != tc.want {
				t.Errorf("List() len = %d, want %d", len(list), tc.want)
			}
			if tc.name == "page size" {
				return
			}
			count, err := env.tasks.Count(ctx, 1, &tc.filter)
			if err != nil {
				t.Fatalf("Count() error = %v", err)
			}
			if int(count) != tc.want {
				t.Errorf("Count() = %d, want %d", count, tc.want)
			}
		})
	}
}

func TestTaskActivityStore_CreateList(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	tk := newTask(1, "with history")
	if err := env.tasks.Create(ctx, tk); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	store := database.NewTaskActivityStore(env.db)
	principal := userID
	for i, kind := range []enum.TaskActivityKind{enum.TaskActivityKindCreated, enum.TaskActivityKindStateChange} {
		a := &types.TaskActivity{
			TaskID:      tk.ID,
			Kind:        kind,
			Payload:     json.RawMessage(`{"from":"abierta","to":"triage"}`),
			PrincipalID: &principal,
			Created:     int64(100 + i),
		}
		if err := store.Create(ctx, a); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if a.ID == 0 {
			t.Error("activity ID not set")
		}
	}

	// activity without payload and principal
	if err := store.Create(ctx, &types.TaskActivity{
		TaskID: tk.ID, Kind: enum.TaskActivityKindComment, Created: 200,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	list, err := store.List(ctx, tk.ID, types.Pagination{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List() len = %d, want 3", len(list))
	}
	if list[0].Kind != enum.TaskActivityKindCreated || list[2].Kind != enum.TaskActivityKindComment {
		t.Errorf("unexpected order: %v, %v, %v", list[0].Kind, list[1].Kind, list[2].Kind)
	}
	if string(list[0].Payload) != `{"from":"abierta","to":"triage"}` || list[0].PrincipalID == nil {
		t.Errorf("unexpected first activity: %+v", list[0])
	}
	if list[2].PrincipalID != nil {
		t.Errorf("expected nil principal, got %v", *list[2].PrincipalID)
	}
}

func TestTaskLinkStore_CreateFindDelete(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	t1 := newTask(1, "one")
	t2 := newTask(1, "two")
	for _, tk := range []*types.Task{t1, t2} {
		if err := env.tasks.Create(ctx, tk); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	store := database.NewTaskLinkStore(env.db)
	link := &types.TaskLink{TaskID: t1.ID, Kind: enum.TaskLinkKindPullReq, RefID: "42", Created: 1}
	// creating twice must be idempotent.
	for range 2 {
		if err := store.Create(ctx, link); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	if err := store.Create(ctx, &types.TaskLink{
		TaskID: t2.ID, Kind: enum.TaskLinkKindPullReq, RefID: "42", Created: 1,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.Create(ctx, &types.TaskLink{
		TaskID: t1.ID, Kind: enum.TaskLinkKindShieldFinding, RefID: "f-1", Created: 1,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	links, err := store.ListByTask(ctx, t1.ID)
	if err != nil {
		t.Fatalf("ListByTask() error = %v", err)
	}
	if len(links) != 2 {
		t.Errorf("ListByTask() len = %d, want 2", len(links))
	}

	found, err := store.FindTasksByRef(ctx, enum.TaskLinkKindPullReq, "42")
	if err != nil {
		t.Fatalf("FindTasksByRef() error = %v", err)
	}
	if len(found) != 2 {
		t.Errorf("FindTasksByRef() len = %d, want 2", len(found))
	}

	if err = store.Delete(ctx, t1.ID, enum.TaskLinkKindPullReq, "42"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	found, err = store.FindTasksByRef(ctx, enum.TaskLinkKindPullReq, "42")
	if err != nil {
		t.Fatalf("FindTasksByRef() error = %v", err)
	}
	if len(found) != 1 || found[0].ID != t2.ID {
		t.Errorf("FindTasksByRef() after delete = %+v", found)
	}
}

func TestTaskLabelStore_AssignListUnassign(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	tk := newTask(1, "labelled")
	if err := env.tasks.Create(ctx, tk); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	repoID := int64(1)
	lbl := &types.Label{
		RepoID: &repoID, Key: "sec", Type: enum.LabelTypeStatic, Color: enum.LabelColorBlue,
		Created: 1, Updated: 1, CreatedBy: userID, UpdatedBy: userID,
	}
	if err := database.NewLabelStore(env.db).Define(ctx, lbl); err != nil {
		t.Fatalf("Define() error = %v", err)
	}

	store := database.NewTaskLabelStore(env.db)
	in := &types.TaskLabel{
		TaskID: tk.ID, LabelID: lbl.ID, Created: 5, Updated: 5, CreatedBy: userID, UpdatedBy: userID,
	}
	for range 2 { // upsert
		if err := store.Assign(ctx, in); err != nil {
			t.Fatalf("Assign() error = %v", err)
		}
	}

	got, err := store.FindByLabelID(ctx, tk.ID, lbl.ID)
	if err != nil {
		t.Fatalf("FindByLabelID() error = %v", err)
	}
	if got.Created != 5 {
		t.Errorf("Created = %d, want 5", got.Created)
	}

	assigned, err := store.ListAssigned(ctx, tk.ID)
	if err != nil {
		t.Fatalf("ListAssigned() error = %v", err)
	}
	if len(assigned) != 1 {
		t.Errorf("ListAssigned() len = %d, want 1", len(assigned))
	}

	infos, err := store.ListAssignedByTaskIDs(ctx, []int64{tk.ID})
	if err != nil {
		t.Fatalf("ListAssignedByTaskIDs() error = %v", err)
	}
	if len(infos[tk.ID]) != 1 || infos[tk.ID][0].LabelKey != "sec" {
		t.Errorf("unexpected infos: %+v", infos)
	}

	removed, err := store.Unassign(ctx, tk.ID, lbl.ID)
	if err != nil || !removed {
		t.Errorf("Unassign() = %v, %v, want true, nil", removed, err)
	}
	removed, err = store.Unassign(ctx, tk.ID, lbl.ID)
	if err != nil || removed {
		t.Errorf("second Unassign() = %v, %v, want false, nil", removed, err)
	}
}

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
	"errors"
	"testing"

	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestTaskStore_UniqueActiveOrigin(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	mk := func(repoID int64, origin enum.TaskOrigin, ref string) *types.Task {
		tk := newTask(repoID, "from origin")
		tk.Origin = origin
		tk.OriginRef = ref
		return tk
	}

	first := mk(1, enum.TaskOriginInsights, "issue-1")
	if err := env.tasks.Create(ctx, first); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err := env.tasks.Create(ctx, mk(1, enum.TaskOriginInsights, "issue-1"))
	if !errors.Is(err, gitness_store.ErrDuplicate) {
		t.Fatalf("Create() duplicate active error = %v, want ErrDuplicate", err)
	}

	// other repo, other ref, other origin and manual tasks are not affected
	for _, tk := range []*types.Task{
		mk(2, enum.TaskOriginInsights, "issue-1"),
		mk(1, enum.TaskOriginInsights, "issue-2"),
		mk(1, enum.TaskOriginShield, "issue-1"),
		mk(1, enum.TaskOriginManual, "x"),
		mk(1, enum.TaskOriginManual, "x"),
	} {
		if err = env.tasks.Create(ctx, tk); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// a dismissed task frees the slot
	first.State = enum.TaskStateDismissed
	if err = env.tasks.Update(ctx, first); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err = env.tasks.Create(ctx, mk(1, enum.TaskOriginInsights, "issue-1")); err != nil {
		t.Fatalf("Create() after dismissal error = %v", err)
	}
}

func TestTaskStore_ListByRepos(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	for _, repoID := range []int64{1, 1, 2} {
		if err := env.tasks.Create(ctx, newTask(repoID, "task")); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	filter := &types.TaskFilter{Page: 1, Size: 10}
	list, err := env.tasks.ListByRepos(ctx, []int64{1, 2}, filter)
	if err != nil {
		t.Fatalf("ListByRepos() error = %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListByRepos() len = %d, want 3", len(list))
	}
	count, err := env.tasks.CountByRepos(ctx, []int64{2}, filter)
	if err != nil || count != 1 {
		t.Fatalf("CountByRepos() = %d, %v, want 1", count, err)
	}
	if list, err = env.tasks.ListByRepos(ctx, nil, filter); err != nil || len(list) != 0 {
		t.Fatalf("ListByRepos(nil) = %d, %v, want empty", len(list), err)
	}
}

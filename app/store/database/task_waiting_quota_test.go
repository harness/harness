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
	"slices"
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestTaskStore_ListWaitingQuotaOrder(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	mk := func(repoID int64, title string, prio enum.TaskPriority, desc string, created int64, state enum.TaskState) {
		tk := newTask(repoID, title)
		tk.Priority, tk.Description, tk.Created, tk.State = prio, desc, created, state
		if err := env.tasks.Create(ctx, tk); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	w := enum.TaskStateWaitingQuota
	mk(1, "rest-old", enum.TaskPriorityP3, "", 1, w)
	mk(2, "p1", enum.TaskPriorityP1, "", 2, w)
	mk(1, "regression", enum.TaskPriorityP2, "- Regression: the issue had been resolved", 3, w)
	mk(2, "p0", enum.TaskPriorityP0, "", 4, w)
	mk(1, "not-waiting", enum.TaskPriorityP0, "", 0, enum.TaskStateOpen)

	got, err := env.tasks.ListWaitingQuota(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, tk := range got {
		titles = append(titles, tk.Title)
	}
	want := []string{"p0", "regression", "p1", "rest-old"}
	if !slices.Equal(titles, want) {
		t.Fatalf("got %v, want %v", titles, want)
	}
	if lim, _ := env.tasks.ListWaitingQuota(ctx, 2); len(lim) != 2 {
		t.Fatalf("limit not applied: %d", len(lim))
	}
}

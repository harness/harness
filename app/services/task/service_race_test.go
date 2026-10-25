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

package task

import (
	"context"
	"testing"

	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// racingTasks simulates a concurrent create: the first lookup misses, and the first insert
// fails with a unique violation because the competitor inserted the task in between.
type racingTasks struct {
	*fakeTasks
	lookups int
	creates int
}

func (r *racingTasks) FindActiveByOrigin(
	ctx context.Context, repoID int64, origin enum.TaskOrigin, ref string,
) (*types.Task, error) {
	r.lookups++
	if r.lookups == 1 {
		return nil, gitness_store.ErrResourceNotFound
	}
	return r.fakeTasks.FindActiveByOrigin(ctx, repoID, origin, ref)
}

func (r *racingTasks) Create(ctx context.Context, t *types.Task) error {
	r.creates++
	if r.creates == 1 {
		winner := *t
		_ = r.fakeTasks.Create(ctx, &winner)
		return gitness_store.ErrDuplicate
	}
	return r.fakeTasks.Create(ctx, t)
}

func TestCreateReturnsExistingOnUniqueViolation(t *testing.T) {
	e := newEnv()
	racing := &racingTasks{fakeTasks: e.tasks}
	e.svc.tasks = racing

	task, created, err := e.svc.Create(context.Background(), 7, CreateInput{
		RepoID: 1, Title: "NPE", Origin: enum.TaskOriginInsights, OriginRef: "issue-1",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created {
		t.Error("created = true, want false (the competitor won)")
	}
	if task == nil || task.ID != 1 {
		t.Fatalf("task = %+v, want the competitor's task", task)
	}
	if racing.creates != 1 || e.tasks.seq != 1 {
		t.Errorf("creates = %d, tasks = %d, want exactly one task", racing.creates, e.tasks.seq)
	}
}

func TestCreateManualDuplicateErrorIsNotSwallowed(t *testing.T) {
	e := newEnv()
	e.svc.tasks = &racingTasks{fakeTasks: e.tasks}

	_, _, err := e.svc.Create(context.Background(), 7, CreateInput{RepoID: 1, Title: "manual"})
	if err == nil {
		t.Fatal("Create() error = nil, want the duplicate error for a manual task")
	}
}

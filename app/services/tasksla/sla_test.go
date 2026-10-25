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

package tasksla

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var now = time.Unix(1_800_000_000, 0)

type fakeTasks struct {
	store.TaskStore
	tasks []*types.Task
}

func (f *fakeTasks) ListDueBefore(_ context.Context, before, after int64, limit int) ([]*types.Task, error) {
	var out []*types.Task
	for _, t := range f.tasks {
		if t.Due != nil && *t.Due <= before && t.ID > after && len(out) < limit {
			out = append(out, t)
		}
	}
	return out, nil
}

type fakeActivities struct {
	store.TaskActivityStore
	rows []*types.TaskActivity
}

func (f *fakeActivities) Create(_ context.Context, a *types.TaskActivity) error {
	f.rows = append(f.rows, a)
	return nil
}

func (f *fakeActivities) ListByKind(_ context.Context, id int64, k enum.TaskActivityKind) ([]*types.TaskActivity, error) {
	var out []*types.TaskActivity
	for _, r := range f.rows {
		if r.TaskID == id && r.Kind == k {
			out = append(out, r)
		}
	}
	return out, nil
}

type notified struct {
	id      int64
	overdue bool
}

type fakeNotifier struct{ calls []notified }

func (f *fakeNotifier) NotifyTaskSLA(_ context.Context, t *types.Task, overdue bool) error {
	f.calls = append(f.calls, notified{t.ID, overdue})
	return nil
}

func due(id int64, d time.Duration) *types.Task {
	ms := now.Add(d).UnixMilli()
	return &types.Task{ID: id, Due: &ms}
}

func TestRunSelectsAndNotifiesOnce(t *testing.T) {
	ctx := context.Background()
	tasks := &fakeTasks{tasks: []*types.Task{
		due(1, 3*time.Hour),   // warning
		due(2, -2*time.Hour),  // overdue
		due(3, 30*time.Hour),  // outside the window
		{ID: 4},               // no due date
		due(5, 23*time.Hour),  // warning
		due(6, -48*time.Hour), // overdue
	}}
	acts := &fakeActivities{}
	n := &fakeNotifier{}
	r := NewRunner(24*time.Hour, tasks, acts, n)
	r.now = func() time.Time { return now }

	res, err := r.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []notified{{1, false}, {2, true}, {5, false}, {6, true}}
	if len(n.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", n.calls, want)
	}
	for i := range want {
		if n.calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", n.calls, want)
		}
	}
	if res.Warned != 2 || res.Overdue != 2 || res.Failed != 0 {
		t.Fatalf("result = %s", res)
	}

	// second run does not notify again
	n.calls = nil
	if _, err = r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != 0 {
		t.Fatalf("repeated notifications: %v", n.calls)
	}

	// time passes: the warned task becomes overdue and is notified once more
	r.now = func() time.Time { return now.Add(4 * time.Hour) }
	if _, err = r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(n.calls) != 1 || n.calls[0] != (notified{1, true}) {
		t.Fatalf("calls after due = %v", n.calls)
	}
}

func TestDecide(t *testing.T) {
	nowMs := now.UnixMilli()
	if got := decide(&types.Task{}, nowMs, nil); got != "" {
		t.Fatalf("no due: %q", got)
	}
	if got := decide(due(1, time.Hour), nowMs, map[string]bool{markerOverdue: true}); got != "" {
		t.Fatalf("overdue marker suppresses warning: %q", got)
	}
	if got := decide(due(1, -time.Hour), nowMs, map[string]bool{markerWarning: true}); got != markerOverdue {
		t.Fatalf("warned then overdue: %q", got)
	}
}

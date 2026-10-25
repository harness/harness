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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type fakeTasks struct {
	byID         map[int64]*types.Task
	seq          int64
	conflictsOnU int // number of Update calls to fail with a version conflict
	updates      int
}

func (f *fakeTasks) Create(_ context.Context, t *types.Task) error {
	f.seq++
	t.ID, t.Number = f.seq, f.seq
	c := *t
	f.byID[t.ID] = &c
	return nil
}

func (f *fakeTasks) Find(_ context.Context, id int64) (*types.Task, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	c := *t
	return &c, nil
}

func (f *fakeTasks) FindByNumber(context.Context, int64, int64) (*types.Task, error) {
	return nil, gitness_store.ErrResourceNotFound
}

func (f *fakeTasks) FindActiveByOrigin(
	_ context.Context, repoID int64, origin enum.TaskOrigin, ref string,
) (*types.Task, error) {
	for id := int64(1); id <= f.seq; id++ {
		t := f.byID[id]
		if t.RepoID == repoID && t.Origin == origin && t.OriginRef == ref && IsActive(t.State) {
			c := *t
			return &c, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f *fakeTasks) Update(_ context.Context, t *types.Task) error {
	f.updates++
	if f.conflictsOnU > 0 {
		f.conflictsOnU--
		return gitness_store.ErrVersionConflict
	}
	cur := f.byID[t.ID]
	if cur.Version != t.Version {
		return gitness_store.ErrVersionConflict
	}
	t.Version++
	c := *t
	f.byID[t.ID] = &c
	return nil
}

func (f *fakeTasks) List(context.Context, int64, *types.TaskFilter) ([]*types.Task, error) {
	return nil, nil
}
func (f *fakeTasks) Count(context.Context, int64, *types.TaskFilter) (int64, error) { return 0, nil }

func (f *fakeTasks) ListByRepos(context.Context, []int64, *types.TaskFilter) ([]*types.Task, error) {
	return nil, nil
}

func (f *fakeTasks) CountByRepos(context.Context, []int64, *types.TaskFilter) (int64, error) {
	return 0, nil
}

func (f *fakeTasks) ListWaitingQuota(context.Context, int) ([]*types.Task, error) {
	return nil, nil
}

func (f *fakeTasks) ListDueBefore(context.Context, int64, int64, int) ([]*types.Task, error) {
	return nil, nil
}

type fakeActivities struct{ list []*types.TaskActivity }

func (f *fakeActivities) ListByKind(
	_ context.Context, taskID int64, kind enum.TaskActivityKind,
) ([]*types.TaskActivity, error) {
	var out []*types.TaskActivity
	for _, a := range f.list {
		if a.TaskID == taskID && a.Kind == kind {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeActivities) Create(_ context.Context, a *types.TaskActivity) error {
	a.ID = int64(len(f.list) + 1)
	f.list = append(f.list, a)
	return nil
}

func (f *fakeActivities) List(_ context.Context, taskID int64, p types.Pagination) ([]*types.TaskActivity, error) {
	var all []*types.TaskActivity
	for _, a := range f.list {
		if a.TaskID == taskID {
			all = append(all, a)
		}
	}
	start := (p.Page - 1) * p.Size
	if start >= len(all) {
		return nil, nil
	}
	end := min(start+p.Size, len(all))
	return all[start:end], nil
}

func (f *fakeActivities) kinds(taskID int64) []enum.TaskActivityKind {
	var out []enum.TaskActivityKind
	for _, a := range f.list {
		if a.TaskID == taskID {
			out = append(out, a.Kind)
		}
	}
	return out
}

type fakeLinks struct{ list []*types.TaskLink }

func (f *fakeLinks) Create(_ context.Context, l *types.TaskLink) error {
	f.list = append(f.list, l)
	return nil
}

func (f *fakeLinks) Delete(_ context.Context, taskID int64, kind enum.TaskLinkKind, ref string) error {
	for i, l := range f.list {
		if l.TaskID == taskID && l.Kind == kind && l.RefID == ref {
			f.list = append(f.list[:i], f.list[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeLinks) ListByTask(_ context.Context, taskID int64) ([]*types.TaskLink, error) {
	var out []*types.TaskLink
	for _, l := range f.list {
		if l.TaskID == taskID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeLinks) FindTasksByRef(context.Context, enum.TaskLinkKind, string) ([]*types.Task, error) {
	return nil, nil
}

const fixedNow = int64(1_700_000_000_000)

type env struct {
	svc   *Service
	tasks *fakeTasks
	acts  *fakeActivities
	links *fakeLinks
}

func newEnv() *env {
	e := &env{
		tasks: &fakeTasks{byID: map[int64]*types.Task{}},
		acts:  &fakeActivities{},
		links: &fakeLinks{},
	}
	e.svc = NewService(DefaultConfig(), fakeTx{}, e.tasks, e.acts, e.links, nil)
	e.svc.now = func() int64 { return fixedNow }
	return e
}

func (e *env) create(t *testing.T, in CreateInput) *types.Task {
	t.Helper()
	if in.RepoID == 0 {
		in.RepoID = 1
	}
	if in.Title == "" {
		in.Title = "title"
	}
	task, _, err := e.svc.Create(context.Background(), 7, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return task
}

func (e *env) move(t *testing.T, id int64, to enum.TaskState, reason string) *types.Task {
	t.Helper()
	task, err := e.svc.Transition(context.Background(), 7, id, to, reason)
	if err != nil {
		t.Fatalf("transition to %s: %v", to, err)
	}
	return task
}

func (e *env) path(t *testing.T, id int64, states ...enum.TaskState) {
	t.Helper()
	for _, s := range states {
		e.move(t, id, s, "")
	}
}

func TestTransitionTable(t *testing.T) {
	all, _ := enum.GetAllTaskStates()

	mustAllow := [][2]enum.TaskState{
		{enum.TaskStateOpen, enum.TaskStateTriage},
		{enum.TaskStateTriage, enum.TaskStateInProgress},
		{enum.TaskStateInProgress, enum.TaskStateInReview},
		{enum.TaskStateInReview, enum.TaskStateResolved},
		{enum.TaskStateResolved, enum.TaskStateVerified},
		{enum.TaskStateResolved, enum.TaskStateReopened},
		{enum.TaskStateVerified, enum.TaskStateReopened},
		{enum.TaskStateInProgress, enum.TaskStateWaitingQuota},
		{enum.TaskStateInProgress, enum.TaskStateBlocked},
		{enum.TaskStateBlocked, enum.TaskStateTriage},
		{enum.TaskStateBlocked, enum.TaskStateInProgress},
		{enum.TaskStateWaitingQuota, enum.TaskStateDismissed},
	}
	for _, p := range mustAllow {
		if !CanTransition(p[0], p[1]) {
			t.Errorf("expected %s -> %s to be allowed", p[0], p[1])
		}
	}
	mustDeny := [][2]enum.TaskState{
		{enum.TaskStateOpen, enum.TaskStateResolved},
		{enum.TaskStateOpen, enum.TaskStateVerified},
		{enum.TaskStateOpen, enum.TaskStateReopened},
		{enum.TaskStateTriage, enum.TaskStateResolved},
		{enum.TaskStateInProgress, enum.TaskStateReopened},
		{enum.TaskStateInReview, enum.TaskStateReopened},
		{enum.TaskStateVerified, enum.TaskStateDismissed},
		{enum.TaskStateVerified, enum.TaskStateInProgress},
		{enum.TaskStateResolved, enum.TaskStateBlocked},
		{enum.TaskStateBlocked, enum.TaskStateWaitingQuota},
		{enum.TaskStateWaitingQuota, enum.TaskStateBlocked},
		{enum.TaskStateWaitingQuota, enum.TaskStateResolved},
	}
	for _, p := range mustDeny {
		if CanTransition(p[0], p[1]) {
			t.Errorf("expected %s -> %s to be denied", p[0], p[1])
		}
	}
	// dismissed is terminal; nothing transitions to itself; every state has a table entry.
	for _, s := range all {
		if CanTransition(s, s) {
			t.Errorf("%s -> %s must not be allowed", s, s)
		}
		if CanTransition(enum.TaskStateDismissed, s) {
			t.Errorf("dismissed -> %s must be denied", s)
		}
		if _, ok := transitions[s]; !ok {
			t.Errorf("state %s missing from the transition table", s)
		}
	}
}

func TestInvalidTransitionError(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})

	_, err := e.svc.Transition(context.Background(), 7, task.ID, enum.TaskStateResolved, "")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
	var terr *TransitionError
	if !errors.As(err, &terr) || terr.From != enum.TaskStateOpen || terr.To != enum.TaskStateResolved {
		t.Fatalf("unexpected transition error: %#v", err)
	}
	var uerr *usererror.Error
	if !errors.As(err, &uerr) || uerr.Status != 400 {
		t.Fatalf("expected a 400 usererror, got %#v", err)
	}

	if _, err = e.svc.Transition(context.Background(), 7, task.ID, enum.TaskStateOpen, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("same state: %v", err)
	}
	if _, err = e.svc.Transition(context.Background(), 7, task.ID, "nope", ""); err == nil {
		t.Fatal("expected error for unknown state")
	}
	if got := e.acts.kinds(task.ID); len(got) != 1 || got[0] != enum.TaskActivityKindCreated {
		t.Fatalf("unexpected activities %v", got)
	}
}

func TestMainFlowTimestampsAndHistory(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})
	e.path(t, task.ID, enum.TaskStateTriage, enum.TaskStateInProgress, enum.TaskStateInReview, enum.TaskStateResolved)
	got := e.tasks.byID[task.ID]
	if got.State != enum.TaskStateResolved || got.Resolved == nil || got.Verified != nil {
		t.Fatalf("unexpected task after resolve: %+v", got)
	}
	e.move(t, task.ID, enum.TaskStateVerified, "")
	if e.tasks.byID[task.ID].Verified == nil {
		t.Fatal("verified timestamp not set")
	}

	// created + 5 state changes
	if k := e.acts.kinds(task.ID); len(k) != 6 {
		t.Fatalf("activities %v", k)
	}
	last := e.acts.list[len(e.acts.list)-1]
	var p stateChangePayload
	if err := json.Unmarshal(last.Payload, &p); err != nil || p.From != enum.TaskStateResolved || p.To != enum.TaskStateVerified {
		t.Fatalf("payload %s err %v", last.Payload, err)
	}
	if last.PrincipalID == nil || *last.PrincipalID != 7 {
		t.Fatalf("principal not recorded: %+v", last.PrincipalID)
	}
}

func TestDismissRequiresReason(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})
	if _, err := e.svc.Transition(context.Background(), 7, task.ID, enum.TaskStateDismissed, "  "); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected error without reason, got %v", err)
	}
	got := e.move(t, task.ID, enum.TaskStateDismissed, "duplicate of upstream")
	if got.StateReason != "duplicate of upstream" {
		t.Fatalf("reason: %q", got.StateReason)
	}
	if _, err := e.svc.Transition(context.Background(), 7, task.ID, enum.TaskStateOpen, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected terminal state, got %v", err)
	}
}

func TestReopenRaisesPriority(t *testing.T) {
	cases := []struct {
		from, want enum.TaskPriority
	}{
		{enum.TaskPriorityP3, enum.TaskPriorityP2},
		{enum.TaskPriorityP2, enum.TaskPriorityP1},
		{enum.TaskPriorityP1, enum.TaskPriorityP0},
		{enum.TaskPriorityP0, enum.TaskPriorityP0},
	}
	for _, c := range cases {
		for _, via := range []enum.TaskState{enum.TaskStateResolved, enum.TaskStateVerified} {
			e := newEnv()
			task := e.create(t, CreateInput{Priority: c.from})
			e.path(t, task.ID, enum.TaskStateTriage, enum.TaskStateInProgress, enum.TaskStateInReview, enum.TaskStateResolved)
			if via == enum.TaskStateVerified {
				e.move(t, task.ID, enum.TaskStateVerified, "")
			}
			got := e.move(t, task.ID, enum.TaskStateReopened, "error is back")
			if got.Priority != c.want {
				t.Errorf("%s via %s: priority %s want %s", c.from, via, got.Priority, c.want)
			}
			if got.Resolved != nil || got.Verified != nil {
				t.Errorf("reopen must clear resolved/verified")
			}
			hasPrio := false
			for _, k := range e.acts.kinds(task.ID) {
				hasPrio = hasPrio || k == enum.TaskActivityKindPriorityChange
			}
			if hasPrio != (c.from != c.want) {
				t.Errorf("%s: priority_change activity present=%v", c.from, hasPrio)
			}
			// reopened tasks are active again and can continue
			e.move(t, task.ID, enum.TaskStateInProgress, "")
		}
	}
}

func TestReopenOnlyFromResolvedOrVerified(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})
	if _, err := e.svc.Transition(context.Background(), 7, task.ID, enum.TaskStateReopened, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("got %v", err)
	}
}

func TestSideStatesReturnToPrevious(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})
	e.path(t, task.ID, enum.TaskStateTriage, enum.TaskStateInProgress, enum.TaskStateWaitingQuota)

	for _, to := range []enum.TaskState{enum.TaskStateInReview, enum.TaskStateOpen} {
		if _, err := e.svc.Transition(context.Background(), 7, task.ID, to, ""); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("to %s: expected invalid transition, got %v", to, err)
		}
	}
	got, err := e.svc.Resume(context.Background(), 7, task.ID, "quota renewed")
	if err != nil || got.State != enum.TaskStateInProgress {
		t.Fatalf("resume: %v %+v", err, got)
	}

	e.move(t, task.ID, enum.TaskStateBlocked, "waiting for infra")
	e.move(t, task.ID, enum.TaskStateInProgress, "")
	e.move(t, task.ID, enum.TaskStateBlocked, "again")
	e.move(t, task.ID, enum.TaskStateTriage, "")

	if _, err = e.svc.Resume(context.Background(), 7, task.ID, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("resume on triage: %v", err)
	}

	e.move(t, task.ID, enum.TaskStateWaitingQuota, "")
	e.move(t, task.ID, enum.TaskStateDismissed, "not needed")
}

func TestCreateDefaultsAndSLA(t *testing.T) {
	day := (24 * time.Hour).Milliseconds()
	cases := []struct {
		prio enum.TaskPriority
		due  *int64
	}{
		{enum.TaskPriorityP0, ptr(fixedNow + day)},
		{enum.TaskPriorityP1, ptr(fixedNow + 3*day)},
		{enum.TaskPriorityP2, ptr(fixedNow + 14*day)},
		{enum.TaskPriorityP3, nil},
		{"", ptr(fixedNow + 14*day)}, // default p2
	}
	for _, c := range cases {
		e := newEnv()
		task := e.create(t, CreateInput{Priority: c.prio})
		if task.State != enum.TaskStateOpen {
			t.Errorf("state %s", task.State)
		}
		if (task.Due == nil) != (c.due == nil) || (c.due != nil && *task.Due != *c.due) {
			t.Errorf("priority %q: due %v want %v", c.prio, task.Due, c.due)
		}
		if k := e.acts.kinds(task.ID); len(k) != 1 || k[0] != enum.TaskActivityKindCreated {
			t.Errorf("activities %v", k)
		}
	}

	e := newEnv()
	task := e.create(t, CreateInput{Priority: enum.TaskPriorityP0, Due: ptr(int64(42))})
	if task.Due == nil || *task.Due != 42 {
		t.Fatalf("due %v", task.Due)
	}

	e = newEnv()
	e.svc.config = Config{SLA: SLAConfig{P3: time.Hour}}
	task = e.create(t, CreateInput{Priority: enum.TaskPriorityP3})
	if task.Due == nil || *task.Due != fixedNow+time.Hour.Milliseconds() {
		t.Fatalf("custom sla due %v", task.Due)
	}
}

func TestReopenRestartsSLA(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{Priority: enum.TaskPriorityP3})
	e.path(t, task.ID, enum.TaskStateTriage, enum.TaskStateInProgress, enum.TaskStateInReview, enum.TaskStateResolved)
	got := e.move(t, task.ID, enum.TaskStateReopened, "")
	want := fixedNow + 14*(24*time.Hour).Milliseconds() // raised to p2
	if got.Due == nil || *got.Due != want {
		t.Fatalf("due %v want %d", got.Due, want)
	}
}

func TestCreateValidation(t *testing.T) {
	e := newEnv()
	bad := []CreateInput{
		{RepoID: 1, Title: "  "},
		{RepoID: 1, Title: "x", Origin: "alien"},
		{RepoID: 1, Title: "x", Priority: "p9"},
		{RepoID: 1, Title: "x", Assignee: &AssigneeInput{PrincipalID: ptr(int64(1)), GroupID: ptr(int64(2))}},
		{RepoID: 1, Title: "x", Assignee: &AssigneeInput{GroupID: ptr(int64(2)), IsAgent: true}},
	}
	for i, in := range bad {
		if _, _, err := e.svc.Create(context.Background(), 7, in); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
	if e.tasks.seq != 0 {
		t.Fatal("nothing should have been created")
	}
}

func TestOriginDedup(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	in := CreateInput{RepoID: 1, Title: "NPE in checkout", Origin: enum.TaskOriginInsights, OriginRef: "issue-1"}

	first, created, err := e.svc.Create(ctx, 7, in)
	if err != nil || !created {
		t.Fatalf("first: %v %v", err, created)
	}
	second, created, err := e.svc.Create(ctx, 7, in)
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("second: %v created=%v id=%d", err, created, second.ID)
	}
	if e.tasks.seq != 1 || len(e.acts.list) != 1 {
		t.Fatalf("dedup must not write: tasks=%d acts=%d", e.tasks.seq, len(e.acts.list))
	}

	in2 := in
	in2.RepoID = 2
	if _, created, _ = e.svc.Create(ctx, 7, in2); !created {
		t.Fatal("different repo must create")
	}
	in3 := in
	in3.Origin = enum.TaskOriginShield
	if _, created, _ = e.svc.Create(ctx, 7, in3); !created {
		t.Fatal("different origin must create")
	}

	e.path(t, first.ID, enum.TaskStateTriage, enum.TaskStateInProgress, enum.TaskStateInReview,
		enum.TaskStateResolved)
	if again, created, _ := e.svc.Create(ctx, 7, in); created || again.ID != first.ID {
		t.Fatal("resolved task is still active")
	}
	e.move(t, first.ID, enum.TaskStateVerified, "")
	again, created, err := e.svc.Create(ctx, 7, in)
	if err != nil || !created || again.ID == first.ID {
		t.Fatalf("after verified a new task is expected: %v %v", err, created)
	}

	m := CreateInput{RepoID: 1, Title: "m", OriginRef: "same"}
	a := e.create(t, m)
	b := e.create(t, m)
	if a.ID == b.ID {
		t.Fatal("manual tasks must not dedup")
	}
}

func TestUpdateWritesActivities(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	task := e.create(t, CreateInput{Priority: enum.TaskPriorityP3})
	if task.Due != nil {
		t.Fatal("p3 has no due date")
	}

	title, desc, prio := "new title", "desc", enum.TaskPriorityP1
	got, err := e.svc.Update(ctx, 9, task.ID, UpdateInput{
		Title: &title, Description: &desc, Priority: &prio,
		Assignee: &AssigneeInput{PrincipalID: ptr(int64(5))},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != title || got.Description != desc || got.Priority != prio {
		t.Fatalf("fields not updated: %+v", got)
	}
	if got.Due == nil || *got.Due != fixedNow+3*(24*time.Hour).Milliseconds() {
		t.Fatalf("SLA of the new priority expected, got %v", got.Due)
	}
	if got.AssigneePrincipalID == nil || *got.AssigneePrincipalID != 5 || got.AssigneeIsAgent {
		t.Fatalf("assignee %+v", got)
	}
	kinds := e.acts.kinds(task.ID)
	want := map[enum.TaskActivityKind]bool{
		enum.TaskActivityKindCreated: true, enum.TaskActivityKindUpdated: true,
		enum.TaskActivityKindPriorityChange: true, enum.TaskActivityKindAssignment: true,
	}
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v", kinds)
	}
	for _, k := range kinds {
		if !want[k] {
			t.Fatalf("unexpected kind %s", k)
		}
	}

	n := len(e.acts.list)
	if _, err = e.svc.Update(ctx, 9, task.ID, UpdateInput{Assignee: &AssigneeInput{IsAgent: true}}); err != nil {
		t.Fatal(err)
	}
	if len(e.acts.list) != n+1 || !e.tasks.byID[task.ID].AssigneeIsAgent || e.tasks.byID[task.ID].AssigneePrincipalID != nil {
		t.Fatal("agent assignment expected")
	}
	updates := e.tasks.updates
	if _, err = e.svc.Update(ctx, 9, task.ID, UpdateInput{Title: &title, Priority: &prio}); err != nil {
		t.Fatal(err)
	}
	if len(e.acts.list) != n+1 || e.tasks.updates != updates {
		t.Fatal("no-op update must not write")
	}

	if _, err = e.svc.Update(ctx, 9, task.ID, UpdateInput{Due: ptr(int64(99))}); err != nil {
		t.Fatal(err)
	}
	if d := e.tasks.byID[task.ID].Due; d == nil || *d != 99 {
		t.Fatalf("due %v", d)
	}
	if _, err = e.svc.Update(ctx, 9, task.ID, UpdateInput{ClearDue: true}); err != nil {
		t.Fatal(err)
	}
	if e.tasks.byID[task.ID].Due != nil {
		t.Fatal("due should be cleared")
	}
	if _, err = e.svc.Update(ctx, 9, task.ID, UpdateInput{Due: ptr(int64(1)), ClearDue: true}); err == nil {
		t.Fatal("due + clear must fail")
	}
	empty := " "
	if _, err = e.svc.Update(ctx, 9, task.ID, UpdateInput{Title: &empty}); err == nil {
		t.Fatal("empty title must fail")
	}
}

func TestOptimisticLockRetry(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})

	e.tasks.conflictsOnU = 2
	got := e.move(t, task.ID, enum.TaskStateTriage, "")
	if got.State != enum.TaskStateTriage {
		t.Fatalf("state %s", got.State)
	}
	// The fake transaction doesn't roll back, but activities are written only after a successful
	// task update, so the failed attempts leave nothing behind.
	n := 0
	for _, k := range e.acts.kinds(task.ID) {
		if k == enum.TaskActivityKindStateChange {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected one state change, got %d", n)
	}

	e.tasks.conflictsOnU = maxTxRetries
	if _, err := e.svc.Transition(context.Background(), 7, task.ID, enum.TaskStateInProgress, ""); !errors.Is(err, gitness_store.ErrVersionConflict) {
		t.Fatalf("expected conflict after retries, got %v", err)
	}
}

func TestLinks(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	task := e.create(t, CreateInput{})

	for _, k := range []enum.TaskLinkKind{
		enum.TaskLinkKindPullReq, enum.TaskLinkKindAITask, enum.TaskLinkKindInsightsIssue, enum.TaskLinkKindShieldFinding,
	} {
		if err := e.svc.Link(ctx, 7, task.ID, k, "ref-"+string(k)); err != nil {
			t.Fatalf("link %s: %v", k, err)
		}
	}
	if len(e.links.list) != 4 {
		t.Fatalf("links %d", len(e.links.list))
	}
	n := len(e.acts.list)
	if err := e.svc.Link(ctx, 7, task.ID, enum.TaskLinkKindPullReq, "ref-pullreq"); err != nil {
		t.Fatal(err)
	}
	if len(e.links.list) != 4 || len(e.acts.list) != n {
		t.Fatal("duplicate link must be a no-op")
	}
	if err := e.svc.Unlink(ctx, 7, task.ID, enum.TaskLinkKindPullReq, "ref-pullreq"); err != nil {
		t.Fatal(err)
	}
	if len(e.links.list) != 3 || len(e.acts.list) != n+1 {
		t.Fatal("unlink must remove link and record history")
	}
	if err := e.svc.Unlink(ctx, 7, task.ID, enum.TaskLinkKindPullReq, "ref-pullreq"); err != nil || len(e.acts.list) != n+1 {
		t.Fatalf("second unlink must be a no-op: %v", err)
	}
	var p linkPayload
	if err := json.Unmarshal(e.acts.list[n].Payload, &p); err != nil || p.Action != "unlinked" || p.Kind != enum.TaskLinkKindPullReq {
		t.Fatalf("payload %s", e.acts.list[n].Payload)
	}
	if err := e.svc.Link(ctx, 7, task.ID, "", "x"); err == nil {
		t.Fatal("empty kind must fail")
	}
	if err := e.svc.Link(ctx, 7, task.ID, enum.TaskLinkKindPullReq, " "); err == nil {
		t.Fatal("empty ref must fail")
	}
	if err := e.svc.Link(ctx, 7, 999, enum.TaskLinkKindPullReq, "x"); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Fatalf("missing task: %v", err)
	}
}

func TestCommentDecisionAgentRun(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	task := e.create(t, CreateInput{})
	version := e.tasks.byID[task.ID].Version

	if err := e.svc.AddComment(ctx, 7, task.ID, "looks like a regression"); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.AddComment(ctx, 7, task.ID, " "); err == nil {
		t.Fatal("empty comment must fail")
	}

	dec := DecisionInput{
		QuestionID: "severity", Value: "high", Confidence: 0.81,
		Probabilities: map[string]float64{"high": 0.81, "low": 0.19},
		Model:         "build-1", Fallback: true,
	}
	if err := e.svc.RecordDecision(ctx, 0, task.ID, dec); err != nil {
		t.Fatal(err)
	}
	bad := dec
	bad.Confidence = 1.5
	if err := e.svc.RecordDecision(ctx, 0, task.ID, bad); err == nil {
		t.Fatal("confidence out of range must fail")
	}
	if err := e.svc.RecordDecision(ctx, 0, task.ID, DecisionInput{}); err == nil {
		t.Fatal("missing question id must fail")
	}

	run := AgentRunInput{Model: "m", RequestsUsed: 12, Outcome: "pr_opened"}
	if err := e.svc.RecordAgentRun(ctx, 0, task.ID, run); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.RecordAgentRun(ctx, 0, task.ID, AgentRunInput{RequestsUsed: -1, Outcome: "x"}); err == nil {
		t.Fatal("negative requests must fail")
	}
	if err := e.svc.RecordAgentRun(ctx, 0, task.ID, AgentRunInput{}); err == nil {
		t.Fatal("missing outcome must fail")
	}
	if err := e.svc.AddComment(ctx, 7, 999, "x"); !errors.Is(err, gitness_store.ErrResourceNotFound) {
		t.Fatalf("missing task: %v", err)
	}

	want := []enum.TaskActivityKind{
		enum.TaskActivityKindCreated, enum.TaskActivityKindComment,
		enum.TaskActivityKindDecision, enum.TaskActivityKindAgentRun,
	}
	kinds := e.acts.kinds(task.ID)
	if len(kinds) != len(want) {
		t.Fatalf("kinds %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("kinds %v want %v", kinds, want)
		}
	}
	if e.tasks.byID[task.ID].Version != version {
		t.Fatal("history-only entries must not bump the task version")
	}

	var gotDec DecisionInput
	decAct := e.acts.list[2]
	if err := json.Unmarshal(decAct.Payload, &gotDec); err != nil ||
		gotDec.QuestionID != "severity" || gotDec.Model != "build-1" || !gotDec.Fallback ||
		gotDec.Probabilities["low"] != 0.19 {
		t.Fatalf("decision payload %s", decAct.Payload)
	}
	if decAct.PrincipalID != nil {
		t.Fatal("system activities have no principal")
	}
	var gotRun AgentRunInput
	if err := json.Unmarshal(e.acts.list[3].Payload, &gotRun); err != nil || gotRun != run {
		t.Fatalf("agent run payload %s", e.acts.list[3].Payload)
	}
}

func TestRaisePriority(t *testing.T) {
	if RaisePriority(enum.TaskPriorityP0) != enum.TaskPriorityP0 || RaisePriority(enum.TaskPriorityP3) != enum.TaskPriorityP2 {
		t.Fatal("unexpected priority raise")
	}
}

func ptr[T any](v T) *T { return &v }

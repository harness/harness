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
	"reflect"
	"testing"

	taskevents "github.com/harness/gitness/app/events/task"
	"github.com/harness/gitness/types/enum"
)

// fakeReporter records the names of the reported events.
type fakeReporter struct {
	names   []string
	state   []*taskevents.StateChangedPayload
	created []*taskevents.CreatedPayload
	assign  []*taskevents.AssignedPayload
}

func (f *fakeReporter) Created(_ context.Context, p *taskevents.CreatedPayload) {
	f.names = append(f.names, "created")
	f.created = append(f.created, p)
}

func (f *fakeReporter) StateChanged(_ context.Context, p *taskevents.StateChangedPayload) {
	f.names = append(f.names, "state_changed")
	f.state = append(f.state, p)
}

func (f *fakeReporter) Assigned(_ context.Context, p *taskevents.AssignedPayload) {
	f.names = append(f.names, "assigned")
	f.assign = append(f.assign, p)
}

func (f *fakeReporter) PriorityChanged(context.Context, *taskevents.PriorityChangedPayload) {
	f.names = append(f.names, "priority_changed")
}

func (f *fakeReporter) Resolved(context.Context, *taskevents.ResolvedPayload) {
	f.names = append(f.names, "resolved")
}

func (f *fakeReporter) Verified(context.Context, *taskevents.VerifiedPayload) {
	f.names = append(f.names, "verified")
}

func (f *fakeReporter) Reopened(context.Context, *taskevents.ReopenedPayload) {
	f.names = append(f.names, "reopened")
}

func (f *fakeReporter) CommentCreated(context.Context, *taskevents.CommentCreatedPayload) {
	f.names = append(f.names, "comment_created")
}

func (f *fakeReporter) AgentNeedsHelp(context.Context, *taskevents.AgentNeedsHelpPayload) {
	f.names = append(f.names, "agent_needs_help")
}

func (f *fakeReporter) reset() { f.names = nil }

func TestEventsEmission(t *testing.T) {
	ctx := context.Background()
	e := newEnv()
	rep := &fakeReporter{}
	e.svc.events = rep

	task := e.create(t, CreateInput{Origin: enum.TaskOriginInsights, OriginRef: "ref-1"})
	if !reflect.DeepEqual(rep.names, []string{"created"}) {
		t.Fatalf("create events: %v", rep.names)
	}
	p := rep.created[0]
	if p.TaskID != task.ID || p.RepoID != task.RepoID || p.Number != task.Number || p.PrincipalID != 7 {
		t.Fatalf("unexpected created payload: %+v", p)
	}

	// deduplicated create does not emit
	rep.reset()
	if _, created, err := e.svc.Create(ctx, 7, CreateInput{
		RepoID: 1, Title: "again", Origin: enum.TaskOriginInsights, OriginRef: "ref-1",
	}); err != nil || created {
		t.Fatalf("expected dedup, created=%v err=%v", created, err)
	}
	if len(rep.names) != 0 {
		t.Fatalf("dedup must not emit: %v", rep.names)
	}

	// a state change emits state_changed with from/to
	e.move(t, task.ID, enum.TaskStateTriage, "")
	if !reflect.DeepEqual(rep.names, []string{"state_changed"}) {
		t.Fatalf("transition events: %v", rep.names)
	}
	if sp := rep.state[0]; sp.From != string(enum.TaskStateOpen) || sp.To != string(enum.TaskStateTriage) {
		t.Fatalf("unexpected state payload: %+v", sp)
	}

	// a rejected transition does not emit
	rep.reset()
	if _, err := e.svc.Transition(ctx, 7, task.ID, enum.TaskStateVerified, ""); err == nil {
		t.Fatal("expected invalid transition")
	}
	if len(rep.names) != 0 {
		t.Fatalf("failed transition must not emit: %v", rep.names)
	}

	// update: assignment and priority
	rep.reset()
	assignee := int64(9)
	prio := enum.TaskPriorityP0
	if _, err := e.svc.Update(ctx, 7, task.ID, UpdateInput{
		Priority: &prio, Assignee: &AssigneeInput{PrincipalID: &assignee},
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if !reflect.DeepEqual(rep.names, []string{"priority_changed", "assigned"}) {
		t.Fatalf("update events: %v", rep.names)
	}
	if ap := rep.assign[0]; ap.AssigneePrincipalID == nil || *ap.AssigneePrincipalID != 9 {
		t.Fatalf("unexpected assigned payload: %+v", ap)
	}

	// a no-op update does not emit
	rep.reset()
	if _, err := e.svc.Update(ctx, 7, task.ID, UpdateInput{Priority: &prio}); err != nil {
		t.Fatalf("noop update: %v", err)
	}
	if len(rep.names) != 0 {
		t.Fatalf("noop update must not emit: %v", rep.names)
	}

	// comment
	if err := e.svc.AddComment(ctx, 7, task.ID, "hello"); err != nil {
		t.Fatalf("comment: %v", err)
	}
	if !reflect.DeepEqual(rep.names, []string{"comment_created"}) {
		t.Fatalf("comment events: %v", rep.names)
	}
}

func TestEventsResolvedVerifiedReopened(t *testing.T) {
	e := newEnv()
	rep := &fakeReporter{}
	e.svc.events = rep
	task := e.create(t, CreateInput{})
	e.path(t, task.ID, enum.TaskStateTriage, enum.TaskStateInProgress, enum.TaskStateInReview)

	rep.reset()
	e.move(t, task.ID, enum.TaskStateResolved, "")
	if !reflect.DeepEqual(rep.names, []string{"state_changed", "resolved"}) {
		t.Fatalf("resolve events: %v", rep.names)
	}
	rep.reset()
	e.move(t, task.ID, enum.TaskStateVerified, "")
	if !reflect.DeepEqual(rep.names, []string{"state_changed", "verified"}) {
		t.Fatalf("verify events: %v", rep.names)
	}
	rep.reset()
	e.move(t, task.ID, enum.TaskStateReopened, "regression")
	want := []string{"state_changed", "reopened", "priority_changed"}
	if !reflect.DeepEqual(rep.names, want) {
		t.Fatalf("reopen events: %v", rep.names)
	}
}

func TestNilReporterIsSafe(t *testing.T) {
	e := newEnv()
	task := e.create(t, CreateInput{})
	e.move(t, task.ID, enum.TaskStateTriage, "")
	if err := e.svc.AddComment(context.Background(), 7, task.ID, "x"); err != nil {
		t.Fatalf("comment: %v", err)
	}
}

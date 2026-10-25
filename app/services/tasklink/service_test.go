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

package tasklink

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/events"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	testRepoID      = int64(10)
	testPrincipalID = int64(-1)
)

// fake implements taskService, taskFinder, linkFinder, pullReqFinder and principalFinder in memory.
type fake struct {
	tasks       map[int64]*types.Task
	links       map[int64][]*types.TaskLink
	pullreqs    map[int64]*types.PullReq
	activities  []string
	transitions []string
}

func newFake() *fake {
	return &fake{
		tasks:    map[int64]*types.Task{},
		links:    map[int64][]*types.TaskLink{},
		pullreqs: map[int64]*types.PullReq{},
	}
}

func (f *fake) addTask(number int64, repoID int64, state enum.TaskState) *types.Task {
	t := &types.Task{ID: 100 + number + repoID*1000, RepoID: repoID, Number: number, State: state}
	f.tasks[t.ID] = t
	return t
}

func (f *fake) addPR(id int64, state enum.PullReqState, title, desc string) *types.PullReq {
	pr := &types.PullReq{
		ID: id, Number: id + 1000, TargetRepoID: testRepoID, State: state,
		Title: title, Description: desc, CreatedBy: 7,
	}
	f.pullreqs[id] = pr
	return pr
}

func (f *fake) Find(_ context.Context, id int64) (*types.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	c := *t
	return &c, nil
}

func (f *fake) FindByNumber(_ context.Context, repoID, number int64) (*types.Task, error) {
	for _, t := range f.tasks {
		if t.RepoID == repoID && t.Number == number {
			c := *t
			return &c, nil
		}
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (f *fake) Transition(
	_ context.Context, principalID, taskID int64, to enum.TaskState, reason string,
) (*types.Task, error) {
	if principalID != testPrincipalID {
		return nil, fmt.Errorf("unexpected principal %d", principalID)
	}
	t := f.tasks[taskID]
	if !task.CanTransition(t.State, to) {
		return nil, &task.TransitionError{From: t.State, To: to, Message: "not allowed"}
	}
	f.activities = append(f.activities, fmt.Sprintf("state:%d:%s->%s:%s", t.Number, t.State, to, reason))
	f.transitions = append(f.transitions, fmt.Sprintf("%d:%s", t.Number, to))
	t.State = to
	c := *t
	return &c, nil
}

func (f *fake) hasLink(taskID int64, refID string) int {
	for i, l := range f.links[taskID] {
		if l.Kind == enum.TaskLinkKindPullReq && l.RefID == refID {
			return i
		}
	}
	return -1
}

func (f *fake) Link(_ context.Context, _, taskID int64, kind enum.TaskLinkKind, refID string) error {
	if f.hasLink(taskID, refID) >= 0 {
		return nil
	}
	f.links[taskID] = append(f.links[taskID], &types.TaskLink{TaskID: taskID, Kind: kind, RefID: refID})
	f.activities = append(f.activities, fmt.Sprintf("link:%d:%s", f.tasks[taskID].Number, refID))
	return nil
}

func (f *fake) Unlink(_ context.Context, _, taskID int64, _ enum.TaskLinkKind, refID string) error {
	i := f.hasLink(taskID, refID)
	if i < 0 {
		return nil
	}
	f.links[taskID] = append(f.links[taskID][:i], f.links[taskID][i+1:]...)
	f.activities = append(f.activities, fmt.Sprintf("unlink:%d:%s", f.tasks[taskID].Number, refID))
	return nil
}

func (f *fake) ListByTask(_ context.Context, taskID int64) ([]*types.TaskLink, error) {
	return f.links[taskID], nil
}

func (f *fake) FindTasksByRef(_ context.Context, _ enum.TaskLinkKind, refID string) ([]*types.Task, error) {
	var out []*types.Task
	for id, links := range f.links {
		for _, l := range links {
			if l.RefID == refID {
				c := *f.tasks[id]
				out = append(out, &c)
			}
		}
	}
	return out, nil
}

type fakePullReqs struct{ f *fake }

func (p fakePullReqs) Find(_ context.Context, id int64) (*types.PullReq, error) {
	pr, ok := p.f.pullreqs[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	c := *pr
	return &c, nil
}

type fakePrincipals struct{}

func (fakePrincipals) Find(_ context.Context, id int64) (*types.Principal, error) {
	return &types.Principal{ID: id, UID: "author"}, nil
}

func newTestService(f *fake) *Service {
	s := newService(f, f, f, fakePullReqs{f}, fakePrincipals{})
	s.principalID = func() int64 { return testPrincipalID }
	return s
}

func base(prID int64) pullreqevents.Base {
	return pullreqevents.Base{PullReqID: prID, TargetRepoID: testRepoID, Number: prID + 1000}
}

func created(prID int64) *events.Event[*pullreqevents.CreatedPayload] {
	return &events.Event[*pullreqevents.CreatedPayload]{Payload: &pullreqevents.CreatedPayload{Base: base(prID)}}
}

func merged(prID int64) *events.Event[*pullreqevents.MergedPayload] {
	return &events.Event[*pullreqevents.MergedPayload]{Payload: &pullreqevents.MergedPayload{Base: base(prID)}}
}

func closed(prID int64) *events.Event[*pullreqevents.ClosedPayload] {
	return &events.Event[*pullreqevents.ClosedPayload]{Payload: &pullreqevents.ClosedPayload{Base: base(prID)}}
}

func updated(prID int64, title, desc bool) *events.Event[*pullreqevents.UpdatedPayload] {
	return &events.Event[*pullreqevents.UpdatedPayload]{Payload: &pullreqevents.UpdatedPayload{
		Base: base(prID), TitleChanged: title, DescriptionChanged: desc,
	}}
}

func (f *fake) state(number int64) enum.TaskState {
	t, _ := f.FindByNumber(context.Background(), testRepoID, number)
	return t.State
}

func TestOnCreated(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		start enum.TaskState
		want  enum.TaskState
	}{
		{"abierta goes through en_curso", enum.TaskStateOpen, enum.TaskStateInReview},
		{"triage goes through en_curso", enum.TaskStateTriage, enum.TaskStateInReview},
		{"en_curso", enum.TaskStateInProgress, enum.TaskStateInReview},
		{"reabierta", enum.TaskStateReopened, enum.TaskStateInReview},
		{"already en_revision", enum.TaskStateInReview, enum.TaskStateInReview},
		{"bloqueada untouched", enum.TaskStateBlocked, enum.TaskStateBlocked},
		{"resuelta untouched", enum.TaskStateResolved, enum.TaskStateResolved},
		{"descartada untouched", enum.TaskStateDismissed, enum.TaskStateDismissed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake()
			f.addTask(1, testRepoID, tt.start)
			f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
			s := newTestService(f)

			require.NoError(t, s.onCreated(ctx, created(1)))
			require.Equal(t, tt.want, f.state(1))
			require.Len(t, f.links[f.mustTask(1).ID], 1)
		})
	}

	t.Run("mention links without transition", func(t *testing.T) {
		f := newFake()
		f.addTask(2, testRepoID, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateOpen, "Work on TASK-2", "")
		require.NoError(t, newTestService(f).onCreated(ctx, created(1)))
		require.Equal(t, enum.TaskStateInProgress, f.state(2))
		require.Len(t, f.links[f.mustTask(2).ID], 1)
		require.Empty(t, f.transitions)
	})

	t.Run("task of another repo and missing task are ignored", func(t *testing.T) {
		f := newFake()
		other := f.addTask(3, testRepoID+1, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-3, TASK-99", "")
		require.NoError(t, newTestService(f).onCreated(ctx, created(1)))
		require.Equal(t, enum.TaskStateInProgress, f.tasks[other.ID].State)
		require.Empty(t, f.activities)
	})

	t.Run("replay is idempotent", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateOpen)
		f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
		s := newTestService(f)
		require.NoError(t, s.onCreated(ctx, created(1)))
		n := len(f.activities)
		require.NoError(t, s.onCreated(ctx, created(1)))
		require.Len(t, f.activities, n)
	})

	t.Run("author is recorded in the reason", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
		require.NoError(t, newTestService(f).onCreated(ctx, created(1)))
		require.Contains(t, f.activities[len(f.activities)-1], "author: author")
	})

	t.Run("non open pull request is ignored", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateMerged, "Fixes TASK-1", "")
		require.NoError(t, newTestService(f).onCreated(ctx, created(1)))
		require.Empty(t, f.activities)
	})
}

func (f *fake) mustTask(number int64) *types.Task {
	t, err := f.FindByNumber(context.Background(), testRepoID, number)
	if err != nil {
		panic(strconv.FormatInt(number, 10))
	}
	return t
}

func TestOnUpdated(t *testing.T) {
	ctx := context.Background()

	t.Run("not title or description change is ignored", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
		require.NoError(t, newTestService(f).onUpdated(ctx, updated(1, false, false)))
		require.Empty(t, f.activities)
	})

	t.Run("added reference links, removed reference unlinks", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		f.addTask(2, testRepoID, enum.TaskStateInProgress)
		pr := f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
		s := newTestService(f)
		require.NoError(t, s.onCreated(ctx, created(1)))
		require.Equal(t, enum.TaskStateInReview, f.state(1))

		pr.Title = "Fixes TASK-2"
		require.NoError(t, s.onUpdated(ctx, updated(1, true, false)))
		require.Empty(t, f.links[f.mustTask(1).ID])
		require.Len(t, f.links[f.mustTask(2).ID], 1)
		require.Equal(t, enum.TaskStateInReview, f.state(2))
		// the first task is not moved back automatically
		require.Equal(t, enum.TaskStateInReview, f.state(1))

		n := len(f.activities)
		require.NoError(t, s.onUpdated(ctx, updated(1, true, false)))
		require.Len(t, f.activities, n)
	})

	t.Run("description change", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateOpen, "Title", "Closes TASK-1")
		require.NoError(t, newTestService(f).onUpdated(ctx, updated(1, false, true)))
		require.Equal(t, enum.TaskStateInReview, f.state(1))
	})
}

func TestOnMerged(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		start enum.TaskState
		want  enum.TaskState
	}{
		{"en_revision", enum.TaskStateInReview, enum.TaskStateResolved},
		{"en_curso goes through en_revision", enum.TaskStateInProgress, enum.TaskStateResolved},
		{"abierta goes through en_revision", enum.TaskStateOpen, enum.TaskStateResolved},
		{"reabierta", enum.TaskStateReopened, enum.TaskStateResolved},
		{"already resuelta", enum.TaskStateResolved, enum.TaskStateResolved},
		{"verificada untouched", enum.TaskStateVerified, enum.TaskStateVerified},
		{"descartada untouched", enum.TaskStateDismissed, enum.TaskStateDismissed},
		{"bloqueada untouched", enum.TaskStateBlocked, enum.TaskStateBlocked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake()
			f.addTask(1, testRepoID, tt.start)
			f.addPR(1, enum.PullReqStateMerged, "Fixes TASK-1", "")
			require.NoError(t, newTestService(f).onMerged(ctx, merged(1)))
			require.Equal(t, tt.want, f.state(1))
		})
	}

	t.Run("mention does not transition and replay is idempotent", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInReview)
		f.addTask(2, testRepoID, enum.TaskStateInReview)
		f.addPR(1, enum.PullReqStateMerged, "Fixes TASK-1", "see TASK-2")
		s := newTestService(f)
		require.NoError(t, s.onMerged(ctx, merged(1)))
		require.Equal(t, enum.TaskStateResolved, f.state(1))
		require.Equal(t, enum.TaskStateInReview, f.state(2))
		n := len(f.activities)
		require.NoError(t, s.onMerged(ctx, merged(1)))
		require.Len(t, f.activities, n)
	})
}

func TestOnClosed(t *testing.T) {
	ctx := context.Background()

	t.Run("task goes back to en_curso", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		pr := f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
		s := newTestService(f)
		require.NoError(t, s.onCreated(ctx, created(1)))
		require.Equal(t, enum.TaskStateInReview, f.state(1))

		pr.State = enum.PullReqStateClosed
		require.NoError(t, s.onClosed(ctx, closed(1)))
		require.Equal(t, enum.TaskStateInProgress, f.state(1))

		n := len(f.activities)
		require.NoError(t, s.onClosed(ctx, closed(1)))
		require.Len(t, f.activities, n)
	})

	t.Run("other open pull request keeps the task in review", func(t *testing.T) {
		f := newFake()
		f.addTask(1, testRepoID, enum.TaskStateInProgress)
		f.addPR(1, enum.PullReqStateOpen, "Fixes TASK-1", "")
		pr2 := f.addPR(2, enum.PullReqStateOpen, "Also TASK-1", "")
		s := newTestService(f)
		require.NoError(t, s.onCreated(ctx, created(1)))
		require.NoError(t, s.onCreated(ctx, created(2)))

		f.pullreqs[1].State = enum.PullReqStateClosed
		require.NoError(t, s.onClosed(ctx, closed(1)))
		require.Equal(t, enum.TaskStateInReview, f.state(1))

		pr2.State = enum.PullReqStateClosed
		require.NoError(t, s.onClosed(ctx, closed(2)))
		require.Equal(t, enum.TaskStateInProgress, f.state(1))
	})

	t.Run("tasks not in review are untouched", func(t *testing.T) {
		f := newFake()
		task1 := f.addTask(1, testRepoID, enum.TaskStateResolved)
		f.addPR(1, enum.PullReqStateClosed, "Fixes TASK-1", "")
		f.links[task1.ID] = []*types.TaskLink{{TaskID: task1.ID, Kind: enum.TaskLinkKindPullReq, RefID: "1"}}
		require.NoError(t, newTestService(f).onClosed(ctx, closed(1)))
		require.Equal(t, enum.TaskStateResolved, f.state(1))
	})

	t.Run("missing pull request returns an error", func(t *testing.T) {
		f := newFake()
		require.Error(t, newTestService(f).onClosed(ctx, closed(5)))
	})
}

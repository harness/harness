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

package notification

import (
	"context"
	"testing"

	taskevents "github.com/harness/gitness/app/events/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type taskStoreStub struct {
	store.TaskStore
	task *types.Task
}

func (s *taskStoreStub) Find(_ context.Context, _ int64) (*types.Task, error) { return s.task, nil }

type taskActivityStoreStub struct {
	store.TaskActivityStore
	comments []*types.TaskActivity
}

func (s *taskActivityStoreStub) ListByKind(
	_ context.Context, _ int64, _ enum.TaskActivityKind,
) ([]*types.TaskActivity, error) {
	return s.comments, nil
}

type taskClient struct {
	testNotificationClient
	calls []*taskCall
}

type taskCall struct {
	recipients []*types.PrincipalInfo
	payload    *TaskPayload
}

func (c *taskClient) SendTaskNotification(
	_ context.Context, r []*types.PrincipalInfo, p *TaskPayload,
) error {
	c.calls = append(c.calls, &taskCall{r, p})
	return nil
}

type taskURLStub struct{ urlProviderStub }

func (taskURLStub) GenerateUIRepoURL(context.Context, string) string { return "https://ui/space/repo" }

func ptr(i int64) *int64 { return &i }

func newTaskService(t *types.Task, comments []*types.TaskActivity, members []int64) (*Service, *taskClient) {
	users := map[int64]*types.PrincipalInfo{}
	for id := int64(1); id <= 9; id++ {
		users[id] = &types.PrincipalInfo{ID: id, Email: "u@example.com", DisplayName: "u"}
	}
	users[9].Email = "" // agent
	c := &taskClient{}
	return &Service{
		taskStore:          &taskStoreStub{task: t},
		taskActivityStore:  &taskActivityStoreStub{comments: comments},
		repoStore:          &repoStoreStub{repo: &types.Repository{ID: 1, Path: "space/repo"}},
		principalInfoCache: &principalInfoCacheStub{byID: users},
		userGroupService:   &userGroupServiceStub{memberIDs: members},
		urlProvider:        &taskURLStub{},
		notificationClient: c,
	}, c
}

func ids(r []*types.PrincipalInfo) []int64 { return recipientIDs(r) }

func base(actor int64) taskevents.Base {
	return taskevents.Base{TaskID: 1, RepoID: 1, Number: 7, PrincipalID: actor}
}

func TestTaskAssignedToUserAndGroup(t *testing.T) {
	task := &types.Task{ID: 1, Number: 7, RepoID: 1, CreatedBy: 1}
	svc, c := newTaskService(task, nil, []int64{3, 4, 2})

	err := svc.notifyTaskAssigned(context.Background(), &events.Event[*taskevents.AssignedPayload]{
		Payload: &taskevents.AssignedPayload{Base: base(2), AssigneePrincipalID: ptr(5), AssigneeGroupID: ptr(10)},
	})
	require.NoError(t, err)
	require.Len(t, c.calls, 1)
	// actor 2 is a group member and is skipped
	require.Equal(t, []int64{5, 3, 4}, ids(c.calls[0].recipients))
	require.Equal(t, TaskNotificationAssigned, c.calls[0].payload.Kind)
	require.Equal(t, "https://ui/space/repo/tasks/7", c.calls[0].payload.TaskURL)
}

func TestTaskAssignedToAgentIsNotNotified(t *testing.T) {
	svc, c := newTaskService(&types.Task{ID: 1}, nil, nil)
	err := svc.notifyTaskAssigned(context.Background(), &events.Event[*taskevents.AssignedPayload]{
		Payload: &taskevents.AssignedPayload{Base: base(2), AssigneePrincipalID: ptr(9), AssigneeIsAgent: true},
	})
	require.NoError(t, err)
	require.Empty(t, c.calls)
}

func TestTaskStateChangedRecipients(t *testing.T) {
	task := &types.Task{ID: 1, RepoID: 1, CreatedBy: 1, AssigneePrincipalID: ptr(2)}
	svc, c := newTaskService(task, nil, nil)
	ev := func(actor int64, to enum.TaskState) *events.Event[*taskevents.StateChangedPayload] {
		return &events.Event[*taskevents.StateChangedPayload]{
			Payload: &taskevents.StateChangedPayload{Base: base(actor), From: "abierta", To: string(to)},
		}
	}

	require.NoError(t, svc.notifyTaskStateChanged(context.Background(), ev(2, enum.TaskStateInProgress)))
	require.Equal(t, []int64{1}, ids(c.calls[0].recipients)) // actor is the assignee

	require.NoError(t, svc.notifyTaskStateChanged(context.Background(), ev(3, enum.TaskStateInProgress)))
	require.Equal(t, []int64{2, 1}, ids(c.calls[1].recipients))

	// reopened state change is reported by the reopened event only
	require.NoError(t, svc.notifyTaskStateChanged(context.Background(), ev(3, enum.TaskStateReopened)))
	require.Len(t, c.calls, 2)

	require.NoError(t, svc.notifyTaskReopened(context.Background(), &events.Event[*taskevents.ReopenedPayload]{
		Payload: &taskevents.ReopenedPayload{Base: base(3)},
	}))
	require.Equal(t, TaskNotificationReopened, c.calls[2].payload.Kind)
	require.Equal(t, []int64{2, 1}, ids(c.calls[2].recipients))
}

func TestTaskBlockedAgentNeedsHelp(t *testing.T) {
	task := &types.Task{ID: 1, RepoID: 1, CreatedBy: 1, AssigneePrincipalID: ptr(9), AssigneeIsAgent: true}
	svc, c := newTaskService(task, nil, nil)
	err := svc.notifyTaskStateChanged(context.Background(), &events.Event[*taskevents.StateChangedPayload]{
		Payload: &taskevents.StateChangedPayload{Base: base(9), From: "en_curso", To: string(enum.TaskStateBlocked)},
	})
	require.NoError(t, err)
	require.Len(t, c.calls, 1)
	require.Equal(t, TaskNotificationAgentNeedsHelp, c.calls[0].payload.Kind)
	require.Equal(t, []int64{1}, ids(c.calls[0].recipients))

	require.NoError(t, svc.notifyTaskAgentNeedsHelp(context.Background(),
		&events.Event[*taskevents.AgentNeedsHelpPayload]{Payload: &taskevents.AgentNeedsHelpPayload{Base: base(0)}}))
	require.Len(t, c.calls, 2)
	require.Equal(t, []int64{1}, ids(c.calls[1].recipients))
}

func TestTaskCommentRecipients(t *testing.T) {
	task := &types.Task{ID: 1, RepoID: 1, CreatedBy: 1, AssigneePrincipalID: ptr(2)}
	comments := []*types.TaskActivity{
		{PrincipalID: ptr(3)}, {PrincipalID: ptr(4)}, {PrincipalID: ptr(3)}, {PrincipalID: ptr(1)}, {},
	}
	svc, c := newTaskService(task, comments, nil)
	err := svc.notifyTaskCommentCreated(context.Background(), &events.Event[*taskevents.CommentCreatedPayload]{
		Payload: &taskevents.CommentCreatedPayload{Base: base(4)},
	})
	require.NoError(t, err)
	// assignee, creator, previous commenters; no duplicates, author 4 excluded
	require.Equal(t, []int64{2, 1, 3}, ids(c.calls[0].recipients))
}

func TestNotifyTaskSLARecipients(t *testing.T) {
	task := &types.Task{ID: 1, RepoID: 1, CreatedBy: 1, AssigneePrincipalID: ptr(2)}
	svc, c := newTaskService(task, nil, nil)
	require.NoError(t, svc.NotifyTaskSLA(context.Background(), task, false))
	require.Equal(t, []int64{2}, ids(c.calls[0].recipients))
	require.Equal(t, TaskNotificationSLAWarning, c.calls[0].payload.Kind)

	// agent assignee has no email: the creator is notified
	agentTask := &types.Task{ID: 1, RepoID: 1, CreatedBy: 1, AssigneePrincipalID: ptr(9), AssigneeIsAgent: true}
	svc, c = newTaskService(agentTask, nil, nil)
	require.NoError(t, svc.NotifyTaskSLA(context.Background(), agentTask, true))
	require.Equal(t, []int64{1}, ids(c.calls[0].recipients))
	require.Equal(t, TaskNotificationSLAOverdue, c.calls[0].payload.Kind)
}

func TestTaskTemplatesRender(t *testing.T) {
	p := &TaskPayload{
		Repo: &types.Repository{Identifier: "repo", Path: "space/repo"},
		Task: &types.Task{Number: 7, Title: "T"}, TaskURL: "u", From: "a", To: "b",
	}
	for kind, name := range taskTemplates {
		p.Kind = kind
		out, err := GetHTMLBody(name, p)
		require.NoError(t, err, name)
		require.Contains(t, string(out), "#7")
	}
}

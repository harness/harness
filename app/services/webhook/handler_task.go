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

package webhook

import (
	"context"
	"errors"
	"fmt"

	taskevents "github.com/harness/gitness/app/events/task"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// TaskInfo describes the task related info for a webhook payload.
// NOTE: don't use types package as we want the payload to be independent from API calls.
type TaskInfo struct {
	Number   int64             `json:"number"`
	Title    string            `json:"title"`
	State    enum.TaskState    `json:"state"`
	Priority enum.TaskPriority `json:"priority"`
	Origin   enum.TaskOrigin   `json:"origin"`
	Assignee TaskAssigneeInfo  `json:"assignee"`
	URL      string            `json:"url"`
}

// TaskAssigneeInfo describes the assignee of a task. All zero values mean "unassigned".
type TaskAssigneeInfo struct {
	PrincipalID *int64 `json:"principal_id,omitempty"`
	GroupID     *int64 `json:"group_id,omitempty"`
	IsAgent     bool   `json:"is_agent"`
}

// TaskChangeInfo describes what changed on a task for task_updated triggers.
type TaskChangeInfo struct {
	// Kind is one of state, assignment or priority.
	Kind string `json:"kind"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	// Reason is the reason of a state change, if any.
	Reason string `json:"reason,omitempty"`
}

// TaskPayload describes the body of the task created and task resolved triggers.
type TaskPayload struct {
	BaseSegment
	Task TaskInfo `json:"task"`
}

// TaskUpdatedPayload describes the body of the task updated trigger.
type TaskUpdatedPayload struct {
	BaseSegment
	Task   TaskInfo       `json:"task"`
	Change TaskChangeInfo `json:"change"`
}

// taskInfoFrom gets the TaskInfo from a types.Task.
func taskInfoFrom(ctx context.Context, task *types.Task, repo *types.Repository, urlProvider url.Provider) TaskInfo {
	return TaskInfo{
		Number:   task.Number,
		Title:    task.Title,
		State:    task.State,
		Priority: task.Priority,
		Origin:   task.Origin,
		Assignee: TaskAssigneeInfo{
			PrincipalID: task.AssigneePrincipalID,
			GroupID:     task.AssigneeGroupID,
			IsAgent:     task.AssigneeIsAgent,
		},
		URL: fmt.Sprintf("%s/tasks/%d", urlProvider.GenerateUIRepoURL(ctx, repo.Path), task.Number),
	}
}

// triggerForEventWithTask triggers all webhooks for the repo of the task and triggerType.
// The task is read when the event is handled, so the payload reflects the current state of the task.
func (s *Service) triggerForEventWithTask(
	ctx context.Context,
	triggerType enum.WebhookTrigger,
	eventID string,
	principalID int64,
	taskID int64,
	createBodyFn func(*types.Principal, *types.Repository, *types.Task) (any, error),
) error {
	task, err := s.taskStore.Find(ctx, taskID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return events.NewDiscardEventErrorf("task with id '%d' doesn't exist anymore", taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to get task for id '%d': %w", taskID, err)
	}

	return s.triggerForEventWithRepo(ctx, triggerType, eventID, principalID, task.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			return createBodyFn(principal, repo, task)
		})
}

func (s *Service) triggerTaskPayload(
	ctx context.Context,
	triggerType enum.WebhookTrigger,
	eventID string,
	base taskevents.Base,
) error {
	return s.triggerForEventWithTask(ctx, triggerType, eventID, base.PrincipalID, base.TaskID,
		func(principal *types.Principal, repo *types.Repository, task *types.Task) (any, error) {
			return &TaskPayload{
				BaseSegment: BaseSegment{
					Trigger:   triggerType,
					Repo:      repositoryInfoFrom(ctx, repo, s.urlProvider),
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				Task: taskInfoFrom(ctx, task, repo, s.urlProvider),
			}, nil
		})
}

func (s *Service) triggerTaskUpdated(
	ctx context.Context,
	eventID string,
	base taskevents.Base,
	change TaskChangeInfo,
) error {
	return s.triggerForEventWithTask(ctx, enum.WebhookTriggerTaskUpdated, eventID, base.PrincipalID, base.TaskID,
		func(principal *types.Principal, repo *types.Repository, task *types.Task) (any, error) {
			return &TaskUpdatedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerTaskUpdated,
					Repo:      repositoryInfoFrom(ctx, repo, s.urlProvider),
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				Task:   taskInfoFrom(ctx, task, repo, s.urlProvider),
				Change: change,
			}, nil
		})
}

// handleEventTaskCreated handles task created events and triggers task created webhooks.
func (s *Service) handleEventTaskCreated(
	ctx context.Context,
	event *events.Event[*taskevents.CreatedPayload],
) error {
	return s.triggerTaskPayload(ctx, enum.WebhookTriggerTaskCreated, event.ID, event.Payload.Base)
}

// handleEventTaskResolved handles task resolved events and triggers task resolved webhooks.
func (s *Service) handleEventTaskResolved(
	ctx context.Context,
	event *events.Event[*taskevents.ResolvedPayload],
) error {
	return s.triggerTaskPayload(ctx, enum.WebhookTriggerTaskResolved, event.ID, event.Payload.Base)
}

// handleEventTaskStateChanged triggers task updated webhooks for state changes.
// Moves to resolved are skipped, they trigger the task resolved webhooks.
func (s *Service) handleEventTaskStateChanged(
	ctx context.Context,
	event *events.Event[*taskevents.StateChangedPayload],
) error {
	if event.Payload.To == string(enum.TaskStateResolved) {
		return nil
	}
	return s.triggerTaskUpdated(ctx, event.ID, event.Payload.Base, TaskChangeInfo{
		Kind: "state", From: event.Payload.From, To: event.Payload.To, Reason: event.Payload.Reason,
	})
}

// handleEventTaskAssigned triggers task updated webhooks for assignment changes.
func (s *Service) handleEventTaskAssigned(
	ctx context.Context,
	event *events.Event[*taskevents.AssignedPayload],
) error {
	return s.triggerTaskUpdated(ctx, event.ID, event.Payload.Base, TaskChangeInfo{Kind: "assignment"})
}

// handleEventTaskPriorityChanged triggers task updated webhooks for priority changes.
func (s *Service) handleEventTaskPriorityChanged(
	ctx context.Context,
	event *events.Event[*taskevents.PriorityChangedPayload],
) error {
	return s.triggerTaskUpdated(ctx, event.ID, event.Payload.Base, TaskChangeInfo{
		Kind: "priority", From: event.Payload.From, To: event.Payload.To,
	})
}

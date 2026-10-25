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
	"fmt"
	"strings"

	taskevents "github.com/harness/gitness/app/events/task"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// TaskNotificationKind identifies the kind of a task notification.
type TaskNotificationKind string

const (
	TaskNotificationAssigned       TaskNotificationKind = "assigned"
	TaskNotificationStateChanged   TaskNotificationKind = "state_changed"
	TaskNotificationReopened       TaskNotificationKind = "reopened"
	TaskNotificationCommentCreated TaskNotificationKind = "comment_created"
	TaskNotificationAgentNeedsHelp TaskNotificationKind = "agent_needs_help"
	TaskNotificationSLAWarning     TaskNotificationKind = "sla_warning"
	TaskNotificationSLAOverdue     TaskNotificationKind = "sla_overdue"
)

// TaskPayload is the data of a task notification.
type TaskPayload struct {
	Kind    TaskNotificationKind
	Repo    *types.Repository
	Task    *types.Task
	Actor   *types.PrincipalInfo // nil if the change was made by the system
	TaskURL string
	From    string
	To      string
	Reason  string
}

// recipientSpec describes who is notified about a task event.
type recipientSpec struct {
	assigneePrincipalID *int64
	assigneeGroupID     *int64
	creator             bool
	commenters          bool
}

func (s *Service) registerTaskHandlers(r *taskevents.Reader) {
	_ = r.RegisterAssigned(s.notifyTaskAssigned)
	_ = r.RegisterStateChanged(s.notifyTaskStateChanged)
	_ = r.RegisterReopened(s.notifyTaskReopened)
	_ = r.RegisterCommentCreated(s.notifyTaskCommentCreated)
	_ = r.RegisterAgentNeedsHelp(s.notifyTaskAgentNeedsHelp)
}

func (s *Service) notifyTaskAssigned(
	ctx context.Context,
	event *events.Event[*taskevents.AssignedPayload],
) error {
	p := event.Payload
	if p.AssigneeIsAgent {
		return nil // agents are not notified by mail
	}
	// the assignee of the event is the recipient, not necessarily the current one
	return s.sendTask(ctx, p.Base, TaskNotificationAssigned, "", "", "", recipientSpec{
		assigneePrincipalID: p.AssigneePrincipalID,
		assigneeGroupID:     p.AssigneeGroupID,
	})
}

func (s *Service) notifyTaskStateChanged(
	ctx context.Context,
	event *events.Event[*taskevents.StateChangedPayload],
) error {
	p := event.Payload
	if p.To == string(enum.TaskStateReopened) {
		return nil // reported by the reopened event
	}
	t, err := s.taskStore.Find(ctx, p.TaskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", p.TaskID, err)
	}
	if p.To == string(enum.TaskStateBlocked) && t.AssigneeIsAgent {
		return s.sendTaskFor(ctx, t, p.PrincipalID, TaskNotificationAgentNeedsHelp, p.From, p.To, p.Reason,
			helpRecipients())
	}
	return s.sendTaskFor(ctx, t, p.PrincipalID, TaskNotificationStateChanged, p.From, p.To, p.Reason,
		currentAssigneeAndCreator(t))
}

func (s *Service) notifyTaskReopened(
	ctx context.Context,
	event *events.Event[*taskevents.ReopenedPayload],
) error {
	p := event.Payload
	t, err := s.taskStore.Find(ctx, p.TaskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", p.TaskID, err)
	}
	return s.sendTaskFor(ctx, t, p.PrincipalID, TaskNotificationReopened, "", string(enum.TaskStateReopened),
		p.Reason, currentAssigneeAndCreator(t))
}

func (s *Service) notifyTaskCommentCreated(
	ctx context.Context,
	event *events.Event[*taskevents.CommentCreatedPayload],
) error {
	p := event.Payload
	t, err := s.taskStore.Find(ctx, p.TaskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", p.TaskID, err)
	}
	spec := currentAssigneeAndCreator(t)
	spec.commenters = true
	return s.sendTaskFor(ctx, t, p.PrincipalID, TaskNotificationCommentCreated, "", "", "", spec)
}

func (s *Service) notifyTaskAgentNeedsHelp(
	ctx context.Context,
	event *events.Event[*taskevents.AgentNeedsHelpPayload],
) error {
	p := event.Payload
	t, err := s.taskStore.Find(ctx, p.TaskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", p.TaskID, err)
	}
	return s.sendTaskFor(ctx, t, p.PrincipalID, TaskNotificationAgentNeedsHelp, "", "", "", helpRecipients())
}

func currentAssigneeAndCreator(t *types.Task) recipientSpec {
	return recipientSpec{
		assigneePrincipalID: t.AssigneePrincipalID,
		assigneeGroupID:     t.AssigneeGroupID,
		creator:             true,
	}
}

// helpRecipients resolves who is asked for help: the task creator. Repository admins and code owners
// are not resolvable without per-repository authorization lookups, so the creator is the conservative choice.
func helpRecipients() recipientSpec {
	return recipientSpec{creator: true}
}

// NotifyTaskSLA notifies about a task that is about to be due (overdue=false) or already overdue.
// The assignee is notified; tasks without a human assignee notify their creator.
func (s *Service) NotifyTaskSLA(ctx context.Context, t *types.Task, overdue bool) error {
	kind := TaskNotificationSLAWarning
	if overdue {
		kind = TaskNotificationSLAOverdue
	}
	recipients, err := s.taskRecipients(ctx, t, 0, recipientSpec{
		assigneePrincipalID: t.AssigneePrincipalID,
		assigneeGroupID:     t.AssigneeGroupID,
	})
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		recipients, err = s.taskRecipients(ctx, t, 0, recipientSpec{creator: true})
		if err != nil {
			return err
		}
	}
	return s.deliverTask(ctx, recipients, t, 0, kind, "", "", "")
}

func (s *Service) sendTask(
	ctx context.Context,
	base taskevents.Base,
	kind TaskNotificationKind,
	from, to, reason string,
	spec recipientSpec,
) error {
	t, err := s.taskStore.Find(ctx, base.TaskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", base.TaskID, err)
	}
	return s.sendTaskFor(ctx, t, base.PrincipalID, kind, from, to, reason, spec)
}

func (s *Service) sendTaskFor(
	ctx context.Context,
	t *types.Task,
	actorID int64,
	kind TaskNotificationKind,
	from, to, reason string,
	spec recipientSpec,
) error {
	recipients, err := s.taskRecipients(ctx, t, actorID, spec)
	if err != nil {
		return err
	}
	return s.deliverTask(ctx, recipients, t, actorID, kind, from, to, reason)
}

func (s *Service) deliverTask(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	t *types.Task,
	actorID int64,
	kind TaskNotificationKind,
	from, to, reason string,
) error {
	if len(recipients) == 0 {
		return nil
	}
	repo, err := s.repoStore.Find(ctx, t.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo %d: %w", t.RepoID, err)
	}
	var actor *types.PrincipalInfo
	if actorID > 0 {
		if actor, err = s.principalInfoCache.Get(ctx, actorID); err != nil {
			actor = nil
		}
	}
	repoURL := strings.TrimRight(s.urlProvider.GenerateUIRepoURL(ctx, repo.Path), "/")
	payload := &TaskPayload{
		Kind:    kind,
		Repo:    repo,
		Task:    t,
		Actor:   actor,
		TaskURL: fmt.Sprintf("%s/tasks/%d", repoURL, t.Number),
		From:    from,
		To:      to,
		Reason:  reason,
	}
	if err = s.notificationClient.SendTaskNotification(ctx, recipients, payload); err != nil {
		return fmt.Errorf("failed to send task %s notification for task %d: %w", kind, t.ID, err)
	}
	return nil
}

// taskRecipients computes the principals to notify: the assignee (user or members of the group),
// the creator and previous commenters as requested by the spec. The actor is never a recipient and
// principals without an email (e.g. agents) are skipped.
func (s *Service) taskRecipients(
	ctx context.Context,
	t *types.Task,
	actorID int64,
	spec recipientSpec,
) ([]*types.PrincipalInfo, error) {
	var ids []int64
	if spec.assigneePrincipalID != nil {
		ids = append(ids, *spec.assigneePrincipalID)
	}
	if spec.assigneeGroupID != nil {
		members, err := s.userGroupService.ListUserIDsByGroupIDs(ctx, []int64{*spec.assigneeGroupID})
		if err != nil {
			return nil, fmt.Errorf("failed to list members of user group %d: %w", *spec.assigneeGroupID, err)
		}
		ids = append(ids, members...)
	}
	if spec.creator {
		ids = append(ids, t.CreatedBy)
	}
	if spec.commenters {
		acts, err := s.taskActivityStore.ListByKind(ctx, t.ID, enum.TaskActivityKindComment)
		if err != nil {
			return nil, fmt.Errorf("failed to list comments of task %d: %w", t.ID, err)
		}
		for _, a := range acts {
			if a.PrincipalID != nil {
				ids = append(ids, *a.PrincipalID)
			}
		}
	}

	ids = uniqueIDsExcluding(ids, actorID)
	if len(ids) == 0 {
		return nil, nil
	}
	infos, err := s.principalInfoCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load recipients of task %d: %w", t.ID, err)
	}
	out := make([]*types.PrincipalInfo, 0, len(ids))
	for _, id := range ids {
		if info, ok := infos[id]; ok && info != nil && info.Email != "" {
			out = append(out, info)
		}
	}
	return out, nil
}

// uniqueIDsExcluding removes duplicates (keeping order) and the excluded id.
func uniqueIDsExcluding(ids []int64, exclude int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if exclude != 0 && id == exclude {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

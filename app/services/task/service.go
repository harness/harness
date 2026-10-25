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

// Package task implements the domain logic of the Tasks module: state machine,
// deduplication by origin, SLA due dates and the per-task history (task_activities).
package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	maxTitleLength = 256
	// maxTxRetries is the number of attempts of a read-modify-write on optimistic lock conflicts.
	maxTxRetries = 5
	// activityPageSize is the page size used when reading the history.
	activityPageSize = 100
)

// SLAConfig holds the default time-to-resolve per priority. A zero duration means no due date.
type SLAConfig struct {
	P0 time.Duration
	P1 time.Duration
	P2 time.Duration
	P3 time.Duration
}

// Config is the configuration of the task service.
type Config struct {
	SLA SLAConfig
}

// DefaultConfig returns the default configuration: p0 1 day, p1 3 days, p2 14 days, p3 none.
func DefaultConfig() Config {
	const day = 24 * time.Hour
	return Config{SLA: SLAConfig{P0: day, P1: 3 * day, P2: 14 * day, P3: 0}}
}

// DueFor returns the due timestamp (unix millis) for a task of the priority starting at fromMillis,
// or nil if the priority has no SLA.
func (c Config) DueFor(priority enum.TaskPriority, fromMillis int64) *int64 {
	var d time.Duration
	switch priority {
	case enum.TaskPriorityP0:
		d = c.SLA.P0
	case enum.TaskPriorityP1:
		d = c.SLA.P1
	case enum.TaskPriorityP2:
		d = c.SLA.P2
	case enum.TaskPriorityP3:
		d = c.SLA.P3
	}
	if d <= 0 {
		return nil
	}
	due := fromMillis + d.Milliseconds()
	return &due
}

// Service implements the task domain logic.
type Service struct {
	config     Config
	tx         dbtx.Transactor
	tasks      store.TaskStore
	activities store.TaskActivityStore
	links      store.TaskLinkStore
	events     EventReporter
	now        func() int64
}

// NewService creates a new task service.
func NewService(
	config Config,
	tx dbtx.Transactor,
	tasks store.TaskStore,
	activities store.TaskActivityStore,
	links store.TaskLinkStore,
	events EventReporter,
) *Service {
	return &Service{
		config:     config,
		tx:         tx,
		tasks:      tasks,
		activities: activities,
		links:      links,
		events:     events,
		now:        func() int64 { return time.Now().UnixMilli() },
	}
}

// AssigneeInput describes the assignee of a task. All zero values mean "unassigned".
// PrincipalID and GroupID are mutually exclusive, and the agent flag can't be combined with a group.
type AssigneeInput struct {
	PrincipalID *int64 `json:"principal_id,omitempty"`
	GroupID     *int64 `json:"group_id,omitempty"`
	IsAgent     bool   `json:"is_agent"`
}

func (a *AssigneeInput) validate() error {
	if a.PrincipalID != nil && a.GroupID != nil {
		return usererror.BadRequest("A task can't be assigned to a user and a group at the same time")
	}
	if a.IsAgent && a.GroupID != nil {
		return usererror.BadRequest("A task can't be assigned to the agent and a group at the same time")
	}
	return nil
}

func assigneeOf(t *types.Task) AssigneeInput {
	return AssigneeInput{
		PrincipalID: t.AssigneePrincipalID,
		GroupID:     t.AssigneeGroupID,
		IsAgent:     t.AssigneeIsAgent,
	}
}

func sameInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// CreateInput is the input of Service.Create.
type CreateInput struct {
	RepoID      int64
	Title       string
	Description string
	// Origin defaults to manual.
	Origin enum.TaskOrigin
	// OriginRef identifies the source entity (Insights issue, Shield finding). It is used for deduplication
	// of non-manual origins.
	OriginRef string
	// Priority defaults to p2.
	Priority enum.TaskPriority
	// Due overrides the SLA based due date.
	Due      *int64
	Assignee *AssigneeInput
	ParentID *int64
}

// Create creates a task. For non manual origins with an origin reference, at most one active task exists
// per (repo, origin, origin_ref): if there is one it is returned and created is false.
func (s *Service) Create(
	ctx context.Context,
	principalID int64,
	in CreateInput,
) (task *types.Task, created bool, err error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, false, usererror.BadRequest("Task title is required")
	}
	if len(title) > maxTitleLength {
		return nil, false, usererror.BadRequestf("Task title can't be longer than %d characters", maxTitleLength)
	}

	origin := in.Origin
	if origin == "" {
		origin = enum.TaskOriginManual
	}
	if _, ok := origin.Sanitize(); !ok {
		return nil, false, usererror.BadRequestf("Unknown task origin %q", origin)
	}

	priority := in.Priority
	if priority == "" {
		priority = enum.TaskPriorityP2
	}
	if _, ok := priority.Sanitize(); !ok {
		return nil, false, usererror.BadRequestf("Unknown task priority %q", priority)
	}

	if in.Assignee != nil {
		if err = in.Assignee.validate(); err != nil {
			return nil, false, err
		}
	}

	dedup := origin != enum.TaskOriginManual && in.OriginRef != ""

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		if dedup {
			existing, findErr := s.tasks.FindActiveByOrigin(ctx, in.RepoID, origin, in.OriginRef)
			if findErr == nil {
				task, created = existing, false
				return nil
			}
			if !errors.Is(findErr, gitness_store.ErrResourceNotFound) {
				return fmt.Errorf("failed to look up task by origin: %w", findErr)
			}
		}

		if in.ParentID != nil {
			parent, findErr := s.tasks.Find(ctx, *in.ParentID)
			if findErr != nil {
				return fmt.Errorf("failed to find parent task: %w", findErr)
			}
			if parent.RepoID != in.RepoID {
				return usererror.BadRequest("Parent task must belong to the same repository")
			}
		}

		now := s.now()
		due := in.Due
		if due == nil {
			due = s.config.DueFor(priority, now)
		}

		t := &types.Task{
			RepoID:      in.RepoID,
			Title:       title,
			Description: in.Description,
			Origin:      origin,
			OriginRef:   in.OriginRef,
			Priority:    priority,
			State:       enum.TaskStateOpen,
			ParentID:    in.ParentID,
			Due:         due,
			CreatedBy:   principalID,
			Created:     now,
			Updated:     now,
		}
		if in.Assignee != nil {
			t.AssigneePrincipalID = in.Assignee.PrincipalID
			t.AssigneeGroupID = in.Assignee.GroupID
			t.AssigneeIsAgent = in.Assignee.IsAgent
		}

		if createErr := s.tasks.Create(ctx, t); createErr != nil {
			return fmt.Errorf("failed to create task: %w", createErr)
		}

		if actErr := s.addActivity(ctx, t.ID, enum.TaskActivityKindCreated, principalID, createdPayload{
			Title:     t.Title,
			Origin:    t.Origin,
			OriginRef: t.OriginRef,
			Priority:  t.Priority,
			Due:       t.Due,
			Assignee:  assigneeOf(t),
		}); actErr != nil {
			return actErr
		}

		task, created = t, true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if created {
		s.emitCreated(ctx, principalID, task)
	}
	return task, created, nil
}

// UpdateInput is the input of Service.Update. Nil fields are left unchanged.
type UpdateInput struct {
	Title       *string
	Description *string
	Priority    *enum.TaskPriority
	// Due sets the due date (unix millis), ClearDue removes it. If the priority changes and the task
	// has no due date (and Due isn't given) the SLA of the new priority is applied.
	Due      *int64
	ClearDue bool
	// Assignee replaces the assignee. A non-nil pointer to a zero value unassigns the task.
	Assignee *AssigneeInput
}

// Update changes the fields of a task and records the changes in the history.
func (s *Service) Update(
	ctx context.Context,
	principalID int64,
	taskID int64,
	in UpdateInput,
) (*types.Task, error) {
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, usererror.BadRequest("Task title is required")
		}
		if len(title) > maxTitleLength {
			return nil, usererror.BadRequestf("Task title can't be longer than %d characters", maxTitleLength)
		}
		in.Title = &title
	}
	if in.Priority != nil {
		if _, ok := in.Priority.Sanitize(); !ok {
			return nil, usererror.BadRequestf("Unknown task priority %q", *in.Priority)
		}
	}
	if in.Assignee != nil {
		if err := in.Assignee.validate(); err != nil {
			return nil, err
		}
	}
	if in.Due != nil && in.ClearDue {
		return nil, usererror.BadRequest("Due and clear due can't be combined")
	}

	return s.modify(ctx, taskID, func(_ context.Context, t *types.Task) ([]activity, error) {
		var acts []activity
		changes := map[string]fieldChange{}

		if in.Title != nil && *in.Title != t.Title {
			changes["title"] = fieldChange{From: t.Title, To: *in.Title}
			t.Title = *in.Title
		}
		if in.Description != nil && *in.Description != t.Description {
			changes["description"] = fieldChange{From: t.Description, To: *in.Description}
			t.Description = *in.Description
		}

		if in.Priority != nil && *in.Priority != t.Priority {
			acts = append(acts, activity{
				Kind: enum.TaskActivityKindPriorityChange,
				Payload: priorityChangePayload{
					From: t.Priority, To: *in.Priority,
				},
			})
			t.Priority = *in.Priority
			if in.Due == nil && !in.ClearDue && t.Due == nil {
				t.Due = s.config.DueFor(t.Priority, s.now())
				if t.Due != nil {
					changes["due"] = fieldChange{From: nil, To: *t.Due}
				}
			}
		}

		switch {
		case in.Due != nil && !sameInt64Ptr(in.Due, t.Due):
			changes["due"] = fieldChange{From: ptrValue(t.Due), To: *in.Due}
			t.Due = in.Due
		case in.ClearDue && t.Due != nil:
			changes["due"] = fieldChange{From: *t.Due, To: nil}
			t.Due = nil
		}

		if in.Assignee != nil {
			if cur := assigneeOf(t); !sameInt64Ptr(cur.PrincipalID, in.Assignee.PrincipalID) ||
				!sameInt64Ptr(cur.GroupID, in.Assignee.GroupID) || cur.IsAgent != in.Assignee.IsAgent {
				acts = append(acts, activity{
					Kind:    enum.TaskActivityKindAssignment,
					Payload: assignmentPayload{From: cur, To: *in.Assignee},
				})
				t.AssigneePrincipalID = in.Assignee.PrincipalID
				t.AssigneeGroupID = in.Assignee.GroupID
				t.AssigneeIsAgent = in.Assignee.IsAgent
			}
		}

		if len(changes) > 0 {
			acts = append([]activity{{Kind: enum.TaskActivityKindUpdated, Payload: updatedPayload{Changes: changes}}},
				acts...)
		}
		return acts, nil
	}, principalID)
}

// Transition moves a task to a new state following the transition table.
// A reason is required for descartada. Reopening (reabierta) raises the priority one level.
func (s *Service) Transition(
	ctx context.Context,
	principalID int64,
	taskID int64,
	to enum.TaskState,
	reason string,
) (*types.Task, error) {
	if _, ok := to.Sanitize(); !ok {
		return nil, usererror.BadRequestf("Unknown task state %q", to)
	}
	return s.modify(ctx, taskID, func(ctx context.Context, t *types.Task) ([]activity, error) {
		return s.applyTransition(ctx, t, to, reason)
	}, principalID)
}

// Resume leaves a side state (esperando_cuota or bloqueada) back to the state the task had before.
// Without known history the task goes to triage.
func (s *Service) Resume(
	ctx context.Context,
	principalID int64,
	taskID int64,
	reason string,
) (*types.Task, error) {
	return s.modify(ctx, taskID, func(ctx context.Context, t *types.Task) ([]activity, error) {
		if !IsSideState(t.State) {
			return nil, newTransitionError(t.State, "", "Task in state %q is not waiting or blocked", t.State)
		}
		prev, err := s.previousState(ctx, t.ID, t.State)
		if err != nil {
			return nil, err
		}
		if prev == "" {
			prev = enum.TaskStateTriage
		}
		return s.applyTransition(ctx, t, prev, reason)
	}, principalID)
}

func (s *Service) applyTransition(
	ctx context.Context,
	t *types.Task,
	to enum.TaskState,
	reason string,
) ([]activity, error) {
	from := t.State
	if from == to {
		return nil, newTransitionError(from, to, "Task is already in state %q", to)
	}
	if !CanTransition(from, to) {
		return nil, newTransitionError(from, to, "Task can't move from state %q to %q", from, to)
	}
	reason = strings.TrimSpace(reason)
	if to == enum.TaskStateDismissed && reason == "" {
		return nil, newTransitionError(from, to, "A reason is required to dismiss a task")
	}
	if IsSideState(from) && to != enum.TaskStateTriage && to != enum.TaskStateDismissed {
		prev, err := s.previousState(ctx, t.ID, from)
		if err != nil {
			return nil, err
		}
		if prev != to {
			return nil, newTransitionError(from, to,
				"Task in state %q can only go back to its previous state, triage or be dismissed", from)
		}
	}

	now := s.now()
	t.State = to
	t.StateReason = reason

	acts := []activity{{
		Kind:    enum.TaskActivityKindStateChange,
		Payload: stateChangePayload{From: from, To: to, Reason: reason},
	}}

	switch to {
	case enum.TaskStateResolved:
		t.Resolved = &now
	case enum.TaskStateVerified:
		t.Verified = &now
	case enum.TaskStateReopened:
		t.Resolved = nil
		t.Verified = nil
		raised := RaisePriority(t.Priority)
		if raised != t.Priority {
			acts = append(acts, activity{
				Kind:    enum.TaskActivityKindPriorityChange,
				Payload: priorityChangePayload{From: t.Priority, To: raised, Reason: "reopened"},
			})
			t.Priority = raised
		}
		// the SLA restarts when the task is reopened
		t.Due = s.config.DueFor(t.Priority, now)
	}
	return acts, nil
}

// previousState returns the state the task was in before it entered the current (side) state.
func (s *Service) previousState(
	ctx context.Context,
	taskID int64,
	current enum.TaskState,
) (enum.TaskState, error) {
	var prev enum.TaskState
	for page := 1; ; page++ {
		list, err := s.activities.List(ctx, taskID, types.Pagination{Page: page, Size: activityPageSize})
		if err != nil {
			return "", fmt.Errorf("failed to list task activities: %w", err)
		}
		for _, a := range list {
			if a.Kind != enum.TaskActivityKindStateChange {
				continue
			}
			var p stateChangePayload
			if json.Unmarshal(a.Payload, &p) == nil && p.To == current {
				prev = p.From
			}
		}
		if len(list) < activityPageSize {
			return prev, nil
		}
	}
}

// Link links a task to an external entity. It's idempotent and only records history on a new link.
func (s *Service) Link(
	ctx context.Context,
	principalID int64,
	taskID int64,
	kind enum.TaskLinkKind,
	refID string,
) error {
	if err := validateLink(kind, refID); err != nil {
		return err
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.tasks.Find(ctx, taskID); err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}
		exists, err := s.hasLink(ctx, taskID, kind, refID)
		if err != nil || exists {
			return err
		}
		if err = s.links.Create(ctx, &types.TaskLink{
			TaskID: taskID, Kind: kind, RefID: refID, Created: s.now(),
		}); err != nil {
			return fmt.Errorf("failed to create task link: %w", err)
		}
		return s.addActivity(ctx, taskID, enum.TaskActivityKindPRLink, principalID,
			linkPayload{Action: "linked", Kind: kind, RefID: refID})
	})
}

// Unlink removes a link. It's idempotent and only records history if the link existed.
func (s *Service) Unlink(
	ctx context.Context,
	principalID int64,
	taskID int64,
	kind enum.TaskLinkKind,
	refID string,
) error {
	if err := validateLink(kind, refID); err != nil {
		return err
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.tasks.Find(ctx, taskID); err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}
		exists, err := s.hasLink(ctx, taskID, kind, refID)
		if err != nil || !exists {
			return err
		}
		if err = s.links.Delete(ctx, taskID, kind, refID); err != nil {
			return fmt.Errorf("failed to delete task link: %w", err)
		}
		return s.addActivity(ctx, taskID, enum.TaskActivityKindPRLink, principalID,
			linkPayload{Action: "unlinked", Kind: kind, RefID: refID})
	})
}

func validateLink(kind enum.TaskLinkKind, refID string) error {
	if _, ok := kind.Sanitize(); !ok || kind == "" {
		return usererror.BadRequestf("Unknown task link kind %q", kind)
	}
	if strings.TrimSpace(refID) == "" {
		return usererror.BadRequest("Task link reference is required")
	}
	return nil
}

func (s *Service) hasLink(ctx context.Context, taskID int64, kind enum.TaskLinkKind, refID string) (bool, error) {
	list, err := s.links.ListByTask(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to list task links: %w", err)
	}
	for _, l := range list {
		if l.Kind == kind && l.RefID == refID {
			return true, nil
		}
	}
	return false, nil
}

// AddComment adds a comment to the history of the task.
func (s *Service) AddComment(ctx context.Context, principalID int64, taskID int64, text string) error {
	if strings.TrimSpace(text) == "" {
		return usererror.BadRequest("Comment text is required")
	}
	var t *types.Task
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var findErr error
		if t, findErr = s.tasks.Find(ctx, taskID); findErr != nil {
			return fmt.Errorf("failed to find task: %w", findErr)
		}
		return s.addActivity(ctx, taskID, enum.TaskActivityKindComment, principalID, commentPayload{Text: text})
	})
	if err != nil {
		return err
	}
	s.emitCommentCreated(ctx, principalID, t)
	return nil
}

// DecisionInput is the outcome of a model decision about a task.
type DecisionInput struct {
	QuestionID    string             `json:"question_id"`
	Value         string             `json:"value"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	// Model is the exact model build that answered.
	Model string `json:"model"`
	// Fallback is true if the decision came from the fallback model or deterministic rules.
	Fallback bool `json:"fallback"`
}

// RecordDecision adds a model decision to the history. principalID may be 0 for the system.
func (s *Service) RecordDecision(ctx context.Context, principalID int64, taskID int64, in DecisionInput) error {
	if strings.TrimSpace(in.QuestionID) == "" {
		return usererror.BadRequest("Decision question id is required")
	}
	if in.Confidence < 0 || in.Confidence > 1 {
		return usererror.BadRequest("Decision confidence must be between 0 and 1")
	}
	return s.recordOnly(ctx, taskID, enum.TaskActivityKindDecision, principalID, in)
}

// AgentRunInput describes a run of the agent on a task.
type AgentRunInput struct {
	Model        string `json:"model"`
	RequestsUsed int    `json:"requests_used"`
	Outcome      string `json:"outcome"`
	// AITaskRef optionally references the AI task of the run.
	AITaskRef string `json:"ai_task_ref,omitempty"`
}

// RecordAgentRun adds an agent run to the history. principalID may be 0 for the system.
func (s *Service) RecordAgentRun(ctx context.Context, principalID int64, taskID int64, in AgentRunInput) error {
	if strings.TrimSpace(in.Outcome) == "" {
		return usererror.BadRequest("Agent run outcome is required")
	}
	if in.RequestsUsed < 0 {
		return usererror.BadRequest("Agent run requests used can't be negative")
	}
	return s.recordOnly(ctx, taskID, enum.TaskActivityKindAgentRun, principalID, in)
}

// recordOnly writes a history entry without changing the task row.
func (s *Service) recordOnly(
	ctx context.Context,
	taskID int64,
	kind enum.TaskActivityKind,
	principalID int64,
	payload any,
) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if _, err := s.tasks.Find(ctx, taskID); err != nil {
			return fmt.Errorf("failed to find task: %w", err)
		}
		return s.addActivity(ctx, taskID, kind, principalID, payload)
	})
}

type activity struct {
	Kind    enum.TaskActivityKind
	Payload any
}

// modify runs a read-modify-write of a task and its history entries in a transaction,
// retrying on optimistic lock conflicts. If fn returns no activities nothing is written.
func (s *Service) modify(
	ctx context.Context,
	taskID int64,
	fn func(ctx context.Context, t *types.Task) ([]activity, error),
	principalID int64,
) (*types.Task, error) {
	var result *types.Task
	var written []activity
	var err error
	for try := 0; try < maxTxRetries; try++ {
		result, written = nil, nil
		err = s.tx.WithTx(ctx, func(ctx context.Context) error {
			t, findErr := s.tasks.Find(ctx, taskID)
			if findErr != nil {
				return fmt.Errorf("failed to find task: %w", findErr)
			}
			acts, fnErr := fn(ctx, t)
			if fnErr != nil {
				return fnErr
			}
			if len(acts) == 0 {
				result = t
				return nil
			}
			t.Updated = s.now()
			if updErr := s.tasks.Update(ctx, t); updErr != nil {
				return fmt.Errorf("failed to update task: %w", updErr)
			}
			for _, a := range acts {
				if actErr := s.addActivity(ctx, t.ID, a.Kind, principalID, a.Payload); actErr != nil {
					return actErr
				}
			}
			result, written = t, acts
			return nil
		})
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	s.emitChanges(ctx, principalID, result, written)
	return result, nil
}

func (s *Service) addActivity(
	ctx context.Context,
	taskID int64,
	kind enum.TaskActivityKind,
	principalID int64,
	payload any,
) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task activity payload: %w", err)
	}
	a := &types.TaskActivity{
		TaskID:  taskID,
		Kind:    kind,
		Payload: raw,
		Created: s.now(),
	}
	if principalID > 0 {
		a.PrincipalID = &principalID
	}
	if err = s.activities.Create(ctx, a); err != nil {
		return fmt.Errorf("failed to create task activity: %w", err)
	}
	return nil
}

func ptrValue(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

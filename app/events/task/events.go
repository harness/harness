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

package events

import (
	"context"

	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

const (
	// CreatedEvent is emitted after a task was created.
	CreatedEvent events.EventType = "created"
	// StateChangedEvent is emitted after a task changed its state.
	StateChangedEvent events.EventType = "state_changed"
	// AssignedEvent is emitted after the assignee of a task changed.
	AssignedEvent events.EventType = "assigned"
	// PriorityChangedEvent is emitted after the priority of a task changed.
	PriorityChangedEvent events.EventType = "priority_changed"
	// ResolvedEvent is emitted after a task moved to the resolved state.
	ResolvedEvent events.EventType = "resolved"
	// VerifiedEvent is emitted after a task moved to the verified state.
	VerifiedEvent events.EventType = "verified"
	// ReopenedEvent is emitted after a task moved to the reopened state.
	ReopenedEvent events.EventType = "reopened"
	// CommentCreatedEvent is emitted after a comment was added to a task.
	CommentCreatedEvent events.EventType = "comment_created"
	// AgentNeedsHelpEvent is emitted after an agent run on a task reported that it needs a human.
	AgentNeedsHelpEvent events.EventType = "agent_needs_help"
)

// Base is the common part of all task event payloads.
type Base struct {
	TaskID      int64 `json:"task_id"`
	RepoID      int64 `json:"repo_id"`
	Number      int64 `json:"number"`
	PrincipalID int64 `json:"principal_id"`
}

type (
	CreatedPayload struct {
		Base
	}

	StateChangedPayload struct {
		Base
		From   string `json:"from"`
		To     string `json:"to"`
		Reason string `json:"reason,omitempty"`
	}

	AssignedPayload struct {
		Base
		AssigneePrincipalID *int64 `json:"assignee_principal_id,omitempty"`
		AssigneeGroupID     *int64 `json:"assignee_group_id,omitempty"`
		AssigneeIsAgent     bool   `json:"assignee_is_agent"`
	}

	PriorityChangedPayload struct {
		Base
		From string `json:"from"`
		To   string `json:"to"`
	}

	ResolvedPayload struct {
		Base
	}

	VerifiedPayload struct {
		Base
	}

	ReopenedPayload struct {
		Base
		Reason string `json:"reason,omitempty"`
	}

	CommentCreatedPayload struct {
		Base
	}

	AgentNeedsHelpPayload struct {
		Base
		Model string `json:"model,omitempty"`
	}
)

// Created reports a task created event.
func (r *Reporter) Created(ctx context.Context, payload *CreatedPayload) {
	r.send(ctx, CreatedEvent, payload, payload != nil)
}

// StateChanged reports a task state changed event.
func (r *Reporter) StateChanged(ctx context.Context, payload *StateChangedPayload) {
	r.send(ctx, StateChangedEvent, payload, payload != nil)
}

// Assigned reports a task assigned event.
func (r *Reporter) Assigned(ctx context.Context, payload *AssignedPayload) {
	r.send(ctx, AssignedEvent, payload, payload != nil)
}

// PriorityChanged reports a task priority changed event.
func (r *Reporter) PriorityChanged(ctx context.Context, payload *PriorityChangedPayload) {
	r.send(ctx, PriorityChangedEvent, payload, payload != nil)
}

// Resolved reports a task resolved event.
func (r *Reporter) Resolved(ctx context.Context, payload *ResolvedPayload) {
	r.send(ctx, ResolvedEvent, payload, payload != nil)
}

// Verified reports a task verified event.
func (r *Reporter) Verified(ctx context.Context, payload *VerifiedPayload) {
	r.send(ctx, VerifiedEvent, payload, payload != nil)
}

// Reopened reports a task reopened event.
func (r *Reporter) Reopened(ctx context.Context, payload *ReopenedPayload) {
	r.send(ctx, ReopenedEvent, payload, payload != nil)
}

// CommentCreated reports a task comment created event.
func (r *Reporter) CommentCreated(ctx context.Context, payload *CommentCreatedPayload) {
	r.send(ctx, CommentCreatedEvent, payload, payload != nil)
}

// AgentNeedsHelp reports that an agent run on a task needs a human.
func (r *Reporter) AgentNeedsHelp(ctx context.Context, payload *AgentNeedsHelpPayload) {
	r.send(ctx, AgentNeedsHelpEvent, payload, payload != nil)
}

func (r *Reporter) send(ctx context.Context, event events.EventType, payload any, ok bool) {
	if !ok {
		return
	}
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, event, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send task %s event", event)
		return
	}
	log.Ctx(ctx).Debug().Msgf("reported task %s event with id '%s'", event, eventID)
}

func (r *Reader) RegisterCreated(fn events.HandlerFunc[*CreatedPayload], opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn, opts...)
}

func (r *Reader) RegisterStateChanged(
	fn events.HandlerFunc[*StateChangedPayload], opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, StateChangedEvent, fn, opts...)
}

func (r *Reader) RegisterAssigned(fn events.HandlerFunc[*AssignedPayload], opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, AssignedEvent, fn, opts...)
}

func (r *Reader) RegisterPriorityChanged(
	fn events.HandlerFunc[*PriorityChangedPayload], opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, PriorityChangedEvent, fn, opts...)
}

func (r *Reader) RegisterResolved(fn events.HandlerFunc[*ResolvedPayload], opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, ResolvedEvent, fn, opts...)
}

func (r *Reader) RegisterVerified(fn events.HandlerFunc[*VerifiedPayload], opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, VerifiedEvent, fn, opts...)
}

func (r *Reader) RegisterReopened(fn events.HandlerFunc[*ReopenedPayload], opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, ReopenedEvent, fn, opts...)
}

func (r *Reader) RegisterCommentCreated(
	fn events.HandlerFunc[*CommentCreatedPayload], opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, CommentCreatedEvent, fn, opts...)
}

func (r *Reader) RegisterAgentNeedsHelp(
	fn events.HandlerFunc[*AgentNeedsHelpPayload], opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, AgentNeedsHelpEvent, fn, opts...)
}

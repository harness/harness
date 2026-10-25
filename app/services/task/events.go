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

	taskevents "github.com/harness/gitness/app/events/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// EventReporter reports the task events. *taskevents.Reporter implements it.
// A nil reporter disables event reporting.
type EventReporter interface {
	Created(ctx context.Context, payload *taskevents.CreatedPayload)
	StateChanged(ctx context.Context, payload *taskevents.StateChangedPayload)
	Assigned(ctx context.Context, payload *taskevents.AssignedPayload)
	PriorityChanged(ctx context.Context, payload *taskevents.PriorityChangedPayload)
	Resolved(ctx context.Context, payload *taskevents.ResolvedPayload)
	Verified(ctx context.Context, payload *taskevents.VerifiedPayload)
	Reopened(ctx context.Context, payload *taskevents.ReopenedPayload)
	CommentCreated(ctx context.Context, payload *taskevents.CommentCreatedPayload)
}

func baseOf(principalID int64, t *types.Task) taskevents.Base {
	return taskevents.Base{TaskID: t.ID, RepoID: t.RepoID, Number: t.Number, PrincipalID: principalID}
}

func (s *Service) emitCreated(ctx context.Context, principalID int64, t *types.Task) {
	if s.events == nil || t == nil {
		return
	}
	s.events.Created(ctx, &taskevents.CreatedPayload{Base: baseOf(principalID, t)})
}

func (s *Service) emitCommentCreated(ctx context.Context, principalID int64, t *types.Task) {
	if s.events == nil || t == nil {
		return
	}
	s.events.CommentCreated(ctx, &taskevents.CommentCreatedPayload{Base: baseOf(principalID, t)})
}

// emitChanges reports the events of the history entries written for a task by a committed change.
func (s *Service) emitChanges(ctx context.Context, principalID int64, t *types.Task, acts []activity) {
	if s.events == nil || t == nil {
		return
	}
	base := baseOf(principalID, t)
	for _, a := range acts {
		switch p := a.Payload.(type) {
		case stateChangePayload:
			s.events.StateChanged(ctx, &taskevents.StateChangedPayload{
				Base: base, From: string(p.From), To: string(p.To), Reason: p.Reason,
			})
			switch p.To {
			case enum.TaskStateResolved:
				s.events.Resolved(ctx, &taskevents.ResolvedPayload{Base: base})
			case enum.TaskStateVerified:
				s.events.Verified(ctx, &taskevents.VerifiedPayload{Base: base})
			case enum.TaskStateReopened:
				s.events.Reopened(ctx, &taskevents.ReopenedPayload{Base: base, Reason: p.Reason})
			}
		case assignmentPayload:
			s.events.Assigned(ctx, &taskevents.AssignedPayload{
				Base:                base,
				AssigneePrincipalID: p.To.PrincipalID,
				AssigneeGroupID:     p.To.GroupID,
				AssigneeIsAgent:     p.To.IsAgent,
			})
		case priorityChangePayload:
			s.events.PriorityChanged(ctx, &taskevents.PriorityChangedPayload{
				Base: base, From: string(p.From), To: string(p.To),
			})
		}
	}
}

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
	"errors"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/types/enum"
)

// ErrInvalidTransition is matched (errors.Is) by every TransitionError.
var ErrInvalidTransition = errors.New("invalid task state transition")

// TransitionError is returned when a requested state change is not allowed.
// It unwraps to a usererror (HTTP 400) so API layers translate it automatically.
type TransitionError struct {
	From    enum.TaskState
	To      enum.TaskState
	Message string
}

func (e *TransitionError) Error() string { return e.Message }

// Is makes errors.Is(err, ErrInvalidTransition) true.
func (e *TransitionError) Is(target error) bool { return target == ErrInvalidTransition }

// Unwrap returns the user facing error.
func (e *TransitionError) Unwrap() error { return usererror.BadRequest(e.Message) }

func newTransitionError(from, to enum.TaskState, format string, args ...any) *TransitionError {
	return &TransitionError{From: from, To: to, Message: fmt.Sprintf(format, args...)}
}

// transitions is the table of allowed state changes (design doc section 6.2).
//
// Main flow: abierta -> triage -> en_curso -> en_revision -> resuelta -> verificada.
//
// Side states esperando_cuota and bloqueada can be entered from any open working state.
// They can only be left towards triage, descartada or the state the task was in before
// entering them (the table lists every possible "previous" state, Service.Transition
// restricts it to the real one; Service.Resume does it in one call).
//
// descartada is terminal. reabierta is only reachable from resuelta/verificada and
// behaves like a working state afterwards.
var transitions = map[enum.TaskState][]enum.TaskState{
	enum.TaskStateOpen: {
		enum.TaskStateTriage, enum.TaskStateInProgress,
		enum.TaskStateWaitingQuota, enum.TaskStateBlocked, enum.TaskStateDismissed,
	},
	enum.TaskStateTriage: {
		enum.TaskStateInProgress,
		enum.TaskStateWaitingQuota, enum.TaskStateBlocked, enum.TaskStateDismissed,
	},
	enum.TaskStateInProgress: {
		enum.TaskStateTriage, enum.TaskStateInReview,
		enum.TaskStateWaitingQuota, enum.TaskStateBlocked, enum.TaskStateDismissed,
	},
	enum.TaskStateInReview: {
		enum.TaskStateInProgress, enum.TaskStateResolved,
		enum.TaskStateWaitingQuota, enum.TaskStateBlocked, enum.TaskStateDismissed,
	},
	enum.TaskStateResolved: {
		enum.TaskStateVerified, enum.TaskStateReopened, enum.TaskStateDismissed,
	},
	enum.TaskStateVerified: {
		enum.TaskStateReopened,
	},
	enum.TaskStateReopened: {
		enum.TaskStateTriage, enum.TaskStateInProgress,
		enum.TaskStateWaitingQuota, enum.TaskStateBlocked, enum.TaskStateDismissed,
	},
	enum.TaskStateWaitingQuota: {
		enum.TaskStateTriage, enum.TaskStateDismissed,
		enum.TaskStateOpen, enum.TaskStateInProgress, enum.TaskStateInReview, enum.TaskStateReopened,
	},
	enum.TaskStateBlocked: {
		enum.TaskStateTriage, enum.TaskStateDismissed,
		enum.TaskStateOpen, enum.TaskStateInProgress, enum.TaskStateInReview, enum.TaskStateReopened,
	},
	enum.TaskStateDismissed: {},
}

// IsSideState reports whether the state is a parking state that remembers where the task came from.
func IsSideState(s enum.TaskState) bool {
	return s == enum.TaskStateWaitingQuota || s == enum.TaskStateBlocked
}

// CanTransition reports whether the table allows going from one state to the other.
// For side states it doesn't check the "previous state" restriction, Service.Transition does.
func CanTransition(from, to enum.TaskState) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// IsActive reports whether a task in the state counts for origin deduplication.
func IsActive(s enum.TaskState) bool {
	return s != enum.TaskStateDismissed && s != enum.TaskStateVerified
}

// RaisePriority returns the priority one level higher (p3 -> p2 -> p1 -> p0, p0 stays p0).
func RaisePriority(p enum.TaskPriority) enum.TaskPriority {
	switch p {
	case enum.TaskPriorityP3:
		return enum.TaskPriorityP2
	case enum.TaskPriorityP2:
		return enum.TaskPriorityP1
	case enum.TaskPriorityP1, enum.TaskPriorityP0:
		return enum.TaskPriorityP0
	}
	return p
}

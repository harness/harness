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

package enum

// TaskState defines the lifecycle state of a task.
type TaskState string

func (TaskState) Enum() []any                   { return toInterfaceSlice(taskStates) }
func (s TaskState) Sanitize() (TaskState, bool) { return Sanitize(s, GetAllTaskStates) }
func GetAllTaskStates() ([]TaskState, TaskState) {
	return taskStates, TaskStateOpen
}

// TaskState enumeration.
const (
	TaskStateOpen         TaskState = "abierta"
	TaskStateTriage       TaskState = "triage"
	TaskStateInProgress   TaskState = "en_curso"
	TaskStateInReview     TaskState = "en_revision"
	TaskStateResolved     TaskState = "resuelta"
	TaskStateVerified     TaskState = "verificada"
	TaskStateWaitingQuota TaskState = "esperando_cuota"
	TaskStateBlocked      TaskState = "bloqueada"
	TaskStateDismissed    TaskState = "descartada"
	TaskStateReopened     TaskState = "reabierta"
)

var taskStates = sortEnum([]TaskState{
	TaskStateOpen,
	TaskStateTriage,
	TaskStateInProgress,
	TaskStateInReview,
	TaskStateResolved,
	TaskStateVerified,
	TaskStateWaitingQuota,
	TaskStateBlocked,
	TaskStateDismissed,
	TaskStateReopened,
})

// TaskOrigin defines where a task originated from.
type TaskOrigin string

func (TaskOrigin) Enum() []any                    { return toInterfaceSlice(taskOrigins) }
func (o TaskOrigin) Sanitize() (TaskOrigin, bool) { return Sanitize(o, GetAllTaskOrigins) }
func GetAllTaskOrigins() ([]TaskOrigin, TaskOrigin) {
	return taskOrigins, TaskOriginManual
}

// TaskOrigin enumeration.
const (
	TaskOriginInsights TaskOrigin = "insights"
	TaskOriginShield   TaskOrigin = "shield"
	TaskOriginManual   TaskOrigin = "manual"
)

var taskOrigins = sortEnum([]TaskOrigin{
	TaskOriginInsights,
	TaskOriginShield,
	TaskOriginManual,
})

// TaskPriority defines the priority of a task.
type TaskPriority string

func (TaskPriority) Enum() []any                      { return toInterfaceSlice(taskPriorities) }
func (p TaskPriority) Sanitize() (TaskPriority, bool) { return Sanitize(p, GetAllTaskPriorities) }
func GetAllTaskPriorities() ([]TaskPriority, TaskPriority) {
	return taskPriorities, TaskPriorityP2
}

// TaskPriority enumeration.
const (
	TaskPriorityP0 TaskPriority = "p0"
	TaskPriorityP1 TaskPriority = "p1"
	TaskPriorityP2 TaskPriority = "p2"
	TaskPriorityP3 TaskPriority = "p3"
)

var taskPriorities = sortEnum([]TaskPriority{
	TaskPriorityP0,
	TaskPriorityP1,
	TaskPriorityP2,
	TaskPriorityP3,
})

// TaskActivityKind defines the kind of an entry in the task timeline.
type TaskActivityKind string

func (TaskActivityKind) Enum() []any { return toInterfaceSlice(taskActivityKinds) }
func (k TaskActivityKind) Sanitize() (TaskActivityKind, bool) {
	return Sanitize(k, GetAllTaskActivityKinds)
}
func GetAllTaskActivityKinds() ([]TaskActivityKind, TaskActivityKind) {
	return taskActivityKinds, ""
}

// TaskActivityKind enumeration.
const (
	TaskActivityKindCreated        TaskActivityKind = "created"
	TaskActivityKindStateChange    TaskActivityKind = "state_change"
	TaskActivityKindDecision       TaskActivityKind = "decision"
	TaskActivityKindAgentRun       TaskActivityKind = "agent_run"
	TaskActivityKindPRLink         TaskActivityKind = "pr_link"
	TaskActivityKindComment        TaskActivityKind = "comment"
	TaskActivityKindAssignment     TaskActivityKind = "assignment"
	TaskActivityKindPriorityChange TaskActivityKind = "priority_change"
	TaskActivityKindUpdated        TaskActivityKind = "updated"
)

var taskActivityKinds = sortEnum([]TaskActivityKind{
	TaskActivityKindCreated,
	TaskActivityKindStateChange,
	TaskActivityKindDecision,
	TaskActivityKindAgentRun,
	TaskActivityKindPRLink,
	TaskActivityKindComment,
	TaskActivityKindAssignment,
	TaskActivityKindPriorityChange,
	TaskActivityKindUpdated,
})

// TaskLinkKind defines the kind of entity a task is linked to.
type TaskLinkKind string

func (TaskLinkKind) Enum() []any                      { return toInterfaceSlice(taskLinkKinds) }
func (k TaskLinkKind) Sanitize() (TaskLinkKind, bool) { return Sanitize(k, GetAllTaskLinkKinds) }
func GetAllTaskLinkKinds() ([]TaskLinkKind, TaskLinkKind) {
	return taskLinkKinds, ""
}

// TaskLinkKind enumeration.
const (
	TaskLinkKindPullReq       TaskLinkKind = "pullreq"
	TaskLinkKindAITask        TaskLinkKind = "ai_task"
	TaskLinkKindInsightsIssue TaskLinkKind = "insights_issue"
	TaskLinkKindShieldFinding TaskLinkKind = "shield_finding"
)

var taskLinkKinds = sortEnum([]TaskLinkKind{
	TaskLinkKindPullReq,
	TaskLinkKindAITask,
	TaskLinkKindInsightsIssue,
	TaskLinkKindShieldFinding,
})

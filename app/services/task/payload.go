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

import "github.com/harness/gitness/types/enum"

// JSON payloads stored in task_activities.task_activity_payload, per activity kind.

// createdPayload is the payload of enum.TaskActivityKindCreated.
type createdPayload struct {
	Title     string            `json:"title"`
	Origin    enum.TaskOrigin   `json:"origin"`
	OriginRef string            `json:"origin_ref,omitempty"`
	Priority  enum.TaskPriority `json:"priority"`
	Due       *int64            `json:"due,omitempty"`
	Assignee  AssigneeInput     `json:"assignee"`
}

// stateChangePayload is the payload of enum.TaskActivityKindStateChange.
type stateChangePayload struct {
	From   enum.TaskState `json:"from"`
	To     enum.TaskState `json:"to"`
	Reason string         `json:"reason,omitempty"`
}

type priorityChangePayload struct {
	From   enum.TaskPriority `json:"from"`
	To     enum.TaskPriority `json:"to"`
	Reason string            `json:"reason,omitempty"`
}

type assignmentPayload struct {
	From AssigneeInput `json:"from"`
	To   AssigneeInput `json:"to"`
}

type fieldChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

type updatedPayload struct {
	Changes map[string]fieldChange `json:"changes"`
}

type linkPayload struct {
	Action string            `json:"action"` // linked or unlinked
	Kind   enum.TaskLinkKind `json:"kind"`
	RefID  string            `json:"ref_id"`
}

type commentPayload struct {
	Text string `json:"text"`
}

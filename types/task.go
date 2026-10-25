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

package types

import (
	"encoding/json"

	"github.com/harness/gitness/types/enum"
)

// Task represents a unit of tracked work in a repository.
type Task struct {
	ID          int64             `json:"-"`
	Version     int64             `json:"-"`
	RepoID      int64             `json:"repo_id"`
	Number      int64             `json:"number"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Origin      enum.TaskOrigin   `json:"origin"`
	OriginRef   string            `json:"origin_ref"`
	Priority    enum.TaskPriority `json:"priority"`
	State       enum.TaskState    `json:"state"`
	StateReason string            `json:"state_reason,omitempty"`
	ParentID    *int64            `json:"parent_id,omitempty"`
	Due         *int64            `json:"due,omitempty"`
	CreatedBy   int64             `json:"created_by"`
	Created     int64             `json:"created"`
	Updated     int64             `json:"updated"`
	Resolved    *int64            `json:"resolved,omitempty"`
	Verified    *int64            `json:"verified,omitempty"`

	AssigneePrincipalID *int64 `json:"assignee_principal_id,omitempty"`
	AssigneeGroupID     *int64 `json:"assignee_group_id,omitempty"`
	AssigneeIsAgent     bool   `json:"assignee_is_agent"`
}

// TaskFilter stores task list query parameters.
type TaskFilter struct {
	Page            int                 `json:"page"`
	Size            int                 `json:"size"`
	States          []enum.TaskState    `json:"state"`
	Priorities      []enum.TaskPriority `json:"priority"`
	Origins         []enum.TaskOrigin   `json:"origin"`
	AssigneeIDs     []int64             `json:"assignee_id"`
	AssigneeGroupID *int64              `json:"assignee_group_id"`
	IsAgent         *bool               `json:"is_agent"`
	LabelIDs        []int64             `json:"label_id"`
	ParentID        *int64              `json:"parent_id"`
	Query           string              `json:"query"`
}

// TaskActivity is an entry of the task timeline.
type TaskActivity struct {
	ID          int64                 `json:"id"`
	TaskID      int64                 `json:"task_id"`
	Kind        enum.TaskActivityKind `json:"kind"`
	Payload     json.RawMessage       `json:"payload"`
	PrincipalID *int64                `json:"principal_id,omitempty"`
	Created     int64                 `json:"created"`
}

// TaskLink links a task to an external entity (pull request, AI task, Insights issue, Shield finding).
type TaskLink struct {
	TaskID  int64             `json:"task_id"`
	Kind    enum.TaskLinkKind `json:"kind"`
	RefID   string            `json:"ref_id"`
	Created int64             `json:"created"`
}

// LabelTaskAssignmentInfo describes a label (and value) assigned to a task.
type LabelTaskAssignmentInfo struct {
	TaskID     int64            `json:"-"`
	LabelID    int64            `json:"id"`
	LabelKey   string           `json:"key"`
	LabelColor enum.LabelColor  `json:"color,omitempty"`
	LabelScope int64            `json:"scope"`
	ValueCount int64            `json:"value_count"`
	ValueID    *int64           `json:"value_id,omitempty"`
	Value      *string          `json:"value,omitempty"`
	ValueColor *enum.LabelColor `json:"value_color,omitempty"`
}

// TaskLabel is a label assignment on a task.
type TaskLabel struct {
	TaskID    int64  `json:"task_id"`
	LabelID   int64  `json:"label_id"`
	ValueID   *int64 `json:"value_id,omitempty"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
	CreatedBy int64  `json:"created_by"`
	UpdatedBy int64  `json:"updated_by"`
}

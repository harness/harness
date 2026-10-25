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

package store

import (
	"context"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type (
	// TaskStore defines the task data storage.
	TaskStore interface {
		// Create creates a new task. It allocates the next per-repo task number
		// (repositories.repo_task_seq) and sets ID and Number on the provided task.
		// Run it inside a transaction to avoid gaps in the numbering if the insert fails.
		Create(ctx context.Context, task *types.Task) error

		// Find finds the task by id.
		Find(ctx context.Context, id int64) (*types.Task, error)

		// FindByNumber finds the task by repo id and per-repo task number.
		FindByNumber(ctx context.Context, repoID, number int64) (*types.Task, error)

		// Update updates the task using optimistic locking on its version.
		// It returns store.ErrVersionConflict if the version doesn't match.
		Update(ctx context.Context, task *types.Task) error

		// List lists the tasks of a repository matching the filter.
		List(ctx context.Context, repoID int64, filter *types.TaskFilter) ([]*types.Task, error)

		// Count counts the tasks of a repository matching the filter.
		Count(ctx context.Context, repoID int64, filter *types.TaskFilter) (int64, error)
	}

	// TaskActivityStore defines the task timeline storage.
	TaskActivityStore interface {
		// Create creates a new task activity.
		Create(ctx context.Context, activity *types.TaskActivity) error

		// List lists the activities of a task ordered by creation time.
		List(ctx context.Context, taskID int64, pagination types.Pagination) ([]*types.TaskActivity, error)
	}

	// TaskLinkStore defines the storage of links between tasks and other entities.
	TaskLinkStore interface {
		// Create creates a link. It is idempotent.
		Create(ctx context.Context, link *types.TaskLink) error

		// Delete deletes a link.
		Delete(ctx context.Context, taskID int64, kind enum.TaskLinkKind, refID string) error

		// ListByTask lists all links of a task.
		ListByTask(ctx context.Context, taskID int64) ([]*types.TaskLink, error)

		// FindTasksByRef finds the tasks linked to the referenced entity.
		FindTasksByRef(ctx context.Context, kind enum.TaskLinkKind, refID string) ([]*types.Task, error)
	}

	// TaskLabelStore defines the storage of task label assignments.
	TaskLabelStore interface {
		// Assign assigns a label to a task (upsert).
		Assign(ctx context.Context, label *types.TaskLabel) error

		// Unassign removes a label from a task. It reports whether an assignment was removed.
		Unassign(ctx context.Context, taskID, labelID int64) (bool, error)

		// ListAssigned lists labels assigned to a task.
		ListAssigned(ctx context.Context, taskID int64) (map[int64]*types.LabelAssignment, error)

		// FindByLabelID finds the label assignment of a task.
		FindByLabelID(ctx context.Context, taskID, labelID int64) (*types.TaskLabel, error)

		// ListAssignedByTaskIDs lists labels assigned to the specified tasks.
		ListAssignedByTaskIDs(ctx context.Context, taskIDs []int64) (map[int64][]*types.LabelTaskAssignmentInfo, error)
	}
)

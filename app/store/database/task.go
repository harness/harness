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

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.TaskStore = (*TaskStore)(nil)

// NewTaskStore returns a new TaskStore.
func NewTaskStore(db *sqlx.DB) *TaskStore {
	return &TaskStore{db: db}
}

// TaskStore implements store.TaskStore backed by a relational database.
type TaskStore struct {
	db *sqlx.DB
}

type task struct {
	ID                  int64             `db:"task_id"`
	RepoID              int64             `db:"task_repo_id"`
	Number              int64             `db:"task_number"`
	Title               string            `db:"task_title"`
	Description         string            `db:"task_description"`
	Origin              enum.TaskOrigin   `db:"task_origin"`
	OriginRef           string            `db:"task_origin_ref"`
	Priority            enum.TaskPriority `db:"task_priority"`
	State               enum.TaskState    `db:"task_state"`
	StateReason         string            `db:"task_state_reason"`
	AssigneePrincipalID null.Int          `db:"task_assignee_principal_id"`
	AssigneeGroupID     null.Int          `db:"task_assignee_group_id"`
	AssigneeIsAgent     bool              `db:"task_assignee_is_agent"`
	Due                 null.Int          `db:"task_due"`
	ParentID            null.Int          `db:"task_parent_id"`
	CreatedBy           int64             `db:"task_created_by"`
	Created             int64             `db:"task_created"`
	Updated             int64             `db:"task_updated"`
	Resolved            null.Int          `db:"task_resolved"`
	Verified            null.Int          `db:"task_verified"`
	Version             int64             `db:"task_version"`
}

const (
	taskColumns = `
		 task_id
		,task_repo_id
		,task_number
		,task_title
		,task_description
		,task_origin
		,task_origin_ref
		,task_priority
		,task_state
		,task_state_reason
		,task_assignee_principal_id
		,task_assignee_group_id
		,task_assignee_is_agent
		,task_due
		,task_parent_id
		,task_created_by
		,task_created
		,task_updated
		,task_resolved
		,task_verified
		,task_version`

	taskSelectBase = `SELECT` + taskColumns + `
		FROM tasks`
)

// Create allocates the next task number of the repository and inserts the task.
func (s *TaskStore) Create(ctx context.Context, t *types.Task) error {
	const sqlSeq = `
		UPDATE repositories
		SET repo_task_seq = repo_task_seq + 1
		WHERE repo_id = $1
		RETURNING repo_task_seq`

	const sqlInsert = `
		INSERT INTO tasks (
			 task_repo_id
			,task_number
			,task_title
			,task_description
			,task_origin
			,task_origin_ref
			,task_priority
			,task_state
			,task_state_reason
			,task_assignee_principal_id
			,task_assignee_group_id
			,task_assignee_is_agent
			,task_due
			,task_parent_id
			,task_created_by
			,task_created
			,task_updated
			,task_resolved
			,task_verified
			,task_version
		) VALUES (
			 :task_repo_id
			,:task_number
			,:task_title
			,:task_description
			,:task_origin
			,:task_origin_ref
			,:task_priority
			,:task_state
			,:task_state_reason
			,:task_assignee_principal_id
			,:task_assignee_group_id
			,:task_assignee_is_agent
			,:task_due
			,:task_parent_id
			,:task_created_by
			,:task_created
			,:task_updated
			,:task_resolved
			,:task_verified
			,:task_version
		) RETURNING task_id`

	db := dbtx.GetAccessor(ctx, s.db)

	var number int64
	if err := db.QueryRowContext(ctx, sqlSeq, t.RepoID).Scan(&number); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to allocate task number")
	}

	dbTask := mapInternalTask(t)
	dbTask.Number = number

	query, arg, err := db.BindNamed(sqlInsert, dbTask)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind task object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&t.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert task")
	}

	t.Number = number

	return nil
}

func (s *TaskStore) Find(ctx context.Context, id int64) (*types.Task, error) {
	const sqlQuery = taskSelectBase + ` WHERE task_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &task{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find task")
	}

	return mapTask(dst), nil
}

func (s *TaskStore) FindByNumber(ctx context.Context, repoID, number int64) (*types.Task, error) {
	const sqlQuery = taskSelectBase + ` WHERE task_repo_id = $1 AND task_number = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &task{}
	if err := db.GetContext(ctx, dst, sqlQuery, repoID, number); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find task by number")
	}

	return mapTask(dst), nil
}

// Update updates the task using optimistic locking.
func (s *TaskStore) Update(ctx context.Context, t *types.Task) error {
	const sqlQuery = `
		UPDATE tasks
		SET
			 task_version = :task_version + 1
			,task_title = :task_title
			,task_description = :task_description
			,task_origin_ref = :task_origin_ref
			,task_priority = :task_priority
			,task_state = :task_state
			,task_state_reason = :task_state_reason
			,task_assignee_principal_id = :task_assignee_principal_id
			,task_assignee_group_id = :task_assignee_group_id
			,task_assignee_is_agent = :task_assignee_is_agent
			,task_due = :task_due
			,task_parent_id = :task_parent_id
			,task_updated = :task_updated
			,task_resolved = :task_resolved
			,task_verified = :task_verified
		WHERE task_id = :task_id AND task_version = :task_version`

	// the query increments the version and matches on the current one.
	dbTask := mapInternalTask(t)

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbTask)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind task object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update task")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated task rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	t.Version++

	return nil
}

func (s *TaskStore) List(ctx context.Context, repoID int64, filter *types.TaskFilter) ([]*types.Task, error) {
	stmt := database.Builder.
		Select(taskColumns).
		From("tasks").
		Where("task_repo_id = ?", repoID)

	stmt = applyTaskFilter(stmt, filter)

	stmt = stmt.
		OrderBy("task_number DESC").
		Limit(database.Limit(filter.Size)).
		Offset(database.Offset(filter.Page, filter.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert task list query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*task{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing task list query")
	}

	return mapTasks(dst), nil
}

func (s *TaskStore) Count(ctx context.Context, repoID int64, filter *types.TaskFilter) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("tasks").
		Where("task_repo_id = ?", repoID)

	stmt = applyTaskFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert task count query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing task count query")
	}

	return count, nil
}

func applyTaskFilter(stmt squirrel.SelectBuilder, filter *types.TaskFilter) squirrel.SelectBuilder {
	if filter == nil {
		return stmt
	}

	if len(filter.States) > 0 {
		stmt = stmt.Where(squirrel.Eq{"task_state": filter.States})
	}
	if len(filter.Priorities) > 0 {
		stmt = stmt.Where(squirrel.Eq{"task_priority": filter.Priorities})
	}
	if len(filter.Origins) > 0 {
		stmt = stmt.Where(squirrel.Eq{"task_origin": filter.Origins})
	}
	if len(filter.AssigneeIDs) > 0 {
		stmt = stmt.Where(squirrel.Eq{"task_assignee_principal_id": filter.AssigneeIDs})
	}
	if filter.AssigneeGroupID != nil {
		stmt = stmt.Where("task_assignee_group_id = ?", *filter.AssigneeGroupID)
	}
	if filter.IsAgent != nil {
		stmt = stmt.Where("task_assignee_is_agent = ?", *filter.IsAgent)
	}
	if filter.ParentID != nil {
		stmt = stmt.Where("task_parent_id = ?", *filter.ParentID)
	}
	if len(filter.LabelIDs) > 0 {
		stmt = stmt.Where(squirrel.Expr(
			"EXISTS (SELECT 1 FROM task_labels WHERE task_label_task_id = task_id AND task_label_label_id IN ("+
				squirrel.Placeholders(len(filter.LabelIDs))+"))",
			int64sToAny(filter.LabelIDs)...))
	}
	if filter.Query != "" {
		stmt = stmt.Where(squirrel.Expr(PartialMatch("task_title", filter.Query)))
	}

	return stmt
}

func int64sToAny(in []int64) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

func mapInternalTask(t *types.Task) *task {
	return &task{
		ID:                  t.ID,
		RepoID:              t.RepoID,
		Number:              t.Number,
		Title:               t.Title,
		Description:         t.Description,
		Origin:              t.Origin,
		OriginRef:           t.OriginRef,
		Priority:            t.Priority,
		State:               t.State,
		StateReason:         t.StateReason,
		AssigneePrincipalID: null.IntFromPtr(t.AssigneePrincipalID),
		AssigneeGroupID:     null.IntFromPtr(t.AssigneeGroupID),
		AssigneeIsAgent:     t.AssigneeIsAgent,
		Due:                 null.IntFromPtr(t.Due),
		ParentID:            null.IntFromPtr(t.ParentID),
		CreatedBy:           t.CreatedBy,
		Created:             t.Created,
		Updated:             t.Updated,
		Resolved:            null.IntFromPtr(t.Resolved),
		Verified:            null.IntFromPtr(t.Verified),
		Version:             t.Version,
	}
}

func mapTask(t *task) *types.Task {
	return &types.Task{
		ID:                  t.ID,
		Version:             t.Version,
		RepoID:              t.RepoID,
		Number:              t.Number,
		Title:               t.Title,
		Description:         t.Description,
		Origin:              t.Origin,
		OriginRef:           t.OriginRef,
		Priority:            t.Priority,
		State:               t.State,
		StateReason:         t.StateReason,
		ParentID:            t.ParentID.Ptr(),
		Due:                 t.Due.Ptr(),
		CreatedBy:           t.CreatedBy,
		Created:             t.Created,
		Updated:             t.Updated,
		Resolved:            t.Resolved.Ptr(),
		Verified:            t.Verified.Ptr(),
		AssigneePrincipalID: t.AssigneePrincipalID.Ptr(),
		AssigneeGroupID:     t.AssigneeGroupID.Ptr(),
		AssigneeIsAgent:     t.AssigneeIsAgent,
	}
}

func mapTasks(in []*task) []*types.Task {
	out := make([]*types.Task, len(in))
	for i, t := range in {
		out[i] = mapTask(t)
	}
	return out
}

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
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

var _ store.TaskRepository = (*taskStore)(nil)

func NewTaskStore(db *sqlx.DB, tx dbtx.Transactor) store.TaskRepository {
	return &taskStore{
		db: db,
		tx: tx,
	}
}

type taskStore struct {
	db *sqlx.DB
	tx dbtx.Transactor
}

// Find returns a task by its key.
func (s *taskStore) Find(ctx context.Context, key string) (*types.Task, error) {
	stmt := databaseg.Builder.
		Select("registry_task_key", "registry_task_kind", "registry_task_payload",
			"registry_task_status", "registry_task_run_again", "registry_task_updated_at").
		From("registry_tasks").
		Where("registry_task_key = ?", key)

	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	taskDB := &TaskDB{}
	if err := s.db.GetContext(ctx, taskDB, query, args...); err != nil {
		return nil, fmt.Errorf("failed to find task with key %s: %w", key, err)
	}
	return taskDB.ToTask(), nil
}

func (s *taskStore) UpsertTask(ctx context.Context, task *types.Task) error {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Insert("registry_tasks").
		Columns("registry_task_key", "registry_task_kind", "registry_task_payload",
			"registry_task_updated_at").
		Values(task.Key, task.Kind, task.Payload, now).
		Suffix("ON CONFLICT (registry_task_key) DO UPDATE " +
			"SET registry_task_updated_at = EXCLUDED.registry_task_updated_at")

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build upsert query: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to upsert task with key %s: %w", task.Key, err)
	}
	return nil
}

func (s *taskStore) UpdateStatus(ctx context.Context, taskKey string, status types.TaskStatus) error {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Update("registry_tasks").
		Set("registry_task_status", status).
		Set("registry_task_updated_at", now).
		Where("registry_task_key = ?", taskKey)

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build update status query: %w", err)
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *taskStore) SetRunAgain(ctx context.Context, taskKey string, runAgain bool) error {
	stmt := databaseg.Builder.
		Update("registry_tasks").
		Set("registry_task_run_again", runAgain).
		Where("registry_task_key = ?", taskKey)

	query, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build set run_again query: %w", err)
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *taskStore) LockForUpdate(ctx context.Context, task *types.Task) (types.TaskStatus, error) {
	stmt := databaseg.Builder.
		Select("registry_task_status").
		From("registry_tasks").
		Where("registry_task_key = ?", task.Key)
	if s.db.DriverName() != SQLITE3 {
		stmt = stmt.Suffix("FOR UPDATE")
	}
	query, args, err := stmt.ToSql()
	if err != nil {
		return "", fmt.Errorf("failed to build lock for update query: %w", err)
	}
	var result string
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&result)
	if err != nil {
		return "", fmt.Errorf("failed to lock task with key %s: %w", task.Key, err)
	}
	return types.TaskStatus(result), nil
}

// CompleteTask updates the task status, output data and returns if it should run again.
func (s *taskStore) CompleteTask(
	ctx context.Context,
	key string,
	status types.TaskStatus,
) (bool, error) {
	now := time.Now().UnixMilli()
	stmt := databaseg.Builder.
		Update("registry_tasks").
		Set("registry_task_status", status).
		Set("registry_task_updated_at", now)

	stmt = stmt.Where("registry_task_key = ?", key).
		Suffix("RETURNING registry_task_run_again")

	query, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build complete task query: %w", err)
	}
	var runAgain bool
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&runAgain)
	if err != nil {
		return runAgain, fmt.Errorf("failed to complete task %s with status %s: %w", key, status, err)
	}
	return runAgain, nil
}

// ListPendingTasks lists tasks with pending status.
func (s *taskStore) ListPendingTasks(ctx context.Context, limit int) ([]*types.Task, error) {
	stmt := databaseg.Builder.
		Select("registry_task_key", "registry_task_kind", "registry_task_payload",
			"registry_task_status", "registry_task_run_again", "registry_task_updated_at").
		From("registry_tasks").
		Where("registry_task_status = ?", string(types.TaskStatusPending)).
		OrderBy("registry_task_updated_at").
		Limit(util.SafeIntToUInt64(limit))

	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build list pending tasks query: %w", err)
	}

	var tasksDB []*TaskDB
	err = s.db.SelectContext(ctx, &tasksDB, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending tasks with limit %d: %w", limit, err)
	}

	tasks := make([]*types.Task, len(tasksDB))
	for i, taskDB := range tasksDB {
		tasks[i] = taskDB.ToTask()
	}

	return tasks, nil
}

// TaskDB represents a database entity for task processing.
type TaskDB struct {
	Key       string          `db:"registry_task_key"`
	Kind      string          `db:"registry_task_kind"`
	Payload   json.RawMessage `db:"registry_task_payload"`
	Status    string          `db:"registry_task_status"`
	RunAgain  bool            `db:"registry_task_run_again"`
	UpdatedAt int64           `db:"registry_task_updated_at"`
}

// ToTask converts a database task entity to a DTO.
func (t *TaskDB) ToTask() *types.Task {
	return &types.Task{
		Key:       t.Key,
		Kind:      types.TaskKind(t.Kind),
		Payload:   t.Payload,
		Status:    types.TaskStatus(t.Status),
		RunAgain:  t.RunAgain,
		UpdatedAt: time.UnixMilli(t.UpdatedAt),
	}
}

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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.TaskActivityStore = (*TaskActivityStore)(nil)

// NewTaskActivityStore returns a new TaskActivityStore.
func NewTaskActivityStore(db *sqlx.DB) *TaskActivityStore {
	return &TaskActivityStore{db: db}
}

// TaskActivityStore implements store.TaskActivityStore backed by a relational database.
type TaskActivityStore struct {
	db *sqlx.DB
}

type taskActivity struct {
	ID          int64                 `db:"task_activity_id"`
	TaskID      int64                 `db:"task_activity_task_id"`
	Kind        enum.TaskActivityKind `db:"task_activity_kind"`
	Payload     string                `db:"task_activity_payload"`
	PrincipalID null.Int              `db:"task_activity_principal_id"`
	Created     int64                 `db:"task_activity_created"`
}

const taskActivityColumns = `
	 task_activity_id
	,task_activity_task_id
	,task_activity_kind
	,task_activity_payload
	,task_activity_principal_id
	,task_activity_created`

func (s *TaskActivityStore) Create(ctx context.Context, a *types.TaskActivity) error {
	const sqlQuery = `
		INSERT INTO task_activities (
			 task_activity_task_id
			,task_activity_kind
			,task_activity_payload
			,task_activity_principal_id
			,task_activity_created
		) VALUES (
			 :task_activity_task_id
			,:task_activity_kind
			,:task_activity_payload
			,:task_activity_principal_id
			,:task_activity_created
		) RETURNING task_activity_id`

	payload := "{}"
	if len(a.Payload) > 0 {
		payload = string(a.Payload)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, &taskActivity{
		TaskID:      a.TaskID,
		Kind:        a.Kind,
		Payload:     payload,
		PrincipalID: null.IntFromPtr(a.PrincipalID),
		Created:     a.Created,
	})
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind task activity object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&a.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert task activity")
	}

	return nil
}

func (s *TaskActivityStore) List(
	ctx context.Context,
	taskID int64,
	pagination types.Pagination,
) ([]*types.TaskActivity, error) {
	stmt := database.Builder.
		Select(taskActivityColumns).
		From("task_activities").
		Where("task_activity_task_id = ?", taskID).
		OrderBy("task_activity_created ASC", "task_activity_id ASC").
		Limit(database.Limit(pagination.Size)).
		Offset(database.Offset(pagination.Page, pagination.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert task activity list query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*taskActivity{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing task activity list query")
	}

	out := make([]*types.TaskActivity, len(dst))
	for i, d := range dst {
		out[i] = &types.TaskActivity{
			ID:          d.ID,
			TaskID:      d.TaskID,
			Kind:        d.Kind,
			Payload:     json.RawMessage(d.Payload),
			PrincipalID: d.PrincipalID.Ptr(),
			Created:     d.Created,
		}
	}

	return out, nil
}

func (s *TaskActivityStore) ListByKind(
	ctx context.Context,
	taskID int64,
	kind enum.TaskActivityKind,
) ([]*types.TaskActivity, error) {
	stmt := database.Builder.
		Select(taskActivityColumns).
		From("task_activities").
		Where("task_activity_task_id = ?", taskID).
		Where("task_activity_kind = ?", kind).
		OrderBy("task_activity_created ASC", "task_activity_id ASC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert task activity list by kind query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*taskActivity{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing task activity list by kind query")
	}

	out := make([]*types.TaskActivity, len(dst))
	for i, d := range dst {
		out[i] = &types.TaskActivity{
			ID:          d.ID,
			TaskID:      d.TaskID,
			Kind:        d.Kind,
			Payload:     json.RawMessage(d.Payload),
			PrincipalID: d.PrincipalID.Ptr(),
			Created:     d.Created,
		}
	}

	return out, nil
}

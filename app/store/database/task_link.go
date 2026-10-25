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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ store.TaskLinkStore = (*TaskLinkStore)(nil)

// NewTaskLinkStore returns a new TaskLinkStore.
func NewTaskLinkStore(db *sqlx.DB) *TaskLinkStore {
	return &TaskLinkStore{db: db}
}

// TaskLinkStore implements store.TaskLinkStore backed by a relational database.
type TaskLinkStore struct {
	db *sqlx.DB
}

type taskLink struct {
	TaskID  int64             `db:"task_link_task_id"`
	Kind    enum.TaskLinkKind `db:"task_link_kind"`
	RefID   string            `db:"task_link_ref_id"`
	Created int64             `db:"task_link_created"`
}

const taskLinkColumns = `
	 task_link_task_id
	,task_link_kind
	,task_link_ref_id
	,task_link_created`

func (s *TaskLinkStore) Create(ctx context.Context, l *types.TaskLink) error {
	const sqlQuery = `
		INSERT INTO task_links (` + taskLinkColumns + `)
		VALUES (
			 :task_link_task_id
			,:task_link_kind
			,:task_link_ref_id
			,:task_link_created
		)
		ON CONFLICT (task_link_task_id, task_link_kind, task_link_ref_id) DO NOTHING`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, &taskLink{
		TaskID:  l.TaskID,
		Kind:    l.Kind,
		RefID:   l.RefID,
		Created: l.Created,
	})
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind task link object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert task link")
	}

	return nil
}

func (s *TaskLinkStore) Delete(ctx context.Context, taskID int64, kind enum.TaskLinkKind, refID string) error {
	const sqlQuery = `
		DELETE FROM task_links
		WHERE task_link_task_id = $1 AND task_link_kind = $2 AND task_link_ref_id = $3`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, taskID, kind, refID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete task link")
	}

	return nil
}

func (s *TaskLinkStore) ListByTask(ctx context.Context, taskID int64) ([]*types.TaskLink, error) {
	const sqlQuery = `SELECT` + taskLinkColumns + `
		FROM task_links
		WHERE task_link_task_id = $1
		ORDER BY task_link_kind, task_link_ref_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*taskLink{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, taskID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list task links")
	}

	out := make([]*types.TaskLink, len(dst))
	for i, d := range dst {
		out[i] = &types.TaskLink{
			TaskID:  d.TaskID,
			Kind:    d.Kind,
			RefID:   d.RefID,
			Created: d.Created,
		}
	}

	return out, nil
}

func (s *TaskLinkStore) FindTasksByRef(
	ctx context.Context,
	kind enum.TaskLinkKind,
	refID string,
) ([]*types.Task, error) {
	const sqlQuery = taskSelectBase + `
		INNER JOIN task_links ON task_link_task_id = task_id
		WHERE task_link_kind = $1 AND task_link_ref_id = $2
		ORDER BY task_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*task{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, kind, refID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find tasks by ref")
	}

	return mapTasks(dst), nil
}

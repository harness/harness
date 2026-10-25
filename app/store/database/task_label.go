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

	"github.com/Masterminds/squirrel"
	"github.com/gotidy/ptr"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.TaskLabelStore = (*taskLabelStore)(nil)

func NewTaskLabelStore(db *sqlx.DB) store.TaskLabelStore {
	return &taskLabelStore{
		db: db,
	}
}

type taskLabelStore struct {
	db *sqlx.DB
}

type taskLabel struct {
	TaskID       int64    `db:"task_label_task_id"`
	LabelID      int64    `db:"task_label_label_id"`
	LabelValueID null.Int `db:"task_label_label_value_id"`
	Created      int64    `db:"task_label_created"`
	Updated      int64    `db:"task_label_updated"`
	CreatedBy    int64    `db:"task_label_created_by"`
	UpdatedBy    int64    `db:"task_label_updated_by"`
}

type taskAssignmentInfo struct {
	TaskID     int64           `db:"task_label_task_id"`
	LabelID    int64           `db:"label_id"`
	LabelKey   string          `db:"label_key"`
	LabelColor enum.LabelColor `db:"label_color"`
	LabelScope int64           `db:"label_scope"`
	ValueCount int64           `db:"label_value_count"`
	ValueID    null.Int        `db:"label_value_id"`
	Value      null.String     `db:"label_value_value"`
	ValueColor null.String     `db:"label_value_color"`
}

const (
	taskLabelColumns = `
		 task_label_task_id
		,task_label_label_id
		,task_label_label_value_id
		,task_label_created
		,task_label_updated
		,task_label_created_by
		,task_label_updated_by`
)

func (s *taskLabelStore) Assign(ctx context.Context, label *types.TaskLabel) error {
	const sqlQuery = `
		INSERT INTO task_labels (` + taskLabelColumns + `)
			values (
				:task_label_task_id
				,:task_label_label_id
				,:task_label_label_value_id
				,:task_label_created
				,:task_label_updated
				,:task_label_created_by
				,:task_label_updated_by
			)
			ON CONFLICT (task_label_task_id, task_label_label_id)
			DO UPDATE SET
				task_label_label_value_id = EXCLUDED.task_label_label_value_id,
				task_label_updated = EXCLUDED.task_label_updated,
				task_label_updated_by = EXCLUDED.task_label_updated_by
			RETURNING task_label_created, task_label_created_by
			`

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, mapInternalTaskLabel(label))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to bind query")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&label.Created, &label.CreatedBy); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to create task label")
	}

	return nil
}

func (s *taskLabelStore) Unassign(ctx context.Context, taskID int64, labelID int64) (bool, error) {
	const sqlQuery = `
		DELETE FROM task_labels
		WHERE task_label_task_id = $1 AND task_label_label_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, taskID, labelID)
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "failed to delete task label")
	}

	n, err := result.RowsAffected()
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "failed to get number of deleted task labels")
	}

	return n > 0, nil
}

func (s *taskLabelStore) FindByLabelID(
	ctx context.Context,
	taskID int64,
	labelID int64,
) (*types.TaskLabel, error) {
	const sqlQuery = `SELECT ` + taskLabelColumns + `
		FROM task_labels
		WHERE task_label_task_id = $1 AND task_label_label_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst taskLabel
	if err := db.GetContext(ctx, &dst, sqlQuery, taskID, labelID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to find task label by id")
	}

	return mapTaskLabel(&dst), nil
}

func (s *taskLabelStore) ListAssigned(
	ctx context.Context,
	taskID int64,
) (map[int64]*types.LabelAssignment, error) {
	const sqlQuery = `
		SELECT
			label_id
			,label_repo_id
			,label_space_id
			,label_key
			,label_value_id
			,label_value_label_id
			,label_value_value
			,label_color
			,label_value_color
			,label_scope
			,label_type
		FROM task_labels
		INNER JOIN labels ON task_label_label_id = label_id
		LEFT JOIN label_values ON task_label_label_value_id = label_value_id
		WHERE task_label_task_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*struct {
		labelInfo
		labelValueInfo
	}
	if err := db.SelectContext(ctx, &dst, sqlQuery, taskID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list assigned task labels")
	}

	ret := make(map[int64]*types.LabelAssignment, len(dst))
	for _, res := range dst {
		li := mapLabelInfo(&res.labelInfo)
		lvi := mapLabeValuelInfo(&res.labelValueInfo)
		ret[li.ID] = &types.LabelAssignment{
			LabelInfo:     *li,
			AssignedValue: lvi,
		}
	}

	return ret, nil
}

func (s *taskLabelStore) ListAssignedByTaskIDs(
	ctx context.Context,
	taskIDs []int64,
) (map[int64][]*types.LabelTaskAssignmentInfo, error) {
	stmt := database.Builder.Select(`
			task_label_task_id
			,label_id
			,label_key
			,label_color
			,label_scope
			,label_value_count
			,label_value_id
			,label_value_value
			,label_value_color
	`).
		From("task_labels").
		InnerJoin("labels ON task_label_label_id = label_id").
		LeftJoin("label_values ON task_label_label_value_id = label_value_id").
		Where(squirrel.Eq{"task_label_task_id": taskIDs})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*taskAssignmentInfo
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list assigned task labels")
	}

	result := make(map[int64][]*types.LabelTaskAssignmentInfo)
	for _, lbl := range dst {
		var valueColor *enum.LabelColor
		if lbl.ValueColor.Valid {
			valueColor = ptr.Of(enum.LabelColor(lbl.ValueColor.String))
		}
		result[lbl.TaskID] = append(result[lbl.TaskID], &types.LabelTaskAssignmentInfo{
			TaskID:     lbl.TaskID,
			LabelID:    lbl.LabelID,
			LabelKey:   lbl.LabelKey,
			LabelColor: lbl.LabelColor,
			LabelScope: lbl.LabelScope,
			ValueCount: lbl.ValueCount,
			ValueID:    lbl.ValueID.Ptr(),
			Value:      lbl.Value.Ptr(),
			ValueColor: valueColor,
		})
	}

	return result, nil
}

func mapInternalTaskLabel(lbl *types.TaskLabel) *taskLabel {
	return &taskLabel{
		TaskID:       lbl.TaskID,
		LabelID:      lbl.LabelID,
		LabelValueID: null.IntFromPtr(lbl.ValueID),
		Created:      lbl.Created,
		Updated:      lbl.Updated,
		CreatedBy:    lbl.CreatedBy,
		UpdatedBy:    lbl.UpdatedBy,
	}
}

func mapTaskLabel(lbl *taskLabel) *types.TaskLabel {
	return &types.TaskLabel{
		TaskID:    lbl.TaskID,
		LabelID:   lbl.LabelID,
		ValueID:   lbl.LabelValueID.Ptr(),
		Created:   lbl.Created,
		Updated:   lbl.Updated,
		CreatedBy: lbl.CreatedBy,
		UpdatedBy: lbl.UpdatedBy,
	}
}

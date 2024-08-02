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
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PullReqLabelAssignmentStore = (*pullReqLabelStore)(nil)

func NewPullReqLabelStore(db *sqlx.DB) store.PullReqLabelAssignmentStore {
	return &pullReqLabelStore{
		db: db,
	}
}

type pullReqLabelStore struct {
	db *sqlx.DB
}

type pullReqLabel struct {
	PullReqID    int64    `db:"pullreq_label_pullreq_id"`
	LabelID      int64    `db:"pullreq_label_label_id"`
	LabelValueID null.Int `db:"pullreq_label_label_value_id"`
	Created      int64    `db:"pullreq_label_created"`
	Updated      int64    `db:"pullreq_label_updated"`
	CreatedBy    int64    `db:"pullreq_label_created_by"`
	UpdatedBy    int64    `db:"pullreq_label_updated_by"`
}

type pullReqAssignmentInfo struct {
	PullReqID  int64           `db:"pullreq_label_pullreq_id"`
	LabelID    int64           `db:"label_id"`
	LabelKey   string          `db:"label_key"`
	LabelColor enum.LabelColor `db:"label_color"`
	ValueCount int64           `db:"label_value_count"`
	Value      null.String     `db:"label_value_value"`
	ValueColor null.String     `db:"label_value_color"`
}

const (
	pullReqLabelColumns = `
		 pullreq_label_pullreq_id
		,pullreq_label_label_id
		,pullreq_label_label_value_id
		,pullreq_label_created
		,pullreq_label_updated
		,pullreq_label_created_by
		,pullreq_label_updated_by`
)

func (s *pullReqLabelStore) Assign(ctx context.Context, label *types.PullReqLabel) error {
	const sqlQuery = `
		INSERT INTO pullreq_labels (` + pullReqLabelColumns + `)
			values (
				:pullreq_label_pullreq_id
				,:pullreq_label_label_id
				,:pullreq_label_label_value_id
				,:pullreq_label_created
				,:pullreq_label_updated
				,:pullreq_label_created_by
				,:pullreq_label_updated_by			
			)
			ON CONFLICT (pullreq_label_pullreq_id, pullreq_label_label_id) 
			DO UPDATE SET 
				pullreq_label_label_value_id = EXCLUDED.pullreq_label_label_value_id,
				pullreq_label_updated = EXCLUDED.pullreq_label_updated,
				pullreq_label_updated_by = EXCLUDED.pullreq_label_updated_by
			RETURNING pullreq_label_created, pullreq_label_created_by
			`

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, mapInternalPullReqLabel(label))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to bind query")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&label.Created, &label.CreatedBy); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to create pull request label")
	}

	return nil
}

func (s *pullReqLabelStore) Unassign(ctx context.Context, pullreqID int64, labelID int64) error {
	const sqlQuery = `
		DELETE FROM pullreq_labels
		WHERE pullreq_label_pullreq_id = $1 AND pullreq_label_label_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, pullreqID, labelID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to delete pullreq label")
	}

	return nil
}

func (s *pullReqLabelStore) FindByLabelID(
	ctx context.Context,
	pullreqID int64,
	labelID int64,
) (*types.PullReqLabel, error) {
	const sqlQuery = `SELECT ` + pullReqLabelColumns + `
		FROM pullreq_labels
		WHERE pullreq_label_pullreq_id = $1 AND pullreq_label_label_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst pullReqLabel
	if err := db.GetContext(ctx, &dst, sqlQuery, pullreqID, labelID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to find pullreq label by id")
	}

	return mapPullReqLabel(&dst), nil
}

func (s *pullReqLabelStore) ListAssigned(
	ctx context.Context,
	pullreqID int64,
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
		FROM pullreq_labels
		INNER JOIN labels ON pullreq_label_label_id = label_id
		LEFT JOIN label_values ON pullreq_label_label_value_id = label_value_id
		WHERE pullreq_label_pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*struct {
		labelInfo
		labelValueInfo
	}
	if err := db.SelectContext(ctx, &dst, sqlQuery, pullreqID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list assigned label")
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

func (s *pullReqLabelStore) ListAssignedByPullreqIDs(
	ctx context.Context,
	pullreqIDs []int64,
) (map[int64][]*types.LabelPullReqAssignmentInfo, error) {
	stmt := database.Builder.Select(`
			pullreq_label_pullreq_id
			,label_id
			,label_key
			,label_color
			,label_value_count
			,label_value_value
			,label_value_color
	`).
		From("pullreq_labels").
		InnerJoin("labels ON pullreq_label_label_id = label_id").
		LeftJoin("label_values ON pullreq_label_label_value_id = label_value_id").
		Where(squirrel.Eq{"pullreq_label_pullreq_id": pullreqIDs})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*pullReqAssignmentInfo
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to list assigned label")
	}

	return mapPullReqAssignmentInfos(dst), nil
}

func (s *pullReqLabelStore) FindValueByLabelID(
	ctx context.Context,
	labelID int64,
) (*types.LabelValue, error) {
	const sqlQuery = `SELECT label_value_id, ` + labelValueColumns + `
		FROM pullreq_labels
		JOIN label_values ON pullreq_label_label_value_id = label_value_id
		WHERE pullreq_label_label_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst labelValue
	if err := db.GetContext(ctx, &dst, sqlQuery, labelID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find label")
	}

	return mapLabelValue(&dst), nil
}

func mapInternalPullReqLabel(lbl *types.PullReqLabel) *pullReqLabel {
	return &pullReqLabel{
		PullReqID:    lbl.PullReqID,
		LabelID:      lbl.LabelID,
		LabelValueID: null.IntFromPtr(lbl.ValueID),
		Created:      lbl.Created,
		Updated:      lbl.Updated,
		CreatedBy:    lbl.CreatedBy,
		UpdatedBy:    lbl.UpdatedBy,
	}
}

func mapPullReqLabel(lbl *pullReqLabel) *types.PullReqLabel {
	return &types.PullReqLabel{
		PullReqID: lbl.PullReqID,
		LabelID:   lbl.LabelID,
		ValueID:   lbl.LabelValueID.Ptr(),
		Created:   lbl.Created,
		Updated:   lbl.Updated,
		CreatedBy: lbl.CreatedBy,
		UpdatedBy: lbl.UpdatedBy,
	}
}

func mapPullReqAssignmentInfo(lbl *pullReqAssignmentInfo) *types.LabelPullReqAssignmentInfo {
	return &types.LabelPullReqAssignmentInfo{
		PullReqID:  lbl.PullReqID,
		LabelID:    lbl.LabelID,
		LabelKey:   lbl.LabelKey,
		LabelColor: lbl.LabelColor,
		ValueCount: lbl.ValueCount,
		Value:      lbl.Value.Ptr(),
		ValueColor: lbl.ValueColor.Ptr(),
	}
}

func mapPullReqAssignmentInfos(
	dbLabels []*pullReqAssignmentInfo,
) map[int64][]*types.LabelPullReqAssignmentInfo {
	result := make(map[int64][]*types.LabelPullReqAssignmentInfo)

	for _, lbl := range dbLabels {
		result[lbl.PullReqID] = append(result[lbl.PullReqID], mapPullReqAssignmentInfo(lbl))
	}

	return result
}

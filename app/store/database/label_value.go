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

const (
	labelValueColumns = `
		 label_value_label_id
		,label_value_value
		,label_value_color
		,label_value_created
		,label_value_updated
		,label_value_created_by
		,label_value_updated_by`

	labelValueSelectBase = `SELECT label_value_id, ` + labelValueColumns + ` FROM label_values`
)

type labelValue struct {
	ID        int64           `db:"label_value_id"`
	LabelID   int64           `db:"label_value_label_id"`
	Value     string          `db:"label_value_value"`
	Color     enum.LabelColor `db:"label_value_color"`
	Created   int64           `db:"label_value_created"`
	Updated   int64           `db:"label_value_updated"`
	CreatedBy int64           `db:"label_value_created_by"`
	UpdatedBy int64           `db:"label_value_updated_by"`
}

type labelValueInfo struct {
	ValueID    null.Int    `db:"label_value_id"`
	LabelID    null.Int    `db:"label_value_label_id"`
	Value      null.String `db:"label_value_value"`
	ValueColor null.String `db:"label_value_color"`
}

type labelValueStore struct {
	db *sqlx.DB
}

func NewLabelValueStore(
	db *sqlx.DB,
) store.LabelValueStore {
	return &labelValueStore{
		db: db,
	}
}

var _ store.LabelValueStore = (*labelValueStore)(nil)

func (s *labelValueStore) Define(ctx context.Context, lblVal *types.LabelValue) error {
	const sqlQuery = `
		INSERT INTO label_values (` + labelValueColumns + `)` + `
			values (
				:label_value_label_id
				,:label_value_value
				,:label_value_color
				,:label_value_created
				,:label_value_updated
				,:label_value_created_by
				,:label_value_updated_by
			) 
		RETURNING label_value_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, mapInternalLabelValue(lblVal))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&lblVal.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to create label value")
	}

	return nil
}

func (s *labelValueStore) Update(ctx context.Context, lblVal *types.LabelValue) error {
	const sqlQuery = `
		UPDATE label_values SET
			 label_value_value = :label_value_value
			,label_value_color = :label_value_color
			,label_value_updated = :label_value_updated
			,label_value_updated_by = :label_value_updated_by
		WHERE label_value_id = :label_value_id`

	db := dbtx.GetAccessor(ctx, s.db)
	query, args, err := db.BindNamed(sqlQuery, mapInternalLabelValue(lblVal))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update label value")
	}

	return nil
}

func (s *labelValueStore) Delete(
	ctx context.Context,
	labelID int64,
	value string,
) error {
	const sqlQuery = `
		DELETE FROM label_values
		WHERE label_value_label_id = $1 AND LOWER(label_value_value) = LOWER($2)`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, labelID, value); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete label")
	}

	return nil
}

func (s *labelValueStore) DeleteMany(
	ctx context.Context,
	labelID int64,
	values []string,
) error {
	stmt := database.Builder.
		Delete("label_values").
		Where("label_value_label_id = ?", labelID).
		Where(squirrel.Eq{"label_value_value": values})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete label")
	}

	return nil
}

// List returns a list of label values for a specified label.
func (s *labelValueStore) List(
	ctx context.Context,
	labelID int64,
	opts *types.ListQueryFilter,
) ([]*types.LabelValue, error) {
	stmt := database.Builder.
		Select(`label_value_id, ` + labelValueColumns).
		From("label_values")

	stmt = stmt.Where("label_value_label_id = ?", labelID)

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*labelValue
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Fail to list labels")
	}

	return mapSliceLabelValue(dst), nil
}

func (s *labelValueStore) ListInfosByLabelIDs(
	ctx context.Context,
	labelIDs []int64,
) (map[int64][]*types.LabelValueInfo, error) {
	stmt := database.Builder.
		Select(`
			 label_value_id
			,label_value_label_id
			,label_value_value
			,label_value_color
		`).
		From("label_values").
		Where(squirrel.Eq{"label_value_label_id": labelIDs}).
		OrderBy("label_value_value")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*labelValueInfo
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Fail to list labels")
	}

	valueInfos := mapLabelValuInfos(dst)
	labelValueMap := make(map[int64][]*types.LabelValueInfo)
	for _, info := range valueInfos {
		labelValueMap[*info.LabelID] = append(labelValueMap[*info.LabelID], info)
	}

	return labelValueMap, nil
}

func (s *labelValueStore) FindByLabelID(
	ctx context.Context,
	labelID int64,
	value string,
) (*types.LabelValue, error) {
	const sqlQuery = labelValueSelectBase + `
		WHERE label_value_label_id = $1 AND LOWER(label_value_value) = LOWER($2)`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst labelValue
	if err := db.GetContext(ctx, &dst, sqlQuery, labelID, value); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find label")
	}

	return mapLabelValue(&dst), nil
}

func (s *labelValueStore) FindByID(ctx context.Context, id int64) (*types.LabelValue, error) {
	const sqlQuery = labelValueSelectBase + `
		WHERE label_value_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst labelValue
	if err := db.GetContext(ctx, &dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find label")
	}

	return mapLabelValue(&dst), nil
}

func mapLabelValue(lbl *labelValue) *types.LabelValue {
	return &types.LabelValue{
		ID:        lbl.ID,
		LabelID:   lbl.LabelID,
		Value:     lbl.Value,
		Color:     lbl.Color,
		Created:   lbl.Created,
		Updated:   lbl.Updated,
		CreatedBy: lbl.CreatedBy,
		UpdatedBy: lbl.UpdatedBy,
	}
}

func mapSliceLabelValue(dbLabelValues []*labelValue) []*types.LabelValue {
	result := make([]*types.LabelValue, len(dbLabelValues))

	for i, lbl := range dbLabelValues {
		result[i] = mapLabelValue(lbl)
	}

	return result
}

func mapInternalLabelValue(lblVal *types.LabelValue) *labelValue {
	return &labelValue{
		ID:        lblVal.ID,
		LabelID:   lblVal.LabelID,
		Value:     lblVal.Value,
		Color:     lblVal.Color,
		Created:   lblVal.Created,
		Updated:   lblVal.Updated,
		CreatedBy: lblVal.CreatedBy,
		UpdatedBy: lblVal.UpdatedBy,
	}
}

func mapLabeValuelInfo(internal *labelValueInfo) *types.LabelValueInfo {
	if !internal.ValueID.Valid {
		return nil
	}
	return &types.LabelValueInfo{
		ID:      internal.ValueID.Ptr(),
		LabelID: internal.LabelID.Ptr(),
		Value:   internal.Value.Ptr(),
		Color:   internal.ValueColor.Ptr(),
	}
}

func mapLabelValuInfos(
	dbLabels []*labelValueInfo,
) []*types.LabelValueInfo {
	result := make([]*types.LabelValueInfo, len(dbLabels))

	for i, lbl := range dbLabels {
		result[i] = mapLabeValuelInfo(lbl)
	}

	return result
}

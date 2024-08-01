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
	labelColumns = `
		 label_space_id
		,label_repo_id
		,label_scope
		,label_key
		,label_description
		,label_type
		,label_color
		,label_created
		,label_updated
		,label_created_by
		,label_updated_by`

	labelSelectBase = `SELECT label_id, ` + labelColumns + ` FROM labels`
)

type label struct {
	ID          int64           `db:"label_id"`
	SpaceID     null.Int        `db:"label_space_id"`
	RepoID      null.Int        `db:"label_repo_id"`
	Scope       int64           `db:"label_scope"`
	Key         string          `db:"label_key"`
	Description string          `db:"label_description"`
	Type        enum.LabelType  `db:"label_type"`
	Color       enum.LabelColor `db:"label_color"`
	ValueCount  int64           `db:"label_value_count"`
	Created     int64           `db:"label_created"`
	Updated     int64           `db:"label_updated"`
	CreatedBy   int64           `db:"label_created_by"`
	UpdatedBy   int64           `db:"label_updated_by"`
}

type labelInfo struct {
	LabelID    int64           `db:"label_id"`
	SpaceID    null.Int        `db:"label_space_id"`
	RepoID     null.Int        `db:"label_repo_id"`
	Scope      int64           `db:"label_scope"`
	Key        string          `db:"label_key"`
	Type       enum.LabelType  `db:"label_type"`
	LabelColor enum.LabelColor `db:"label_color"`
}

type labelStore struct {
	db *sqlx.DB
}

func NewLabelStore(
	db *sqlx.DB,
) store.LabelStore {
	return &labelStore{
		db: db,
	}
}

var _ store.LabelStore = (*labelStore)(nil)

func (s *labelStore) Define(ctx context.Context, lbl *types.Label) error {
	const sqlQuery = `
		INSERT INTO labels (` + labelColumns + `)` + `
		values (
			:label_space_id
			,:label_repo_id
			,:label_scope
			,:label_key
			,:label_description
			,:label_type
			,:label_color
			,:label_created
			,:label_updated 
			,:label_created_by
			,:label_updated_by 
		) 
		RETURNING label_id`

	db := dbtx.GetAccessor(ctx, s.db)
	query, args, err := db.BindNamed(sqlQuery, mapInternalLabel(lbl))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&lbl.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to create label")
	}

	return nil
}

func (s *labelStore) Update(ctx context.Context, lbl *types.Label) error {
	const sqlQuery = `
		UPDATE labels SET
			 label_key = :label_key
			,label_description = :label_description
			,label_type = :label_type
			,label_color = :label_color
			,label_updated = :label_updated
			,label_updated_by = :label_updated_by
		WHERE label_id = :label_id`

	db := dbtx.GetAccessor(ctx, s.db)
	query, args, err := db.BindNamed(sqlQuery, mapInternalLabel(lbl))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update label")
	}

	return nil
}

func (s *labelStore) IncrementValueCount(
	ctx context.Context,
	labelID int64,
	increment int,
) (int64, error) {
	const sqlQuery = `
		UPDATE labels
		SET label_value_count = label_value_count + $1
		WHERE label_id = $2
		RETURNING label_value_count
	`

	db := dbtx.GetAccessor(ctx, s.db)

	var valueCount int64
	if err := db.QueryRowContext(ctx, sqlQuery, increment, labelID).Scan(&valueCount); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to increment label_value_count")
	}

	return valueCount, nil
}

func (s *labelStore) Find(
	ctx context.Context,
	spaceID, repoID *int64,
	key string,
) (*types.Label, error) {
	const sqlQuery = labelSelectBase + `
	WHERE (label_space_id = $1 OR label_repo_id = $2) AND LOWER(label_key) = LOWER($3)`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst label
	if err := db.GetContext(ctx, &dst, sqlQuery, spaceID, repoID, key); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find label")
	}

	return mapLabel(&dst), nil
}

func (s *labelStore) FindByID(ctx context.Context, id int64) (*types.Label, error) {
	const sqlQuery = labelSelectBase + `
		WHERE label_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst label
	if err := db.GetContext(ctx, &dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find label")
	}

	return mapLabel(&dst), nil
}

func (s *labelStore) Delete(ctx context.Context, spaceID, repoID *int64, name string) error {
	const sqlQuery = `
		DELETE FROM labels
		WHERE (label_space_id = $1 OR label_repo_id = $2) AND LOWER(label_key) = LOWER($3)`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, spaceID, repoID, name); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete label")
	}

	return nil
}

// List returns a list of pull requests for a repo or space.
func (s *labelStore) List(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.LabelFilter,
) ([]*types.Label, error) {
	stmt := database.Builder.
		Select(`label_id, ` + labelColumns + `, label_value_count`).
		From("labels").
		OrderBy("label_key")

	stmt = stmt.Where("(label_space_id = ? OR label_repo_id = ?)", spaceID, repoID)
	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))
	if filter.Query != "" {
		stmt = stmt.Where(
			"LOWER(label_key) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*label
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list labels")
	}

	return mapSliceLabel(dst), nil
}

func (s *labelStore) ListInScopes(
	ctx context.Context,
	repoID int64,
	spaceIDs []int64,
	filter *types.LabelFilter,
) ([]*types.Label, error) {
	stmt := database.Builder.
		Select(`label_id, ` + labelColumns + `, label_value_count`).
		From("labels")

	stmt = stmt.Where(squirrel.Or{
		squirrel.Eq{"label_space_id": spaceIDs},
		squirrel.Eq{"label_repo_id": repoID},
	}).
		OrderBy("label_key").
		OrderBy("label_scope")

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))
	if filter.Query != "" {
		stmt = stmt.Where(
			"LOWER(label_key) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*label
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list labels in hierarchy")
	}

	return mapSliceLabel(dst), nil
}

func (s *labelStore) ListInfosInScopes(
	ctx context.Context,
	repoID int64,
	spaceIDs []int64,
	filter *types.AssignableLabelFilter,
) ([]*types.LabelInfo, error) {
	stmt := database.Builder.
		Select(`
			 label_id
			,label_space_id
			,label_repo_id
			,label_scope
			,label_key
			,label_type
			,label_color`).
		From("labels").
		Where(squirrel.Or{
			squirrel.Eq{"label_space_id": spaceIDs},
			squirrel.Eq{"label_repo_id": repoID},
		}).
		OrderBy("label_key").
		OrderBy("label_scope")

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))
	if filter.Query != "" {
		stmt = stmt.Where(
			"LOWER(label_key) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*labelInfo
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list labels")
	}

	return mapLabelInfos(dst), nil
}

func (s *labelStore) CountInSpace(
	ctx context.Context,
	spaceID int64,
	filter *types.LabelFilter,
) (int64, error) {
	const sqlQuery = `SELECT COUNT(*) FROM labels WHERE label_space_id = $1`

	return s.count(ctx, sqlQuery, spaceID, filter)
}

func (s *labelStore) CountInRepo(
	ctx context.Context,
	repoID int64,
	filter *types.LabelFilter,
) (int64, error) {
	const sqlQuery = `SELECT COUNT(*) FROM labels WHERE label_repo_id = $1`

	return s.count(ctx, sqlQuery, repoID, filter)
}

func (s labelStore) count(
	ctx context.Context,
	sqlQuery string,
	scopeID int64,
	filter *types.LabelFilter,
) (int64, error) {
	sqlQuery += `
		AND LOWER(label_key) LIKE '%' || LOWER($2) || '%'`

	db := dbtx.GetAccessor(ctx, s.db)
	var count int64
	if err := db.QueryRowContext(ctx, sqlQuery, scopeID, filter.Query).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to count labels")
	}

	return count, nil
}

func (s *labelStore) CountInScopes(
	ctx context.Context,
	repoID int64,
	spaceIDs []int64,
	filter *types.LabelFilter,
) (int64, error) {
	stmt := database.Builder.Select("COUNT(*)").
		From("labels").
		Where(squirrel.Or{
			squirrel.Eq{"label_space_id": spaceIDs},
			squirrel.Eq{"label_repo_id": repoID},
		}).
		Where("LOWER(label_key) LIKE '%' || LOWER(?) || '%'", filter.Query)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to count labels in scopes")
	}

	return count, nil
}

func mapLabel(lbl *label) *types.Label {
	return &types.Label{
		ID:          lbl.ID,
		SpaceID:     lbl.SpaceID.Ptr(),
		RepoID:      lbl.RepoID.Ptr(),
		Scope:       lbl.Scope,
		Key:         lbl.Key,
		Type:        lbl.Type,
		Description: lbl.Description,
		ValueCount:  lbl.ValueCount,
		Color:       lbl.Color,
		Created:     lbl.Created,
		Updated:     lbl.Updated,
		CreatedBy:   lbl.CreatedBy,
		UpdatedBy:   lbl.UpdatedBy,
	}
}

func mapSliceLabel(dbLabels []*label) []*types.Label {
	result := make([]*types.Label, len(dbLabels))

	for i, lbl := range dbLabels {
		result[i] = mapLabel(lbl)
	}

	return result
}

func mapInternalLabel(lbl *types.Label) *label {
	return &label{
		ID:          lbl.ID,
		SpaceID:     null.IntFromPtr(lbl.SpaceID),
		RepoID:      null.IntFromPtr(lbl.RepoID),
		Scope:       lbl.Scope,
		Key:         lbl.Key,
		Description: lbl.Description,
		Type:        lbl.Type,
		Color:       lbl.Color,
		Created:     lbl.Created,
		Updated:     lbl.Updated,
		CreatedBy:   lbl.CreatedBy,
		UpdatedBy:   lbl.UpdatedBy,
	}
}

func mapLabelInfo(internal *labelInfo) *types.LabelInfo {
	return &types.LabelInfo{
		ID:      internal.LabelID,
		RepoID:  internal.RepoID.Ptr(),
		SpaceID: internal.SpaceID.Ptr(),
		Scope:   internal.Scope,
		Key:     internal.Key,
		Type:    internal.Type,
		Color:   internal.LabelColor,
	}
}

func mapLabelInfos(
	dbLabels []*labelInfo,
) []*types.LabelInfo {
	result := make([]*types.LabelInfo, len(dbLabels))

	for i, lbl := range dbLabels {
		result[i] = mapLabelInfo(lbl)
	}

	return result
}

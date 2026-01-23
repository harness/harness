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
	"time"

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
	autolinkColumns = `
		 autolink_space_id
		,autolink_repo_id
		,autolink_type
		,autolink_pattern
		,autolink_target_url
		,autolink_created
		,autolink_updated
		,autolink_created_by
		,autolink_updated_by`

	autolinkColumnsWithID = autolinkColumns + `,autolink_id`
)

type autoLink struct {
	ID        int64             `db:"autolink_id"`
	SpaceID   null.Int          `db:"autolink_space_id"`
	RepoID    null.Int          `db:"autolink_repo_id"`
	Type      enum.AutoLinkType `db:"autolink_type"`
	Pattern   string            `db:"autolink_pattern"`
	TargetURL string            `db:"autolink_target_url"`
	Created   int64             `db:"autolink_created"`
	Updated   int64             `db:"autolink_updated"`
	CreatedBy int64             `db:"autolink_created_by"`
	UpdatedBy int64             `db:"autolink_updated_by"`
}

var _ store.AutoLinkStore = (*AutoLinkStore)(nil)

type AutoLinkStore struct {
	db *sqlx.DB
}

func NewAutoLinkStore(db *sqlx.DB) store.AutoLinkStore {
	return &AutoLinkStore{
		db: db,
	}
}

func (s *AutoLinkStore) Create(ctx context.Context, autolink *types.AutoLink) error {
	const sqlQuery = `
		INSERT INTO autolinks (` + autolinkColumns + `)` + `
		values (
			:autolink_space_id
			,:autolink_repo_id
			,:autolink_type
			,:autolink_pattern
			,:autolink_target_url
			,:autolink_created
			,:autolink_updated
			,:autolink_created_by
			,:autolink_updated_by
		)
		RETURNING autolink_id`

	db := dbtx.GetAccessor(ctx, s.db)
	query, args, err := db.BindNamed(sqlQuery, mapInternalAutoLink(autolink))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&autolink.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to create autolink")
	}

	return nil
}

func (s *AutoLinkStore) Update(ctx context.Context, autolink *types.AutoLink) error {
	const sqlQuery = `
		UPDATE autolinks
		SET
			 autolink_type = :autolink_type
			,autolink_pattern = :autolink_pattern
			,autolink_target_url = :autolink_target_url
			,autolink_updated = :autolink_updated
			,autolink_updated_by = :autolink_updated_by
		WHERE autolink_id = :autolink_id`

	dbAutolink := mapInternalAutoLink(autolink)

	dbAutolink.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)
	query, args, err := db.BindNamed(sqlQuery, dbAutolink)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update autolink")
	}

	return nil
}

func (s *AutoLinkStore) Find(ctx context.Context, id int64) (*types.AutoLink, error) {
	stmt := database.Builder.
		Select(autolinkColumnsWithID).
		From("autolinks").
		Where("autolink_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)
	var dst autoLink
	if err := db.GetContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to get autolink")
	}

	return mapAutoLink(&dst), nil
}

func (s *AutoLinkStore) List(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.AutoLinkFilter,
) ([]*types.AutoLink, error) {
	stmt := database.Builder.
		Select(autolinkColumnsWithID).
		From("autolinks").
		Where("(autolink_space_id = ? OR autolink_repo_id = ?)", spaceID, repoID).
		OrderBy("autolink_created asc").
		Limit(database.Limit(filter.Size)).
		Offset(database.Offset(filter.Page, filter.Size))

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(autolink_pattern) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*autoLink
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list autolinks")
	}

	return mapAutoLinks(dst), nil
}

func (s *AutoLinkStore) ListInScopes(
	ctx context.Context,
	repoID int64,
	spaceIDs []int64,
	filter *types.AutoLinkFilter,
) ([]*types.AutoLink, error) {
	stmt := database.Builder.
		Select(autolinkColumnsWithID).
		From("autolinks").
		Where(squirrel.Or{
			squirrel.Eq{"autolink_space_id": spaceIDs},
			squirrel.Eq{"autolink_repo_id": repoID},
		}).
		OrderBy("autolink_created asc").
		Limit(database.Limit(filter.Size)).
		Offset(database.Offset(filter.Page, filter.Size))

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(autolink_pattern) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*autoLink
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list autolinks in scopes")
	}

	return mapAutoLinks(dst), nil
}

func (s *AutoLinkStore) Count(
	ctx context.Context,
	spaceID, repoID *int64,
	filter *types.AutoLinkFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("autolinks").
		Where("(autolink_space_id = ? OR autolink_repo_id = ?)", spaceID, repoID)

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(autolink_pattern) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err := db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to count autolinks")
	}

	return count, nil
}

func (s *AutoLinkStore) CountInScopes(
	ctx context.Context,
	repoID int64,
	spaceIDs []int64,
	filter *types.AutoLinkFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("autolinks").
		Where(squirrel.Or{
			squirrel.Eq{"autolink_space_id": spaceIDs},
			squirrel.Eq{"autolink_repo_id": repoID},
		})

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(autolink_pattern) LIKE '%' || LOWER(?) || '%'", filter.Query)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err := db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed to count autolinks in scopes")
	}

	return count, nil
}

func (s *AutoLinkStore) Delete(ctx context.Context, autolinkID int64) error {
	stmt := database.Builder.
		Delete("autolinks").
		Where("autolink_id = ?", autolinkID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete autolink")
	}

	return nil
}

func mapInternalAutoLink(autolink *types.AutoLink) *autoLink {
	return &autoLink{
		ID:        autolink.ID,
		SpaceID:   null.IntFromPtr(autolink.SpaceID),
		RepoID:    null.IntFromPtr(autolink.RepoID),
		Type:      autolink.Type,
		Pattern:   autolink.Pattern,
		TargetURL: autolink.TargetURL,
		Created:   autolink.Created,
		Updated:   autolink.Updated,
		CreatedBy: autolink.CreatedBy,
		UpdatedBy: autolink.UpdatedBy,
	}
}

func mapAutoLink(internal *autoLink) *types.AutoLink {
	return &types.AutoLink{
		ID:        internal.ID,
		SpaceID:   internal.SpaceID.Ptr(),
		RepoID:    internal.RepoID.Ptr(),
		Type:      internal.Type,
		Pattern:   internal.Pattern,
		TargetURL: internal.TargetURL,
		Created:   internal.Created,
		Updated:   internal.Updated,
		CreatedBy: internal.CreatedBy,
		UpdatedBy: internal.UpdatedBy,
	}
}

func mapAutoLinks(dbAutolinks []*autoLink) []*types.AutoLink {
	result := make([]*types.AutoLink, len(dbAutolinks))

	for i, autolink := range dbAutolinks {
		result[i] = mapAutoLink(autolink)
	}

	return result
}

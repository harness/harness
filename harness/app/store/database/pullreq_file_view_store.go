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

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PullReqFileViewStore = (*PullReqFileViewStore)(nil)

// NewPullReqFileViewStore returns a new PullReqFileViewStore.
func NewPullReqFileViewStore(
	db *sqlx.DB,
) *PullReqFileViewStore {
	return &PullReqFileViewStore{
		db: db,
	}
}

// PullReqFileViewStore implements store.PullReqFileViewStore backed by a relational database.
type PullReqFileViewStore struct {
	db *sqlx.DB
}

type pullReqFileView struct {
	PullReqID   int64 `db:"pullreq_file_view_pullreq_id"`
	PrincipalID int64 `db:"pullreq_file_view_principal_id"`

	Path     string `db:"pullreq_file_view_path"`
	SHA      string `db:"pullreq_file_view_sha"`
	Obsolete bool   `db:"pullreq_file_view_obsolete"`

	Created int64 `db:"pullreq_file_view_created"`
	Updated int64 `db:"pullreq_file_view_updated"`
}

const (
	pullReqFileViewsColumn = `
		 pullreq_file_view_pullreq_id
		,pullreq_file_view_principal_id
		,pullreq_file_view_path
		,pullreq_file_view_sha
		,pullreq_file_view_obsolete
		,pullreq_file_view_created
		,pullreq_file_view_updated`
)

// Upsert inserts or updates the latest viewed sha for a file in a PR.
func (s *PullReqFileViewStore) Upsert(ctx context.Context, view *types.PullReqFileView) error {
	const sqlQuery = `
	INSERT INTO pullreq_file_views (
		 pullreq_file_view_pullreq_id
		,pullreq_file_view_principal_id
		,pullreq_file_view_path
		,pullreq_file_view_sha
		,pullreq_file_view_obsolete
		,pullreq_file_view_created
		,pullreq_file_view_updated
	) VALUES (
		 :pullreq_file_view_pullreq_id
		,:pullreq_file_view_principal_id
		,:pullreq_file_view_path
		,:pullreq_file_view_sha
		,:pullreq_file_view_obsolete
		,:pullreq_file_view_created
		,:pullreq_file_view_updated
	)
	ON CONFLICT (pullreq_file_view_pullreq_id, pullreq_file_view_principal_id, pullreq_file_view_path) DO
	UPDATE SET
		 pullreq_file_view_updated = :pullreq_file_view_updated
		,pullreq_file_view_sha = :pullreq_file_view_sha
		,pullreq_file_view_obsolete = :pullreq_file_view_obsolete
	RETURNING pullreq_file_view_created`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapToInternalPullreqFileView(view))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind pullreq file view object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&view.Created); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Upsert query failed")
	}

	return nil
}

// DeleteByFileForPrincipal deletes the entry for the specified PR, principal, and file.
func (s *PullReqFileViewStore) DeleteByFileForPrincipal(
	ctx context.Context,
	prID int64,
	principalID int64,
	filePath string,
) error {
	const sqlQuery = `
	DELETE from pullreq_file_views
	WHERE pullreq_file_view_pullreq_id = $1 AND
		  pullreq_file_view_principal_id = $2 AND
		  pullreq_file_view_path = $3`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, prID, principalID, filePath); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "delete query failed")
	}

	return nil
}

// MarkObsolete updates all entries of the files as obsolete for the PR.
func (s *PullReqFileViewStore) MarkObsolete(ctx context.Context, prID int64, filePaths []string) error {
	stmt := database.Builder.
		Update("pullreq_file_views").
		Set("pullreq_file_view_obsolete", true).
		Set("pullreq_file_view_updated", time.Now().UnixMilli()).
		Where("pullreq_file_view_pullreq_id = ?", prID).
		Where(squirrel.Eq{"pullreq_file_view_path": filePaths}).
		Where("pullreq_file_view_obsolete = ?", false)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to create sql query")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to execute update query")
	}

	return nil
}

// List lists all files marked as viewed by the user for the specified PR.
func (s *PullReqFileViewStore) List(
	ctx context.Context,
	prID int64,
	principalID int64,
) ([]*types.PullReqFileView, error) {
	stmt := database.Builder.
		Select(pullReqFileViewsColumn).
		From("pullreq_file_views").
		Where("pullreq_file_view_pullreq_id = ?", prID).
		Where("pullreq_file_view_principal_id = ?", principalID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*pullReqFileView
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to execute list query")
	}

	return mapToPullreqFileViews(dst), nil
}

func mapToInternalPullreqFileView(view *types.PullReqFileView) *pullReqFileView {
	return &pullReqFileView{
		PullReqID:   view.PullReqID,
		PrincipalID: view.PrincipalID,
		Path:        view.Path,
		SHA:         view.SHA,
		Obsolete:    view.Obsolete,
		Created:     view.Created,
		Updated:     view.Updated,
	}
}

func mapToPullreqFileView(view *pullReqFileView) *types.PullReqFileView {
	return &types.PullReqFileView{
		PullReqID:   view.PullReqID,
		PrincipalID: view.PrincipalID,
		Path:        view.Path,
		SHA:         view.SHA,
		Obsolete:    view.Obsolete,
		Created:     view.Created,
		Updated:     view.Updated,
	}
}

func mapToPullreqFileViews(views []*pullReqFileView) []*types.PullReqFileView {
	m := make([]*types.PullReqFileView, len(views))
	for i, view := range views {
		m[i] = mapToPullreqFileView(view)
	}
	return m
}

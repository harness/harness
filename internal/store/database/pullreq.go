// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PullReqStore = (*PullReqStore)(nil)

// NewPullReqStore returns a new PullReqStore.
func NewPullReqStore(db *sqlx.DB) *PullReqStore {
	return &PullReqStore{
		db: db,
	}
}

// PullReqStore implements store.PullReqStore backed by a relational database.
type PullReqStore struct {
	db *sqlx.DB
}

const (
	pullReqColumns = `
		 pullreq_id
		,pullreq_created_by
		,pullreq_created
		,pullreq_updated
		,pullreq_state
	    ,pullreq_number
		,pullreq_title
		,pullreq_description
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_merged_by
		,pullreq_merged
		,pullreq_merge_strategy
		,author.principal_id as "author_id"
		,author.principal_displayName as "author_name"
		,author.principal_email as "author_email"
		,merger.principal_id as "merger_id"
		,merger.principal_displayName as "merger_name"
		,merger.principal_email as "merger_email"`

	pullReqSelectBase = `
	SELECT` + pullReqColumns + `
	FROM pullreq
	INNER JOIN principals author on author.principal_id = pullreq_created_by
	LEFT  JOIN principals merger on merger.principal_id = pullreq_merged_by`
)

// Find finds the pull request by id.
func (s *PullReqStore) Find(ctx context.Context, id int64) (*types.PullReqInfo, error) {
	const sqlQuery = pullReqSelectBase + `
	WHERE pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.PullReqInfo)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

// FindByNumber finds the pull request by repo ID and pull request number.
func (s *PullReqStore) FindByNumber(ctx context.Context, repoID, number int64) (*types.PullReqInfo, error) {
	const sqlQuery = pullReqSelectBase + `
	WHERE pullreq_target_repo_id = $1 AND pullreq_number = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.PullReqInfo)
	if err := db.GetContext(ctx, dst, sqlQuery, repoID, number); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

// Create creates a new pull request.
func (s *PullReqStore) Create(ctx context.Context, pullReq *types.PullReq) error {
	const sqlQuery = `
	INSERT INTO pullreq (
		 pullreq_created_by
		,pullreq_created
		,pullreq_updated
		,pullreq_state
		,pullreq_number
		,pullreq_title
		,pullreq_description
		,pullreq_source_repo_id
		,pullreq_source_branch
		,pullreq_target_repo_id
		,pullreq_target_branch
		,pullreq_merged_by
		,pullreq_merged
		,pullreq_merge_strategy
	) values (
		 :pullreq_created_by
		,:pullreq_created
		,:pullreq_updated
		,:pullreq_state
		,:pullreq_number
		,:pullreq_title
		,:pullreq_description
		,:pullreq_source_repo_id
		,:pullreq_source_branch
		,:pullreq_target_repo_id
		,:pullreq_target_branch
		,:pullreq_merged_by
		,:pullreq_merged
		,:pullreq_merge_strategy
	) RETURNING pullreq_id
`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, pullReq)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pullReq object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&pullReq.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the pull request.
func (s *PullReqStore) Update(ctx context.Context, pullReq *types.PullReq) error {
	const sqlQuery = `
	UPDATE pullreq
	SET
		 pullreq_updated = :pullreq_updated
		,pullreq_state = :pullreq_state
		,pullreq_title = :pullreq_title
		,pullreq_description = :pullreq_description
		,pullreq_merged_by = :pullreq_merged_by
		,pullreq_merged = :pullreq_merged
		,pullreq_merge_strategy = :pullreq_merge_strategy
	WHERE pullreq_id = :pullreq_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, pullReq)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind pull request object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return nil
}

// Delete the pull request.
func (s *PullReqStore) Delete(ctx context.Context, id int64) error {
	const pullReqDelete = `DELETE FROM pullreq WHERE pullreq_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, pullReqDelete, id); err != nil {
		return processSQLErrorf(err, "the delete query failed")
	}

	return nil
}

// LastNumber return the number of the most recent pull request.
func (s *PullReqStore) LastNumber(ctx context.Context, repoID int64) (int64, error) {
	const sqlQuery = `select coalesce(max(pullreq_number), 0) from pullreq where pullreq_target_repo_id = $1 limit 1`

	db := dbtx.GetAccessor(ctx, s.db)

	row := db.QueryRowContext(ctx, sqlQuery, repoID)
	if err := row.Err(); err != nil {
		return 0, err
	}

	var lastNumber int64
	if err := row.Scan(&lastNumber); err != nil {
		return 0, err
	}

	return lastNumber, nil
}

// Count of pull requests for a repo.
func (s *PullReqStore) Count(ctx context.Context, repoID int64, opts *types.PullReqFilter) (int64, error) {
	stmt := builder.
		Select("count(*)").
		From("pullreq").
		Where("pullreq_target_repo_id = ?", repoID)

	if len(opts.States) == 1 {
		stmt = stmt.Where("pullreq_state = ?", opts.States[0])
	} else if len(opts.States) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_state": opts.States})
	}

	if opts.CreatedBy != 0 {
		stmt = stmt.Where("pullreq_created_by = ?", opts.CreatedBy)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of pull requests for a repo.
func (s *PullReqStore) List(ctx context.Context, repoID int64, opts *types.PullReqFilter) ([]*types.PullReqInfo, error) {
	stmt := builder.
		Select(pullReqColumns).
		From("pullreq").
		InnerJoin("principals author on author.principal_id = pullreq_created_by").
		LeftJoin("principals merger on merger.principal_id = pullreq_merged_by").
		Where("pullreq_target_repo_id = ?", repoID)

	if len(opts.States) == 1 {
		stmt = stmt.Where("pullreq_state = ?", opts.States[0])
	} else if len(opts.States) > 1 {
		stmt = stmt.Where(squirrel.Eq{"pullreq_state": opts.States})
	}

	if opts.Query != "" {
		stmt = stmt.Where("pullreq_title LIKE ?", "%"+opts.Query+"%")
	}

	if opts.CreatedBy > 0 {
		stmt = stmt.Where("pullreq_created_by = ?", opts.CreatedBy)
	}

	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	// NOTE: string concatenation is safe because the
	// order attribute is an enum and is not user-defined,
	// and is therefore not subject to injection attacks.
	switch opts.Sort {
	case enum.PullReqSortNumber, enum.PullReqSortNone:
		stmt = stmt.OrderBy("pullreq_number " + opts.Order.String())
	case enum.PullReqSortCreated:
		stmt = stmt.OrderBy("pullreq_created " + opts.Order.String())
	case enum.PullReqSortUpdated:
		stmt = stmt.OrderBy("pullreq_updated " + opts.Order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	dst := make([]*types.PullReqInfo, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

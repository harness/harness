// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.RepoStore = (*RepoStore)(nil)

// NewRepoStore returns a new RepoStore.
func NewRepoStore(db *sqlx.DB, pathCache store.PathCache) *RepoStore {
	return &RepoStore{
		db:        db,
		pathCache: pathCache,
	}
}

// RepoStore implements a store.RepoStore backed by a relational database.
type RepoStore struct {
	db        *sqlx.DB
	pathCache store.PathCache
}

const (
	repoColumnsForJoin = `
		repo_id
		,repo_version
		,repo_parent_id
		,repo_uid
		,paths.path_value AS repo_path
		,repo_description
		,repo_is_public
		,repo_created_by
		,repo_created
		,repo_updated
		,repo_git_uid
		,repo_default_branch
		,repo_pullreq_seq
		,repo_fork_id
		,repo_num_forks
		,repo_num_pulls
		,repo_num_closed_pulls
		,repo_num_open_pulls
		,repo_num_merged_pulls`

	repoSelectBaseWithJoin = `
		SELECT` + repoColumnsForJoin + `
		FROM repositories
		INNER JOIN paths
		ON repositories.repo_id=paths.path_repo_id AND paths.path_is_primary=true`
)

// Find finds the repo by id.
func (s *RepoStore) Find(ctx context.Context, id int64) (*types.Repository, error) {
	const sqlQuery = repoSelectBaseWithJoin + `
		WHERE repo_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Repository)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, processSQLErrorf(err, "Failed to find repo")
	}
	return dst, nil
}

// FindByRef finds the repo using the repoRef as either the id or the repo path.
func (s *RepoStore) FindByRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	// ASSUMPTION: digits only is not a valid repo path
	id, err := strconv.ParseInt(repoRef, 10, 64)
	if err != nil {
		var path *types.Path
		path, err = s.pathCache.Get(ctx, repoRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get path: %w", err)
		}

		if path.TargetType != enum.PathTargetTypeRepo {
			// IMPORTANT: expose as not found error as we didn't find the repo!
			return nil, fmt.Errorf("path is not targeting a repo - %w", store.ErrResourceNotFound)
		}

		id = path.TargetID
	}

	return s.Find(ctx, id)
}

// Create creates a new repository.
func (s *RepoStore) Create(ctx context.Context, repo *types.Repository) error {
	const sqlQuery = `
		INSERT INTO repositories (
			repo_version                      
			,repo_parent_id
			,repo_uid
			,repo_description
			,repo_is_public
			,repo_created_by
			,repo_created
			,repo_updated
			,repo_git_uid
			,repo_default_branch
			,repo_fork_id
			,repo_pullreq_seq
			,repo_num_forks
			,repo_num_pulls
			,repo_num_closed_pulls
			,repo_num_open_pulls
			,repo_num_merged_pulls
		) values (
			:repo_version
			,:repo_parent_id
			,:repo_uid
			,:repo_description
			,:repo_is_public
			,:repo_created_by
			,:repo_created
			,:repo_updated
			,:repo_git_uid
			,:repo_default_branch
			,:repo_fork_id
			,:repo_pullreq_seq
			,:repo_num_forks
			,:repo_num_pulls
			,:repo_num_closed_pulls
			,:repo_num_open_pulls
			,:repo_num_merged_pulls
		) RETURNING repo_id`

	db := dbtx.GetAccessor(ctx, s.db)

	// insert repo first so we get id
	query, arg, err := db.BindNamed(sqlQuery, repo)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&repo.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the repo details.
func (s *RepoStore) Update(ctx context.Context, repo *types.Repository) error {
	const sqlQuery = `
		UPDATE repositories
		SET
			repo_version			= :repo_version
			,repo_updated			= :repo_updated
			,repo_parent_id			= :repo_parent_id
			,repo_uid				= :repo_uid
			,repo_description		= :repo_description
			,repo_is_public			= :repo_is_public
			,repo_pullreq_seq		= :repo_pullreq_seq
			,repo_num_forks			= :repo_num_forks
			,repo_num_pulls			= :repo_num_pulls
			,repo_num_closed_pulls	= :repo_num_closed_pulls
			,repo_num_open_pulls	= :repo_num_open_pulls
			,repo_num_merged_pulls	= :repo_num_merged_pulls
		WHERE repo_id = :repo_id AND repo_version = :repo_version - 1`

	updatedAt := time.Now()

	repo.Version++
	repo.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, repo)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind repo object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to update repository")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return processSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return store.ErrVersionConflict
	}

	return nil
}

// UpdateOptLock updates the repository using the optimistic locking mechanism.
func (s *RepoStore) UpdateOptLock(ctx context.Context,
	repo *types.Repository,
	mutateFn func(repository *types.Repository) error) (*types.Repository, error) {
	for {
		dup := *repo

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, store.ErrVersionConflict) {
			return nil, err
		}

		repo, err = s.Find(ctx, repo.ID)
		if err != nil {
			return nil, err
		}
	}
}

// Delete the repository.
func (s *RepoStore) Delete(ctx context.Context, id int64) error {
	const repoDelete = `
		DELETE FROM repositories
		WHERE repo_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, repoDelete, id); err != nil {
		return processSQLErrorf(err, "the delete query failed")
	}

	return nil
}

// Count of repos in a space.
func (s *RepoStore) Count(ctx context.Context, parentID int64, opts *types.RepoFilter) (int64, error) {
	stmt := builder.
		Select("count(*)").
		From("repositories").
		Where("repo_parent_id = ?", parentID)

	if opts.Query != "" {
		stmt = stmt.Where("repo_uid LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
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

// List returns a list of repos in a space.
func (s *RepoStore) List(ctx context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error) {
	stmt := builder.
		Select(repoColumnsForJoin).
		From("repositories").
		InnerJoin("paths ON repositories.repo_id=paths.path_repo_id AND paths.path_is_primary=true").
		Where("repo_parent_id = ?", fmt.Sprint(parentID))

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(repo_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.RepoAttrUID, enum.RepoAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("repo_uid " + opts.Order.String())
		//TODO: Postgres does not support COLLATE NOCASE for UTF8
		// stmt = stmt.OrderBy("repo_uid COLLATE NOCASE " + opts.Order.String())
	case enum.RepoAttrCreated:
		stmt = stmt.OrderBy("repo_created " + opts.Order.String())
	case enum.RepoAttrUpdated:
		stmt = stmt.OrderBy("repo_updated " + opts.Order.String())
	case enum.RepoAttrPath:
		stmt = stmt.OrderBy("repo_path " + opts.Order.String())
		//TODO: Postgres does not support COLLATE NOCASE for UTF8
		// stmt = stmt.OrderBy("repo_path COLLATE NOCASE " + opts.Order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Repository{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

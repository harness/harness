// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.RepoStore = (*RepoStore)(nil)

// NewRepoStore returns a new RepoStore.
func NewRepoStore(db *sqlx.DB, pathTransformation store.PathTransformation) *RepoStore {
	return &RepoStore{
		db:                 db,
		pathTransformation: pathTransformation,
	}
}

// RepoStore implements a store.RepoStore backed by a relational database.
type RepoStore struct {
	db                 *sqlx.DB
	pathTransformation store.PathTransformation
}

// Find finds the repo by id.
func (s *RepoStore) Find(ctx context.Context, id int64) (*types.Repository, error) {
	dst := new(types.Repository)
	if err := s.db.GetContext(ctx, dst, repoSelectByID, id); err != nil {
		return nil, processSQLErrorf(err, "Failed to find repo")
	}
	return dst, nil
}

// FindByPath finds the repo by path.
func (s *RepoStore) FindByPath(ctx context.Context, path string) (*types.Repository, error) {
	// ensure we transform path before searching (otherwise casing might be wrong)
	pathUnique, err := s.pathTransformation(path)
	if err != nil {
		return nil, fmt.Errorf("failed to transform path '%s': %w", path, err)
	}

	dst := new(types.Repository)
	if err = s.db.GetContext(ctx, dst, repoSelectByPathUnique, pathUnique); err != nil {
		return nil, processSQLErrorf(err, "Failed to find repo by path")
	}

	return dst, nil
}

// FindByGitUID the repo by git uid.
func (s *RepoStore) FindByGitUID(ctx context.Context, gitUID string) (*types.Repository, error) {
	dst := new(types.Repository)
	if err := s.db.GetContext(ctx, dst, repoSelectByGitUID, gitUID); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

func (s *RepoStore) FindRepoFromRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	// check if ref is repoId - ASSUMPTION: digit only is no valid repo name
	id, err := strconv.ParseInt(repoRef, 10, 64)
	if err == nil {
		return s.Find(ctx, id)
	}

	return s.FindByPath(ctx, repoRef)
}

// Create creates a new repository.
func (s *RepoStore) Create(ctx context.Context, repo *types.Repository) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// insert repo first so we get id
	query, arg, err := s.db.BindNamed(repoInsert, repo)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind repo object")
	}

	if err = tx.QueryRow(query, arg...).Scan(&repo.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	// Get parent path (repo always has a parent)
	parentPath, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, repo.ParentID)
	if err != nil {
		return errors.Wrap(err, "Failed to find path of parent space")
	}

	// All existing paths are valid, repo uid is assumed to be valid => new path is valid
	path := paths.Concatinate(parentPath.Value, repo.UID)

	// create path only once we know the id of the repo
	p := &types.Path{
		TargetType: enum.PathTargetTypeRepo,
		TargetID:   repo.ID,
		IsAlias:    false,
		Value:      path,
		CreatedBy:  repo.CreatedBy,
		Created:    repo.Created,
		Updated:    repo.Updated,
	}

	if err = CreatePathTx(ctx, s.db, tx, p, s.pathTransformation); err != nil {
		return errors.Wrap(err, "Failed to create primary path of repo")
	}

	// commit
	if err = tx.Commit(); err != nil {
		return processSQLErrorf(err, "Failed to commit transaction")
	}

	// update path in repo object
	repo.Path = p.Value

	return nil
}

// Move moves an existing space.
func (s *RepoStore) Move(ctx context.Context, principalID int64, repoID int64, newParentID int64, newName string,
	keepAsAlias bool) (*types.Repository, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback() // should we take care about rollbacks errors?
	}(tx)

	// get current path of repo
	currentPath, err := FindPathTx(ctx, tx, enum.PathTargetTypeRepo, repoID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find the primary path of the repo")
	}

	// get path of new parent space
	spacePath, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, newParentID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find the primary path of the new space")
	}

	newPath := paths.Concatinate(spacePath.Value, newName)

	// path is exactly the same => nothing to do
	if newPath == currentPath.Value {
		return nil, store.ErrNoChangeInRequestedMove
	}

	p := &types.Path{
		TargetType: enum.PathTargetTypeRepo,
		TargetID:   repoID,
		IsAlias:    false,
		Value:      newPath,
		CreatedBy:  principalID,
		Created:    time.Now().UnixMilli(),
		Updated:    time.Now().UnixMilli(),
	}

	// replace the primary path (also updates all child primary paths)
	if err = ReplacePathTx(ctx, s.db, tx, p, keepAsAlias, s.pathTransformation); err != nil {
		return nil, errors.Wrap(err, "Failed to update the primary path of the repo")
	}

	// Rename the repo itself
	if _, err = tx.ExecContext(ctx, repoUpdateUIDAndParentID, newName, newParentID, repoID); err != nil {
		return nil, processSQLErrorf(err, "Query for renaming and updating the space id failed")
	}

	// TODO: return repo as part of rename db operation?
	dst := new(types.Repository)
	if err = tx.GetContext(ctx, dst, repoSelectByID, repoID); err != nil {
		return nil, processSQLErrorf(err, "Select query to get the repo's latest state failed")
	}

	// commit
	if err = tx.Commit(); err != nil {
		return nil, processSQLErrorf(err, "Failed to commit transaction")
	}

	return dst, nil
}

// Update updates the repo details.
func (s *RepoStore) Update(ctx context.Context, repo *types.Repository) error {
	const repoUpdate = `
	UPDATE repositories
	SET
	 repo_version           = :repo_version
	,repo_description		= :repo_description
	,repo_is_public			= :repo_is_public
	,repo_updated			= :repo_updated
	,repo_pullreq_seq		= :repo_pullreq_seq
	,repo_num_forks			= :repo_num_forks
	,repo_num_pulls			= :repo_num_pulls
	,repo_num_closed_pulls	= :repo_num_closed_pulls
	,repo_num_open_pulls	= :repo_num_open_pulls
	WHERE repo_id = :repo_id AND repo_version = :repo_version - 1`

	updatedAt := time.Now()

	repo.Version++
	repo.Updated = updatedAt.UnixMilli()

	query, arg, err := s.db.BindNamed(repoUpdate, repo)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind repo object")
	}

	result, err := s.db.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to update repository")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return processSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return store.ErrConflict
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
		if !errors.Is(err, store.ErrConflict) {
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

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// delete all paths
	if err = DeleteAllPaths(ctx, tx, enum.PathTargetTypeRepo, id); err != nil {
		return fmt.Errorf("failed to delete all paths of the repo: %w", err)
	}

	// delete the repo
	if _, err = tx.ExecContext(ctx, repoDelete, id); err != nil {
		return processSQLErrorf(err, "the delete query failed")
	}

	if err = tx.Commit(); err != nil {
		return processSQLErrorf(err, "failed to commit transaction")
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

	var count int64
	err = s.db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// List returns a list of repos in a space.
func (s *RepoStore) List(ctx context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error) {
	dst := []*types.Repository{}

	// construct the sql statement.
	stmt := builder.
		Select("repositories.*,paths.path_value AS repo_path").
		From("repositories").
		InnerJoin("paths ON repositories.repo_id=paths.path_target_id AND paths.path_target_type='repo' "+
			"AND paths.path_is_alias=false").
		Where("repo_parent_id = ?", fmt.Sprint(parentID))

	if opts.Query != "" {
		stmt = stmt.Where("repo_uid LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
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

	if err = s.db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// CountPaths returns a count of all paths of a repo.
func (s *RepoStore) CountPaths(ctx context.Context, id int64, opts *types.PathFilter) (int64, error) {
	return CountPaths(ctx, s.db, enum.PathTargetTypeRepo, id, opts)
}

// ListPaths returns a list of all paths of a repo.
func (s *RepoStore) ListPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error) {
	return ListPaths(ctx, s.db, enum.PathTargetTypeRepo, id, opts)
}

// CreatePath creates an alias for a repository.
func (s *RepoStore) CreatePath(ctx context.Context, repoID int64, params *types.PathParams) (*types.Path, error) {
	p := &types.Path{
		TargetType: enum.PathTargetTypeRepo,
		TargetID:   repoID,
		IsAlias:    true,

		// get remaining info from params
		Value:     params.Path,
		CreatedBy: params.CreatedBy,
		Created:   params.Created,
		Updated:   params.Updated,
	}

	return p, CreateAliasPath(ctx, s.db, p, s.pathTransformation)
}

// DeletePath an alias of a repository.
func (s *RepoStore) DeletePath(ctx context.Context, repoID int64, pathID int64) error {
	return DeletePath(ctx, s.db, pathID)
}

const repoSelectBase = `
SELECT
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
,repo_fork_id
,repo_num_forks
,repo_num_pulls
,repo_num_closed_pulls
,repo_num_open_pulls
`

const repoSelectBaseWithJoin = repoSelectBase + `
FROM repositories
INNER JOIN paths
ON repositories.repo_id=paths.path_target_id AND paths.path_target_type='repo' AND paths.path_is_alias=false
`

const repoSelectByID = repoSelectBaseWithJoin + `
WHERE repo_id = $1
`

const repoSelectByGitUID = repoSelectBaseWithJoin + `
WHERE repo_git_uid = $1
`

const repoSelectByPathUnique = repoSelectBase + `
FROM paths paths1
INNER JOIN repositories ON repositories.repo_id=paths1.path_target_id AND paths1.path_target_type='repo' 
  AND paths1.path_value_unique = $1
INNER JOIN paths ON repositories.repo_id=paths.path_target_id
  AND paths.path_target_type='repo' AND paths.path_is_alias=false
`

// TODO: do we have to worry about SQL injection for description?
const repoInsert = `
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
) RETURNING repo_id`

const repoUpdateUIDAndParentID = `
UPDATE repositories
SET
repo_uid        = $1
,repo_parent_id = $2
WHERE repo_id   = $3
`

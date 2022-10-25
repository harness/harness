// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/pkg/errors"

	"github.com/jmoiron/sqlx"
)

var _ store.RepoStore = (*RepoStore)(nil)

// Returns a new RepoStore.
func NewRepoStore(db *sqlx.DB) *RepoStore {
	return &RepoStore{db}
}

// Implements a RepoStore backed by a relational database.
type RepoStore struct {
	db *sqlx.DB
}

// Finds the repo by id.
func (s *RepoStore) Find(ctx context.Context, id int64) (*types.Repository, error) {
	dst := new(types.Repository)
	if err := s.db.GetContext(ctx, dst, repoSelectByID, id); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

// Finds the repo by path.
func (s *RepoStore) FindByPath(ctx context.Context, path string) (*types.Repository, error) {
	dst := new(types.Repository)
	if err := s.db.GetContext(ctx, dst, repoSelectByPath, path); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
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
	parentPath, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, repo.SpaceID)
	if err != nil {
		return errors.Wrap(err, "Failed to find path of parent space")
	}

	// all existing paths are valid, repo name is assumed to be valid.
	path := paths.Concatinate(parentPath.Value, repo.PathName)

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

	if err = CreatePathTx(ctx, s.db, tx, p); err != nil {
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
func (s *RepoStore) Move(ctx context.Context, principalID int64, repoID int64, newSpaceID int64, newName string,
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
	spacePath, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, newSpaceID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find the primary path of the new space")
	}
	newPath := paths.Concatinate(spacePath.Value, newName)

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
	if err = ReplacePathTx(ctx, s.db, tx, p, keepAsAlias); err != nil {
		return nil, errors.Wrap(err, "Failed to update the primary path of the repo")
	}

	// Rename the repo itself
	if _, err = tx.ExecContext(ctx, repoUpdateNameAndSpaceID, newName, newSpaceID, repoID); err != nil {
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

// Updates the repo details.
func (s *RepoStore) Update(ctx context.Context, repo *types.Repository) error {
	query, arg, err := s.db.BindNamed(repoUpdate, repo)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind repo object")
	}

	if _, err = s.db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return nil
}

// Delete the repository.
func (s *RepoStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// delete all paths
	err = DeleteAllPaths(ctx, tx, enum.PathTargetTypeRepo, id)
	if err != nil {
		return errors.Wrap(err, "Failed to delete all paths of the repo")
	}

	// delete the repo
	if _, err = tx.ExecContext(ctx, repoDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}

	if err = tx.Commit(); err != nil {
		return processSQLErrorf(err, "Failed to commit transaction")
	}

	return nil
}

// Count of repos in a space.
func (s *RepoStore) Count(ctx context.Context, spaceID int64, opts *types.RepoFilter) (int64, error) {
	stmt := builder.
		Select("count(*)").
		From("repositories").
		Where("repo_spaceId = ?", spaceID)

	if opts.Query != "" {
		stmt = stmt.Where("repo_pathName LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
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
func (s *RepoStore) List(ctx context.Context, spaceID int64, opts *types.RepoFilter) ([]*types.Repository, error) {
	dst := []*types.Repository{}

	// construct the sql statement.
	stmt := builder.
		Select("repositories.*,path_value AS repo_path").
		From("repositories").
		InnerJoin("paths ON repositories.repo_id=paths.path_targetId AND paths.path_targetType='repo' "+
			"AND paths.path_isAlias=0").
		Where("repo_spaceId = ?", fmt.Sprint(spaceID))

	if opts.Query != "" {
		stmt = stmt.Where("repo_pathName LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
	}

	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.RepoAttrName, enum.RepoAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("repo_name COLLATE NOCASE " + opts.Order.String())
	case enum.RepoAttrCreated:
		stmt = stmt.OrderBy("repo_created " + opts.Order.String())
	case enum.RepoAttrUpdated:
		stmt = stmt.OrderBy("repo_updated " + opts.Order.String())
	case enum.RepoAttrPathName:
		stmt = stmt.OrderBy("repo_pathName COLLATE NOCASE " + opts.Order.String())
	case enum.RepoAttrPath:
		stmt = stmt.OrderBy("repo_path COLLATE NOCASE " + opts.Order.String())
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

		// get remaining infor from params
		Value:     params.Path,
		CreatedBy: params.CreatedBy,
		Created:   params.Created,
		Updated:   params.Updated,
	}

	return p, CreateAliasPath(ctx, s.db, p)
}

// DeletePath an alias of a repository.
func (s *RepoStore) DeletePath(ctx context.Context, repoID int64, pathID int64) error {
	return DeletePath(ctx, s.db, pathID)
}

const repoSelectBase = `
SELECT
repo_id
,repo_pathName
,repo_spaceId
,paths.path_value AS repo_path
,repo_name
,repo_description
,repo_isPublic
,repo_createdBy
,repo_created
,repo_updated
,repo_gitUid
,repo_defaultBranch
,repo_forkId
,repo_numForks
,repo_numPulls
,repo_numClosedPulls
,repo_numOpenPulls
`

const repoSelectBaseWithJoin = repoSelectBase + `
FROM repositories
INNER JOIN paths
ON repositories.repo_id=paths.path_targetId AND paths.path_targetType='repo' AND paths.path_isAlias=0
`

const repoSelectByID = repoSelectBaseWithJoin + `
WHERE repo_id = $1
`

const repoSelectByPath = repoSelectBase + `
FROM paths paths1
INNER JOIN repositories ON repositories.repo_id=paths1.path_targetId AND paths1.path_targetType='repo' 
AND paths1.path_value = $1
INNER JOIN paths ON repositories.repo_id=paths.path_targetId AND paths.path_targetType='repo' AND paths.path_isAlias=0
`

const repoDelete = `
DELETE FROM repositories
WHERE repo_id = $1
`

// TODO: do we have to worry about SQL injection for description?
const repoInsert = `
INSERT INTO repositories (
	repo_pathName
	,repo_spaceId
	,repo_name
	,repo_description
	,repo_isPublic
	,repo_createdBy
	,repo_created
	,repo_updated
	,repo_gitUid
	,repo_defaultBranch
	,repo_forkId
	,repo_numForks
	,repo_numPulls
	,repo_numClosedPulls
	,repo_numOpenPulls
) values (
	:repo_pathName
	,:repo_spaceId
	,:repo_name
	,:repo_description
	,:repo_isPublic
	,:repo_createdBy
	,:repo_created
	,:repo_updated
	,:repo_gitUid
	,:repo_defaultBranch
	,:repo_forkId
	,:repo_numForks
	,:repo_numPulls
	,:repo_numClosedPulls
	,:repo_numOpenPulls
) RETURNING repo_id
`

const repoUpdate = `
UPDATE repositories
SET
repo_name		        = :repo_name
,repo_description		= :repo_description
,repo_isPublic			= :repo_isPublic
,repo_updated			= :repo_updated
,repo_numForks			= :repo_numForks
,repo_numPulls			= :repo_numPulls
,repo_numClosedPulls	= :repo_numClosedPulls
,repo_numOpenPulls		= :repo_numOpenPulls
WHERE repo_id = :repo_id
`

const repoUpdateNameAndSpaceID = `
UPDATE repositories
SET
repo_pathName = $1
,repo_spaceId = $2
WHERE repo_id = $3
`

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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.RepoStore = (*RepoStore)(nil)

// NewRepoStore returns a new RepoStore.
func NewRepoStore(
	db *sqlx.DB,
	spacePathCache store.SpacePathCache,
	spacePathStore store.SpacePathStore,
	spaceStore store.SpaceStore,
) *RepoStore {
	return &RepoStore{
		db:             db,
		spacePathCache: spacePathCache,
		spacePathStore: spacePathStore,
		spaceStore:     spaceStore,
	}
}

// RepoStore implements a store.RepoStore backed by a relational database.
type RepoStore struct {
	db             *sqlx.DB
	spacePathCache store.SpacePathCache
	spacePathStore store.SpacePathStore
	spaceStore     store.SpaceStore
}

type repository struct {
	// TODO: int64 ID doesn't match DB
	ID          int64    `db:"repo_id"`
	Version     int64    `db:"repo_version"`
	ParentID    int64    `db:"repo_parent_id"`
	Identifier  string   `db:"repo_uid"`
	Description string   `db:"repo_description"`
	CreatedBy   int64    `db:"repo_created_by"`
	Created     int64    `db:"repo_created"`
	Updated     int64    `db:"repo_updated"`
	Deleted     null.Int `db:"repo_deleted"`

	Size        int64 `db:"repo_size"`
	SizeUpdated int64 `db:"repo_size_updated"`

	GitUID        string `db:"repo_git_uid"`
	DefaultBranch string `db:"repo_default_branch"`
	ForkID        int64  `db:"repo_fork_id"`
	PullReqSeq    int64  `db:"repo_pullreq_seq"`

	NumForks       int `db:"repo_num_forks"`
	NumPulls       int `db:"repo_num_pulls"`
	NumClosedPulls int `db:"repo_num_closed_pulls"`
	NumOpenPulls   int `db:"repo_num_open_pulls"`
	NumMergedPulls int `db:"repo_num_merged_pulls"`

	State   enum.RepoState `db:"repo_state"`
	IsEmpty bool           `db:"repo_is_empty"`
}

const (
	repoColumnsForJoin = `
		repo_id
		,repo_version
		,repo_parent_id
		,repo_uid
		,repo_description
		,repo_created_by
		,repo_created
		,repo_updated
		,repo_deleted
		,repo_size
		,repo_size_updated
		,repo_git_uid
		,repo_default_branch
		,repo_pullreq_seq
		,repo_fork_id
		,repo_num_forks
		,repo_num_pulls
		,repo_num_closed_pulls
		,repo_num_open_pulls
		,repo_num_merged_pulls
		,repo_state
		,repo_is_empty`
)

// Find finds the repo by id.
func (s *RepoStore) Find(ctx context.Context, id int64) (*types.Repository, error) {
	return s.find(ctx, id, nil)
}

// find is a wrapper to find a repo by id w/o deleted timestamp.
func (s *RepoStore) find(ctx context.Context, id int64, deletedAt *int64) (*types.Repository, error) {
	stmt := database.Builder.
		Select(repoColumnsForJoin).
		From("repositories").
		Where("repo_id = ?", id)

	if deletedAt != nil {
		stmt = stmt.Where("repo_deleted = ?", *deletedAt)
	} else {
		stmt = stmt.Where("repo_deleted IS NULL")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(repository)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return s.mapToRepo(ctx, dst)
}

func (s *RepoStore) findByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
	deletedAt *int64,
) (*types.Repository, error) {
	stmt := database.Builder.
		Select(repoColumnsForJoin).
		From("repositories").
		Where("repo_parent_id = ? AND LOWER(repo_uid) = ?", spaceID, strings.ToLower(identifier))

	if deletedAt != nil {
		stmt = stmt.Where("repo_deleted = ?", *deletedAt)
	} else {
		stmt = stmt.Where("repo_deleted IS NULL")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(repository)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return s.mapToRepo(ctx, dst)
}

func (s *RepoStore) findByRef(ctx context.Context, repoRef string, deletedAt *int64) (*types.Repository, error) {
	// ASSUMPTION: digits only is not a valid repo path
	id, err := strconv.ParseInt(repoRef, 10, 64)
	if err != nil {
		spacePath, repoIdentifier, err := paths.DisectLeaf(repoRef)
		if err != nil {
			return nil, fmt.Errorf("failed to disect leaf for path '%s': %w", repoRef, err)
		}
		pathObject, err := s.spacePathCache.Get(ctx, spacePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get space path: %w", err)
		}

		return s.findByIdentifier(ctx, pathObject.SpaceID, repoIdentifier, deletedAt)
	}
	return s.find(ctx, id, deletedAt)
}

// FindByRef finds the repo using the repoRef as either the id or the repo path.
func (s *RepoStore) FindByRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	return s.findByRef(ctx, repoRef, nil)
}

// FindByRefAndDeletedAt finds the repo using the repoRef and deleted timestamp.
func (s *RepoStore) FindByRefAndDeletedAt(
	ctx context.Context,
	repoRef string,
	deletedAt int64,
) (*types.Repository, error) {
	return s.findByRef(ctx, repoRef, &deletedAt)
}

// Create creates a new repository.
func (s *RepoStore) Create(ctx context.Context, repo *types.Repository) error {
	const sqlQuery = `
		INSERT INTO repositories (
			repo_version                      
			,repo_parent_id
			,repo_uid
			,repo_description
			,repo_created_by
			,repo_created
			,repo_updated
			,repo_deleted
			,repo_size
			,repo_size_updated	
			,repo_git_uid
			,repo_default_branch
			,repo_fork_id
			,repo_pullreq_seq
			,repo_num_forks
			,repo_num_pulls
			,repo_num_closed_pulls
			,repo_num_open_pulls
			,repo_num_merged_pulls
			,repo_state
			,repo_is_empty
		) values (
			:repo_version
			,:repo_parent_id
			,:repo_uid
			,:repo_description
			,:repo_created_by
			,:repo_created
			,:repo_updated
			,:repo_deleted
			,:repo_size
			,:repo_size_updated
			,:repo_git_uid
			,:repo_default_branch
			,:repo_fork_id
			,:repo_pullreq_seq
			,:repo_num_forks
			,:repo_num_pulls
			,:repo_num_closed_pulls
			,:repo_num_open_pulls
			,:repo_num_merged_pulls
			,:repo_state
			,:repo_is_empty
		) RETURNING repo_id`

	db := dbtx.GetAccessor(ctx, s.db)

	// insert repo first so we get id
	query, arg, err := db.BindNamed(sqlQuery, mapToInternalRepo(repo))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&repo.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	repo.Path, err = s.getRepoPath(ctx, repo.ParentID, repo.Identifier)
	if err != nil {
		return err
	}

	return nil
}

// Update updates the repo details.
func (s *RepoStore) Update(ctx context.Context, repo *types.Repository) error {
	const sqlQuery = `
		UPDATE repositories
		SET
			 repo_version = :repo_version
			,repo_updated = :repo_updated
			,repo_deleted = :repo_deleted
			,repo_parent_id = :repo_parent_id
			,repo_uid = :repo_uid
			,repo_git_uid = :repo_git_uid
			,repo_description = :repo_description
			,repo_default_branch = :repo_default_branch
			,repo_pullreq_seq = :repo_pullreq_seq
			,repo_num_forks = :repo_num_forks
			,repo_num_pulls = :repo_num_pulls
			,repo_num_closed_pulls = :repo_num_closed_pulls
			,repo_num_open_pulls = :repo_num_open_pulls
			,repo_num_merged_pulls = :repo_num_merged_pulls
			,repo_state = :repo_state
			,repo_is_empty = :repo_is_empty
		WHERE repo_id = :repo_id AND repo_version = :repo_version - 1`

	dbRepo := mapToInternalRepo(repo)

	// update Version (used for optimistic locking) and Updated time
	dbRepo.Version++
	dbRepo.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbRepo)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update repository")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	repo.Version = dbRepo.Version
	repo.Updated = dbRepo.Updated

	// update path in case parent/identifier changed (its most likely cached anyway)
	repo.Path, err = s.getRepoPath(ctx, repo.ParentID, repo.Identifier)
	if err != nil {
		return err
	}

	return nil
}

// UpdateSize updates the size of a specific repository in the database (size is in KiB).
func (s *RepoStore) UpdateSize(ctx context.Context, id int64, sizeInKiB int64) error {
	stmt := database.Builder.
		Update("repositories").
		Set("repo_size", sizeInKiB).
		Set("repo_size_updated", time.Now().UnixMilli()).
		Where("repo_id = ? AND repo_deleted IS NULL", id)

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to create sql query")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update repo size")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return fmt.Errorf("repo %d size not updated: %w", id, gitness_store.ErrResourceNotFound)
	}

	return nil
}

// GetSize returns the repo size.
func (s *RepoStore) GetSize(ctx context.Context, id int64) (int64, error) {
	query := "SELECT repo_size FROM repositories WHERE repo_id = $1 AND repo_deleted IS NULL;"
	db := dbtx.GetAccessor(ctx, s.db)

	var size int64
	if err := db.GetContext(ctx, &size, query, id); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to get repo size")
	}
	return size, nil
}

// UpdateOptLock updates the active repository using the optimistic locking mechanism.
func (s *RepoStore) UpdateOptLock(
	ctx context.Context,
	repo *types.Repository,
	mutateFn func(repository *types.Repository) error,
) (*types.Repository, error) {
	return s.updateOptLock(
		ctx,
		repo,
		func(r *types.Repository) error {
			if repo.Deleted != nil {
				return gitness_store.ErrResourceNotFound
			}
			return mutateFn(r)
		},
	)
}

// UpdateDeletedOptLock updates a deleted repository using the optimistic locking mechanism.
func (s *RepoStore) updateDeletedOptLock(ctx context.Context,
	repo *types.Repository,
	mutateFn func(repository *types.Repository) error,
) (*types.Repository, error) {
	return s.updateOptLock(
		ctx,
		repo,
		func(r *types.Repository) error {
			if repo.Deleted == nil {
				return gitness_store.ErrResourceNotFound
			}
			return mutateFn(r)
		},
	)
}

// updateOptLock updates the repository using the optimistic locking mechanism.
func (s *RepoStore) updateOptLock(
	ctx context.Context,
	repo *types.Repository,
	mutateFn func(repository *types.Repository) error,
) (*types.Repository, error) {
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
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		repo, err = s.find(ctx, repo.ID, repo.Deleted)
		if err != nil {
			return nil, err
		}
	}
}

// SoftDelete deletes a repo softly by setting the deleted timestamp.
func (s *RepoStore) SoftDelete(ctx context.Context, repo *types.Repository, deletedAt int64) error {
	_, err := s.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		r.Deleted = &deletedAt
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to soft delete repo: %w", err)
	}
	return nil
}

// Purge deletes the repo permanently.
func (s *RepoStore) Purge(ctx context.Context, id int64, deletedAt *int64) error {
	stmt := database.Builder.
		Delete("repositories").
		Where("repo_id = ?", id)

	if deletedAt != nil {
		stmt = stmt.Where("repo_deleted = ?", *deletedAt)
	} else {
		stmt = stmt.Where("repo_deleted IS NULL")
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge repo query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

// Restore restores a deleted repo.
func (s *RepoStore) Restore(
	ctx context.Context,
	repo *types.Repository,
	newIdentifier *string,
	newParentID *int64,
) (*types.Repository, error) {
	repo, err := s.updateDeletedOptLock(ctx, repo, func(r *types.Repository) error {
		r.Deleted = nil
		if newIdentifier != nil {
			r.Identifier = *newIdentifier
		}
		if newParentID != nil {
			r.ParentID = *newParentID
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// Count of active repos in a space. if parentID (space) is zero then it will count all repositories in the system.
// Count deleted repos requires opts.DeletedBeforeOrAt filter.
func (s *RepoStore) Count(
	ctx context.Context,
	parentID int64,
	filter *types.RepoFilter,
) (int64, error) {
	if filter.Recursive {
		return s.countAll(ctx, parentID, filter)
	}
	return s.count(ctx, parentID, filter)
}

func (s *RepoStore) count(
	ctx context.Context,
	parentID int64,
	filter *types.RepoFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("repositories")

	if parentID > 0 {
		stmt = stmt.Where("repo_parent_id = ?", parentID)
	}

	stmt = applyQueryFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (s *RepoStore) countAll(
	ctx context.Context,
	parentID int64,
	filter *types.RepoFilter,
) (int64, error) {
	query := `WITH RECURSIVE SpaceHierarchy AS (
    SELECT space_id, space_parent_id
    FROM spaces
    WHERE space_id = $1
    
    UNION
    
    SELECT s.space_id, s.space_parent_id
    FROM spaces s
    JOIN SpaceHierarchy h ON s.space_parent_id = h.space_id
)
SELECT space_id
FROM SpaceHierarchy h1;`

	db := dbtx.GetAccessor(ctx, s.db)

	var spaceIDs []int64
	if err := db.SelectContext(ctx, &spaceIDs, query, parentID); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to retrieve spaces")
	}

	stmt := database.Builder.
		Select("COUNT(repo_id)").
		From("repositories").
		Where(squirrel.Eq{"repo_parent_id": spaceIDs})

	stmt = applyQueryFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	var numRepos int64
	if err := db.GetContext(ctx, &numRepos, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to count repositories")
	}

	return numRepos, nil
}

// List returns a list of active repos in a space.
// With "DeletedBeforeOrAt" filter, lists deleted repos by opts.DeletedBeforeOrAt.
func (s *RepoStore) List(
	ctx context.Context,
	parentID int64,
	filter *types.RepoFilter,
) ([]*types.Repository, error) {
	if filter.Recursive {
		return s.listAll(ctx, parentID, filter)
	}
	return s.list(ctx, parentID, filter)
}

func (s *RepoStore) list(
	ctx context.Context,
	parentID int64,
	filter *types.RepoFilter,
) ([]*types.Repository, error) {
	stmt := database.Builder.
		Select(repoColumnsForJoin).
		From("repositories").
		Where("repo_parent_id = ?", fmt.Sprint(parentID))

	stmt = applyQueryFilter(stmt, filter)
	stmt = applySortFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*repository{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToRepos(ctx, dst)
}

func (s *RepoStore) listAll(
	ctx context.Context,
	parentID int64,
	filter *types.RepoFilter,
) ([]*types.Repository, error) {
	where := `WITH RECURSIVE SpaceHierarchy AS (
    SELECT space_id, space_parent_id
    FROM spaces
    WHERE space_id = $1
    
    UNION
    
    SELECT s.space_id, s.space_parent_id
    FROM spaces s
    JOIN SpaceHierarchy h ON s.space_parent_id = h.space_id
)
SELECT space_id
FROM SpaceHierarchy h1;`

	db := dbtx.GetAccessor(ctx, s.db)

	var spaceIDs []int64
	if err := db.SelectContext(ctx, &spaceIDs, where, parentID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to retrieve spaces")
	}

	stmt := database.Builder.
		Select(repoColumnsForJoin).
		From("repositories").
		Where(squirrel.Eq{"repo_parent_id": spaceIDs})

	stmt = applyQueryFilter(stmt, filter)
	stmt = applySortFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}
	repos := []*repository{}
	if err := db.SelectContext(ctx, &repos, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to count repositories")
	}

	return s.mapToRepos(ctx, repos)
}

type repoSize struct {
	ID          int64  `db:"repo_id"`
	GitUID      string `db:"repo_git_uid"`
	Size        int64  `db:"repo_size"`
	SizeUpdated int64  `db:"repo_size_updated"`
}

func (s *RepoStore) ListSizeInfos(ctx context.Context) ([]*types.RepositorySizeInfo, error) {
	stmt := database.Builder.
		Select("repo_id", "repo_git_uid", "repo_size", "repo_size_updated").
		From("repositories").
		Where("repo_deleted IS NULL")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*repoSize{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToRepoSizes(dst), nil
}

func (s *RepoStore) mapToRepo(
	ctx context.Context,
	in *repository,
) (*types.Repository, error) {
	var err error
	res := &types.Repository{
		ID:             in.ID,
		Version:        in.Version,
		ParentID:       in.ParentID,
		Identifier:     in.Identifier,
		Description:    in.Description,
		Created:        in.Created,
		CreatedBy:      in.CreatedBy,
		Updated:        in.Updated,
		Deleted:        in.Deleted.Ptr(),
		Size:           in.Size,
		SizeUpdated:    in.SizeUpdated,
		GitUID:         in.GitUID,
		DefaultBranch:  in.DefaultBranch,
		ForkID:         in.ForkID,
		PullReqSeq:     in.PullReqSeq,
		NumForks:       in.NumForks,
		NumPulls:       in.NumPulls,
		NumClosedPulls: in.NumClosedPulls,
		NumOpenPulls:   in.NumOpenPulls,
		NumMergedPulls: in.NumMergedPulls,
		State:          in.State,
		IsEmpty:        in.IsEmpty,
		// Path: is set below
	}

	res.Path, err = s.getRepoPath(ctx, in.ParentID, in.Identifier)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *RepoStore) getRepoPath(ctx context.Context, parentID int64, repoIdentifier string) (string, error) {
	spacePath, err := s.spacePathStore.FindPrimaryBySpaceID(ctx, parentID)
	// try to re-create the space path if was soft deleted.
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		return getPathForDeletedSpace(ctx, s.db, parentID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get primary path for space %d: %w", parentID, err)
	}
	return paths.Concatenate(spacePath.Value, repoIdentifier), nil
}

func (s *RepoStore) mapToRepos(
	ctx context.Context,
	repos []*repository,
) ([]*types.Repository, error) {
	var err error
	res := make([]*types.Repository, len(repos))
	for i := range repos {
		res[i], err = s.mapToRepo(ctx, repos[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *RepoStore) mapToRepoSize(
	in *repoSize,
) *types.RepositorySizeInfo {
	return &types.RepositorySizeInfo{
		ID:          in.ID,
		GitUID:      in.GitUID,
		Size:        in.Size,
		SizeUpdated: in.SizeUpdated,
	}
}

func (s *RepoStore) mapToRepoSizes(
	repoSizes []*repoSize,
) []*types.RepositorySizeInfo {
	res := make([]*types.RepositorySizeInfo, len(repoSizes))
	for i := range repoSizes {
		res[i] = s.mapToRepoSize(repoSizes[i])
	}
	return res
}

func mapToInternalRepo(in *types.Repository) *repository {
	return &repository{
		ID:             in.ID,
		Version:        in.Version,
		ParentID:       in.ParentID,
		Identifier:     in.Identifier,
		Description:    in.Description,
		Created:        in.Created,
		CreatedBy:      in.CreatedBy,
		Updated:        in.Updated,
		Deleted:        null.IntFromPtr(in.Deleted),
		Size:           in.Size,
		SizeUpdated:    in.SizeUpdated,
		GitUID:         in.GitUID,
		DefaultBranch:  in.DefaultBranch,
		ForkID:         in.ForkID,
		PullReqSeq:     in.PullReqSeq,
		NumForks:       in.NumForks,
		NumPulls:       in.NumPulls,
		NumClosedPulls: in.NumClosedPulls,
		NumOpenPulls:   in.NumOpenPulls,
		NumMergedPulls: in.NumMergedPulls,
		State:          in.State,
		IsEmpty:        in.IsEmpty,
	}
}

func applyQueryFilter(stmt squirrel.SelectBuilder, filter *types.RepoFilter) squirrel.SelectBuilder {
	if filter.Query != "" {
		stmt = stmt.Where("LOWER(repo_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
	}
	//nolint:gocritic
	if filter.DeletedAt != nil {
		stmt = stmt.Where("repo_deleted = ?", filter.DeletedAt)
	} else if filter.DeletedBeforeOrAt != nil {
		stmt = stmt.Where("repo_deleted <= ?", filter.DeletedBeforeOrAt)
	} else {
		stmt = stmt.Where("repo_deleted IS NULL")
	}
	return stmt
}

func applySortFilter(stmt squirrel.SelectBuilder, filter *types.RepoFilter) squirrel.SelectBuilder {
	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	switch filter.Sort {
	// TODO [CODE-1363]: remove after identifier migration.
	case enum.RepoAttrUID, enum.RepoAttrIdentifier, enum.RepoAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("repo_state desc, repo_uid " + filter.Order.String())
	case enum.RepoAttrCreated:
		stmt = stmt.OrderBy("repo_created " + filter.Order.String())
	case enum.RepoAttrUpdated:
		stmt = stmt.OrderBy("repo_updated " + filter.Order.String())
	case enum.RepoAttrDeleted:
		stmt = stmt.OrderBy("repo_deleted " + filter.Order.String())
	}

	return stmt
}

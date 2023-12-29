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

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.RepoStore = (*RepoStore)(nil)

// NewRepoStore returns a new RepoStore.
func NewRepoStore(
	db *sqlx.DB,
	spacePathCache store.SpacePathCache,
	spacePathStore store.SpacePathStore,
) *RepoStore {
	return &RepoStore{
		db:             db,
		spacePathCache: spacePathCache,
		spacePathStore: spacePathStore,
	}
}

// RepoStore implements a store.RepoStore backed by a relational database.
type RepoStore struct {
	db             *sqlx.DB
	spacePathCache store.SpacePathCache
	spacePathStore store.SpacePathStore
}

type repository struct {
	// TODO: int64 ID doesn't match DB
	ID          int64  `db:"repo_id"`
	Version     int64  `db:"repo_version"`
	ParentID    int64  `db:"repo_parent_id"`
	UID         string `db:"repo_uid"`
	Description string `db:"repo_description"`
	IsPublic    bool   `db:"repo_is_public"`
	CreatedBy   int64  `db:"repo_created_by"`
	Created     int64  `db:"repo_created"`
	Updated     int64  `db:"repo_updated"`

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

	Importing bool `db:"repo_importing"`
}

const (
	repoColumnsForJoin = `
		repo_id
		,repo_version
		,repo_parent_id
		,repo_uid
		,repo_description
		,repo_is_public
		,repo_created_by
		,repo_created
		,repo_updated
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
		,repo_importing`

	repoSelectBase = `
		SELECT` + repoColumnsForJoin + `
		FROM repositories`
)

// Find finds the repo by id.
func (s *RepoStore) Find(ctx context.Context, id int64) (*types.Repository, error) {
	const sqlQuery = repoSelectBase + `
		WHERE repo_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(repository)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find repo")
	}

	return s.mapToRepo(ctx, dst)
}

// Find finds the repo with the given UID in the given space ID.
func (s *RepoStore) FindByUID(ctx context.Context, spaceID int64, uid string) (*types.Repository, error) {
	const sqlQuery = repoSelectBase + `
		WHERE repo_parent_id = $1 AND LOWER(repo_uid) = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(repository)
	if err := db.GetContext(ctx, dst, sqlQuery, spaceID, strings.ToLower(uid)); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find repo")
	}

	return s.mapToRepo(ctx, dst)
}

// FindByRef finds the repo using the repoRef as either the id or the repo path.
func (s *RepoStore) FindByRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	// ASSUMPTION: digits only is not a valid repo path
	id, err := strconv.ParseInt(repoRef, 10, 64)
	if err != nil {
		spacePath, repoUID, err := paths.DisectLeaf(repoRef)
		if err != nil {
			return nil, fmt.Errorf("failed to disect leaf for path '%s': %w", repoRef, err)
		}
		pathObject, err := s.spacePathCache.Get(ctx, spacePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get space path: %w", err)
		}

		return s.FindByUID(ctx, pathObject.SpaceID, repoUID)
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
			,repo_importing
		) values (
			:repo_version
			,:repo_parent_id
			,:repo_uid
			,:repo_description
			,:repo_is_public
			,:repo_created_by
			,:repo_created
			,:repo_updated
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
			,:repo_importing
		) RETURNING repo_id`

	db := dbtx.GetAccessor(ctx, s.db)

	// insert repo first so we get id
	query, arg, err := db.BindNamed(sqlQuery, mapToInternalRepo(repo))
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&repo.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Insert query failed")
	}

	repo.Path, err = s.getRepoPath(ctx, repo.ParentID, repo.UID)
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
			,repo_parent_id = :repo_parent_id
			,repo_uid = :repo_uid
			,repo_git_uid = :repo_git_uid
			,repo_description = :repo_description
			,repo_is_public = :repo_is_public
			,repo_default_branch = :repo_default_branch
			,repo_pullreq_seq = :repo_pullreq_seq
			,repo_num_forks = :repo_num_forks
			,repo_num_pulls = :repo_num_pulls
			,repo_num_closed_pulls = :repo_num_closed_pulls
			,repo_num_open_pulls = :repo_num_open_pulls
			,repo_num_merged_pulls = :repo_num_merged_pulls
			,repo_importing = :repo_importing
		WHERE repo_id = :repo_id AND repo_version = :repo_version - 1`

	dbRepo := mapToInternalRepo(repo)

	// update Version (used for optimistic locking) and Updated time
	dbRepo.Version++
	dbRepo.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbRepo)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind repo object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update repository")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	repo.Version = dbRepo.Version
	repo.Updated = dbRepo.Updated

	// update path in case parent/uid changed (its most likely cached anyway)
	repo.Path, err = s.getRepoPath(ctx, repo.ParentID, repo.UID)
	if err != nil {
		return err
	}

	return nil
}

// UpdateSize updates the size of a specific repository in the database.
func (s *RepoStore) UpdateSize(ctx context.Context, repoID int64, size int64) error {
	stmt := database.Builder.
		Update("repositories").
		Set("repo_size", size).
		Set("repo_size_updated", time.Now().UnixMilli()).
		Where("repo_id = ?", repoID)

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to create sql query")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update repo size")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return fmt.Errorf("repo %d size not updated: %w", repoID, gitness_store.ErrResourceNotFound)
	}

	return nil
}

// UpdateOptLock updates the repository using the optimistic locking mechanism.
func (s *RepoStore) UpdateOptLock(ctx context.Context,
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
		return database.ProcessSQLErrorf(err, "the delete query failed")
	}

	return nil
}

// Count of repos in a space. if parentID (space) is zero then it will count all repositories in the system.
func (s *RepoStore) Count(ctx context.Context, parentID int64, opts *types.RepoFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("repositories")

	if parentID > 0 {
		stmt = stmt.Where("repo_parent_id = ?", parentID)
	}

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(repo_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// List returns a list of repos in a space.
func (s *RepoStore) List(ctx context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error) {
	stmt := database.Builder.
		Select(repoColumnsForJoin).
		From("repositories").
		Where("repo_parent_id = ?", fmt.Sprint(parentID))

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(repo_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	switch opts.Sort {
	case enum.RepoAttrUID, enum.RepoAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("repo_importing desc, repo_uid " + opts.Order.String())
	case enum.RepoAttrCreated:
		stmt = stmt.OrderBy("repo_created " + opts.Order.String())
	case enum.RepoAttrUpdated:
		stmt = stmt.OrderBy("repo_updated " + opts.Order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*repository{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return s.mapToRepos(ctx, dst)
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
		From("repositories")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*repoSize{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
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
		UID:            in.UID,
		Description:    in.Description,
		IsPublic:       in.IsPublic,
		Created:        in.Created,
		CreatedBy:      in.CreatedBy,
		Updated:        in.Updated,
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
		Importing:      in.Importing,
		// Path: is set below
	}

	res.Path, err = s.getRepoPath(ctx, in.ParentID, in.UID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *RepoStore) getRepoPath(ctx context.Context, parentID int64, repoUID string) (string, error) {
	spacePath, err := s.spacePathStore.FindPrimaryBySpaceID(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("failed to get primary path for space %d: %w", parentID, err)
	}
	return paths.Concatinate(spacePath.Value, repoUID), nil
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
		UID:            in.UID,
		Description:    in.Description,
		IsPublic:       in.IsPublic,
		Created:        in.Created,
		CreatedBy:      in.CreatedBy,
		Updated:        in.Updated,
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
		Importing:      in.Importing,
	}
}

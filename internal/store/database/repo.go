// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

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
	err := s.db.Get(dst, repoSelectID, id)
	return dst, err
}

// Finds the repo by the full qualified repo name.
func (s *RepoStore) FindFqn(ctx context.Context, fqn string) (*types.Repository, error) {
	dst := new(types.Repository)
	err := s.db.Get(dst, repoSelectFqn, fqn)
	return dst, err
}

// Creates a new repo
func (s *RepoStore) Create(ctx context.Context, repo *types.Repository) error {
	// TODO: Ensure parent exists!!
	query, arg, err := s.db.BindNamed(repoInsert, repo)
	if err != nil {
		return err
	}
	return s.db.QueryRow(query, arg...).Scan(&repo.ID)
}

// Updates the repo details.
func (s *RepoStore) Update(ctx context.Context, repo *types.Repository) error {
	query, arg, err := s.db.BindNamed(repoUpdate, repo)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(query, arg...)
	return err
}

// Deletes the repo.
func (s *RepoStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// delete the repo
	if _, err := tx.Exec(repoDelete, id); err != nil {
		return err
	}
	return tx.Commit()
}

// List returns a list of repos in a space.
func (s *RepoStore) List(ctx context.Context, spaceId int64, opts types.RepoFilter) ([]*types.Repository, error) {
	dst := []*types.Repository{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.RepoAttrNone {
		err := s.db.Select(&dst, repoSelect, spaceId, limit(opts.Size), offset(opts.Page, opts.Size))
		return dst, err
	}

	// else we construct the sql statement.
	stmt := builder.Select("*").From("repositories").Where("repo_spaceId = " + fmt.Sprint(spaceId))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.RepoAttrCreated:
		// NOTE: string concatination is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("repo_created " + opts.Order.String())
	case enum.RepoAttrUpdated:
		stmt = stmt.OrderBy("repo_updated " + opts.Order.String())
	case enum.RepoAttrId:
		stmt = stmt.OrderBy("repo_id " + opts.Order.String())
	case enum.RepoAttrName:
		stmt = stmt.OrderBy("repo_name " + opts.Order.String())
	case enum.RepoAttrFqn:
		stmt = stmt.OrderBy("repo_fqn " + opts.Order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return dst, err
	}

	err = s.db.Select(&dst, sql)
	return dst, err
}

// Count of repos in a space.
func (s *RepoStore) Count(ctx context.Context, spaceId int64) (int64, error) {
	var count int64
	err := s.db.QueryRow(repoCount, spaceId).Scan(&count)
	return count, err
}

const repoBase = `
SELECT
repo_id
,repo_name
,repo_spaceId
,repo_fqn
,repo_displayName
,repo_description
,repo_isPublic
,repo_createdBy
,repo_created
,repo_updated
,repo_forkId
,repo_numForks
,repo_numPulls
,repo_numClosedPulls
,repo_numOpenPulls
FROM repositories
`
const repoSelect = repoBase + `
WHERE repo_spaceId = $1
ORDER BY repo_fqn ASC
LIMIT $2 OFFSET $3
`

const repoCount = `
SELECT count(*)
FROM repositories
WHERE repo_spaceId = $1
`

const repoSelectID = repoBase + `
WHERE repo_id = $1
`

const repoSelectFqn = repoBase + `
WHERE repo_fqn = $1
`

const repoDelete = `
DELETE FROM repositories
WHERE repo_id = $1
`

const repoInsert = `
INSERT INTO repositories (
	repo_name
	,repo_spaceId
	,repo_fqn
	,repo_displayName
	,repo_description
	,repo_isPublic
	,repo_createdBy
	,repo_created
	,repo_updated
	,repo_forkId
	,repo_numForks
	,repo_numPulls
	,repo_numClosedPulls
	,repo_numOpenPulls
) values (
	:repo_name
	,:repo_spaceId
	,:repo_fqn
	,:repo_displayName
	,:repo_description
	,:repo_isPublic
	,:repo_createdBy
	,:repo_created
	,:repo_updated
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
,repo_displayName		= :repo_displayName
,repo_description		= :repo_description
,repo_isPublic			= :repo_isPublic
,repo_updated			= :repo_updated
,repo_numForks			= :repo_numForks
,repo_numPulls			= :repo_numPulls
,repo_numClosedPulls	= :repo_numClosedPulls
,repo_numOpenPulls		= :repo_numOpenPulls
WHERE repo_id = :repo_id
`

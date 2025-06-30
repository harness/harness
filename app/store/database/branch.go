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
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

const (
	branchColumns = `
		 branch_repo_id
		,branch_name
		,branch_sha
		,branch_created_by
		,branch_created
		,branch_updated_by
		,branch_updated`

	branchSelectBase = `
		SELECT` + branchColumns + `
		FROM branches`
)

// branch represents the internal database model for a branch.
type branch struct {
	RepoID    int64  `db:"branch_repo_id"`
	Name      string `db:"branch_name"`
	SHA       string `db:"branch_sha"`
	CreatedBy int64  `db:"branch_created_by"`
	Created   int64  `db:"branch_created"`
	UpdatedBy int64  `db:"branch_updated_by"`
	Updated   int64  `db:"branch_updated"`
}

// branchStore implements store.BranchStore interface to manage branch data.
type branchStore struct {
	db *sqlx.DB
}

// NewBranchStore returns a new branchStore that implements store.BranchStore.
func NewBranchStore(db *sqlx.DB) store.BranchStore {
	return &branchStore{
		db: db,
	}
}

// ToType converts the internal branch type to the external Branch type.
func (b *branch) ToType() types.BranchTable {
	shaObj, _ := sha.New(b.SHA) // Error ignored since DB values should be valid

	return types.BranchTable{
		Name:      b.Name,
		SHA:       shaObj,
		CreatedBy: b.CreatedBy,
		Created:   b.Created,
		UpdatedBy: b.UpdatedBy,
		Updated:   b.Updated,
	}
}

// mapInternalBranch converts the external branch type to the internal branch type.
func mapInternalBranch(b *types.BranchTable, repoID int64) branch {
	return branch{
		Name:      b.Name,
		RepoID:    repoID,
		SHA:       b.SHA.String(),
		CreatedBy: b.CreatedBy,
		Created:   b.Created,
		UpdatedBy: b.UpdatedBy,
		Updated:   b.Updated,
	}
}

// FindBranchesWithoutPRs finds branches without pull requests for a repository.
func (s *branchStore) FindBranchesWithoutPRs(
	ctx context.Context,
	repoID int64,
	principalID int64,
	cutOffTime int64,
	limit uint64,
) ([]types.BranchTable, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	const sqlQuery = branchSelectBase + `
		LEFT JOIN pullreqs ON branch_repo_id = pullreq_source_repo_id
			AND branch_name = pullreq_source_branch 
			AND pullreq_state = 'open'
		WHERE branch_repo_id = $1
			AND branch_updated_by = $2
			AND branch_updated > $3
			AND pullreq_id IS NULL
		ORDER BY branch_updated DESC
		LIMIT $4
	`

	dst := make([]*branch, 0, limit)
	err := db.SelectContext(
		ctx,
		&dst,
		sqlQuery,
		repoID,
		principalID,
		cutOffTime,
		limit,
	)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find branches without PRs")
	}

	// Convert to external types
	result := make([]types.BranchTable, len(dst))
	for i, b := range dst {
		result[i] = b.ToType()
	}

	return result, nil
}

// Delete deletes a branch by repo ID and branch name.
func (s *branchStore) Delete(ctx context.Context, repoID int64, name string) error {
	db := dbtx.GetAccessor(ctx, s.db)

	const sqlQuery = `
		DELETE FROM branches
		WHERE branch_repo_id = $1 AND branch_name = $2
	`

	if _, err := db.ExecContext(ctx, sqlQuery, repoID, name); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete branch")
	}
	return nil
}

// Upsert creates a new branch or updates an existing one if it already exists.
func (s *branchStore) Upsert(ctx context.Context, repoID int64, branch *types.BranchTable) error {
	db := dbtx.GetAccessor(ctx, s.db)

	const sqlQuery = `
		INSERT INTO branches (
			 branch_repo_id
			,branch_name
			,branch_sha
			,branch_created_by
			,branch_created
			,branch_updated_by
			,branch_updated
		) VALUES (
			 :branch_repo_id
			,:branch_name
			,:branch_sha
			,:branch_created_by
			,:branch_created
			,:branch_updated_by
			,:branch_updated
		) ON CONFLICT (branch_repo_id, branch_name) DO UPDATE SET
			 branch_sha = EXCLUDED.branch_sha
			,branch_updated_by = EXCLUDED.branch_updated_by
			,branch_updated = EXCLUDED.branch_updated
	`

	query, args, err := db.BindNamed(sqlQuery, mapInternalBranch(branch, repoID))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind branch parameters")
	}

	_, err = db.ExecContext(ctx, query, args...)

	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to upsert branch")
	}

	return nil
}

// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.RepoGitInfoView = (*RepoGitInfoView)(nil)

// NewRepoGitInfoView returns a new RepoGitInfoView.
// It's used by the repository git UID cache.
func NewRepoGitInfoView(db *sqlx.DB) *RepoGitInfoView {
	return &RepoGitInfoView{
		db: db,
	}
}

type RepoGitInfoView struct {
	db *sqlx.DB
}

func (s *RepoGitInfoView) Find(ctx context.Context, id int64) (*types.RepositoryGitInfo, error) {
	const sqlQuery = `
		SELECT repo_git_uid
		FROM repositories
		WHERE repo_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	v := db.QueryRowContext(ctx, sqlQuery, id)
	if err := v.Err(); err != nil {
		return nil, database.ProcessSQLErrorf(err, "failed to find git uid by repository id")
	}

	var result = types.RepositoryGitInfo{ID: id}

	if err := v.Scan(&result.GitUID); err != nil {
		return nil, database.ProcessSQLErrorf(err, "failed to scan git uid")
	}

	return &result, nil
}

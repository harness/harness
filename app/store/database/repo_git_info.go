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
		SELECT repo_git_uid, repo_parent_id
		FROM repositories
		WHERE repo_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	v := db.QueryRowContext(ctx, sqlQuery, id)
	if err := v.Err(); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to find git uid by repository id")
	}

	var result = types.RepositoryGitInfo{ID: id}

	if err := v.Scan(&result.GitUID, &result.ParentID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to scan git uid")
	}

	return &result, nil
}

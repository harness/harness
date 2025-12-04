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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.LinkedRepoStore = (*LinkedRepoStore)(nil)

func NewLinkedRepoStore(db *sqlx.DB) *LinkedRepoStore {
	return &LinkedRepoStore{
		db: db,
	}
}

// LinkedRepoStore implements store.LinkedRepoStore backed by a relational database.
type LinkedRepoStore struct {
	db *sqlx.DB
}

type linkedRepo struct {
	RepoID              int64  `db:"linked_repo_id"`
	Version             int64  `db:"linked_repo_version"`
	Created             int64  `db:"linked_repo_created"`
	Updated             int64  `db:"linked_repo_updated"`
	LastFullSync        int64  `db:"linked_repo_last_full_sync"`
	ConnectorPath       string `db:"linked_repo_connector_path"`
	ConnectorIdentifier string `db:"linked_repo_connector_identifier"`
	ConnectorRepo       string `db:"linked_repo_connector_repo"`
}

const (
	linkedRepoColumns = `
		 linked_repo_id
		,linked_repo_version
		,linked_repo_created
		,linked_repo_updated
		,linked_repo_last_full_sync
		,linked_repo_connector_path
		,linked_repo_connector_identifier
		,linked_repo_connector_repo`

	linkedRepoSelectBase = `
	SELECT` + linkedRepoColumns + `
	FROM linked_repositories`
)

func (s *LinkedRepoStore) Find(ctx context.Context, repoID int64) (*types.LinkedRepo, error) {
	const sqlQuery = linkedRepoSelectBase + `
	WHERE linked_repo_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &linkedRepo{}
	if err := db.GetContext(ctx, dst, sqlQuery, repoID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find linked repo")
	}

	return (*types.LinkedRepo)(dst), nil
}

func (s *LinkedRepoStore) Create(ctx context.Context, v *types.LinkedRepo) error {
	const sqlQuery = `
	INSERT INTO linked_repositories (
		 linked_repo_id
		,linked_repo_version
		,linked_repo_created
		,linked_repo_updated
		,linked_repo_last_full_sync
		,linked_repo_connector_path
		,linked_repo_connector_identifier
		,linked_repo_connector_repo
	) values (
		 :linked_repo_id
		,:linked_repo_version
		,:linked_repo_created
		,:linked_repo_updated
		,:linked_repo_last_full_sync
		,:linked_repo_connector_path
		,:linked_repo_connector_identifier
		,:linked_repo_connector_repo
	)`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, (*linkedRepo)(v))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind linked repo object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert linked repo")
	}

	return nil
}

func (s *LinkedRepoStore) Update(ctx context.Context, linked *types.LinkedRepo) error {
	const sqlQuery = `
		UPDATE linked_repositories
		SET
			 linked_repo_version = :linked_repo_version
			,linked_repo_updated = :linked_repo_updated
			,linked_repo_last_full_sync = :linked_repo_last_full_sync
		WHERE linked_repo_id = :linked_repo_id AND linked_repo_version = :linked_repo_version - 1`

	dbLinked := linkedRepo(*linked)
	dbLinked.Version++
	dbLinked.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbLinked)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind linked repository object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update linked repository")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated linked repository rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	linked.Version = dbLinked.Version
	linked.Updated = dbLinked.Updated

	return nil
}

func (s *LinkedRepoStore) UpdateOptLock(
	ctx context.Context,
	r *types.LinkedRepo,
	mutateFn func(*types.LinkedRepo) error,
) (*types.LinkedRepo, error) {
	for {
		dup := *r

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

		r, err = s.Find(ctx, r.RepoID)
		if err != nil {
			return nil, err
		}
	}
}

func (s *LinkedRepoStore) List(ctx context.Context, limit int) ([]types.LinkedRepo, error) {
	stmt := database.Builder.
		Select(linkedRepoColumns).
		From("linked_repositories").
		InnerJoin("repositories ON repo_id = linked_repo_id").
		Where("repo_deleted IS NULL").
		OrderBy("linked_repo_last_full_sync ASC").
		Limit(uint64(limit)) //nolint:gosec

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert linked repo list query to sql: %w", err)
	}

	dst := make([]linkedRepo, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing list linked repos query")
	}

	result := make([]types.LinkedRepo, len(dst))
	for i, r := range dst {
		result[i] = types.LinkedRepo(r)
	}

	return result, nil
}

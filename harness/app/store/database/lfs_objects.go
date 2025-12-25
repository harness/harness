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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var _ store.LFSObjectStore = (*LFSObjectStore)(nil)

func NewLFSObjectStore(db *sqlx.DB) *LFSObjectStore {
	return &LFSObjectStore{
		db: db,
	}
}

type LFSObjectStore struct {
	db *sqlx.DB
}

type lfsObject struct {
	ID        int64  `db:"lfs_object_id"`
	OID       string `db:"lfs_object_oid"`
	Size      int64  `db:"lfs_object_size"`
	Created   int64  `db:"lfs_object_created"`
	CreatedBy int64  `db:"lfs_object_created_by"`
	RepoID    int64  `db:"lfs_object_repo_id"`
}

const (
	lfsObjectColumns = `
		lfs_object_id
		,lfs_object_oid
		,lfs_object_size
		,lfs_object_created
		,lfs_object_created_by
		,lfs_object_repo_id`
)

func (s *LFSObjectStore) Find(
	ctx context.Context,
	repoID int64,
	oid string,
) (*types.LFSObject, error) {
	stmt := database.Builder.
		Select(lfsObjectColumns).
		From("lfs_objects").
		Where("lfs_object_repo_id = ? AND lfs_object_oid = ?", repoID, oid)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &lfsObject{}
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return mapLFSObject(dst), nil
}

func (s *LFSObjectStore) FindMany(
	ctx context.Context,
	repoID int64,
	oids []string,
) ([]*types.LFSObject, error) {
	stmt := database.Builder.
		Select(lfsObjectColumns).
		From("lfs_objects").
		Where("lfs_object_repo_id = ?", repoID).
		Where(squirrel.Eq{"lfs_object_oid": oids})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}
	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*lfsObject
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return mapLFSObjects(dst), nil
}

func (s *LFSObjectStore) Create(ctx context.Context, obj *types.LFSObject) error {
	const sqlQuery = `
		INSERT INTO lfs_objects (
			 lfs_object_oid
			,lfs_object_size
			,lfs_object_created
			,lfs_object_created_by
			,lfs_object_repo_id
		) VALUES (
			 :lfs_object_oid
			,:lfs_object_size
			,:lfs_object_created
			,:lfs_object_created_by
			,:lfs_object_repo_id
		) RETURNING lfs_object_id`

	db := dbtx.GetAccessor(ctx, s.db)
	query, args, err := db.BindNamed(sqlQuery, mapInternalLFSObject(obj))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind query")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&obj.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to create LFS object")
	}

	return nil
}

// GetSizeInKBByRepoID returns the total size of LFS objects in KiB for a specified repo.
func (s *LFSObjectStore) GetSizeInKBByRepoID(ctx context.Context, repoID int64) (int64, error) {
	stmt := database.Builder.
		Select("CAST(COALESCE(SUM(lfs_object_size) / 1024, 0) AS BIGINT)").
		From("lfs_objects").
		Where("lfs_object_repo_id = ?", repoID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var size int64
	if err := db.GetContext(ctx, &size, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return size, nil
}

func mapInternalLFSObject(obj *types.LFSObject) *lfsObject {
	return &lfsObject{
		ID:        obj.ID,
		OID:       obj.OID,
		Size:      obj.Size,
		Created:   obj.Created,
		CreatedBy: obj.CreatedBy,
		RepoID:    obj.RepoID,
	}
}

func mapLFSObject(obj *lfsObject) *types.LFSObject {
	return &types.LFSObject{
		ID:        obj.ID,
		OID:       obj.OID,
		Size:      obj.Size,
		Created:   obj.Created,
		CreatedBy: obj.CreatedBy,
		RepoID:    obj.RepoID,
	}
}

func mapLFSObjects(objs []*lfsObject) []*types.LFSObject {
	res := make([]*types.LFSObject, len(objs))
	for i := range objs {
		res[i] = mapLFSObject(objs[i])
	}

	return res
}

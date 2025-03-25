//  Copyright 2023 Harness, Inc.
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
	"database/sql"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type GenericBlobDao struct {
	sqlDB *sqlx.DB
}

func (g GenericBlobDao) FindByID(ctx context.Context, id string) (*types.GenericBlob, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(GenericBlob{}), ",")).
		From("generic_blobs").
		Where("generic_blob_id = ?", id)

	db := dbtx.GetAccessor(ctx, g.sqlDB)

	dst := new(GenericBlob)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find generic blob with id %s", id)
	}

	return g.mapToGenericBlob(ctx, dst)
}

func (g GenericBlobDao) TotalSizeByRootParentID(ctx context.Context, rootID int64) (int64, error) {
	q := databaseg.Builder.
		Select("COALESCE(SUM(generic_blob_size), 0) AS size").
		From("generic_blobs").
		Where("generic_blob_root_parent_id = ?", rootID)

	db := dbtx.GetAccessor(ctx, g.sqlDB)

	var size int64
	sqlQuery, args, err := q.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&size); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return 0,
			databaseg.ProcessSQLErrorf(ctx, err, "Failed to find total blob size for root parent with id %d", rootID)
	}
	return size, nil
}

func (g GenericBlobDao) FindBySha256AndRootParentID(ctx context.Context,
	sha256 string, rootParentID int64) (
	*types.GenericBlob, error) {
	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(GenericBlob{}), ",")).
		From("generic_blob").
		Where("generic_blob_root_parent_id = ? AND generic_blob_sha256 = ?", rootParentID, sha256)

	db := dbtx.GetAccessor(ctx, g.sqlDB)

	dst := new(GenericBlob)
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find generic blob with sha256 %s", sha256)
	}

	return g.mapToGenericBlob(ctx, dst)
}

func (g GenericBlobDao) Create(ctx context.Context, gb *types.GenericBlob) (bool, error) {
	const sqlQuery = `
        INSERT INTO generic_blobs (
            generic_blob_id,
            generic_blob_root_parent_id,
            generic_blob_sha_1,
            generic_blob_sha_256,
            generic_blob_sha_512,
            generic_blob_md5,
            generic_blob_size,
            generic_blob_created_at,
            generic_blob_created_by
        ) VALUES (
            :generic_blob_id,
            :generic_blob_root_parent_id,
            :generic_blob_sha_1,
            :generic_blob_sha_256,
            :generic_blob_sha_512,
            :generic_blob_md5,
            :generic_blob_size,
            :generic_blob_created_at,
            :generic_blob_created_by
        ) ON CONFLICT (generic_blob_root_parent_id, generic_blob_sha_256)
        DO UPDATE SET generic_blob_id = generic_blobs.generic_blob_id
        RETURNING generic_blob_id`

	db := dbtx.GetAccessor(ctx, g.sqlDB)
	query, arg, err := db.BindNamed(sqlQuery, g.mapToInternalGenericBlob(ctx, gb))
	if err != nil {
		return false, databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind generic blob object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&gb.ID); err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, store2.ErrDuplicate) {
			return false, nil
		}
		return false, databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return true, nil
}

func (g GenericBlobDao) DeleteByID(_ context.Context, _ string) error {
	// TODO implement me
	panic("implement me")
}

func (g GenericBlobDao) mapToGenericBlob(_ context.Context, dst *GenericBlob) (*types.GenericBlob, error) {
	return &types.GenericBlob{
		ID:           dst.ID,
		RootParentID: dst.RootParentID,
		Sha256:       dst.Sha256,
		Sha1:         dst.Sha1,
		Sha512:       dst.Sha512,
		MD5:          dst.MD5,
		Size:         dst.Size,
		CreatedAt:    time.UnixMilli(dst.CreatedAt),
		CreatedBy:    dst.CreatedBy,
	}, nil
}

func (g GenericBlobDao) mapToInternalGenericBlob(ctx context.Context, gb *types.GenericBlob) interface{} {
	session, _ := request.AuthSessionFrom(ctx)

	if gb.CreatedAt.IsZero() {
		gb.CreatedAt = time.Now()
	}
	if gb.CreatedBy == 0 {
		gb.CreatedBy = session.Principal.ID
	}
	if gb.ID == "" {
		gb.ID = uuid.NewString()
	}

	return &GenericBlob{
		ID:           gb.ID,
		Sha256:       gb.Sha256,
		Sha1:         gb.Sha1,
		Sha512:       gb.Sha512,
		MD5:          gb.MD5,
		Size:         gb.Size,
		RootParentID: gb.RootParentID,
		CreatedAt:    gb.CreatedAt.UnixMilli(),
		CreatedBy:    gb.CreatedBy,
	}
}

func NewGenericBlobDao(sqlDB *sqlx.DB) store.GenericBlobRepository {
	return &GenericBlobDao{
		sqlDB: sqlDB,
	}
}

type GenericBlob struct {
	ID           string `db:"generic_blob_id"`
	RootParentID int64  `db:"generic_blob_root_parent_id"`
	Sha1         string `db:"generic_blob_sha_1"`
	Sha256       string `db:"generic_blob_sha_256"`
	Sha512       string `db:"generic_blob_sha_512"`
	MD5          string `db:"generic_blob_md5"`
	Size         int64  `db:"generic_blob_size"`
	CreatedAt    int64  `db:"generic_blob_created_at"`
	CreatedBy    int64  `db:"generic_blob_created_by"`
}

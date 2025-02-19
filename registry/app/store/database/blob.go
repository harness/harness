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
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/opencontainers/go-digest"
	errors2 "github.com/pkg/errors"
)

type blobDao struct {
	db *sqlx.DB

	//FIXME: Arvind: Move this to controller layer later
	mtRepository store.MediaTypesRepository
}

func NewBlobDao(db *sqlx.DB, mtRepository store.MediaTypesRepository) store.BlobRepository {
	return &blobDao{
		db:           db,
		mtRepository: mtRepository,
	}
}

var (
	PrimaryQuery = database.Builder.Select("blobs.blob_id as blob_id", "blob_media_type_id", "mt_media_type",
		"blob_digest", "blob_size", "blob_created_at", "blob_root_parent_id").
		From("blobs").
		Join("media_types ON mt_id = blobs.blob_media_type_id")
)

type blobDB struct {
	ID           int64  `db:"blob_id"`
	RootParentID int64  `db:"blob_root_parent_id"`
	Digest       []byte `db:"blob_digest"`
	MediaTypeID  int64  `db:"blob_media_type_id"`
	Size         int64  `db:"blob_size"`
	CreatedAt    int64  `db:"blob_created_at"`
	CreatedBy    int64  `db:"blob_created_by"`
}

type blobMetadataDB struct {
	blobDB
	MediaType string `db:"mt_media_type"`
}

func (bd blobDao) FindByDigestAndRootParentID(ctx context.Context, d digest.Digest,
	rootParentID int64) (*types.Blob, error) {
	dgst, err := types.NewDigest(d)
	if err != nil {
		return nil, err
	}

	digestBytes, err := util.GetHexDecodedBytes(string(dgst))
	if err != nil {
		return nil, err
	}

	stmt := PrimaryQuery.
		Where("blob_root_parent_id = ?", rootParentID).
		Where("blob_digest = ?", digestBytes)

	db := dbtx.GetAccessor(ctx, bd.db)

	dst := new(blobMetadataDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors2.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find blob")
	}

	return bd.mapToBlob(dst)
}

func (bd blobDao) TotalSizeByRootParentID(ctx context.Context, rootID int64) (int64, error) {
	q := database.Builder.Select("COALESCE(SUM(blob_size), 0) AS size").
		From("blobs").
		Where("blob_root_parent_id = ?", rootID)

	db := dbtx.GetAccessor(ctx, bd.db)

	var size int64
	sqlQuery, args, err := q.ToSql()
	if err != nil {
		return 0, errors2.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&size); err != nil &&
		!errors2.Is(err, sql.ErrNoRows) {
		return 0,
			database.ProcessSQLErrorf(ctx, err, "Failed to find total blob size for root parent with id %d", rootID)
	}
	return size, nil
}

func (bd blobDao) FindByID(ctx context.Context, id int64) (*types.Blob, error) {
	stmt := PrimaryQuery.
		Where("blob_id = ?", id)

	db := dbtx.GetAccessor(ctx, bd.db)

	dst := new(blobMetadataDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors2.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find blob")
	}

	return bd.mapToBlob(dst)
}

func (bd blobDao) FindByDigestAndRepoID(ctx context.Context, d digest.Digest, repoID int64,
	imageName string) (*types.Blob, error) {
	dgst, err := types.NewDigest(d)
	if err != nil {
		return nil, err
	}

	digestBytes, err := util.GetHexDecodedBytes(string(dgst))
	if err != nil {
		return nil, err
	}

	stmt := PrimaryQuery.
		Join("registry_blobs ON rblob_blob_id = blobs.blob_id").
		Where("rblob_registry_id = ?", repoID).
		Where("rblob_image_name = ?", imageName).
		Where("blob_digest = ?", digestBytes)

	db := dbtx.GetAccessor(ctx, bd.db)

	dst := new(blobMetadataDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors2.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find blob")
	}

	return bd.mapToBlob(dst)
}

func (bd blobDao) CreateOrFind(ctx context.Context, b *types.Blob) (*types.Blob, error) {
	sqlQuery := `INSERT INTO blobs (
                   blob_digest, 
                   blob_root_parent_id, 
                   blob_media_type_id, 
                   blob_size,
                   blob_created_at,
                   blob_created_by
        ) VALUES (
                  :blob_digest, 
                  :blob_root_parent_id,
                  :blob_media_type_id,
                  :blob_size,
                  :blob_created_at,
                  :blob_created_by
        ) ON CONFLICT (
            blob_digest, blob_root_parent_id
        ) DO NOTHING 
          RETURNING blob_id`

	mediaTypeID, err := bd.mtRepository.MapMediaType(ctx, b.MediaType)
	if err != nil {
		return nil, err
	}
	b.MediaTypeID = mediaTypeID

	db := dbtx.GetAccessor(ctx, bd.db)
	blob, err := mapToInternalBlob(ctx, b)
	if err != nil {
		return nil, err
	}
	query, arg, err := db.BindNamed(sqlQuery, blob)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&b.ID); err != nil {
		err := database.ProcessSQLErrorf(ctx, err, "Insert query failed")
		if !errors2.Is(err, store2.ErrResourceNotFound) {
			return nil, err
		}
	}

	return bd.FindByDigestAndRootParentID(ctx, b.Digest, b.RootParentID)
}

func (bd blobDao) DeleteByID(ctx context.Context, id int64) error {
	stmt := database.Builder.Delete("blobs").
		Where("blob_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge blob query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, bd.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (bd blobDao) ExistsBlob(ctx context.Context, repoID int64,
	d digest.Digest, image string) (bool, error) {
	stmt := database.Builder.Select("EXISTS (SELECT 1 FROM registry_blobs " +
		"JOIN blobs as b ON rblob_blob_id = b.blob_id " +
		"WHERE rblob_registry_id = ? AND " +
		"rblob_image_name = ? AND " +
		"b.blob_digest = ?)")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert exists blob query to sql: %w", err)
	}

	var exists bool
	db := dbtx.GetAccessor(ctx, bd.db)
	newDigest, err := types.NewDigest(d)
	if err != nil {
		return false, err
	}
	bytes, err := util.GetHexDecodedBytes(string(newDigest))
	if err != nil {
		return false, err
	}
	args = append(args, repoID, image, bytes)

	if err = db.GetContext(ctx, &exists, sql, args...); err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Failed to check exists blob")
	}

	return exists, nil
}

func mapToInternalBlob(ctx context.Context, in *types.Blob) (*blobDB, error) {
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.CreatedBy = -1
	newDigest, err := types.NewDigest(in.Digest)
	if err != nil {
		return nil, err
	}

	digestBytes, err := util.GetHexDecodedBytes(string(newDigest))
	if err != nil {
		return nil, err
	}

	return &blobDB{
		ID:           in.ID,
		RootParentID: in.RootParentID,
		MediaTypeID:  in.MediaTypeID,
		Digest:       digestBytes,
		Size:         in.Size,
		CreatedAt:    in.CreatedAt.UnixMilli(),
		CreatedBy:    in.CreatedBy,
	}, nil
}

func (bd blobDao) mapToBlob(dst *blobMetadataDB) (*types.Blob, error) {
	createdBy := int64(-1)
	dig := types.Digest(util.GetHexEncodedString(dst.Digest))
	parsedDigest, err := dig.Parse()
	if err != nil {
		return nil, err
	}
	return &types.Blob{
		ID:           dst.ID,
		RootParentID: dst.RootParentID,
		MediaTypeID:  dst.MediaTypeID,
		MediaType:    dst.MediaType,
		Digest:       parsedDigest,
		Size:         dst.Size,
		CreatedAt:    time.UnixMilli(dst.CreatedAt),
		CreatedBy:    createdBy,
	}, nil
}

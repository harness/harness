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
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type registryBlobDao struct {
	db *sqlx.DB
}

func NewRegistryBlobDao(db *sqlx.DB) store.RegistryBlobRepository {
	return &registryBlobDao{
		db: db,
	}
}

// registryBlobDB holds the record of a registry_blobs in DB.
type registryBlobDB struct {
	ID         int64  `db:"rblob_id"`
	RegistryID int64  `db:"rblob_registry_id"`
	BlobID     int64  `db:"rblob_blob_id"`
	ImageName  string `db:"rblob_image_name"`
	CreatedAt  int64  `db:"rblob_created_at"`
	UpdatedAt  int64  `db:"rblob_updated_at"`
	CreatedBy  int64  `db:"rblob_created_by"`
	UpdatedBy  int64  `db:"rblob_updated_by"`
}

func (r registryBlobDao) LinkBlob(
	ctx context.Context, imageName string,
	registry *types.Registry, blobID int64,
) error {
	sqlQuery := `
		INSERT INTO registry_blobs (
			rblob_blob_id,
    		rblob_registry_id,        
			rblob_image_name,
			rblob_created_at,
			rblob_updated_at,
			rblob_created_by,
			rblob_updated_by
        ) VALUES (
			:rblob_blob_id,
        	:rblob_registry_id,
			:rblob_image_name,
			:rblob_created_at,
			:rblob_updated_at,
			:rblob_created_by,
			:rblob_updated_by
        ) ON CONFLICT (
            rblob_registry_id, rblob_blob_id, rblob_image_name
        ) DO NOTHING 
          RETURNING rblob_registry_id`

	rblob := mapToInternalRegistryBlob(ctx, registry.ID, blobID, imageName)
	db := dbtx.GetAccessor(ctx, r.db)
	query, arg, err := db.BindNamed(sqlQuery, rblob)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	var registryBlobID int64

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&registryBlobID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	log.Ctx(ctx).Info().Msgf("Linking blob to registry %d with id: %d", registry.ID, registryBlobID)

	return nil
}

// UnlinkBlob unlinks a blob from a repository. It does nothing if not linked. A boolean is returned to denote whether
// the link was deleted or not. This avoids the need for a separate preceding `SELECT` to find if it exists.
func (r registryBlobDao) UnlinkBlob(
	ctx context.Context, imageName string,
	registry *types.Registry, blobID int64,
) (bool, error) {
	stmt := databaseg.Builder.Delete("registry_blobs").
		Where("rblob_registry_id = ? AND rblob_blob_id = ? "+
			"AND rblob_image_name = ?", registry.ID, blobID, imageName)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert purge registry query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, databaseg.ProcessSQLErrorf(ctx, err, "error unlinking blobs")
	}

	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (r registryBlobDao) UnlinkBlobByImageName(
	ctx context.Context, registryID int64,
	imageName string,
) (bool, error) {
	stmt := databaseg.Builder.Delete("registry_blobs").
		Where("rblob_registry_id = ? AND rblob_image_name = ?",
			registryID, imageName)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert purge registry query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, databaseg.ProcessSQLErrorf(ctx, err, "error unlinking blobs")
	}

	affected, err := result.RowsAffected()
	return affected > 0, err
}

func mapToInternalRegistryBlob(
	ctx context.Context, registryID int64, blobID int64,
	imageName string,
) *registryBlobDB {
	creationTime := time.Now().UnixMilli()
	session, _ := request.AuthSessionFrom(ctx)
	return &registryBlobDB{
		RegistryID: registryID,
		BlobID:     blobID,
		ImageName:  imageName,
		CreatedAt:  creationTime,
		UpdatedAt:  creationTime,
		CreatedBy:  session.Principal.ID,
		UpdatedBy:  session.Principal.ID,
	}
}

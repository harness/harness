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
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

type ArtifactStatDao struct {
	db *sqlx.DB
}

func NewArtifactStatDao(db *sqlx.DB) store.ArtifactStatRepository {
	return &ArtifactStatDao{
		db: db,
	}
}

type artifactStatDB struct {
	ID            int64 `db:"artifact_stat_id"`
	ArtifactID    int64 `db:"artifact_stat_artifact_id"`
	Date          int64 `db:"artifact_stat_date"`
	DownloadCount int64 `db:"artifact_stat_download_count"`
	UploadBytes   int64 `db:"artifact_stat_upload_bytes"`
	DownloadBytes int64 `db:"artifact_stat_download_bytes"`
	CreatedAt     int64 `db:"artifact_stat_created_at"`
	UpdatedAt     int64 `db:"artifact_stat_updated_at"`
	CreatedBy     int64 `db:"artifact_stat_created_by"`
	UpdatedBy     int64 `db:"artifact_stat_updated_by"`
}

func (a ArtifactStatDao) CreateOrUpdate(ctx context.Context, artifactStat *types.ArtifactStat) error {
	const sqlQuery = `
		INSERT INTO artifact_stats ( 
		         artifact_stat_artifact_id
				,artifact_stat_date
				,artifact_stat_download_count
				,artifact_stat_upload_bytes
				,artifact_stat_download_bytes
				,artifact_stat_created_at
				,artifact_stat_updated_at
				,artifact_stat_created_by
				,artifact_stat_updated_by		
		    ) VALUES (
						 :artifact_stat_artifact_id
						,:artifact_stat_date
						,:artifact_stat_download_count
						,:artifact_stat_upload_bytes
						,:artifact_stat_download_bytes
						,:artifact_stat_created_at
						,:artifact_stat_updated_at
				        ,:artifact_stat_created_by
				        ,:artifact_stat_updated_by							

		    ) 
            ON CONFLICT (artifact_stat_artifact_id, artifact_stat_date)
	        DO UPDATE SET
		    artifact_stat_download_count = 
	            artifact_stats.artifact_stat_download_count + EXCLUDED.artifact_stat_download_count,
		    artifact_stat_upload_bytes = 
	            artifact_stats.artifact_stat_upload_bytes + EXCLUDED.artifact_stat_upload_bytes,
		    artifact_stat_download_bytes = 
	            artifact_stats.artifact_stat_download_bytes + EXCLUDED.artifact_stat_download_bytes			   
        RETURNING artifact_stat_id`

	db := dbtx.GetAccessor(ctx, a.db)
	query, arg, err := db.BindNamed(sqlQuery, a.mapToInternalArtifactStat(ctx, artifactStat))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind artifact object")
	}

	if err = db.QueryRowContext(ctx, query,
		arg...).Scan(&artifactStat.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (a ArtifactStatDao) mapToInternalArtifactStat(ctx context.Context, in *types.ArtifactStat) *artifactStatDB {
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}

	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.UpdatedAt = time.Now()

	return &artifactStatDB{
		ID:            in.ID,
		ArtifactID:    in.ArtifactID,
		Date:          in.Date,
		DownloadCount: in.DownloadCount,
		UploadBytes:   in.UploadBytes,
		DownloadBytes: in.DownloadBytes,
		CreatedAt:     in.CreatedAt.UnixMilli(),
		UpdatedAt:     in.UpdatedAt.UnixMilli(),
		CreatedBy:     in.CreatedBy,
		UpdatedBy:     session.Principal.ID,
	}
}

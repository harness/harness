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

type DownloadStatDao struct {
	db *sqlx.DB
}

func NewDownloadStatDao(db *sqlx.DB) store.DownloadStatRepository {
	return &DownloadStatDao{
		db: db,
	}
}

type downloadStatDB struct {
	ID         int64 `db:"download_stat_id"`
	ArtifactID int64 `db:"download_stat_artifact_id"`
	Timestamp  int64 `db:"download_stat_timestamp"`
	CreatedAt  int64 `db:"download_stat_created_at"`
	UpdatedAt  int64 `db:"download_stat_updated_at"`
	CreatedBy  int64 `db:"download_stat_created_by"`
	UpdatedBy  int64 `db:"download_stat_updated_by"`
}

func (d DownloadStatDao) Create(ctx context.Context, downloadStat *types.DownloadStat) error {
	const sqlQuery = `
		INSERT INTO download_stats ( 
		         download_stat_artifact_id
				,download_stat_timestamp
				,download_stat_created_at
				,download_stat_updated_at
				,download_stat_created_by
				,download_stat_updated_by		
		    ) VALUES (
						 :download_stat_artifact_id
						,:download_stat_timestamp
						,:download_stat_created_at
						,:download_stat_updated_at
				        ,:download_stat_created_by
				        ,:download_stat_updated_by							
		    ) 		   
        RETURNING download_stat_id`

	db := dbtx.GetAccessor(ctx, d.db)
	query, arg, err := db.BindNamed(sqlQuery, d.mapToInternalDownloadStat(ctx, downloadStat))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind download stat object")
	}

	if err = db.QueryRowContext(ctx, query,
		arg...).Scan(&downloadStat.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (d DownloadStatDao) mapToInternalDownloadStat(ctx context.Context,
	in *types.DownloadStat) *downloadStatDB {
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}

	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.UpdatedAt = time.Now()

	return &downloadStatDB{
		ID:         in.ID,
		ArtifactID: in.ArtifactID,
		Timestamp:  time.Now().UnixMilli(),
		CreatedAt:  in.CreatedAt.UnixMilli(),
		UpdatedAt:  in.UpdatedAt.UnixMilli(),
		CreatedBy:  in.CreatedBy,
		UpdatedBy:  session.Principal.ID,
	}
}

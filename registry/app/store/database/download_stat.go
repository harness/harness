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
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

func (d DownloadStatDao) CreateByRegistryIDImageAndArtifactName(ctx context.Context,
	regID int64, image string, version string) error {
	selectQuery := databaseg.Builder.
		Select(
			"a.artifact_id",
			"?",
			"?",
			"?",
			"?",
			"?",
		).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("a.artifact_version = ? AND i.image_registry_id = ? AND i.image_name = ?").
		Limit(1)

	insertQuery := databaseg.Builder.
		Insert("download_stats").
		Columns(
			"download_stat_artifact_id",
			"download_stat_timestamp",
			"download_stat_created_at",
			"download_stat_updated_at",
			"download_stat_created_by",
			"download_stat_updated_by",
		).
		Select(selectQuery)

	// Convert query to SQL string and args
	sqlStr, _, err := insertQuery.ToSql()
	if err != nil {
		return fmt.Errorf("failed to generate SQL: %w", err)
	}

	session, _ := request.AuthSessionFrom(ctx)
	user := session.Principal.ID
	db := dbtx.GetAccessor(ctx, d.db)

	// Execute the query with parameters
	_, err = db.ExecContext(ctx, sqlStr,
		time.Now().UnixMilli(), time.Now().UnixMilli(), time.Now().UnixMilli(),
		user, user, version, regID, image)
	if err != nil {
		return fmt.Errorf("failed to insert download stat: %w", err)
	}

	return nil
}
func (d DownloadStatDao) GetTotalDownloadsForImage(ctx context.Context, imageID int64) (int64, error) {
	q := databaseg.Builder.Select(`count(*)`).
		From("artifacts art").Where("art.artifact_image_id = ?", imageID).
		Join("download_stats ds ON ds.download_stat_artifact_id = art.artifact_id")

	sql, args, err := q.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}
	// Log the final sql query
	finalQuery := util.FormatQuery(sql, args)
	log.Ctx(ctx).Debug().Str("sql", finalQuery).Msg("Executing GetTotalDownloadsForImage query")
	// Execute query
	db := dbtx.GetAccessor(ctx, d.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (d DownloadStatDao) GetTotalDownloadsForArtifactID(ctx context.Context, artifactID int64) (int64, error) {
	q := databaseg.Builder.Select(`count(*)`).
		From("download_stats ds").Where("ds.download_stat_artifact_id = ?", artifactID)

	sql, args, err := q.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}
	// Log the final sql query
	finalQuery := util.FormatQuery(sql, args)
	log.Ctx(ctx).Debug().Str("sql", finalQuery).Msg("Executing GetTotalDownloadsForArtifact query")
	// Execute query
	db := dbtx.GetAccessor(ctx, d.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (d DownloadStatDao) GetTotalDownloadsForManifests(
	ctx context.Context,
	artifactVersions []string,
	imageID int64,
) (map[string]int64, error) {
	q := databaseg.Builder.Select(`art.artifact_version, count(*)`).
		From("artifacts art").
		Join("download_stats ds ON ds.download_stat_artifact_id = art.artifact_id").Where(sq.And{
		sq.Eq{"artifact_image_id": imageID},
		sq.Eq{"artifact_version": artifactVersions},
	}, "art").GroupBy("art.artifact_version")

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}
	// Log the final sql query
	finalQuery := util.FormatQuery(sql, args)
	log.Ctx(ctx).Debug().Str("sql", finalQuery).Msg("Executing GetTotalDownloadsForManifests query")
	// Execute query
	db := dbtx.GetAccessor(ctx, d.db)

	dst := []*versionsCountDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing query")
	}

	// Convert the slice to a map
	result := make(map[string]int64)
	for _, v := range dst {
		result[v.Version] = v.Count
	}

	return result, nil
}

func (d DownloadStatDao) mapToInternalDownloadStat(
	ctx context.Context,
	in *types.DownloadStat,
) *downloadStatDB {
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

type versionsCountDB struct {
	Version string `db:"artifact_version"`
	Count   int64  `db:"count"`
}

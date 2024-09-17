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
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type ArtifactDao struct {
	db *sqlx.DB
}

func NewArtifactDao(db *sqlx.DB) store.ArtifactRepository {
	return &ArtifactDao{
		db: db,
	}
}

type artifactDB struct {
	ID        int64  `db:"artifact_id"`
	Version   string `db:"artifact_version"`
	ImageID   int64  `db:"artifact_image_id"`
	CreatedAt int64  `db:"artifact_created_at"`
	UpdatedAt int64  `db:"artifact_updated_at"`
	CreatedBy int64  `db:"artifact_created_by"`
	UpdatedBy int64  `db:"artifact_updated_by"`
}

func (a ArtifactDao) GetByName(ctx context.Context, imageID int64,
	version string) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts").
		Where("artifact_image_id = ? AND artifact_version = ?", imageID, version)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapToArtifact(ctx, dst)
}

func (a ArtifactDao) CreateOrUpdate(ctx context.Context, artifact *types.Artifact) error {
	const sqlQuery = `
		INSERT INTO artifacts ( 
		         artifact_image_id
				,artifact_version
				,artifact_created_at
				,artifact_updated_at
				,artifact_created_by
				,artifact_updated_by
		    ) VALUES (
						 :artifact_image_id
						,:artifact_version
						,:artifact_created_at
						,:artifact_updated_at
						,:artifact_created_by
						,:artifact_updated_by
		    ) 
            ON CONFLICT (artifact_image_id, artifact_version)
		    DO NOTHING 
            RETURNING artifact_id`

	db := dbtx.GetAccessor(ctx, a.db)
	query, arg, err := db.BindNamed(sqlQuery, a.mapToInternalArtifact(ctx, artifact))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind artifact object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&artifact.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (a ArtifactDao) mapToInternalArtifact(ctx context.Context, in *types.Artifact) *artifactDB {
	session, _ := request.AuthSessionFrom(ctx)

	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.UpdatedAt = time.Now()
	in.UpdatedBy = session.Principal.ID

	return &artifactDB{
		ID:        in.ID,
		Version:   in.Version,
		ImageID:   in.ImageID,
		CreatedAt: in.CreatedAt.UnixMilli(),
		UpdatedAt: in.UpdatedAt.UnixMilli(),
		CreatedBy: in.CreatedBy,
		UpdatedBy: in.UpdatedBy,
	}
}

func (a ArtifactDao) mapToArtifact(_ context.Context, dst *artifactDB) (*types.Artifact, error) {
	createdBy := dst.CreatedBy
	updatedBy := dst.UpdatedBy
	return &types.Artifact{
		ID:        dst.ID,
		Version:   dst.Version,
		ImageID:   dst.ImageID,
		CreatedAt: time.UnixMilli(dst.CreatedAt),
		UpdatedAt: time.UnixMilli(dst.UpdatedAt),
		CreatedBy: createdBy,
		UpdatedBy: updatedBy,
	}, nil
}

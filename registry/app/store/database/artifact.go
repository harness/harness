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
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
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

type ArtifactDao struct {
	db *sqlx.DB
}

func NewArtifactDao(db *sqlx.DB) store.ArtifactRepository {
	return &ArtifactDao{
		db: db,
	}
}

type artifactDB struct {
	ID        int64            `db:"artifact_id"`
	Version   string           `db:"artifact_version"`
	ImageID   int64            `db:"artifact_image_id"`
	Metadata  *json.RawMessage `db:"artifact_metadata"`
	CreatedAt int64            `db:"artifact_created_at"`
	UpdatedAt int64            `db:"artifact_updated_at"`
	CreatedBy int64            `db:"artifact_created_by"`
	UpdatedBy int64            `db:"artifact_updated_by"`
}

func (a ArtifactDao) GetByName(ctx context.Context, imageID int64, version string) (*types.Artifact, error) {
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
				,artifact_metadata
				,artifact_updated_at
				,artifact_created_by
				,artifact_updated_by
		    ) VALUES (
						 :artifact_image_id
						,:artifact_version
						,:artifact_created_at
						,:artifact_metadata
						,:artifact_updated_at
						,:artifact_created_by
						,:artifact_updated_by
		    ) 
            ON CONFLICT (artifact_image_id, artifact_version)
		    DO UPDATE SET artifact_metadata = :artifact_metadata
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

func (a ArtifactDao) Count(ctx context.Context) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("artifacts")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (a ArtifactDao) mapToInternalArtifact(ctx context.Context, in *types.Artifact) *artifactDB {
	session, _ := request.AuthSessionFrom(ctx)

	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	var metadata = json.RawMessage("null")
	if in.Metadata != nil {
		metadata = in.Metadata
	}
	in.UpdatedAt = time.Now()
	in.UpdatedBy = session.Principal.ID

	return &artifactDB{
		ID:        in.ID,
		Version:   in.Version,
		ImageID:   in.ImageID,
		Metadata:  &metadata,
		CreatedAt: in.CreatedAt.UnixMilli(),
		UpdatedAt: in.UpdatedAt.UnixMilli(),
		CreatedBy: in.CreatedBy,
		UpdatedBy: in.UpdatedBy,
	}
}

func (a ArtifactDao) mapToArtifact(_ context.Context, dst *artifactDB) (*types.Artifact, error) {
	createdBy := dst.CreatedBy
	updatedBy := dst.UpdatedBy
	var metadata json.RawMessage
	if dst.Metadata != nil {
		metadata = *dst.Metadata
	}
	return &types.Artifact{
		ID:        dst.ID,
		Version:   dst.Version,
		ImageID:   dst.ImageID,
		Metadata:  metadata,
		CreatedAt: time.UnixMilli(dst.CreatedAt),
		UpdatedAt: time.UnixMilli(dst.UpdatedAt),
		CreatedBy: createdBy,
		UpdatedBy: updatedBy,
	}, nil
}

func (a ArtifactDao) GetAllArtifactsByParentID(
	ctx context.Context,
	parentID int64,
	registryIDs *[]string,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
	latestVersion bool,
	packageTypes []string,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, 
		i.image_name as name, 
		r.registry_package_type as package_type, 
		a.artifact_version as version, 
		a.artifact_updated_at as modified_at, 
		i.image_labels as labels, 
		COALESCE(t2.download_count,0) as download_count `,
	).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Join("registries r ON r.registry_id = i.image_registry_id").
		Where("r.registry_parent_id = ?", parentID).
		LeftJoin(
			`( SELECT i.image_name, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_image_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id 
			GROUP BY a.artifact_image_id ) as t1 
			JOIN images i ON i.image_id = t1.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? GROUP BY i.image_name) as t2 
			ON i.image_name = t2.image_name`, parentID,
		)

	if latestVersion {
		q = q.Join(
			`(SELECT t.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY t.artifact_image_id
			ORDER BY t.artifact_updated_at DESC) AS rank FROM artifacts t 
			JOIN images i ON t.artifact_image_id = i.image_id
            JOIN registries r ON i.image_registry_id = r.registry_id
			WHERE r.registry_parent_id = ? ) AS a1 
			ON a.artifact_id = a1.id`, parentID, // nolint:goconst
		).
			Where("a1.rank = 1")
	}

	if len(*registryIDs) > 0 {
		q = q.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if len(packageTypes) > 0 {
		q = q.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	if search != "" {
		q = q.Where("image_name LIKE ?", sqlPartialMatch(search))
	}
	sortField := "i." + sortByField
	if sortByField == downloadCount {
		sortField = downloadCount
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return a.mapToArtifactMetadataList(ctx, dst)
}

func (a ArtifactDao) CountAllArtifactsByParentID(
	ctx context.Context, parentID int64,
	registryIDs *[]string, search string, latestVersion bool, packageTypes []string,
) (int64, error) {
	// nolint:goconst
	q := databaseg.Builder.Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id"). // nolint:goconst
		Where("r.registry_parent_id = ?", parentID)

	if latestVersion {
		q = q.Join(
			`(SELECT t.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY t.artifact_image_id
			ORDER BY t.artifact_updated_at DESC) AS rank FROM artifacts t 
			JOIN images i ON t.artifact_image_id = i.image_id
            JOIN registries r ON i.image_registry_id = r.registry_id
			WHERE r.registry_parent_id = ? ) AS a1 
			ON a.artifact_id = a1.id`, parentID, // nolint:goconst
		).
			Where("a1.rank = 1")
	}
	if len(*registryIDs) > 0 {
		q = q.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if search != "" {
		q = q.Where("image_name LIKE ?", sqlPartialMatch(search))
	}

	if len(packageTypes) > 0 {
		q = q.Where(sq.Eq{"registry_package_type": packageTypes})
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}
	db := dbtx.GetAccessor(ctx, a.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (a ArtifactDao) GetAllArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey string,
	sortByField string, sortByOrder string, limit int, offset int, search string,
	labels []string,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, i.image_name as name, 
		r.registry_package_type as package_type, a.artifact_version as latest_version, 
		a.artifact_updated_at as modified_at, i.image_labels as labels, 
		COALESCE(t2.download_count, 0) as download_count `,
	).
		From("artifacts a").
		Join(
			`(SELECT a.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY a.artifact_image_id
			ORDER BY a.artifact_updated_at DESC) AS rank FROM artifacts a 
            JOIN images i ON i.image_id = a.artifact_image_id  
			JOIN registries r ON i.image_registry_id = r.registry_id  
			WHERE r.registry_parent_id = ? AND r.registry_name = ? ) AS a1 
			ON a.artifact_id = a1.id`, parentID, repoKey, // nolint:goconst
		).
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		LeftJoin(
			`( SELECT i.image_name, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_image_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a 
			JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id GROUP BY 
			a.artifact_image_id ) as t1 
			JOIN images i ON i.image_id = t1.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? AND r.registry_name = ? GROUP BY i.image_name) as t2 
			ON i.image_name = t2.image_name`, parentID, repoKey,
		).
		Where("a1.rank = 1 ")

	if search != "" {
		q = q.Where("image_name LIKE ?", sqlPartialMatch(search))
	}

	if len(labels) > 0 {
		sort.Strings(labels)
		labelsVal := util.GetEmptySQLString(util.ArrToString(labels))
		labelsVal.String = labelSeparatorStart + labelsVal.String + labelSeparatorEnd
		q = q.Where("'^_' || i.image_labels || '^_' LIKE ?", labelsVal)
	}

	// nolint:goconst
	sortField := "image_" + sortByField
	if sortByField == downloadCount {
		sortField = downloadCount
	} else if sortByField == imageName {
		sortField = name
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return a.mapToArtifactMetadataList(ctx, dst)
}

// nolint:goconst
func (a ArtifactDao) CountAllArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey string,
	search string, labels []string,
) (int64, error) {
	q := databaseg.Builder.Select("COUNT(*)").
		From("artifacts a").
		Join(
			`(SELECT a.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY a.artifact_image_id 
			ORDER BY a.artifact_updated_at DESC) AS rank FROM artifacts a 
			JOIN registries r ON t.tag_registry_id = r.registry_id 
			WHERE r.registry_parent_id = ? AND r.registry_name = ? ) AS a1 ON a.artifact_id = a1.id`, parentID, repoKey,
		).
		Join(
			"images i ON i.image_id = a.artifact_image_id AND").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("a1.rank = 1 ")

	if search != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}

	if len(labels) > 0 {
		sort.Strings(labels)
		labelsVal := util.GetEmptySQLString(util.ArrToString(labels))
		labelsVal.String = labelSeparatorStart + labelsVal.String + labelSeparatorEnd
		q = q.Where("'^_' || i.image_labels || '^_' LIKE ?", labelsVal)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}
	db := dbtx.GetAccessor(ctx, a.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (a ArtifactDao) GetLatestArtifactMetadata(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
) (*types.ArtifactMetadata, error) {
	// Precomputed download count subquery
	downloadCountSubquery := `
		SELECT 
			i.image_name, 
			i.image_registry_id,
			SUM(COALESCE(dc.download_count, 0)) AS download_count
		FROM 
			images i
		LEFT JOIN (
			SELECT 
				a.artifact_image_id, 
				COUNT(d.download_stat_id) AS download_count
			FROM 
				artifacts a
			JOIN 
				download_stats d ON d.download_stat_artifact_id = a.artifact_id
			GROUP BY 
				a.artifact_image_id
		) AS dc ON i.image_id = dc.artifact_image_id
		GROUP BY 
			i.image_name, i.image_registry_id
	`
	var q sq.SelectBuilder
	if a.db.DriverName() == SQLITE3 {
		q = databaseg.Builder.Select(
			`r.registry_name AS repo_name, r.registry_package_type AS package_type,
     i.image_name AS name, a.artifact_version AS latest_version,
     a.artifact_created_at AS created_at, a.artifact_updated_at AS modified_at,
     i.image_labels AS labels, COALESCE(dc_subquery.download_count, 0) AS download_count`,
		).
			From("artifacts a").
			Join("images i ON i.image_id = a.artifact_image_id").
			Join("registries r ON i.image_registry_id = r.registry_id"). // nolint:goconst
			LeftJoin(fmt.Sprintf("(%s) AS dc_subquery ON dc_subquery.image_name = i.image_name "+
				"AND dc_subquery.image_registry_id = r.registry_id", downloadCountSubquery)).
			Where(
				"r.registry_parent_id = ? AND r.registry_name = ? AND i.image_name = ?",
				parentID, repoKey, imageName,
			).
			OrderBy("a.artifact_updated_at DESC").Limit(1)
	} else {
		q = databaseg.Builder.Select(
			`r.registry_name AS repo_name,
         r.registry_package_type AS package_type,
         i.image_name AS name,
         a.artifact_version AS latest_version,
         a.artifact_created_at AS created_at,
         a.artifact_updated_at AS modified_at,
         i.image_labels AS labels,
         COALESCE(t2.download_count, 0) AS download_count`,
		).
			From("artifacts a").
			Join("images i ON i.image_id = a.artifact_image_id").
			Join("registries r ON i.image_registry_id = r.registry_id"). // nolint:goconst
			LeftJoin(fmt.Sprintf("LATERAL (%s) AS t2 ON i.image_name = t2.image_name", downloadCountSubquery)).
			Where(
				"r.registry_parent_id = ? AND r.registry_name = ? AND i.image_name = ?",
				parentID, repoKey, imageName,
			).
			OrderBy("a.artifact_updated_at DESC").Limit(1)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}
	// Log the final sql query
	finalQuery := util.FormatQuery(sql, args)
	log.Ctx(ctx).Debug().Str("sql", finalQuery).Msg("Executing GetLatestTagMetadata query")
	// Execute query
	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactMetadataDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}

	return a.mapToArtifactMetadata(ctx, dst)
}

func (a ArtifactDao) mapToArtifactMetadataList(
	ctx context.Context,
	dst []*artifactMetadataDB,
) (*[]types.ArtifactMetadata, error) {
	artifacts := make([]types.ArtifactMetadata, 0, len(dst))
	for _, d := range dst {
		artifact, err := a.mapToArtifactMetadata(ctx, d)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, *artifact)
	}
	return &artifacts, nil
}

func (a ArtifactDao) GetAllVersionsByRepoAndImage(
	ctx context.Context, parentID int64, repoKey string,
	image string, sortByField string, sortByOrder string, limit int, offset int,
	search string,
) (*[]types.NonOCIArtifactMetadata, error) {
	// Define download count subquery
	downloadCountSubquery := `
    SELECT 
        a.artifact_image_id, 
        COUNT(d.download_stat_id) AS download_count, 
        i.image_name, 
        i.image_registry_id
    FROM artifacts a
    JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id
    JOIN images i ON i.image_id = a.artifact_image_id
    GROUP BY a.artifact_image_id, i.image_name, i.image_registry_id
`

	// Build the main query
	q := databaseg.Builder.
		Select(`
        a.artifact_version AS name, 
        a.artifact_metadata ->> 'size' AS size, 
        a.artifact_metadata ->> 'file_count' AS file_count, 
        r.registry_package_type AS package_type, 
        a.artifact_updated_at AS modified_at,
        COALESCE(dc.download_count, 0) AS download_count
    `)

	if a.db.DriverName() == SQLITE3 {
		q = databaseg.Builder.Select(`
        a.artifact_version AS name, 
        json_extract(a.artifact_metadata, '$.size') AS size,
        json_extract(a.artifact_metadata, '$.file_count') AS file_count,
        r.registry_package_type AS package_type, 
        a.artifact_updated_at AS modified_at,
        COALESCE(dc.download_count, 0) AS download_count
    `)
	}

	q = q.From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		LeftJoin(fmt.Sprintf(
			"(%s) AS dc ON a.artifact_image_id = dc.artifact_image_id",
			downloadCountSubquery,
		)).
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ? AND i.image_name = ?",
			parentID, repoKey, image,
		)
	if search != "" {
		q = q.Where("artifact_version LIKE ?", sqlPartialMatch(search))
	}
	// nolint:goconst
	sortField := "image_" + sortByField
	if sortByField == downloadCount {
		sortField = downloadCount
	} else if sortByField == name {
		sortField = name
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*nonOCIArtifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return a.mapToNonOCIMetadataList(dst)
}

func (a ArtifactDao) CountAllVersionsByRepoAndImage(
	ctx context.Context, parentID int64,
	repoKey string, image string, search string,
) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ? "+
				"AND i.image_name = ?", parentID, repoKey, image,
		)

	if search != "" {
		stmt = stmt.Where("artifact_version LIKE ?", sqlPartialMatch(search))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (a ArtifactDao) GetLatestVersionName(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
) (string, error) {
	q := databaseg.Builder.Select("artifact_version as name").
		From("artifacts a").
		Join("images ON i.image_id = a.artifact_image_id").
		Join("registries ON i.image_registry_id = registry_id").
		Where(
			"registry_parent_id = ? AND registry_name = ? AND i.image_name = ?",
			parentID, repoKey, imageName,
		).
		OrderBy("artifact_updated_at DESC").Limit(1)

	sql, args, err := q.ToSql()
	if err != nil {
		return "", errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var tag string
	err = db.QueryRowContext(ctx, sql, args...).Scan(&tag)
	if err != nil {
		return tag, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing get tag name query")
	}
	return tag, nil
}

func (a ArtifactDao) GetArtifactMetadata(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
	name string,
) (*types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		"r.registry_package_type as package_type, a.artifact_version as name,"+
			"a.artifact_updated_at as modified_at").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = registry_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ?"+
				" AND i.image_name = ? AND a.artifact_version = ?", parentID, repoKey, imageName, name,
		)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactMetadataDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
	}

	return a.mapToArtifactMetadata(ctx, dst)
}

func (a ArtifactDao) mapToArtifactMetadata(
	_ context.Context,
	dst *artifactMetadataDB,
) (*types.ArtifactMetadata, error) {
	return &types.ArtifactMetadata{
		Name:          dst.Name,
		RepoName:      dst.RepoName,
		DownloadCount: dst.DownloadCount,
		PackageType:   dst.PackageType,
		LatestVersion: dst.LatestVersion,
		Labels:        util.StringToArr(dst.Labels.String),
		CreatedAt:     time.UnixMilli(dst.CreatedAt),
		ModifiedAt:    time.UnixMilli(dst.ModifiedAt),
		Version:       dst.Version,
	}, nil
}

func (a ArtifactDao) mapToNonOCIMetadata(
	dst *nonOCIArtifactMetadataDB) *types.NonOCIArtifactMetadata {
	var size string
	var fileCount int64

	if dst.Size != nil {
		size = *dst.Size
	}

	if dst.FileCount != nil {
		fileCount = *dst.FileCount
	}
	return &types.NonOCIArtifactMetadata{
		Name:            dst.Name,
		DownloadCount:   dst.DownloadCount,
		PackageType:     dst.PackageType,
		Size:            size,
		FileCount:       fileCount,
		IsLatestVersion: dst.IsLatestVersion,
		ModifiedAt:      time.UnixMilli(dst.ModifiedAt),
	}
}

func (a ArtifactDao) mapToNonOCIMetadataList(
	dst []*nonOCIArtifactMetadataDB) (*[]types.NonOCIArtifactMetadata, error) {
	metadataList := make([]types.NonOCIArtifactMetadata, 0, len(dst))
	for _, d := range dst {
		metadata := a.mapToNonOCIMetadata(d)
		metadataList = append(metadataList, *metadata)
	}
	return &metadataList, nil
}

type nonOCIArtifactMetadataDB struct {
	Name            string               `db:"name"`
	Size            *string              `db:"size"`
	PackageType     artifact.PackageType `db:"package_type"`
	FileCount       *int64               `db:"file_count"`
	IsLatestVersion bool                 `db:"is_latest_version"`
	ModifiedAt      int64                `db:"modified_at"`
	DownloadCount   int64                `db:"download_count"`
}

type GenericMetadata struct {
	Files       []File `json:"files"`
	Description string `json:"desc"`
	FileCount   int64  `json:"file_count"`
}

type MavenMetadata struct {
	Files     []File `json:"files"`
	FileCount int64  `json:"file_count"`
}

type File struct {
	Size      int64  `json:"size"`
	Filename  string `json:"file_name"`
	CreatedAt int64  `json:"created_at"`
}

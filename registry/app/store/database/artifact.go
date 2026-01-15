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
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	gitness_store "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
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
	UUID      string           `db:"artifact_uuid"`
	Version   string           `db:"artifact_version"`
	ImageID   int64            `db:"artifact_image_id"`
	Metadata  *json.RawMessage `db:"artifact_metadata"`
	CreatedAt int64            `db:"artifact_created_at"`
	UpdatedAt int64            `db:"artifact_updated_at"`
	CreatedBy int64            `db:"artifact_created_by"`
	UpdatedBy int64            `db:"artifact_updated_by"`
}

func (a ArtifactDao) GetByUUID(ctx context.Context, uuid string) (*types.Artifact, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts").
		Where("artifact_uuid = ?", uuid)

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find artifact by uuid")
	}

	return a.mapToArtifact(ctx, dst)
}

func (a ArtifactDao) Get(ctx context.Context, id int64) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts").
		Where("artifact_id = ?", id)

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

func (a ArtifactDao) GetByRegistryImageAndVersion(
	ctx context.Context, registryID int64, image string, version string,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ?", registryID).
		Where("i.image_name = ?", image).
		Where("a.artifact_version = ?", version)

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

func (a ArtifactDao) GetByRegistryImageVersionAndArtifactType(
	ctx context.Context, registryID int64, image string, version string, artifactType string,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ?", registryID).
		Where("i.image_name = ?", image).
		Where("i.image_type = ?", artifactType).
		Where("a.artifact_version = ?", version)

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

func (a ArtifactDao) GetByRegistryIDAndImage(ctx context.Context, registryID int64, image string) (
	*[]types.Artifact,
	error,
) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ? AND i.image_name = ? AND i.image_type IS NULL", registryID, image).
		OrderBy("a.artifact_created_at DESC")

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []artifactDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifacts")
	}

	artifacts := make([]types.Artifact, len(dst))
	for i := range dst {
		d := dst[i]
		art, err := a.mapToArtifact(ctx, &d)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to map artifact")
		}
		artifacts[i] = *art
	}

	return &artifacts, nil
}

func (a ArtifactDao) GetLatestByImageID(ctx context.Context, imageID int64) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts").
		Where("artifact_image_id = ?", imageID).OrderBy("artifact_updated_at DESC").Limit(1)

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

func (a ArtifactDao) CreateOrUpdate(ctx context.Context, artifact *types.Artifact) (int64, error) {
	if commons.IsEmpty(artifact.Version) {
		return 0, errors.New("version is empty")
	}

	const sqlQuery = `
		INSERT INTO artifacts ( 
		         artifact_image_id
				,artifact_version
				,artifact_created_at
				,artifact_metadata
				,artifact_updated_at
				,artifact_created_by
				,artifact_updated_by
				,artifact_uuid
		    ) VALUES (
						 :artifact_image_id
						,:artifact_version
						,:artifact_created_at
						,:artifact_metadata
						,:artifact_updated_at
						,:artifact_created_by
						,:artifact_updated_by
						,:artifact_uuid
		    ) 
            ON CONFLICT (artifact_image_id, artifact_version)
		    DO UPDATE SET artifact_metadata = :artifact_metadata
            RETURNING artifact_id`

	db := dbtx.GetAccessor(ctx, a.db)
	query, arg, err := db.BindNamed(sqlQuery, a.mapToInternalArtifact(ctx, artifact))
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind artifact object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&artifact.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return artifact.ID, nil
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

func (a ArtifactDao) DuplicateArtifact(
	ctx context.Context, sourceArtifact *types.Artifact, targetImageID int64,
) (*types.Artifact, error) {
	targetArtifact := &types.Artifact{
		ImageID:  targetImageID,
		Version:  sourceArtifact.Version,
		Metadata: sourceArtifact.Metadata,
	}

	_, err := a.CreateOrUpdate(ctx, targetArtifact)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to duplicate artifact")
	}

	return targetArtifact, nil
}

func (a ArtifactDao) DeleteByImageNameAndRegistryID(ctx context.Context, regID int64, image string) (err error) {
	var delStmt sq.DeleteBuilder
	switch a.db.DriverName() {
	case SQLITE3:
		delStmt = databaseg.Builder.Delete("artifacts").
			Where("artifact_id IN (SELECT a.artifact_id FROM artifacts a JOIN images i ON i.image_id = a.artifact_image_id"+
				" WHERE i.image_name = ? AND i.image_registry_id = ?)", image, regID)
	default:
		delStmt = databaseg.Builder.Delete("artifacts a USING images i").
			Where("a.artifact_image_id = i.image_id").
			Where("i.image_name = ? AND i.image_registry_id = ?", image, regID)
	}

	db := dbtx.GetAccessor(ctx, a.db)

	delQuery, delArgs, err := delStmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert delete query to sql: %w", err)
	}

	_, err = db.ExecContext(ctx, delQuery, delArgs...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (a ArtifactDao) DeleteByVersionAndImageName(
	ctx context.Context, image string,
	version string, regID int64,
) (err error) {
	var delStmt sq.DeleteBuilder
	switch a.db.DriverName() {
	case SQLITE3:
		delStmt = databaseg.Builder.Delete("artifacts").
			Where("artifact_id IN (SELECT a.artifact_id FROM artifacts a JOIN images i ON i.image_id = a.artifact_image_id"+
				" WHERE a.artifact_version = ? AND i.image_name = ? AND i.image_registry_id = ?)", version, image,
				regID)

	default:
		delStmt = databaseg.Builder.Delete("artifacts a USING images i").
			Where("a.artifact_image_id = i.image_id").
			Where("a.artifact_version = ? AND i.image_name = ? AND i.image_registry_id = ?", version, image, regID)
	}

	sql, args, err := delStmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
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

	var metadata = json.RawMessage("null")
	if in.Metadata != nil {
		metadata = in.Metadata
	}
	in.UpdatedAt = time.Now()
	in.UpdatedBy = session.Principal.ID

	if in.UUID == "" {
		in.UUID = uuid.NewString()
	}

	return &artifactDB{
		ID:        in.ID,
		UUID:      in.UUID,
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
		UUID:      dst.UUID,
		Version:   dst.Version,
		ImageID:   dst.ImageID,
		Metadata:  metadata,
		CreatedAt: time.UnixMilli(dst.CreatedAt),
		UpdatedAt: time.UnixMilli(dst.UpdatedAt),
		CreatedBy: createdBy,
		UpdatedBy: updatedBy,
	}, nil
}

func (a ArtifactDao) SearchLatestByName(
	ctx context.Context, regID int64, name string, limit int, offset int,
) (*[]types.Artifact, error) {
	subQuery := `
	SELECT artifact_image_id, MAX(artifact_created_at) AS max_created_at
	FROM artifacts
	GROUP BY artifact_image_id`

	q := databaseg.Builder.
		Select("a.artifact_metadata,"+
			"a.artifact_created_at").
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Join(fmt.Sprintf(`(%s) latest
	ON a.artifact_image_id = latest.artifact_image_id
	AND a.artifact_created_at = latest.max_created_at
`, subQuery)).
		Where("i.image_name LIKE ? AND i.image_registry_id = ?", "%"+name+"%", regID).
		Limit(util.SafeIntToUInt64(limit)).
		Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build SQL for latest artifact metadata with pagination")
	}
	db := dbtx.GetAccessor(ctx, a.db)

	var metadataList []*artifactDB
	if err := db.SelectContext(ctx, &metadataList, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
	}

	artifactList, err := a.mapArtifactToArtifactMetadataList(ctx, metadataList)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to map artifact metadata")
	}

	return artifactList, nil
}

func (a ArtifactDao) CountLatestByName(
	ctx context.Context, regID int64, name string,
) (int64, error) {
	subQuery := `
	SELECT artifact_image_id, MAX(artifact_created_at) AS max_created_at
	FROM artifacts
	GROUP BY artifact_image_id`

	// Main count query
	q := databaseg.Builder.
		Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Join(fmt.Sprintf(`(%s) latest
	ON a.artifact_image_id = latest.artifact_image_id
	AND a.artifact_created_at = latest.max_created_at
`, subQuery)).
		Where("i.image_name LIKE ? AND i.image_registry_id = ?", "%"+name+"%", regID)

	sql, args, err := q.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to build count SQL")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var count int64
	if err := db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed to count artifact metadata")
	}

	return count, nil
}

func (a ArtifactDao) SearchByImageName(
	ctx context.Context, regID int64, name string, limit int,
	offset int,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`i.image_name as name,
        a.artifact_id as artifact_id, a.artifact_version as version, a.artifact_metadata as metadata`,
	).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ?", regID)
	if name != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(name))
	}
	q = q.OrderBy("i.image_name ASC, a.artifact_version ASC").
		Limit(util.SafeIntToUInt64(limit)).
		Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build SQL for"+
			" artifact metadata with pagination")
	}
	db := dbtx.GetAccessor(ctx, a.db)

	var dst []*artifactMetadataDB
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
	}
	return a.mapToArtifactMetadataList(dst)
}

func (a ArtifactDao) CountByImageName(
	ctx context.Context, regID int64, name string,
) (int64, error) {
	q := databaseg.Builder.
		Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ?", regID)
	if name != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(name))
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to build count SQL")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var count int64
	if err := db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed to count artifact metadata")
	}

	return count, nil
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
		a.artifact_metadata as metadata,
		COALESCE(t2.download_count,0) as download_count `,
	).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Join("registries r ON r.registry_id = i.image_registry_id").
		Where("r.registry_parent_id = ?", parentID).
		LeftJoin(
			`( SELECT i.image_id, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_image_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id 
			GROUP BY a.artifact_image_id ) as t1 
			JOIN images i ON i.image_id = t1.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? GROUP BY i.image_id) as t2 
			ON i.image_id = t2.image_id`, parentID,
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
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}
	sortField := "i." + sortByField
	if sortByField == downloadCount {
		sortField = downloadCount
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(util.SafeIntToUInt64(limit)).Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return a.mapToArtifactMetadataList(dst)
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

func (a ArtifactDao) GetArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey string, sortByField string,
	sortByOrder string, limit int, offset int, search string, labels []string,
	artifactType *artifact.ArtifactType,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, i.image_name as name, i.image_uuid as uuid,
		r.registry_uuid as registry_uuid,
		r.registry_package_type as package_type, a.artifact_version as latest_version, 
		a.artifact_updated_at as modified_at, i.image_labels as labels, i.image_type as artifact_type,
		COALESCE(t2.download_count, 0) as download_count`,
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
			`( SELECT i.image_id, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_image_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a 
			JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id GROUP BY 
			a.artifact_image_id ) as t1 
			JOIN images i ON i.image_id = t1.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? AND r.registry_name = ? GROUP BY i.image_id) as t2 
			ON i.image_id = t2.image_id`, parentID, repoKey,
		).
		Where("a1.rank = 1 ")

	if search != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}
	if artifactType != nil && *artifactType != "" {
		q = q.Where("i.image_type = ?", *artifactType)
	}

	if len(labels) > 0 {
		sort.Strings(labels)
		labelsVal := util.GetEmptySQLString(util.ArrToString(labels))
		labelsVal.String = labelSeparatorStart + labelsVal.String + labelSeparatorEnd
		q = q.Where("'^_' || i.image_labels || '^_' LIKE ?", labelsVal)
	}

	// nolint:goconst
	sortField := "image_" + sortByField
	switch sortByField {
	case downloadCount:
		sortField = downloadCount
	case imageName:
		sortField = name
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(util.SafeIntToUInt64(limit)).Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return a.mapToArtifactMetadataList(dst)
}

// nolint:goconst
func (a ArtifactDao) CountArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey, search string, labels []string,
	artifactType *artifact.ArtifactType,
) (int64, error) {
	q := databaseg.Builder.Select("COUNT(*)").
		From("artifacts a").
		Join(
			"images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ?", parentID, repoKey)
	if search != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}
	if artifactType != nil && *artifactType != "" {
		q = q.Where("i.image_type = ?", *artifactType)
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

	return a.mapToArtifactMetadata(dst)
}

func (a ArtifactDao) mapArtifactToArtifactMetadataList(
	ctx context.Context,
	dst []*artifactDB,
) (*[]types.Artifact, error) {
	artifacts := make([]types.Artifact, 0, len(dst))
	for _, d := range dst {
		artifact, err := a.mapToArtifact(ctx, d)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, *artifact)
	}
	return &artifacts, nil
}

func (a ArtifactDao) mapToArtifactMetadataList(
	dst []*artifactMetadataDB,
) (*[]types.ArtifactMetadata, error) {
	artifacts := make([]types.ArtifactMetadata, 0, len(dst))
	for _, d := range dst {
		artifact, err := a.mapToArtifactMetadata(d)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, *artifact)
	}
	return &artifacts, nil
}

func (a ArtifactDao) GetAllVersionsByRepoAndImage(
	ctx context.Context, regID int64, image string,
	sortByField string, sortByOrder string, limit int, offset int, search string,
	artifactType *artifact.ArtifactType,
) (*[]types.NonOCIArtifactMetadata, error) {
	// Build the main query
	q := databaseg.Builder.
		Select(`
        a.artifact_id as id,
				a.artifact_uuid as uuid,
        a.artifact_version AS name, 
        a.artifact_metadata ->> 'size' AS size, 
        a.artifact_metadata ->> 'file_count' AS file_count,
        a.artifact_updated_at AS modified_at,
        (qp.quarantined_path_id IS NOT NULL) AS is_quarantined,
         qp.quarantined_path_reason as quarantine_reason,
		i.image_type as artifact_type`,
		)

	if a.db.DriverName() == SQLITE3 {
		q = databaseg.Builder.Select(`
        a.artifact_id as id,
				a.artifact_uuid as uuid,
        a.artifact_version AS name, 
        json_extract(a.artifact_metadata, '$.size') AS size,
        json_extract(a.artifact_metadata, '$.file_count') AS file_count,
        a.artifact_updated_at AS modified_at,
        (qp.quarantined_path_id IS NOT NULL) AS is_quarantined,
         qp.quarantined_path_reason as quarantine_reason,
		i.image_type as artifact_type`,
		)
	}

	q = q.From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		LeftJoin("quarantined_paths qp ON ((qp.quarantined_path_artifact_id = a.artifact_id "+
			"OR qp.quarantined_path_artifact_id IS NULL) "+
			"AND qp.quarantined_path_image_id = i.image_id) AND qp.quarantined_path_registry_id = ?", regID).
		Where(
			" i.image_registry_id = ? AND i.image_name = ?",
			regID, image,
		)
	if artifactType != nil && *artifactType != "" {
		q = q.Where("i.image_type = ?", *artifactType)
	}
	if search != "" {
		q = q.Where("artifact_version LIKE ?", sqlPartialMatch(search))
	}
	// nolint:goconst
	sortField := "artifact_" + sortByField
	if sortByField == name || sortByField == downloadCount {
		sortField = name
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(util.SafeIntToUInt64(limit)).Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*nonOCIArtifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	artifactIDs := make([]any, 0, len(dst))
	for _, art := range dst {
		artifactIDs = append(artifactIDs, art.ID)
	}
	err = a.fetchDownloadStatsForArtifacts(ctx, artifactIDs, dst, sortByField == downloadCount, sortByOrder)
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to fetch the download count for artifacts")
	}
	return a.mapToNonOCIMetadataList(dst)
}

func (a ArtifactDao) fetchDownloadStatsForArtifacts(
	ctx context.Context,
	artifactIDs []any, dst []*nonOCIArtifactMetadataDB, sortByDownloadCount bool, sortOrder string,
) error {
	if len(artifactIDs) == 0 {
		return nil
	}

	query, args, err := databaseg.Builder.
		Select("download_stat_artifact_id", "COUNT(*) AS download_count").
		From("download_stats").
		Where(sq.Eq{"download_stat_artifact_id": artifactIDs}).
		GroupBy("download_stat_artifact_id").
		ToSql()
	if err != nil {
		return errors.Wrap(err, "building download stats query")
	}

	results := []downloadCountResult{}
	if err := a.db.SelectContext(ctx, &results, query, args...); err != nil {
		return errors.Wrap(err, "executing download stats query")
	}

	// Map artifact ID -> count
	countMap := make(map[string]int64, len(results))
	for _, r := range results {
		countMap[r.ArtifactID] = r.DownloadCount
	}

	// Update download counts in dst
	for _, artifact := range dst {
		if count, ok := countMap[artifact.ID]; ok {
			artifact.DownloadCount = count
		} else {
			artifact.DownloadCount = 0
		}
	}

	if sortByDownloadCount {
		sort.Slice(dst, func(i, j int) bool {
			if sortOrder == "DESC" {
				return dst[i].DownloadCount > dst[j].DownloadCount
			}
			return dst[i].DownloadCount < dst[j].DownloadCount
		})
	}

	return nil
}

func (a ArtifactDao) CountAllVersionsByRepoAndImage(
	ctx context.Context, parentID int64, repoKey string, image string,
	search string, artifactType *artifact.ArtifactType,
) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ? "+
				"AND i.image_name = ?", parentID, repoKey, image,
		)

	if artifactType != nil && *artifactType != "" {
		stmt = stmt.Where("i.image_type = ?", *artifactType)
	}

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

func (a ArtifactDao) GetArtifactMetadata(
	ctx context.Context, id int64, identifier string, image string, version string,
	artifactType *artifact.ArtifactType,
) (*types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		"r.registry_package_type as package_type, a.artifact_version as name, a.artifact_uuid as uuid,"+
			"a.artifact_updated_at as modified_at, r.registry_uuid as registry_uuid, "+
			"COALESCE(COUNT(dc.download_stat_id), 0) as download_count").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = registry_id").
		LeftJoin("download_stats dc ON dc.download_stat_artifact_id = a.artifact_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ?"+
				" AND i.image_name = ? AND a.artifact_version = ?", id, identifier, image, version,
		).
		GroupBy("r.registry_package_type, a.artifact_version, a.artifact_updated_at, " +
			"a.artifact_uuid, r.registry_uuid")

	if artifactType != nil && *artifactType != "" {
		q = q.Where("i.image_type = ?", *artifactType)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactMetadataDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
	}

	return a.mapToArtifactMetadata(dst)
}

func (a ArtifactDao) UpdateArtifactMetadata(
	ctx context.Context, metadata json.RawMessage,
	artifactID int64,
) (err error) {
	var principalID int64
	session, _ := request.AuthSessionFrom(ctx)
	if session != nil {
		principalID = session.Principal.ID
	}

	q := databaseg.Builder.Update("artifacts").
		Set("artifact_metadata", metadata).
		Set("artifact_updated_at", time.Now().UnixMilli()).
		Set("artifact_updated_by", principalID).
		Where("artifact_id = ?", artifactID)

	sql, args, err := q.ToSql()

	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind artifacts object")
	}

	result, err := a.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to update artifact")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrResourceNotFound
	}

	return nil
}

func (a ArtifactDao) GetLatestArtifactsByRepo(
	ctx context.Context, registryID int64, batchSize int, artifactID int64,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, i.image_name as name,
		a.artifact_id as artifact_id, a.artifact_version as version, a.artifact_metadata as metadata`,
	).
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Join(
			`(SELECT t.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY t.artifact_image_id
			ORDER BY t.artifact_updated_at DESC) AS rank FROM artifacts t 
			JOIN images i ON t.artifact_image_id = i.image_id
			JOIN registries r ON i.image_registry_id = r.registry_id
			WHERE r.registry_id = ? ) AS a1 
			ON a.artifact_id = a1.id`, registryID,
		).
		Where("a.artifact_id > ? AND r.registry_id = ?", artifactID, registryID).
		Where("a1.rank = 1").
		OrderBy("a.artifact_id ASC").
		Limit(util.SafeIntToUInt64(batchSize))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var dst []*artifactMetadataDB
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing GetLatestArtifactsByRepo query")
	}

	return a.mapToArtifactMetadataList(dst)
}

func (a ArtifactDao) GetAllArtifactsByRepo(
	ctx context.Context, registryID int64, batchSize int, artifactID int64,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, i.image_name as name,
        a.artifact_id as artifact_id, a.artifact_version as version, a.artifact_metadata as metadata`,
	).
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("artifact_id > ? AND r.registry_id = ?", artifactID, registryID).
		OrderBy("artifact_id ASC").
		Limit(util.SafeIntToUInt64(batchSize))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var dst []*artifactMetadataDB
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing GetAllArtifactsByRepo query")
	}

	return a.mapToArtifactMetadataList(dst)
}

// GetArtifactsByRepoAndImageBatch retrieves a batch of artifacts by repository and image name.
// Initial lastArtifactID can be set to 0 to start from the beginning.
// max batchSize is the maximum number of artifacts to return, capped at 100.
// If there is an error executing the query, the function will return an error.
func (a ArtifactDao) GetArtifactsByRepoAndImageBatch(
	ctx context.Context, registryID int64, imageName string, batchSize int, lastArtifactID int64,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, i.image_name as name,
        a.artifact_id as artifact_id, a.artifact_version as version, a.artifact_metadata as metadata`,
	).
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("artifact_id > ? AND r.registry_id = ?", lastArtifactID, registryID).
		Where("i.image_name = ?", imageName).
		OrderBy("artifact_id ASC").
		Limit(util.SafeIntToUInt64(util.MinInt(batchSize, 100)))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	var dst []*artifactMetadataDB
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing GetAllArtifactsByRepoAndImage query")
	}

	return a.mapToArtifactMetadataList(dst)
}

func (a ArtifactDao) mapToArtifactMetadata(
	dst *artifactMetadataDB,
) (*types.ArtifactMetadata, error) {
	artifactMetadata := &types.ArtifactMetadata{
		ID:               dst.ID,
		UUID:             dst.UUID,
		RegistryUUID:     dst.RegistryUUID,
		Name:             dst.Name,
		RepoName:         dst.RepoName,
		DownloadCount:    dst.DownloadCount,
		PackageType:      dst.PackageType,
		LatestVersion:    dst.LatestVersion,
		Labels:           util.StringToArr(dst.Labels.String),
		CreatedAt:        time.UnixMilli(dst.CreatedAt),
		ModifiedAt:       time.UnixMilli(dst.ModifiedAt),
		Version:          dst.Version,
		IsQuarantined:    dst.IsQuarantined,
		QuarantineReason: dst.QuarantineReason,
		ArtifactType:     dst.ArtifactType,
	}
	if dst.Metadata != nil {
		artifactMetadata.Metadata = *dst.Metadata
	}
	return artifactMetadata, nil
}

func (a ArtifactDao) mapToNonOCIMetadata(
	dst *nonOCIArtifactMetadataDB,
) *types.NonOCIArtifactMetadata {
	var size string
	var fileCount int64
	if dst.Size != nil {
		size = *dst.Size
	}

	if dst.FileCount != nil {
		fileCount = *dst.FileCount
	}
	return &types.NonOCIArtifactMetadata{
		Name:             dst.Name,
		UUID:             dst.UUID,
		DownloadCount:    dst.DownloadCount,
		PackageType:      dst.PackageType,
		Size:             size,
		FileCount:        fileCount,
		ModifiedAt:       time.UnixMilli(dst.ModifiedAt),
		IsQuarantined:    dst.IsQuarantined,
		QuarantineReason: dst.QuarantineReason,
		ArtifactType:     dst.ArtifactType,
	}
}

func (a ArtifactDao) mapToNonOCIMetadataList(
	dst []*nonOCIArtifactMetadataDB,
) (*[]types.NonOCIArtifactMetadata, error) {
	metadataList := make([]types.NonOCIArtifactMetadata, 0, len(dst))
	for _, d := range dst {
		metadata := a.mapToNonOCIMetadata(d)
		metadataList = append(metadataList, *metadata)
	}
	return &metadataList, nil
}

type nonOCIArtifactMetadataDB struct {
	ID               string                 `db:"id"`
	UUID             string                 `db:"uuid"`
	Name             string                 `db:"name"`
	Size             *string                `db:"size"`
	PackageType      artifact.PackageType   `db:"package_type"`
	FileCount        *int64                 `db:"file_count"`
	ModifiedAt       int64                  `db:"modified_at"`
	DownloadCount    int64                  `db:"download_count"`
	IsQuarantined    bool                   `db:"is_quarantined"`
	QuarantineReason *string                `db:"quarantine_reason"`
	ArtifactType     *artifact.ArtifactType `db:"artifact_type"`
}

type downloadCountResult struct {
	ArtifactID    string `db:"download_stat_artifact_id"`
	DownloadCount int64  `db:"download_count"`
}

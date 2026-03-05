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
	DeletedAt *int64           `db:"artifact_deleted_at"`
	DeletedBy *int64           `db:"artifact_deleted_by"`
}

// GetByUUID gets an artifact by its UUID (includes soft-deleted artifacts).
func (a ArtifactDao) GetByUUID(
	ctx context.Context, uuid string,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Where("a.artifact_uuid = ?", uuid)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing find query")
	}

	return a.mapArtifactDB(ctx, dst)
}

func (a ArtifactDao) Get(
	ctx context.Context, id int64,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Where("a.artifact_id = ?", id)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing find query")
	}

	return a.mapArtifactDB(ctx, dst)
}

func (a ArtifactDao) GetByName(
	ctx context.Context, imageID int64, version string, opts ...types.QueryOption,
) (*types.Artifact, error) {
	deleteFilter := types.ExtractDeleteFilter(opts...)
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Where("a.artifact_image_id = ?", imageID).
		Where("a.artifact_version = ?", version)

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		q = q.Where("a.artifact_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		q = q.Where("a.artifact_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filtering
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapArtifactDB(ctx, dst)
}

func (a ArtifactDao) GetByRegistryImageAndVersion(
	ctx context.Context, registryID int64, image string, version string,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ? AND i.image_name = ? AND a.artifact_version = ?", registryID, image, version).
		Where("a.artifact_deleted_at IS NULL")

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapArtifactDB(ctx, dst)
}

// GetByRegistryImageVersionAndArtifactType gets artifact by registry, image, version and type.
func (a ArtifactDao) GetByRegistryImageVersionAndArtifactType(
	ctx context.Context, registryID int64, image string, version string, artifactType string,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ?", registryID).
		Where("i.image_name = ?", image).
		Where("i.image_type = ?", artifactType).
		Where("a.artifact_version = ?", version).
		Where("a.artifact_deleted_at IS NULL")

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapArtifactDB(ctx, dst)
}

// GetByRegistryIDAndImage gets artifacts by registry and image with soft delete filtering.
func (a ArtifactDao) GetByRegistryIDAndImage(
	ctx context.Context, registryID int64, image string,
) (
	*[]types.Artifact,
	error,
) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ? AND i.image_name = ? AND i.image_type IS NULL", registryID, image).
		Where("a.artifact_deleted_at IS NULL")

	q = q.OrderBy("a.artifact_created_at DESC")

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
		art, err := a.mapArtifactDB(ctx, &d)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to map artifact")
		}
		artifacts[i] = *art
	}

	return &artifacts, nil
}

func (a ArtifactDao) GetLatestByImageID(
	ctx context.Context, imageID int64,
) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Where("a.artifact_image_id = ?", imageID).
		Where("a.artifact_deleted_at IS NULL")

	q = q.OrderBy("a.artifact_updated_at DESC").Limit(1)

	query, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, query, args...); err != nil {
		// If no artifacts found for this image, return nil instead of error
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil
		}
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get latest artifact")
	}

	var metadata json.RawMessage
	if dst.Metadata != nil {
		metadata = *dst.Metadata
	}

	var deletedAt *time.Time
	if dst.DeletedAt != nil {
		t := time.UnixMilli(*dst.DeletedAt)
		deletedAt = &t
	}

	return &types.Artifact{
		ID:        dst.ID,
		UUID:      dst.UUID,
		Version:   dst.Version,
		ImageID:   dst.ImageID,
		Metadata:  metadata,
		CreatedAt: time.UnixMilli(dst.CreatedAt),
		UpdatedAt: time.UnixMilli(dst.UpdatedAt),
		CreatedBy: dst.CreatedBy,
		UpdatedBy: dst.UpdatedBy,
		DeletedAt: deletedAt,
	}, nil
}

// checkSoftDeletedArtifactExists checks if a soft-deleted artifact exists with the same image_id and version.
func (a ArtifactDao) checkSoftDeletedArtifactExists(ctx context.Context, imageID int64, version string) error {
	checkQuery := databaseg.Builder.Select("artifact_id").
		From("artifacts").
		Where("artifact_image_id = ? AND artifact_version = ? AND artifact_deleted_at IS NOT NULL",
			imageID, version)

	checkSQL, checkArgs, err := checkQuery.ToSql()
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to build check query")
	}

	db := dbtx.GetAccessor(ctx, a.db)
	var existingID int64
	err = db.QueryRowContext(ctx, checkSQL, checkArgs...).Scan(&existingID)
	if err == nil {
		// Soft-deleted artifact exists with same identifier
		return errors.New("artifact with same version already exists but is soft-deleted")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to check for soft-deleted artifact")
	}
	return nil
}

func (a ArtifactDao) CreateOrUpdate(ctx context.Context, artifact *types.Artifact) (int64, error) {
	if commons.IsEmpty(artifact.Version) {
		return 0, errors.New("version is empty")
	}

	// Check if a soft-deleted artifact exists with the same image_id and version
	if err := a.checkSoftDeletedArtifactExists(ctx, artifact.ImageID, artifact.Version); err != nil {
		return 0, err
	}

	// Proceed with INSERT ... ON CONFLICT (only matches non-deleted records)
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
		From("artifacts a").
		Where("a.artifact_deleted_at IS NULL")

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

func (a ArtifactDao) mapArtifactDB(_ context.Context, dst *artifactDB) (*types.Artifact, error) {
	var metadata json.RawMessage
	if dst.Metadata != nil {
		metadata = *dst.Metadata
	}

	var deletedAt *time.Time
	if dst.DeletedAt != nil {
		t := time.UnixMilli(*dst.DeletedAt)
		deletedAt = &t
	}

	return &types.Artifact{
		ID:        dst.ID,
		UUID:      dst.UUID,
		Version:   dst.Version,
		ImageID:   dst.ImageID,
		Metadata:  metadata,
		CreatedAt: time.UnixMilli(dst.CreatedAt),
		UpdatedAt: time.UnixMilli(dst.UpdatedAt),
		CreatedBy: dst.CreatedBy,
		UpdatedBy: dst.UpdatedBy,
		DeletedAt: deletedAt,
	}, nil
}

func (a ArtifactDao) SearchLatestByName(
	ctx context.Context, regID int64, name string, limit int, offset int,
) (*[]types.Artifact, error) {
	subQuery := `
	SELECT artifact_image_id, MAX(artifact_created_at) AS max_created_at
	FROM artifacts
	WHERE artifact_deleted_at IS NULL
	GROUP BY artifact_image_id`

	q := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Join(fmt.Sprintf(`(%s) latest
	ON a.artifact_image_id = latest.artifact_image_id
	AND a.artifact_created_at = latest.max_created_at
`, subQuery)).
		Where("i.image_name LIKE ? AND i.image_registry_id = ?", "%"+name+"%", regID).
		Where("a.artifact_deleted_at IS NULL")

	q = q.Limit(util.SafeIntToUInt64(limit)).
		Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build SQL for latest artifact metadata with pagination")
	}
	db := dbtx.GetAccessor(ctx, a.db)

	var artifactList []artifactDB
	if err := db.SelectContext(ctx, &artifactList, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
	}

	artifacts := make([]types.Artifact, len(artifactList))
	for i := range artifactList {
		art, err := a.mapArtifactDB(ctx, &artifactList[i])
		if err != nil {
			return nil, errors.Wrap(err, "Failed to map artifact")
		}
		artifacts[i] = *art
	}

	return &artifacts, nil
}

func (a ArtifactDao) CountLatestByName(
	ctx context.Context, regID int64, name string,
) (int64, error) {
	// Count distinct images that have artifacts matching the name pattern
	q := databaseg.Builder.
		Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_name LIKE ? AND i.image_registry_id = ?", "%"+name+"%", regID).
		Where("a.artifact_deleted_at IS NULL")

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
	ctx context.Context, regID int64, name string,
	limit int, offset int,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`i.image_name as name,
        a.artifact_id as artifact_id, a.artifact_version as version, a.artifact_metadata as metadata,
        a.artifact_deleted_at as artifact_deleted_at`,
	).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Where("i.image_registry_id = ?", regID)
	if name != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(name))
	}
	q = q.Where("a.artifact_deleted_at IS NULL")

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
	q = q.Where("a.artifact_deleted_at IS NULL")

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
	ctx context.Context, parentID int64,
	registryIDs *[]string, sortByField string,
	sortByOrder string, limit int, offset int, search string,
	latestVersion bool, packageTypes []string,
) (*[]types.ArtifactMetadata, error) {
	softDeleteClause := " AND a.artifact_deleted_at IS NULL"

	// Build download count subquery - per artifact, with soft delete filter
	downloadCountSubquery := fmt.Sprintf(`( SELECT a.artifact_id, COUNT(d.download_stat_id) as download_count 
		FROM artifacts a 
		JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id 
		JOIN images i ON i.image_id = a.artifact_image_id 
		JOIN registries r ON r.registry_id = i.image_registry_id 
		WHERE r.registry_parent_id = ? %s 
		GROUP BY a.artifact_id ) as t2`, softDeleteClause)

	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, 
		i.image_name as name, 
		r.registry_package_type as package_type, 
		a.artifact_version as version, 
		a.artifact_updated_at as modified_at, 
		i.image_labels as labels, 
		a.artifact_metadata as metadata,
		COALESCE(t2.download_count,0) as download_count,
		a.artifact_deleted_at as artifact_deleted_at`,
	).
		From("artifacts a").
		Join("images i ON a.artifact_image_id = i.image_id").
		Join("registries r ON r.registry_id = i.image_registry_id").
		Where("r.registry_parent_id = ?", parentID).
		LeftJoin(downloadCountSubquery+` ON a.artifact_id = t2.artifact_id`, parentID)

	if latestVersion {
		rankSubquery := fmt.Sprintf(`(SELECT t.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY t.artifact_image_id
			ORDER BY t.artifact_updated_at DESC) AS rank FROM artifacts t 
			JOIN images i ON t.artifact_image_id = i.image_id
			JOIN registries r ON i.image_registry_id = r.registry_id
			WHERE r.registry_parent_id = ? %s) AS a1`, softDeleteClause)
		q = q.Join(rankSubquery+` ON a.artifact_id = a1.id`, parentID).Where("a1.rank = 1")
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

	q = q.Where("a.artifact_deleted_at IS NULL")

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
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
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
		whereClause := " AND t.artifact_deleted_at IS NULL"
		baseSubquery := `(SELECT t.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY t.artifact_image_id
			ORDER BY t.artifact_updated_at DESC) AS rank FROM artifacts t 
			JOIN images i ON t.artifact_image_id = i.image_id
			JOIN registries r ON i.image_registry_id = r.registry_id
			WHERE r.registry_parent_id = ?`
		rowNumSubquery := baseSubquery + whereClause + `) AS a1`
		q = q.Join(rowNumSubquery+` ON a.artifact_id = a1.id`, parentID).Where("a1.rank = 1")
	}

	if len(*registryIDs) > 0 {
		q = q.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if search != "" {
		q = q.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}

	if len(packageTypes) > 0 {
		q = q.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	q = q.Where("a.artifact_deleted_at IS NULL")

	sql, args, err := q.ToSql()
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

func (a ArtifactDao) GetArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey string, sortByField string, sortByOrder string,
	limit int, offset int, search string, labels []string,
	artifactType *artifact.ArtifactType,
) (*[]types.ArtifactMetadata, error) {
	softDeleteClause := " AND a.artifact_deleted_at IS NULL"

	// Build rank subquery with soft delete filter
	rankSubquery := fmt.Sprintf(`(SELECT a.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY a.artifact_image_id 
		ORDER BY a.artifact_updated_at DESC) AS rank FROM artifacts a 
		JOIN images i ON i.image_id = a.artifact_image_id  
		JOIN registries r ON i.image_registry_id = r.registry_id  
		WHERE r.registry_parent_id = ? AND r.registry_name = ? %s) AS a1`, softDeleteClause)

	// Build download count subquery - per artifact, with soft delete filter
	downloadCountSubquery := fmt.Sprintf(`( SELECT a.artifact_id, COUNT(d.download_stat_id) as download_count 
		FROM artifacts a 
		JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id 
		JOIN images i ON i.image_id = a.artifact_image_id 
		JOIN registries r ON r.registry_id = i.image_registry_id 
		WHERE r.registry_parent_id = ? AND r.registry_name = ? %s 
		GROUP BY a.artifact_id ) as t2`, softDeleteClause)

	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, i.image_name as name, i.image_uuid as uuid,
		r.registry_uuid as registry_uuid,
		r.registry_package_type as package_type, a.artifact_version as latest_version, 
		a.artifact_updated_at as modified_at, i.image_labels as labels, i.image_type as artifact_type,
		COALESCE(t2.download_count, 0) as download_count`,
	).
		From("artifacts a").
		Join(rankSubquery+` ON a.artifact_id = a1.id`, parentID, repoKey).
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		LeftJoin(downloadCountSubquery+` ON a.artifact_id = t2.artifact_id`, parentID, repoKey).
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

	q = q.Where("a.artifact_deleted_at IS NULL")

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

func (a ArtifactDao) CountArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey, search string, labels []string,
	artifactType *artifact.ArtifactType,
) (int64, error) {
	q := databaseg.Builder.Select("COUNT(*)").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ?", parentID, repoKey).
		Where("a.artifact_deleted_at IS NULL")

	if artifactType != nil && *artifactType != "" {
		q = q.Where("i.image_type = ?", *artifactType)
	}

	if search != "" {
		q = q.Where("i.image_name LIKE ?", "%"+search+"%")
	}

	if len(labels) > 0 {
		sort.Strings(labels)
		labelsVal := util.GetEmptySQLString(util.ArrToString(labels))
		labelsVal.String = labelSeparatorStart + labelsVal.String + labelSeparatorEnd
		q = q.Where("'^_' || i.image_labels || '^_' LIKE ?", labelsVal)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)
	var count int64
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&count); err != nil {
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
	db := dbtx.GetAccessor(ctx, a.db)

	// Step 1: Find the latest artifact ID
	latestArtifactQuery := databaseg.Builder.Select("a.artifact_id").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ? AND i.image_name = ?", parentID, repoKey, imageName).
		Where("a.artifact_deleted_at IS NULL")

	latestArtifactQuery = latestArtifactQuery.OrderBy("a.artifact_updated_at DESC").Limit(1)

	sql1, args1, err := latestArtifactQuery.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert latest artifact query to sql")
	}

	var latestArtifactID int64
	if err = db.GetContext(ctx, &latestArtifactID, sql1, args1...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No artifact exists - return nil.
			return nil, nil //nolint:nilnil
		}
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get latest artifact ID")
	}

	// Step 2: Fetch full metadata with download count for this specific artifact
	metadataQuery := databaseg.Builder.Select(
		`r.registry_name AS repo_name,
         r.registry_package_type AS package_type,
         i.image_name AS name,
         a.artifact_version AS latest_version,
         a.artifact_created_at AS created_at,
         a.artifact_updated_at AS modified_at,
         i.image_labels AS labels,
         (SELECT COUNT(*) FROM download_stats WHERE download_stat_artifact_id = a.artifact_id) AS download_count`,
	).
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("a.artifact_id = ?", latestArtifactID)

	sql2, args2, err := metadataQuery.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert metadata query to sql")
	}

	finalQuery := util.FormatQuery(sql2, args2)
	log.Ctx(ctx).Debug().Str("sql", finalQuery).Msg("Executing GetLatestArtifactMetadata query")

	dst := new(artifactMetadataDB)
	if err = db.GetContext(ctx, dst, sql2, args2...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact metadata")
	}

	return a.mapToArtifactMetadata(dst)
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

	q = q.Where("a.artifact_deleted_at IS NULL")

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
		).
		Where("a.artifact_deleted_at IS NULL")

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
	artifactType *artifact.ArtifactType, opts ...types.QueryOption,
) (*types.ArtifactMetadata, error) {
	deleteFilter := types.ExtractDeleteFilter(opts...)
	q := databaseg.Builder.Select(
		"r.registry_package_type as package_type, a.artifact_version as name,"+
			"a.artifact_updated_at as modified_at, "+
			"COALESCE(COUNT(dc.download_stat_id), 0) as download_count, "+
			"a.artifact_deleted_at").
		From("artifacts a").
		Join("images i ON i.image_id = a.artifact_image_id").
		Join("registries r ON i.image_registry_id = registry_id").
		LeftJoin("download_stats dc ON dc.download_stat_artifact_id = a.artifact_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ?"+
				" AND i.image_name = ? AND a.artifact_version = ?", id, identifier, image, version,
		).
		GroupBy("r.registry_package_type, a.artifact_version, a.artifact_updated_at, " +
			"a.artifact_deleted_at")

	if artifactType != nil && *artifactType != "" {
		q = q.Where("i.image_type = ?", *artifactType)
	}

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		q = q.Where("a.artifact_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		q = q.Where("a.artifact_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filtering
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := &artifactMetadataDB{}
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
			WHERE r.registry_id = ? 
			  AND t.artifact_deleted_at IS NULL) AS a1 
			ON a.artifact_id = a1.id`, registryID,
		).
		Where("a.artifact_id > ? AND r.registry_id = ?", artifactID, registryID).
		Where("a1.rank = 1").
		Where("a.artifact_deleted_at IS NULL").
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
		Where("a.artifact_deleted_at IS NULL").
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
		Where("a.artifact_deleted_at IS NULL").
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
	var deletedAt *time.Time
	if dst.ArtifactDeletedAt != nil {
		t := time.UnixMilli(*dst.ArtifactDeletedAt)
		deletedAt = &t
	}

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
		DeletedAt:        deletedAt,
		RegistryType:     dst.RegistryType,
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

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
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	gitness_store "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type registryDao struct {
	db *sqlx.DB

	//FIXME: Arvind: Move this to controller layer later
	mtRepository store.MediaTypesRepository
}

func NewRegistryDao(db *sqlx.DB, mtRepository store.MediaTypesRepository) store.RegistryRepository {
	return &registryDao{
		db: db,

		//FIXME: Arvind: Move this to controller layer later
		mtRepository: mtRepository,
	}
}

// registryDB holds the record of a registry in DB.
type registryDB struct {
	ID              int64                 `db:"registry_id"`
	Name            string                `db:"registry_name"`
	ParentID        int64                 `db:"registry_parent_id"`
	RootParentID    int64                 `db:"registry_root_parent_id"`
	Description     sql.NullString        `db:"registry_description"`
	Type            artifact.RegistryType `db:"registry_type"`
	PackageType     artifact.PackageType  `db:"registry_package_type"`
	UpstreamProxies sql.NullString        `db:"registry_upstream_proxies"`
	AllowedPattern  sql.NullString        `db:"registry_allowed_pattern"`
	BlockedPattern  sql.NullString        `db:"registry_blocked_pattern"`
	Labels          sql.NullString        `db:"registry_labels"`
	CreatedAt       int64                 `db:"registry_created_at"`
	UpdatedAt       int64                 `db:"registry_updated_at"`
	CreatedBy       int64                 `db:"registry_created_by"`
	UpdatedBy       int64                 `db:"registry_updated_by"`
}

func (r registryDao) Get(ctx context.Context, id int64) (*types.Registry, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_id = ?", id)

	db := dbtx.GetAccessor(ctx, r.db)

	dst := new(registryDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return r.mapToRegistry(ctx, dst)
}

func (r registryDao) GetByParentIDAndName(
	ctx context.Context, parentID int64,
	name string,
) (*types.Registry, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_parent_id = ? AND registry_name = ?", parentID, name)

	db := dbtx.GetAccessor(ctx, r.db)

	dst := new(registryDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return r.mapToRegistry(ctx, dst)
}

func (r registryDao) GetByRootParentIDAndName(
	ctx context.Context, parentID int64,
	name string,
) (*types.Registry, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_root_parent_id = ? AND registry_name = ?", parentID, name)

	db := dbtx.GetAccessor(ctx, r.db)

	dst := new(registryDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return r.mapToRegistry(ctx, dst)
}

func (r registryDao) FetchUpstreamProxyKeys(
	ctx context.Context,
	ids []int64,
) (repokeys []string, err error) {
	dst := make([]string, 0)
	if commons.IsEmpty(ids) {
		return dst, nil
	}

	stmt := databaseg.Builder.
		Select("registry_name").
		From("registries").
		Where(sq.Eq{"registry_id": ids}).
		Where("registry_type = ?", artifact.RegistryTypeUPSTREAM)

	db := dbtx.GetAccessor(ctx, r.db)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}
	return dst, nil
}

func (r registryDao) GetByIDIn(ctx context.Context, parentID int64, ids []int64) (*[]types.Registry, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_parent_id = ?", parentID).
		Where(sq.Eq{"registry_id": ids})

	db := dbtx.GetAccessor(ctx, r.db)

	dst := []*registryDB{}
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	return r.mapToRegistries(ctx, dst)
}

type RegistryMetadataDB struct {
	RegIdentifier string                `db:"reg_identifier"`
	Description   sql.NullString        `db:"description"`
	PackageType   artifact.PackageType  `db:"package_type"`
	Type          artifact.RegistryType `db:"type"`
	LastModified  int64                 `db:"last_modified"`
	URL           sql.NullString        `db:"url"`
	ArtifactCount int64                 `db:"artifact_count"`
	DownloadCount int64                 `db:"download_count"`
	Size          int64                 `db:"size"`
	Labels        sql.NullString        `db:"registry_labels"`
}

func (r registryDao) GetAll(
	ctx context.Context,
	parentID int64,
	packageTypes []string,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
	repoType string,
) (repos *[]store.RegistryMetadata, err error) {
	q := databaseg.Builder.Select(
		"r.registry_name as reg_identifier,"+
			" r.registry_description as description , "+
			"r.registry_package_type as package_type, r.registry_type as type, r.registry_updated_at as last_modified,"+
			" u.upstream_proxy_config_url as url, COALESCE(t2.artifact_count,0) as artifact_count, "+
			"COALESCE(t3.size,0) as size , r.registry_labels, COALESCE(t4.download_count,0) as download_count ",
	).
		From("registries r").
		LeftJoin("upstream_proxy_configs u ON r.registry_id = u.upstream_proxy_config_registry_id").
		LeftJoin(
			"(SELECT r.registry_id, count(a.image_id) as artifact_count FROM"+
				" registries r LEFT JOIN images a ON r.registry_id = a.image_registry_id"+
				" WHERE r.registry_parent_id = ? AND a.image_enabled = true GROUP BY r.registry_id ) as t2"+
				" ON r.registry_id = t2.registry_id ", parentID,
		).
		LeftJoin(
			"(SELECT r.registry_id , COALESCE(sum(b.blob_size),0) as size FROM "+
				"registries r LEFT JOIN registry_blobs rb ON r.registry_id = rblob_registry_id "+
				"LEFT JOIN blobs b ON rblob_blob_id = b.blob_id WHERE r.registry_parent_id = ? "+
				"GROUP BY r.registry_id) as t3 ON r.registry_id = t3.registry_id ", parentID,
		).
		LeftJoin(
			"(SELECT i.image_registry_id, COUNT(d.download_stat_id) as download_count "+
				"FROM artifacts a "+
				" JOIN images i on i.image_id = a.artifact_image_id"+
				" JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id"+
				" WHERE i.image_enabled = true GROUP BY i.image_registry_id ) as t4 "+
				" ON r.registry_id = t4.image_registry_id").
		Where("r.registry_parent_id = ?", parentID)

	if search != "" {
		q = q.Where("r.registry_name LIKE ?", "%"+search+"%")
	}

	if len(packageTypes) > 0 {
		q = q.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}
	if repoType != "" {
		q = q.Where("r.registry_type = ?", repoType)
	}

	if sortByField == "artifact_count" || sortByField == "size" || sortByField == "download_count" {
		q = q.OrderBy(sortByField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset))
	} else {
		q = q.OrderBy("r.registry_" + sortByField + " " + sortByOrder).
			Limit(uint64(limit)).Offset(uint64(offset))
	}
	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := []*RegistryMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return r.mapToRegistryMetadataList(ctx, dst)
}

func (r registryDao) CountAll(
	ctx context.Context, parentID int64,
	packageTypes []string, search string, repoType string,
) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("registries").
		Where("registry_parent_id = ?", parentID)

	if !commons.IsEmpty(search) {
		stmt = stmt.Where("registry_name LIKE ?", "%"+search+"%")
	}

	if len(packageTypes) > 0 {
		stmt = stmt.Where(sq.Eq{"registry_package_type": packageTypes})
	}

	if repoType != "" {
		stmt = stmt.Where("registry_type = ?", repoType)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (r registryDao) Create(ctx context.Context, registry *types.Registry) (id int64, err error) {
	const sqlQuery = `
		INSERT INTO registries ( 
			 registry_name
			,registry_root_parent_id
			,registry_parent_id
			,registry_description
			,registry_type
			,registry_package_type
			,registry_upstream_proxies
			,registry_allowed_pattern
			,registry_blocked_pattern
			,registry_created_at
			,registry_updated_at
			,registry_created_by
			,registry_updated_by
			,registry_labels
		) VALUES (
			:registry_name
			,:registry_root_parent_id
			,:registry_parent_id
			,:registry_description
			,:registry_type
			,:registry_package_type
			,:registry_upstream_proxies
			,:registry_allowed_pattern
			,:registry_blocked_pattern
			,:registry_created_at
			,:registry_updated_at
			,:registry_created_by
			,:registry_updated_by
			,:registry_labels
		) RETURNING registry_id`

	db := dbtx.GetAccessor(ctx, r.db)
	query, arg, err := db.BindNamed(sqlQuery, mapToInternalRegistry(ctx, registry))
	if err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&registry.ID); err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return registry.ID, nil
}

func mapToInternalRegistry(ctx context.Context, in *types.Registry) *registryDB {
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	return &registryDB{
		ID:              in.ID,
		Name:            in.Name,
		ParentID:        in.ParentID,
		RootParentID:    in.RootParentID,
		Description:     util.GetEmptySQLString(in.Description),
		Type:            in.Type,
		PackageType:     in.PackageType,
		UpstreamProxies: util.GetEmptySQLString(util.Int64ArrToString(in.UpstreamProxies)),
		AllowedPattern:  util.GetEmptySQLString(util.ArrToString(in.AllowedPattern)),
		BlockedPattern:  util.GetEmptySQLString(util.ArrToString(in.BlockedPattern)),
		Labels:          util.GetEmptySQLString(util.ArrToString(in.Labels)),
		CreatedAt:       in.CreatedAt.UnixMilli(),
		UpdatedAt:       in.UpdatedAt.UnixMilli(),
		CreatedBy:       in.CreatedBy,
		UpdatedBy:       in.UpdatedBy,
	}
}

func (r registryDao) Delete(ctx context.Context, parentID int64, name string) (err error) {
	stmt := databaseg.Builder.Delete("registries").
		Where("registry_parent_id = ? AND registry_name = ?", parentID, name)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge registry query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (r registryDao) Update(ctx context.Context, registry *types.Registry) (err error) {
	var sqlQuery = " UPDATE registries SET " + util.GetSetDBKeys(registryDB{}, "registry_id") +
		" WHERE registry_id = :registry_id "

	dbRepo := mapToInternalRegistry(ctx, registry)

	// update Version (used for optimistic locking) and Updated time
	dbRepo.UpdatedAt = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, r.db)

	query, arg, err := db.BindNamed(sqlQuery, dbRepo)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to update repository")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	return nil
}

func (r registryDao) FetchUpstreamProxyIDs(
	ctx context.Context,
	repokeys []string,
	parentID int64,
) (ids []int64, err error) {
	var repoIDs []int64
	stmt := databaseg.Builder.Select("registry_id").
		From("registries").
		Where("registry_parent_id = ?", parentID).
		Where(sq.Eq{"registry_name": repokeys}).
		Where("registry_type = ?", artifact.RegistryTypeUPSTREAM)

	db := dbtx.GetAccessor(ctx, r.db)

	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &repoIDs, query, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find registries")
	}

	return repoIDs, nil
}

func (r registryDao) mapToRegistries(ctx context.Context, dst []*registryDB) (*[]types.Registry, error) {
	registries := make([]types.Registry, 0, len(dst))
	for _, d := range dst {
		repo, err := r.mapToRegistry(ctx, d)
		if err != nil {
			return nil, err
		}
		registries = append(registries, *repo)
	}
	return &registries, nil
}

func (r registryDao) mapToRegistry(_ context.Context, dst *registryDB) (*types.Registry, error) {
	return &types.Registry{
		ID:              dst.ID,
		Name:            dst.Name,
		ParentID:        dst.ParentID,
		RootParentID:    dst.RootParentID,
		Description:     dst.Description.String,
		Type:            dst.Type,
		PackageType:     dst.PackageType,
		UpstreamProxies: util.StringToInt64Arr(dst.UpstreamProxies.String),
		AllowedPattern:  util.StringToArr(dst.AllowedPattern.String),
		BlockedPattern:  util.StringToArr(dst.BlockedPattern.String),
		Labels:          util.StringToArr(dst.Labels.String),
		CreatedAt:       time.UnixMilli(dst.CreatedAt),
		UpdatedAt:       time.UnixMilli(dst.UpdatedAt),
		CreatedBy:       dst.CreatedBy,
		UpdatedBy:       dst.UpdatedBy,
	}, nil
}

func (r registryDao) mapToRegistryMetadataList(
	ctx context.Context,
	dst []*RegistryMetadataDB,
) (*[]store.RegistryMetadata, error) {
	repos := make([]store.RegistryMetadata, 0, len(dst))
	for _, d := range dst {
		repo := r.mapToRegistryMetadata(ctx, d)
		repos = append(repos, *repo)
	}
	return &repos, nil
}

func (r registryDao) mapToRegistryMetadata(_ context.Context, dst *RegistryMetadataDB) *store.RegistryMetadata {
	return &store.RegistryMetadata{
		RegIdentifier: dst.RegIdentifier,
		Description:   dst.Description.String,
		PackageType:   dst.PackageType,
		Type:          dst.Type,
		LastModified:  time.UnixMilli(dst.LastModified),
		URL:           dst.URL.String,
		ArtifactCount: dst.ArtifactCount,
		DownloadCount: dst.DownloadCount,
		Size:          dst.Size,
		Labels:        util.StringToArr(dst.Labels.String),
	}
}

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
	"strconv"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/registry/utils"
	gitnessstore "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const SQLITE3 = "sqlite3"

type registryDao struct {
	db *sqlx.DB

	//FIXME: Arvind: Move this to controller layer later
	mtRepository        store.MediaTypesRepository
	downloadCountFinder store.DownloadCountFinder
}

func NewRegistryDao(
	db *sqlx.DB, mtRepository store.MediaTypesRepository,
	downloadCountFinder store.DownloadCountFinder,
) store.RegistryRepository {
	return &registryDao{
		db: db,
		//FIXME: Arvind: Move this to controller layer later
		mtRepository:        mtRepository,
		downloadCountFinder: downloadCountFinder,
	}
}

// registryDB holds the record of a registry in DB.
type registryDB struct {
	ID              int64                 `db:"registry_id"`
	UUID            string                `db:"registry_uuid"`
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
	Config          sql.NullString        `db:"registry_config"`
	CreatedAt       int64                 `db:"registry_created_at"`
	UpdatedAt       int64                 `db:"registry_updated_at"`
	CreatedBy       int64                 `db:"registry_created_by"`
	UpdatedBy       int64                 `db:"registry_updated_by"`
	DeletedAt       *int64                `db:"registry_deleted_at"`
	DeletedBy       *int64                `db:"registry_deleted_by"`
}

type registryNameID struct {
	ID   int64  `db:"registry_id"`
	Name string `db:"registry_name"`
}

// GetByUUID gets a registry by its UUID (includes soft-deleted registries).
func (r registryDao) GetByUUID(ctx context.Context, uuid string) (*types.Registry, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_uuid = ?", uuid)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := new(registryDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom find query")
	}

	return r.mapToRegistry(ctx, dst)
}

func (r registryDao) Get(
	ctx context.Context, id int64, opts ...types.QueryOption,
) (*types.Registry, error) {
	deleteFilter := types.ExtractDeleteFilter(opts...)
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_id = ?", id)

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filter
	}

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
	name string, opts ...types.QueryOption,
) (*types.Registry, error) {
	deleteFilter := types.ExtractDeleteFilter(opts...)
	log.Ctx(ctx).Info().Msgf("GetByParentIDAndName: parentID: %d, name: %s", parentID, name)
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_parent_id = ? AND registry_name = ?", parentID, name)

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filtering
	}

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
	name string, opts ...types.QueryOption,
) (*types.Registry, error) {
	deleteFilter := types.ExtractDeleteFilter(opts...)
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where("registry_root_parent_id = ? AND registry_name = ?", parentID, name)

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filtering
	}

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

func (r registryDao) Count(ctx context.Context) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("registries").
		Where("registry_deleted_at IS NULL")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (r registryDao) FetchUpstreamProxyKeys(
	ctx context.Context,
	ids []int64,
	opts ...types.QueryOption,
) (repokeys []string, err error) {
	orderedRepoKeys := make([]string, 0)

	if commons.IsEmpty(ids) {
		return orderedRepoKeys, nil
	}

	deleteFilter := types.ExtractDeleteFilter(opts...)
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryNameID{}), ",")).
		From("registries").
		Where(sq.Eq{"registry_id": ids})

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filtering
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := []registryNameID{}
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find repo")
	}

	// Create a map
	recordMap := make(map[int64]registryNameID)
	for _, record := range dst {
		recordMap[record.ID] = record
	}

	// Reorder the fetched records based on the ID list
	for _, id := range ids {
		if record, exists := recordMap[id]; exists {
			orderedRepoKeys = append(orderedRepoKeys, record.Name)
		} else {
			log.Ctx(ctx).Error().Msgf("failed to map upstream registry: %d", id)
			orderedRepoKeys = append(orderedRepoKeys, "")
		}
	}

	return orderedRepoKeys, nil
}

func (r registryDao) GetByIDIn(
	ctx context.Context, ids []int64, opts ...types.QueryOption,
) (*[]types.Registry, error) {
	deleteFilter := types.ExtractDeleteFilter(opts...)
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(registryDB{}), ",")).
		From("registries").
		Where(sq.Eq{"registry_id": ids})

	switch deleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filtering
	}

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
	RegUUID       string                `db:"registry_uuid"`
	RegID         string                `db:"registry_id"`
	ParentID      int64                 `db:"parent_id"`
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
	Config        sql.NullString        `db:"registry_config"`
	DeletedAt     *int64                `db:"registry_deleted_at"`
}

func (r registryDao) GetAll(
	ctx context.Context,
	parentIDs []int64,
	packageTypes []string,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
	repoType string,
) (repos *[]store.RegistryMetadata, err error) {
	if limit < 0 || offset < 0 {
		return nil, fmt.Errorf("limit and offset must be non-negative")
	}

	// Step 1: Fetch base registry data.
	selectFields := `
		r.registry_uuid AS registry_uuid,
		r.registry_id AS registry_id,
		r.registry_parent_id AS parent_id,
		r.registry_name AS reg_identifier,
		COALESCE(r.registry_description, '') AS description, 
		r.registry_package_type AS package_type,
		r.registry_type AS type,
		r.registry_updated_at AS last_modified, 
		COALESCE(u.upstream_proxy_config_url, '') AS url,
		r.registry_config,
		r.registry_labels,
		r.registry_deleted_at
	`

	var query sq.SelectBuilder
	query = databaseg.Builder.
		Select(selectFields).
		From("registries r").
		LeftJoin("upstream_proxy_configs u ON r.registry_id = u.upstream_proxy_config_registry_id").
		Where(sq.Eq{"r.registry_parent_id": parentIDs}).
		Where("r.registry_deleted_at IS NULL")

	// Apply search filter
	if search != "" {
		query = query.Where("r.registry_name LIKE ?", "%"+search+"%")
	}

	// Apply package types filter
	if len(packageTypes) > 0 {
		query = query.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	// Apply repository type filter
	if repoType != "" {
		query = query.Where("r.registry_type = ?", repoType)
	}

	// Sorting and pagination
	validSortFields := map[string]bool{
		"artifact_count": true,
		"size":           true,
		"download_count": true,
	}
	if validSortFields[sortByField] {
		query = query.OrderBy(fmt.Sprintf("%s %s", sortByField, sortByOrder))
	} else {
		query = query.OrderBy(fmt.Sprintf("r.registry_%s %s", sortByField, sortByOrder))
	}
	query = query.Limit(utils.SafeUint64(limit)).Offset(utils.SafeUint64(offset))

	// Convert query to SQL
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to SQL")
	}

	// Execute main query
	db := dbtx.GetAccessor(ctx, r.db)
	dst := []*RegistryMetadataDB{}
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing main registry query")
	}

	// If no registries found, return empty list
	if len(dst) == 0 {
		return r.mapToRegistryMetadataList(ctx, dst)
	}

	// Extract registry IDs for subsequent queries
	registryIDs := make([]int64, len(dst))
	registryIDStrings := make([]string, len(dst))
	for i, reg := range dst {
		regID, err := strconv.ParseInt(reg.RegID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse registry ID %s: %w", reg.RegID, err)
		}
		registryIDs[i] = regID
		registryIDStrings[i] = reg.RegID
	}

	// Fetch aggregate data sequentially with soft delete filtering
	artifactCounts, err := r.fetchArtifactCounts(ctx, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artifact counts: %w", err)
	}

	ociSizes, err := r.fetchOCIBlobSizes(ctx, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OCI blob sizes: %w", err)
	}

	genericSizes, err := r.fetchGenericBlobSizes(ctx, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch generic blob sizes: %w", err)
	}

	downloadCounts, err := r.fetchDownloadCounts(ctx, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch download counts: %w", err)
	}

	// Merge aggregate data into registry results
	for _, reg := range dst {
		regID, _ := strconv.ParseInt(reg.RegID, 10, 64)

		// Set artifact count
		if count, ok := artifactCounts[regID]; ok {
			reg.ArtifactCount = count
		}

		// Set size (OCI takes precedence, fallback to generic)
		if size, ok := ociSizes[regID]; ok && size > 0 {
			reg.Size = size
		} else if genericSize, ok := genericSizes[regID]; ok && genericSize > 0 {
			reg.Size = genericSize
		}

		// Set download count
		if count, ok := downloadCounts[regID]; ok {
			reg.DownloadCount = count
		}
	}

	// Map results to response type
	return r.mapToRegistryMetadataList(ctx, dst)
}

// fetchArtifactCounts fetches artifact counts for given registry IDs with soft delete filtering.
func (r registryDao) fetchArtifactCounts(
	ctx context.Context, registryIDs []int64,
) (map[int64]int64, error) {
	if len(registryIDs) == 0 {
		return make(map[int64]int64), nil
	}

	query := `
		SELECT image_registry_id, COUNT(image_id) AS count
		FROM images
		WHERE image_registry_id IN (?) AND image_enabled = TRUE AND image_deleted_at IS NULL
		GROUP BY image_registry_id
	`

	db := dbtx.GetAccessor(ctx, r.db)
	sql, args, err := sqlx.In(query, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build artifact counts query: %w", err)
	}
	sql = db.Rebind(sql)

	type result struct {
		RegistryID int64 `db:"image_registry_id"`
		Count      int64 `db:"count"`
	}
	var results []result
	if err := db.SelectContext(ctx, &results, sql, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch artifact counts: %w", err)
	}

	counts := make(map[int64]int64, len(results))
	for _, r := range results {
		counts[r.RegistryID] = r.Count
	}

	return counts, nil
}

// fetchOCIBlobSizes fetches OCI blob sizes for given registry IDs with soft delete filtering.
func (r registryDao) fetchOCIBlobSizes(
	ctx context.Context, registryIDs []int64,
) (map[int64]int64, error) {
	if len(registryIDs) == 0 {
		return make(map[int64]int64), nil
	}

	query := `
		SELECT rb.rblob_registry_id, COALESCE(SUM(blobs.blob_size), 0) AS total_size
		FROM registry_blobs rb
		LEFT JOIN blobs ON rb.rblob_blob_id = blobs.blob_id
		WHERE rb.rblob_registry_id IN (?)
		GROUP BY rb.rblob_registry_id
	`

	db := dbtx.GetAccessor(ctx, r.db)
	sql, args, err := sqlx.In(query, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build OCI blob sizes query: %w", err)
	}
	sql = db.Rebind(sql)

	type result struct {
		RegistryID int64 `db:"rblob_registry_id"`
		TotalSize  int64 `db:"total_size"`
	}
	var results []result
	if err := db.SelectContext(ctx, &results, sql, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch OCI blob sizes: %w", err)
	}

	sizes := make(map[int64]int64, len(results))
	for _, r := range results {
		sizes[r.RegistryID] = r.TotalSize
	}

	return sizes, nil
}

// fetchGenericBlobSizes fetches generic blob sizes for given registry IDs.
// NOTE: Soft delete filtering is NOT applied for Generic artifacts due to schema limitations.
// Filtering would require expensive SPLIT_PART operations on node_path for every row.
// TODO: Add node_artifact_id FK and node_deleted_at column to enable proper soft delete filtering.
func (r registryDao) fetchGenericBlobSizes(
	ctx context.Context, registryIDs []int64,
) (map[int64]int64, error) {
	if len(registryIDs) == 0 {
		return make(map[int64]int64), nil
	}

	query := `
		SELECT nodes.node_registry_id, COALESCE(SUM(generic_blobs.generic_blob_size), 0) AS total_size
		FROM nodes
		LEFT JOIN generic_blobs ON generic_blob_id = nodes.node_generic_blob_id
		WHERE node_is_file AND generic_blob_id IS NOT NULL
		  AND node_registry_id IN (?)
		GROUP BY nodes.node_registry_id
	`

	db := dbtx.GetAccessor(ctx, r.db)
	sql, args, err := sqlx.In(query, registryIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to build generic blob sizes query: %w", err)
	}
	sql = db.Rebind(sql)

	type result struct {
		RegistryID int64 `db:"node_registry_id"`
		TotalSize  int64 `db:"total_size"`
	}
	var results []result
	if err := db.SelectContext(ctx, &results, sql, args...); err != nil {
		return nil, fmt.Errorf("failed to fetch generic blob sizes: %w", err)
	}

	sizes := make(map[int64]int64, len(results))
	for _, r := range results {
		sizes[r.RegistryID] = r.TotalSize
	}

	return sizes, nil
}

// fetchDownloadCounts fetches download counts for given registry IDs using the cache.
func (r registryDao) fetchDownloadCounts(ctx context.Context, registryIDs []int64) (map[int64]int64, error) {
	if len(registryIDs) == 0 {
		return make(map[int64]int64), nil
	}
	return r.downloadCountFinder.FindByRegistryIDs(ctx, registryIDs)
}

func (r registryDao) CountAll(
	ctx context.Context,
	parentIDs []int64,
	packageTypes []string,
	search string,
	repoType string,
) (count int64, err error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("registries").
		Where(sq.Eq{"registry_parent_id": parentIDs}).
		Where("registry_deleted_at IS NULL")

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

	var countResult int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&countResult)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return countResult, nil
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
			,registry_config
			,registry_uuid
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
			,:registry_config
			,:registry_uuid
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

	if in.UUID == "" {
		in.UUID = uuid.NewString()
	}

	// Serialize config to JSON
	var configStr string
	if in.Config != nil {
		configBytes, err := json.Marshal(in.Config)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal registry config")
		} else {
			configStr = string(configBytes)
		}
	}

	return &registryDB{
		ID:              in.ID,
		UUID:            in.UUID,
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
		Config:          util.GetEmptySQLString(configStr),
		CreatedAt:       in.CreatedAt.UnixMilli(),
		UpdatedAt:       in.UpdatedAt.UnixMilli(),
		CreatedBy:       in.CreatedBy,
		UpdatedBy:       in.UpdatedBy,
	}
}

func (r registryDao) Delete(ctx context.Context, parentID int64, name string) error {
	stmt := databaseg.Builder.Delete("registries").
		Where("registry_parent_id = ? AND registry_name = ?", parentID, name)

	db := dbtx.GetAccessor(ctx, r.db)

	query, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert query to sql")
	}

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to delete registry")
	}

	return nil
}

func (r registryDao) Update(ctx context.Context, registry *types.Registry) (err error) {
	var sqlQuery = " UPDATE registries SET " + util.GetSetDBKeys(registryDB{}, "registry_id", "registry_uuid") +
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
		return gitnessstore.ErrVersionConflict
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
		Where("registry_type = ?", artifact.RegistryTypeUPSTREAM).
		Where("registry_deleted_at IS NULL")

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

func (r registryDao) FetchRegistriesIDByUpstreamProxyID(
	ctx context.Context, upstreamProxyID string,
	rootParentID int64,
	opts ...types.QueryOption,
) (ids []int64, err error) {
	var registryIDs []int64

	stmt := databaseg.Builder.Select("registry_id").
		From("registries").
		Where("registry_root_parent_id = ?", rootParentID).
		Where(
			"(registry_upstream_proxies = ? "+
				"OR registry_upstream_proxies LIKE ? "+
				"OR registry_upstream_proxies LIKE ? "+
				"OR registry_upstream_proxies LIKE ?)",
			upstreamProxyID,
			upstreamProxyID+"^_%",
			"%^_"+upstreamProxyID+"^_%",
			"%^_"+upstreamProxyID,
		).
		Where("registry_type = ?", artifact.RegistryTypeVIRTUAL)

	// Apply query options
	queryOpts := types.MakeQueryOptions(opts...)

	// Apply delete filter
	switch queryOpts.DeleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filter - include all
	}

	db := dbtx.GetAccessor(ctx, r.db)

	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &registryIDs, query, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find registries")
	}

	return registryIDs, nil
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

func (r registryDao) mapToRegistry(ctx context.Context, dst *registryDB) (*types.Registry, error) {
	// Deserialize config from JSON
	var config *types.RegistryConfig
	if dst.Config.String != "" {
		config = &types.RegistryConfig{}
		if err := json.Unmarshal([]byte(dst.Config.String), config); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal registry config")
			config = nil
		}
	}

	var deletedAt *time.Time
	if dst.DeletedAt != nil {
		t := time.UnixMilli(*dst.DeletedAt)
		deletedAt = &t
	}

	return &types.Registry{
		ID:              dst.ID,
		UUID:            dst.UUID,
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
		Config:          config,
		CreatedAt:       time.UnixMilli(dst.CreatedAt),
		UpdatedAt:       time.UnixMilli(dst.UpdatedAt),
		CreatedBy:       dst.CreatedBy,
		UpdatedBy:       dst.UpdatedBy,
		DeletedAt:       deletedAt,
		DeletedBy:       dst.DeletedBy,
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

func (r registryDao) mapToRegistryMetadata(ctx context.Context, dst *RegistryMetadataDB) *store.RegistryMetadata {
	// Deserialize config from JSON
	var config *types.RegistryConfig
	if dst.Config.String != "" {
		config = &types.RegistryConfig{}
		if err := json.Unmarshal([]byte(dst.Config.String), config); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to unmarshal registry config in metadata")
			config = nil
		}
	}

	var deletedAt *time.Time
	if dst.DeletedAt != nil {
		t := time.UnixMilli(*dst.DeletedAt)
		deletedAt = &t
	}

	return &store.RegistryMetadata{
		RegUUID:       dst.RegUUID,
		RegID:         dst.RegID,
		ParentID:      dst.ParentID,
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
		Config:        config,
		DeletedAt:     deletedAt,
	}
}

// GetIDsByParentSpace returns all registry IDs under a given parent space.
func (r registryDao) GetIDsByParentSpace(
	ctx context.Context, parentSpaceID int64, opts ...types.QueryOption,
) ([]int64, error) {
	stmt := databaseg.Builder.
		Select("registry_id").
		From("registries").
		Where("registry_parent_id = ?", parentSpaceID)

	// Apply query options
	queryOpts := types.MakeQueryOptions(opts...)

	// Apply delete filter
	switch queryOpts.DeleteFilter {
	case types.DeleteFilterExcludeDeleted:
		stmt = stmt.Where("registry_deleted_at IS NULL")
	case types.DeleteFilterOnlyDeleted:
		stmt = stmt.Where("registry_deleted_at IS NOT NULL")
	case types.DeleteFilterIncludeDeleted:
		// No filter - include all
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert select query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)
	var registryIDs []int64
	err = db.SelectContext(ctx, &registryIDs, sql, args...)
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "failed to fetch registry IDs by parent space")
	}

	return registryIDs, nil
}

// UpdateParentSpace updates the parent space ID for all registries under a given source space to target space.
// This is used during space move operations to update registry ownership.
// Returns the number of registries updated.
func (r registryDao) UpdateParentSpace(ctx context.Context, srcSpaceID int64, targetSpaceID int64) (int64, error) {
	stmt := databaseg.Builder.
		Update("registries").
		Set("registry_parent_id", targetSpaceID).
		Where("registry_parent_id = ?", srcSpaceID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert update parent space query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)
	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "failed to update registry parent space")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

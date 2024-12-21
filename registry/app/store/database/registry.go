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
	"github.com/rs/zerolog/log"
)

const SQLITE3 = "sqlite3"

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

func (r registryDao) Count(ctx context.Context) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("registries")

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
	RegID         string                `db:"registry_id"`
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
	recursive bool,
) (repos *[]store.RegistryMetadata, err error) {
	// Select only required fields
	selectFields := `
		r.registry_id AS registry_id,
		r.registry_name AS reg_identifier,
		COALESCE(r.registry_description, '') AS description, 
		r.registry_package_type AS package_type,
		r.registry_type AS type,
		r.registry_updated_at AS last_modified, 
		COALESCE(u.upstream_proxy_config_url, '') AS url, 
		COALESCE(artifact_count.count, 0) AS artifact_count,
		COALESCE(blob_sizes.total_size, 0) AS size,
		r.registry_labels,
		COALESCE(download_stats.download_count, 0) AS download_count
	`

	// Subqueries with optimizations for reduced joins and grouping
	artifactCountSubquery := `
		SELECT image_registry_id, COUNT(image_id) AS count
		FROM images
		WHERE image_enabled = TRUE
		GROUP BY 1
	`

	blobSizesSubquery := `
		SELECT rblob_registry_id AS rblob_registry_id, SUM(b.blob_size) AS total_size
		FROM registry_blobs rb
		JOIN blobs b ON rb.rblob_blob_id = b.blob_id
		GROUP BY 1
	`

	downloadStatsSubquery := `
		SELECT i.image_registry_id AS registry_id, COUNT(d.download_stat_id) AS download_count
		FROM download_stats d
		JOIN artifacts a ON d.download_stat_artifact_id = a.artifact_id
		JOIN images i ON a.artifact_image_id = i.image_id
		WHERE i.image_enabled = TRUE
		GROUP BY 1
	`

	var query sq.SelectBuilder
	if recursive {
		query = databaseg.Builder.
			Select(selectFields).
			From("registry_hierarchy_u rh").
			InnerJoin("registries r ON rh.registry_id = r.registry_id").
			LeftJoin("upstream_proxy_configs u ON r.registry_id = u.upstream_proxy_config_registry_id").
			LeftJoin(fmt.Sprintf("(%s) AS artifact_count ON r.registry_id = artifact_count.image_registry_id",
				artifactCountSubquery)).
			LeftJoin(fmt.Sprintf("(%s) AS blob_sizes ON r.registry_id = blob_sizes.rblob_registry_id", blobSizesSubquery)).
			LeftJoin(fmt.Sprintf("(%s) AS download_stats ON r.registry_id = download_stats.registry_id", downloadStatsSubquery))
	} else {
		query = databaseg.Builder.
			Select(selectFields).
			From("registries r").
			LeftJoin("upstream_proxy_configs u ON r.registry_id = u.upstream_proxy_config_registry_id").
			LeftJoin(fmt.Sprintf("(%s) AS artifact_count ON r.registry_id = artifact_count.image_registry_id",
				artifactCountSubquery)).
			LeftJoin(fmt.Sprintf("(%s) AS blob_sizes ON r.registry_id = blob_sizes.rblob_registry_id", blobSizesSubquery)).
			LeftJoin(fmt.Sprintf("(%s) AS download_stats ON r.registry_id = download_stats.registry_id", downloadStatsSubquery)).
			Where("r.registry_parent_id = ?", parentID)
	}
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
	query = query.Limit(uint64(limit)).Offset(uint64(offset))

	// Convert query to SQL
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to SQL")
	}

	if recursive {
		var cte string
		if r.db.DriverName() == SQLITE3 {
			cte = buildCTESqlite(parentID)
		} else {
			cte = buildCTE(parentID)
		}
		// Add CTE to the query
		sql = cte + sql
	}
	// Log the final query
	finalQuery := util.ConstructQuery(sql, args)
	log.Ctx(ctx).Debug().
		Str("sql", finalQuery).
		Msg("Executing query")

	// Execute query
	db := dbtx.GetAccessor(ctx, r.db)
	dst := []*RegistryMetadataDB{}
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing query")
	}

	// Map results to response type
	return r.mapToRegistryMetadataList(ctx, dst)
}

// Helper function to construct the Common Table Expression (CTE)
// This CTE, named `registry_hierarchy`, is designed to recursively traverse
// a hierarchy of registries starting from a given parent ID (`parentID`).
// The goal is to create a structure that includes all registries related
// to the specified parent registry, along with their relationships and levels in the hierarchy.
//
// Logic:
// 1. **Base Case**:
//    - The query starts by selecting registries whose `registry_parent_id` matches the given `parentID`.
//    - It initializes the recursion level (`recursion_level`) to `1` and creates a path array (`path`) to track
//    -	visited nodes.
//
// 2. **Recursive Step**:
//    - For each registry from the base case, the query looks for child registries by joining
//      the `registry_parent_parent_id` of the current node with the `registry_parent_id` of the next node.
//    - It increments the recursion level and appends the current registry ID to the path.
//    - This process repeats for up to 10 levels (`recursion_level < 10`), preventing infinite recursion.
//
// 3. **Cycle Prevention**:
//    - To prevent cycles in the hierarchy, the query ensures that a node is not revisited
//      by checking that its ID is not already in the `path` array.
//
// 4. **Final Distinct Set**:
//    - After completing the recursive traversal, the resulting CTE (`registry_hierarchy_u`)
//      selects a distinct list of all registry IDs found in the hierarchy.
//      This is used to build the final query for fetching data from related tables.
//

func buildCTE(parentID int64) string {
	cte := `
	WITH RECURSIVE registry_hierarchy AS (
		-- Base case: Start with nodes having registry_parent_id = $1
		SELECT
			r.registry_id,
			r.registry_parent_id,
			r.registry_root_parent_id,
			s.space_parent_id AS registry_parent_parent_id,
			r.registry_name,
			1::integer AS recursion_level, -- Initialize recursion level
			ARRAY[r.registry_id] AS path -- Track visited nodes
		FROM
			registries r
		LEFT JOIN
			spaces s ON r.registry_parent_id = s.space_id -- Fetch registry_parent_parent_id from spaces
		WHERE
			r.registry_parent_id = %d
		UNION 
		-- Recursive step: Traverse the hierarchy upward to the root
		SELECT 
			r.registry_id,
			r.registry_parent_id,
			r.registry_root_parent_id,
			s.space_parent_id AS registry_parent_parent_id,
			r.registry_name,
			rh.recursion_level + 1 AS recursion_level, -- Increment recursion level
			rh.path || r.registry_id -- Append current node to the path
		FROM
			registries r
		LEFT JOIN
			spaces s ON r.registry_parent_id = s.space_id -- Fetch registry_parent_parent_id
		INNER JOIN
			registry_hierarchy rh ON rh.registry_parent_parent_id = r.registry_parent_id -- Match parent to child
		WHERE
			NOT r.registry_id = ANY(rh.path) -- Avoid revisiting nodes
			AND rh.registry_parent_parent_id IS NOT NULL
			AND rh.recursion_level < 10 -- Limit recursion depth
	),
	registry_hierarchy_u AS (
		SELECT DISTINCT registry_id FROM registry_hierarchy)
`
	cte = fmt.Sprintf(cte, parentID)
	return cte
}

// buildCTESqlite is equivalent to buildCTE but for SQLite.
func buildCTESqlite(parentID int64) string {
	cte := `
	WITH RECURSIVE registry_hierarchy AS (
		-- Base case: Start with nodes having registry_parent_id = %d
		SELECT
			r.registry_id,
			r.registry_parent_id,
			r.registry_root_parent_id,
			s.space_parent_id AS registry_parent_parent_id,
			r.registry_name,
			CAST(1 AS INTEGER) AS recursion_level, -- Initialize recursion level
			json_array(r.registry_id) AS path -- Track visited nodes
		FROM
			registries r
		LEFT JOIN
			spaces s ON r.registry_parent_id = s.space_id -- Fetch registry_parent_parent_id from spaces
		WHERE
			r.registry_parent_id = %d
		UNION 
		-- Recursive step: Traverse the hierarchy upward to the root
		SELECT 
			r.registry_id,
			r.registry_parent_id,
			r.registry_root_parent_id,
			s.space_parent_id AS registry_parent_parent_id,
			r.registry_name,
			rh.recursion_level + 1 AS recursion_level, -- Increment recursion level
			json(rh.path || ',' || json(r.registry_id)) -- Append current node to the path
		FROM
			registries r
		LEFT JOIN
			spaces s ON r.registry_parent_id = s.space_id -- Fetch registry_parent_parent_id
		INNER JOIN
			registry_hierarchy rh ON rh.registry_parent_parent_id = r.registry_parent_id -- Match parent to child
		WHERE
			r.registry_id NOT IN (SELECT value FROM json_each(rh.path)) -- Avoid revisiting nodes
			AND rh.registry_parent_parent_id IS NOT NULL
			AND rh.recursion_level < 10 -- Limit recursion depth
	),
	registry_hierarchy_u AS (
		SELECT DISTINCT registry_id FROM registry_hierarchy)
	`
	// Correctly format the string using parentID
	return fmt.Sprintf(cte, parentID, parentID)
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

func (r registryDao) FetchRegistriesIDByUpstreamProxyID(
	ctx context.Context, upstreamProxyID string,
	parentID int64,
) (ids []int64, err error) {
	var registryIDs []int64

	stmt := databaseg.Builder.Select("registry_id").
		From("registries").
		Where("registry_parent_id = ?", parentID).
		Where("registry_upstream_proxies LIKE ?", "%"+upstreamProxyID+"%").
		Where("registry_type = ?", artifact.RegistryTypeVIRTUAL)

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
		RegID:         dst.RegID,
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

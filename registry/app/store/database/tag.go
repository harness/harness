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
	"strings"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	// OrderDesc is the normalized string to be used for sorting results in descending order.
	OrderDesc           types.SortOrder = "desc"
	lessThan            string          = "<"
	greaterThan         string          = ">"
	labelSeparatorStart string          = "%^_"
	labelSeparatorEnd   string          = "^_%"
	downloadCount       string          = "download_count"
	imageName           string          = "image_name"
	name                string          = "name"
	postgresStringAgg   string          = "string_agg"
	sqliteGroupConcat   string          = "group_concat"
)

type tagDao struct {
	db *sqlx.DB
}

func NewTagDao(db *sqlx.DB) store.TagRepository {
	return &tagDao{
		db: db,
	}
}

// tagDB holds the record of a tag in DB.
type tagDB struct {
	ID         int64         `db:"tag_id"`
	Name       string        `db:"tag_name"`
	ImageName  string        `db:"tag_image_name"`
	RegistryID int64         `db:"tag_registry_id"`
	ManifestID int64         `db:"tag_manifest_id"`
	CreatedAt  int64         `db:"tag_created_at"`
	UpdatedAt  int64         `db:"tag_updated_at"`
	CreatedBy  sql.NullInt64 `db:"tag_created_by"`
	UpdatedBy  sql.NullInt64 `db:"tag_updated_by"`
}

type artifactMetadataDB struct {
	ID               int64                  `db:"artifact_id"`
	Name             string                 `db:"name"`
	RepoName         string                 `db:"repo_name"`
	DownloadCount    int64                  `db:"download_count"`
	PackageType      artifact.PackageType   `db:"package_type"`
	Labels           sql.NullString         `db:"labels"`
	LatestVersion    string                 `db:"latest_version"`
	CreatedAt        int64                  `db:"created_at"`
	ModifiedAt       int64                  `db:"modified_at"`
	Tag              *string                `db:"tag"`
	Version          string                 `db:"version"`
	Metadata         *json.RawMessage       `db:"metadata"`
	IsQuarantined    bool                   `db:"is_quarantined"`
	QuarantineReason *string                `db:"quarantine_reason"`
	ArtifactType     *artifact.ArtifactType `db:"artifact_type"`
	Tags             *string                `db:"tags"`
}

type tagMetadataDB struct {
	Name          string               `db:"name"`
	Size          string               `db:"size"`
	PackageType   artifact.PackageType `db:"package_type"`
	DigestCount   int                  `db:"digest_count"`
	ModifiedAt    int64                `db:"modified_at"`
	SchemaVersion int                  `db:"manifest_schema_version"`
	NonConformant bool                 `db:"manifest_non_conformant"`
	Payload       []byte               `db:"manifest_payload"`
	MediaType     string               `db:"mt_media_type"`
	Digest        []byte               `db:"manifest_digest"`
	DownloadCount int64                `db:"download_count"`
}

type ociVersionMetadataDB struct {
	Size          string               `db:"size"`
	PackageType   artifact.PackageType `db:"package_type"`
	DigestCount   int                  `db:"digest_count"`
	ModifiedAt    int64                `db:"modified_at"`
	SchemaVersion int                  `db:"manifest_schema_version"`
	NonConformant bool                 `db:"manifest_non_conformant"`
	Payload       []byte               `db:"manifest_payload"`
	MediaType     string               `db:"mt_media_type"`
	Digest        []byte               `db:"manifest_digest"`
	DownloadCount int64                `db:"download_count"`
	Tags          *string              `db:"tags"`
}

type tagDetailDB struct {
	ID            int64  `db:"id"`
	Name          string `db:"name"`
	ImageName     string `db:"image_name"`
	CreatedAt     int64  `db:"created_at"`
	UpdatedAt     int64  `db:"updated_at"`
	Size          string `db:"size"`
	DownloadCount int64  `db:"download_count"`
}

type tagInfoDB struct {
	Name   string `db:"name"`
	Digest []byte `db:"manifest_digest"`
}

func (t tagDao) CreateOrUpdate(ctx context.Context, tag *types.Tag) error {
	const sqlQuery = `
		INSERT INTO tags ( 
			tag_name
			,tag_image_name
			,tag_registry_id
			,tag_manifest_id
			,tag_created_at
			,tag_updated_at
			,tag_created_by
			,tag_updated_by
		) VALUES (
			:tag_name
			,:tag_image_name
			,:tag_registry_id
			,:tag_manifest_id
			,:tag_created_at
			,:tag_updated_at
			,:tag_created_by
			,:tag_updated_by
		) 
			ON CONFLICT (tag_registry_id, tag_name, tag_image_name)
		    DO UPDATE SET
			   tag_manifest_id = :tag_manifest_id,
		       tag_updated_at = :tag_updated_at
			WHERE
			   tags.tag_manifest_id <> :tag_manifest_id
	   RETURNING
		   tag_id, tag_created_at, tag_updated_at`

	db := dbtx.GetAccessor(ctx, t.db)
	tagDB := t.mapToInternalTag(ctx, tag)
	query, arg, err := db.BindNamed(sqlQuery, tagDB)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(
		&tagDB.ID,
		&tagDB.CreatedAt, &tagDB.UpdatedAt,
	); err != nil {
		err := databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
		if !errors.Is(err, store2.ErrResourceNotFound) {
			return err
		}
	}
	return nil
}

// LockTagByNameForUpdate locks a tag by name within a repository using SELECT FOR UPDATE.
// It returns a boolean indicating whether the tag exists and was successfully locked.
func (t tagDao) LockTagByNameForUpdate(
	ctx context.Context, repoID int64,
	name string,
) (bool, error) {
	// Since tag_registry_id is not unique in the DB schema, we use LIMIT 1 to ensure that
	// only one record is locked and processed.
	stmt := databaseg.Builder.Select("1").
		From("tags").
		Where("tag_registry_id = ? AND tag_name = ?", repoID, name).
		Limit(1).
		Suffix("FOR UPDATE")

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert select for update query to SQL: %w", err)
	}

	db := dbtx.GetAccessor(ctx, t.db)

	var exists int
	err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // Tag does not exist
		}
		return false, databaseg.ProcessSQLErrorf(ctx, err, "the select for update query failed")
	}
	return true, nil
}

// DeleteTagByName deletes a tag by name within a repository. A boolean is returned to denote whether the tag was
// deleted or not. This avoids the need for a separate preceding `SELECT` to find if it exists.
func (t tagDao) DeleteTagByName(
	ctx context.Context, repoID int64,
	name string,
) (bool, error) {
	stmt := databaseg.Builder.Delete("tags").
		Where("tag_registry_id = ? AND tag_name = ?", repoID, name)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert purge tag query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, t.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	count, _ := result.RowsAffected()
	return count == 1, nil
}

// DeleteTagByName deletes a tag by name within a repository.
//
//	A boolean is returned to denote whether the tag was
//
// deleted or not. This avoids the need for a separate preceding
//
//	`SELECT` to find if it exists.
func (t tagDao) DeleteTagByManifestID(
	ctx context.Context,
	repoID int64,
	manifestID int64,
) (bool, error) {
	stmt := databaseg.Builder.Delete("tags").
		Where("tag_registry_id = ? AND tag_manifest_id = ?", repoID, manifestID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert purge tag query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, t.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	count, _ := result.RowsAffected()
	return count > 0, nil
}

// TagsPaginated finds up to `filters.MaxEntries` tags of a given
// repository with name lexicographically after `filters.LastEntry`.
// This is used exclusively for the GET /v2/<name>/tags/list API route,
// where pagination is done with a marker (`filters.LastEntry`).
// Even if there is no tag with a name of `filters.LastEntry`,
// the returned tags will always be those with a path lexicographically after
// `filters.LastEntry`. Finally, tags are lexicographically sorted.
// These constraints exists to preserve the existing API behaviour
// (when doing a filesystem walk based pagination).
func (t tagDao) TagsPaginated(
	ctx context.Context, repoID int64,
	image string, filters types.FilterParams,
) ([]*types.Tag, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(tagDB{}), ",")).
		From("tags").
		Where(
			"tag_registry_id = ? AND tag_image_name = ? AND tag_name > ?",
			repoID, image, filters.LastEntry,
		).
		OrderBy("tag_name").Limit(uint64(filters.MaxEntries)) //nolint:gosec

	db := dbtx.GetAccessor(ctx, t.db)

	dst := []*tagDB{}
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find tag")
	}
	return t.mapToTagList(ctx, dst)
}

func (t tagDao) HasTagsAfterName(
	ctx context.Context, repoID int64,
	filters types.FilterParams,
) (bool, error) {
	stmt := databaseg.Builder.
		Select("COUNT(*)").
		From("tags").
		Where(
			"tag_registry_id = ? AND tag_name LIKE ? ",
			repoID, sqlPartialMatch(filters.Name),
		)
	comparison := greaterThan
	if filters.SortOrder == OrderDesc {
		comparison = lessThan
	}

	if filters.OrderBy != "published_at" {
		stmt = stmt.Where("tag_name "+comparison+" ?", filters.LastEntry)
	} else {
		stmt = stmt.Where(
			"AND (GREATEST(tag_created_at, tag_updated_at), tag_name) "+comparison+" (? ?)",
			filters.PublishedAt, filters.LastEntry,
		)
	}
	stmt = stmt.OrderBy("tag_name").GroupBy("tag_name").Limit(uint64(filters.MaxEntries)) //nolint:gosec

	db := dbtx.GetAccessor(ctx, t.db)

	var count int64
	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return false, errors.Wrap(err, "Failed to convert query to sqlQuery")
	}

	if err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return false,
			databaseg.ProcessSQLErrorf(ctx, err, "Failed to find tag")
	}
	return count == 1, nil
}

// sqlPartialMatch builds a string that can be passed as value
//
//	for a SQL `LIKE` expression. Besides surrounding the
//
// input value with `%` wildcard characters for a partial match,
//
//	this function also escapes the `_` and `%`
//
// metacharacters supported in Postgres `LIKE` expressions.
// See https://www.postgresql.org/docs/current/
// functions-matching.html#FUNCTIONS-LIKE for more details.
func sqlPartialMatch(value string) string {
	value = strings.ReplaceAll(value, "_", `\_`)
	value = strings.ReplaceAll(value, "%", `\%`)

	return fmt.Sprintf("%%%s%%", value)
}

func (t tagDao) GetAllArtifactsByParentID(
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
	q1 := t.GetAllArtifactOnParentIDQueryForNonOCI(parentID, latestVersion, registryIDs, packageTypes, search, false)

	q2 := t.GetAllArtifactsQueryByParentIDForOCI(parentID, latestVersion, registryIDs, packageTypes, search)

	q1SQL, q1Args, err := q1.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	q2SQL, _, err := q2.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	// Combine q1 and q2 with UNION ALL
	finalQuery := fmt.Sprintf(`
    SELECT repo_name, name, package_type, version, modified_at,
           labels, download_count, is_quarantined, quarantine_reason, artifact_type
    FROM (%s UNION ALL %s) AS combined
`, q1SQL, q2SQL)

	// Combine query arguments
	finalArgs := q1Args

	// Apply sorting based on provided field
	sortField := "modified_at"
	switch sortByField {
	case downloadCount:
		sortField = "download_count"
	case imageName:
		sortField = "name"
	}

	finalQuery = fmt.Sprintf("%s ORDER BY %s %s", finalQuery, sortField, sortByOrder)

	// Add pagination (LIMIT and OFFSET) **after** the WHERE and ORDER BY clauses
	finalQuery = fmt.Sprintf("%s LIMIT %d OFFSET %d", finalQuery, limit, offset)

	db := dbtx.GetAccessor(ctx, t.db)

	dst := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, finalQuery, finalArgs...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return t.mapToArtifactMetadataList(dst)
}

func (t tagDao) GetAllArtifactsQueryByParentIDForOCI(
	parentID int64, latestVersion bool, registryIDs *[]string,
	packageTypes []string, search string,
) sq.SelectBuilder {
	q2 := databaseg.Builder.Select(
		`r.registry_name as repo_name, 
		t.tag_image_name as name, 
		r.registry_package_type as package_type, 
		t.tag_name as version, 
		t.tag_updated_at as modified_at, 
		i.image_labels as labels, 
		COALESCE(t2.download_count,0) as download_count,
        false as is_quarantined,
		'' as quarantine_reason,
        i.image_type as artifact_type `,
	).
		From("tags t").
		Join("registries r ON t.tag_registry_id = r.registry_id").
		Where("r.registry_parent_id = ?", parentID).
		Join(
			"images i ON i.image_registry_id = t.tag_registry_id AND"+
				" i.image_name = t.tag_image_name",
		).
		LeftJoin(
			`( SELECT i.image_name, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_image_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id 
			GROUP BY a.artifact_image_id ) as t1 
			JOIN images i ON i.image_id = t1.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? GROUP BY i.image_name) as t2 
			ON t.tag_image_name = t2.image_name`, parentID,
		)

	if latestVersion {
		q2 = q2.Join(
			`(SELECT t.tag_id as id, ROW_NUMBER() OVER (PARTITION BY t.tag_registry_id, t.tag_image_name 
			ORDER BY t.tag_updated_at DESC) AS rank FROM tags t 
			JOIN registries r ON t.tag_registry_id = r.registry_id 
			WHERE r.registry_parent_id = ? ) AS a 
			ON t.tag_id = a.id`, parentID, // nolint:goconst
		).
			Where("a.rank = 1")
	}

	if len(*registryIDs) > 0 {
		q2 = q2.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if len(packageTypes) > 0 {
		q2 = q2.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	if search != "" {
		q2 = q2.Where("t.tag_image_name LIKE ?", sqlPartialMatch(search))
	}
	return q2
}

func (t tagDao) GetAllArtifactsByParentIDUntagged(
	ctx context.Context,
	parentID int64,
	registryIDs *[]string,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
	search string,
	packageTypes []string,
) (*[]types.ArtifactMetadata, error) {
	// Query 1: Get core artifact data with pagination
	coreQuery := t.getCoreArtifactsQuery(
		parentID, registryIDs, packageTypes, search, sortByField, sortByOrder, limit, offset)

	coreSQL, coreArgs, err := coreQuery.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert core query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)
	coreResults := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &coreResults, coreSQL, coreArgs...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing core artifacts query")
	}

	if len(coreResults) == 0 {
		return &[]types.ArtifactMetadata{}, nil
	}

	// Extract artifact IDs for enrichment query
	artifactIDs := make([]int64, len(coreResults))
	for i, result := range coreResults {
		artifactIDs[i] = result.ID
	}

	// Query 2: Get download counts and tags only for returned artifacts
	enrichmentData, err := t.getArtifactEnrichmentData(ctx, artifactIDs)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get enrichment data")
	}

	// Merge the data
	for _, result := range coreResults {
		if enrichment, exists := enrichmentData[result.ID]; exists {
			result.DownloadCount = enrichment.DownloadCount
			if enrichment.Tags != "" {
				result.Tags = &enrichment.Tags
			}
		}
	}

	return t.mapToArtifactMetadataList(coreResults)
}

// getCoreArtifactsQuery returns core artifact data with pagination (no download counts or tags).
func (t tagDao) getCoreArtifactsQuery(
	parentID int64,
	registryIDs *[]string,
	packageTypes []string,
	search string,
	sortByField string,
	sortByOrder string,
	limit int,
	offset int,
) sq.SelectBuilder {
	query := databaseg.Builder.Select(
		`ar.artifact_id,
		r.registry_name as repo_name, 
		i.image_name as name, 
		r.registry_package_type as package_type,
		ar.artifact_version as version, 
		ar.artifact_updated_at as modified_at, 
		i.image_type as artifact_type`,
	).
		From("artifacts ar").
		Join("images i ON i.image_id = ar.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ?", parentID)

	// Apply filters
	if len(*registryIDs) > 0 {
		query = query.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if len(packageTypes) > 0 {
		query = query.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	if search != "" {
		query = query.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}

	// Apply sorting and pagination
	sortField := "ar.artifact_updated_at"
	if sortByField == imageName {
		sortField = "i.image_name"
	}

	return query.OrderBy(fmt.Sprintf("%s %s", sortField, sortByOrder)).
		Limit(uint64(limit)).  // nolint:gosec
		Offset(uint64(offset)) // nolint:gosec
}

type enrichmentData struct {
	DownloadCount int64
	Tags          string
}

func (t tagDao) getArtifactEnrichmentData(ctx context.Context, artifactIDs []int64) (map[int64]enrichmentData, error) {
	if len(artifactIDs) == 0 {
		return make(map[int64]enrichmentData), nil
	}

	db := dbtx.GetAccessor(ctx, t.db)
	driver := db.DriverName()

	// Pick aggregation function and decode function depending on database
	var tagAggExpr, decodeFunction string
	if driver == "postgres" {
		tagAggExpr = "STRING_AGG(t.tag_name, ',')"
		decodeFunction = "decode(ar.artifact_version, 'hex')"
	} else {
		tagAggExpr = "GROUP_CONCAT(t.tag_name, ',')"
		decodeFunction = "unhex(ar.artifact_version)"
	}

	// Build placeholders for the 3 IN clauses
	var placeholders1, placeholders2, placeholders3 string
	args := make([]any, 0, len(artifactIDs)*3)
	const placeholderSeparator = ", ?"

	// nolint:nestif
	if driver == "postgres" {
		// PostgreSQL: sequential parameter numbering across entire query
		paramIndex := 1

		// First IN clause (download_counts)
		for i := range artifactIDs {
			if i > 0 {
				placeholders1 += fmt.Sprintf(", $%d", paramIndex)
			} else {
				placeholders1 += fmt.Sprintf("$%d", paramIndex)
			}
			paramIndex++
		}

		// Second IN clause (tag_lists)
		for i := range artifactIDs {
			if i > 0 {
				placeholders2 += fmt.Sprintf(", $%d", paramIndex)
			} else {
				placeholders2 += fmt.Sprintf("$%d", paramIndex)
			}
			paramIndex++
		}

		// Third IN clause (main WHERE)
		for i := range artifactIDs {
			if i > 0 {
				placeholders3 += fmt.Sprintf(", $%d", paramIndex)
			} else {
				placeholders3 += fmt.Sprintf("$%d", paramIndex)
			}
			paramIndex++
		}
	} else {
		// SQLite: ? placeholders can be reused
		for i := range artifactIDs {
			if i > 0 {
				placeholders1 += placeholderSeparator
				placeholders2 += placeholderSeparator
				placeholders3 += placeholderSeparator
			} else {
				placeholders1 += "?"
				placeholders2 += "?"
				placeholders3 += "?"
			}
		}
	}

	// Arguments: artifactIDs repeated 3 times for the 3 IN clauses
	for range 3 {
		for _, id := range artifactIDs {
			args = append(args, id)
		}
	}

	query := fmt.Sprintf(`
    WITH download_counts AS (
        SELECT 
            ds.download_stat_artifact_id AS artifact_id,
            COUNT(ds.download_stat_id) AS download_count
        FROM download_stats ds
        WHERE ds.download_stat_artifact_id IN (%s)
        GROUP BY ds.download_stat_artifact_id
    ),
    tag_lists AS (
        SELECT 
            ar.artifact_id,
            %s AS tags
        FROM artifacts ar
        JOIN images i ON i.image_id = ar.artifact_image_id
        JOIN manifests m ON m.manifest_registry_id = i.image_registry_id 
                         AND m.manifest_image_name = i.image_name
                         AND m.manifest_digest = %s
        JOIN tags t ON t.tag_manifest_id = m.manifest_id
        WHERE ar.artifact_id IN (%s)
        GROUP BY ar.artifact_id
    )
    SELECT 
        ar.artifact_id,
        COALESCE(dc.download_count, 0) AS download_count,
        COALESCE(tl.tags, '') AS tags
    FROM artifacts ar
    LEFT JOIN download_counts dc ON dc.artifact_id = ar.artifact_id
    LEFT JOIN tag_lists tl ON tl.artifact_id = ar.artifact_id
    WHERE ar.artifact_id IN (%s);
    `, placeholders1, tagAggExpr, decodeFunction, placeholders2, placeholders3)

	type enrichmentRow struct {
		ArtifactID    int64  `db:"artifact_id"`
		DownloadCount int64  `db:"download_count"`
		Tags          string `db:"tags"`
	}

	var rows []enrichmentRow
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing enrichment query")
	}

	result := make(map[int64]enrichmentData, len(rows))
	for _, row := range rows {
		result[row.ArtifactID] = enrichmentData{
			DownloadCount: row.DownloadCount,
			Tags:          row.Tags,
		}
	}

	return result, nil
}

func (t tagDao) GetAllArtifactOnParentIDQueryForNonOCI(
	parentID int64, latestVersion bool, registryIDs *[]string,
	packageTypes []string, search string, queryTags bool,
) sq.SelectBuilder {
	suffix := " "
	if queryTags {
		suffix = ", NULL AS tags "
	}
	q1 := databaseg.Builder.Select(
		`r.registry_name as repo_name, 
		i.image_name as name, 
		r.registry_package_type as package_type,
		ar.artifact_version as version, 
		ar.artifact_updated_at as modified_at, 
		i.image_labels as labels, 
		COALESCE(t2.download_count, 0) as download_count,
        (qp.quarantined_path_id IS NOT NULL) AS is_quarantined,
        qp.quarantined_path_reason as quarantine_reason,
        i.image_type as artifact_type`+suffix,
	).
		From("artifacts ar").
		Join("images i ON i.image_id = ar.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ? AND r.registry_package_type NOT IN ('DOCKER', 'HELM')", parentID).
		LeftJoin("quarantined_paths qp ON ((qp.quarantined_path_artifact_id = ar.artifact_id "+
			"OR qp.quarantined_path_artifact_id IS NULL) "+
			"AND qp.quarantined_path_image_id = i.image_id) AND qp.quarantined_path_registry_id = r.registry_id").
		LeftJoin(
			`( SELECT a.artifact_version, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id 
			GROUP BY a.artifact_id ) as t1 
            JOIN artifacts a ON a.artifact_id = t1.artifact_id 
			JOIN images i ON i.image_id = a.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? GROUP BY a.artifact_version) as t2 
			ON ar.artifact_version = t2.artifact_version`, parentID,
		)

	if latestVersion {
		q1 = q1.Join(
			`(SELECT ar.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY ar.artifact_image_id 
			ORDER BY ar.artifact_updated_at DESC) AS rank FROM artifacts ar 
            JOIN images i ON i.image_id = ar.artifact_image_id 
			JOIN registries r ON i.image_registry_id = r.registry_id 
			WHERE r.registry_parent_id = ? ) AS a 
			ON ar.artifact_id = a.id`, parentID, // nolint:goconst
		).
			Where("a.rank = 1")
	}

	if len(*registryIDs) > 0 {
		q1 = q1.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if len(packageTypes) > 0 {
		q1 = q1.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	if search != "" {
		q1 = q1.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}
	return q1
}

func (t tagDao) CountAllOCIArtifactsByParentID(
	ctx context.Context, parentID int64,
	registryIDs *[]string, search string, latestVersion bool, packageTypes []string,
) (int64, error) {
	// nolint:goconst
	q := databaseg.Builder.Select("COUNT(*)").
		From("tags t").
		Join("registries r ON t.tag_registry_id = r.registry_id"). // nolint:goconst
		Where("r.registry_parent_id = ?", parentID).
		Join(
			"images ar ON ar.image_registry_id = t.tag_registry_id" +
				" AND ar.image_name = t.tag_image_name",
		)

	if latestVersion {
		q = q.Join(
			`(SELECT t.tag_id as id, ROW_NUMBER() OVER (PARTITION BY t.tag_registry_id, t.tag_image_name 
			ORDER BY t.tag_updated_at DESC) AS rank FROM tags t 
			JOIN registries r ON t.tag_registry_id = r.registry_id 
			WHERE r.registry_parent_id = ? ) AS a 
			ON t.tag_id = a.id`, parentID, // nolint:goconst
		).Where("a.rank = 1")
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
	db := dbtx.GetAccessor(ctx, t.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (t tagDao) CountAllArtifactsByParentID(
	ctx context.Context, parentID int64,
	registryIDs *[]string, search string, latestVersion bool, packageTypes []string, untaggedImagesEnabled bool,
) (int64, error) {
	if untaggedImagesEnabled {
		// Use the new unified count function for all artifacts
		return t.CountAllArtifactsByParentIDUntagged(ctx, parentID, registryIDs, search, latestVersion, packageTypes)
	}

	q := databaseg.Builder.Select("COUNT(*)").
		From("artifacts ar").
		Join("images i ON i.image_id = ar.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ? AND r.registry_package_type NOT IN ('DOCKER', 'HELM')", parentID)

	if latestVersion {
		q = q.Join(
			`(SELECT ar.artifact_id as id, ROW_NUMBER() OVER (PARTITION BY ar.artifact_image_id 
			ORDER BY ar.artifact_updated_at DESC) AS rank FROM artifacts ar 
            JOIN images i ON i.image_id = ar.artifact_image_id 
			JOIN registries r ON i.image_registry_id = r.registry_id 
			WHERE r.registry_parent_id = ? ) AS a 
			ON ar.artifact_id = a.id`, parentID,
		).Where("a.rank = 1")
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
	db := dbtx.GetAccessor(ctx, t.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	var ociCount int64
	ociCount, err = t.CountAllOCIArtifactsByParentID(ctx, parentID, registryIDs, search,
		latestVersion, packageTypes)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count + ociCount, nil
}

func (t tagDao) CountAllArtifactsByParentIDUntagged(
	ctx context.Context, parentID int64,
	registryIDs *[]string, search string, _ bool, packageTypes []string,
) (int64, error) {
	query := databaseg.Builder.Select("COUNT(*)").
		From("artifacts ar").
		Join("images i ON i.image_id = ar.artifact_image_id").
		Join("registries r ON i.image_registry_id = r.registry_id").
		Where("r.registry_parent_id = ?", parentID)

	// Apply filters
	if len(*registryIDs) > 0 {
		query = query.Where(sq.Eq{"r.registry_name": registryIDs})
	}

	if len(packageTypes) > 0 {
		query = query.Where(sq.Eq{"r.registry_package_type": packageTypes})
	}

	if search != "" {
		query = query.Where("i.image_name LIKE ?", sqlPartialMatch(search))
	}

	querySQL, queryArgs, err := query.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert count query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)
	var count int64
	if err := db.GetContext(ctx, &count, querySQL, queryArgs...); err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

func (t tagDao) GetTagDetail(
	ctx context.Context, repoID int64, imageName string,
	name string,
) (*types.TagDetail, error) {
	// Define subquery for download counts
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
	// Build main query
	q := databaseg.Builder.
		Select(`
            t.tag_id AS id, 
            t.tag_name AS name, 
            t.tag_image_name AS image_name, 
            t.tag_created_at AS created_at, 
            t.tag_updated_at AS updated_at, 
            m.manifest_total_size AS size, 
            COALESCE(dc.download_count, 0) AS download_count
        `).
		From("tags AS t").
		Join("manifests AS m ON m.manifest_id = t.tag_manifest_id").
		LeftJoin(fmt.Sprintf("(%s) AS dc ON t.tag_image_name = dc.image_name "+
			"AND t.tag_registry_id = dc.image_registry_id", downloadCountSubquery)).
		Where(
			"t.tag_registry_id = ? AND t.tag_image_name = ? AND t.tag_name = ?",
			repoID, imageName, name,
		)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	dst := new(tagDetailDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}

	return t.mapToTagDetail(ctx, dst)
}

func (t tagDao) DeleteTag(ctx context.Context, registryID int64, imageName string, name string) (err error) {
	stmt := databaseg.Builder.Delete("tags").
		Where("tag_registry_id = ? AND tag_image_name = ? AND tag_name = ?", registryID, imageName, name)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge tags query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, t.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}
	return nil
}

func (t tagDao) GetLatestTagMetadata(
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
	if t.db.DriverName() == SQLITE3 {
		q = databaseg.Builder.Select(
			`r.registry_name AS repo_name, r.registry_package_type AS package_type,
     t.tag_image_name AS name, t.tag_name AS latest_version,
     t.tag_created_at AS created_at, t.tag_updated_at AS modified_at,
     ar.image_labels AS labels, COALESCE(dc_subquery.download_count, 0) AS download_count`,
		).
			From("tags t").
			Join("registries r ON t.tag_registry_id = r.registry_id"). // nolint:goconst
			Join("images ar ON ar.image_registry_id = t.tag_registry_id AND ar.image_name = t.tag_image_name").
			LeftJoin(fmt.Sprintf("(%s) AS dc_subquery ON dc_subquery.image_name = t.tag_image_name "+
				"AND dc_subquery.image_registry_id = r.registry_id", downloadCountSubquery)).
			Where(
				"r.registry_parent_id = ? AND r.registry_name = ? AND t.tag_image_name = ?",
				parentID, repoKey, imageName,
			).
			OrderBy("t.tag_updated_at DESC").Limit(1)
	} else {
		q = databaseg.Builder.Select(
			`r.registry_name AS repo_name,
         r.registry_package_type AS package_type,
         t.tag_image_name AS name,
         t.tag_name AS latest_version,
         t.tag_created_at AS created_at,
         t.tag_updated_at AS modified_at,
         ar.image_labels AS labels,
         COALESCE(t2.download_count, 0) AS download_count`,
		).
			From("tags t").
			Join("registries r ON t.tag_registry_id = r.registry_id"). // nolint:goconst
			Join("images ar ON ar.image_registry_id = t.tag_registry_id AND ar.image_name = t.tag_image_name").
			LeftJoin(fmt.Sprintf("LATERAL (%s) AS t2 ON t.tag_image_name = t2.image_name", downloadCountSubquery)).
			Where(
				"r.registry_parent_id = ? AND r.registry_name = ? AND t.tag_image_name = ?",
				parentID, repoKey, imageName,
			).
			OrderBy("t.tag_updated_at DESC").Limit(1)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}
	// Log the final sql query
	finalQuery := util.FormatQuery(sql, args)
	log.Ctx(ctx).Debug().Str("sql", finalQuery).Msg("Executing GetLatestTagMetadata query")
	// Execute query
	db := dbtx.GetAccessor(ctx, t.db)

	dst := new(artifactMetadataDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}

	return t.mapToArtifactMetadata(dst)
}

func (t tagDao) GetLatestTagName(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
) (string, error) {
	q := databaseg.Builder.Select("tag_name as name").
		From("tags").
		Join("registries ON tag_registry_id = registry_id").
		Where(
			"registry_parent_id = ? AND registry_name = ? AND tag_image_name = ?",
			parentID, repoKey, imageName,
		).
		OrderBy("tag_updated_at DESC").Limit(1)

	sql, args, err := q.ToSql()
	if err != nil {
		return "", errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	var tag string
	err = db.QueryRowContext(ctx, sql, args...).Scan(&tag)
	if err != nil {
		return tag, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing get tag name query")
	}
	return tag, nil
}

func (t tagDao) GetTagMetadata(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
	name string,
) (*types.OciVersionMetadata, error) {
	q := databaseg.Builder.Select(
		"registry_package_type as package_type, tag_name as name,"+
			"tag_updated_at as modified_at, manifest_total_size as size",
	).
		From("tags").
		Join("registries ON tag_registry_id = registry_id").
		Join("manifests ON manifest_id = tag_manifest_id").
		Where(
			"registry_parent_id = ? AND registry_name = ?"+
				" AND tag_image_name = ? AND tag_name = ?", parentID, repoKey, imageName, name,
		)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	dst := new(tagMetadataDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag metadata")
	}

	return t.mapToTagMetadata(ctx, dst)
}

func (t tagDao) GetOCIVersionMetadata(
	ctx context.Context,
	parentID int64,
	repoKey string,
	imageName string,
	dgst string,
) (*types.OciVersionMetadata, error) {
	digestBytes, err := types.GetDigestBytes(digest.Digest(dgst))
	if err != nil {
		return nil, err
	}
	q := databaseg.Builder.Select(
		"registry_package_type as package_type, manifest_digest, "+
			"manifest_created_at as modified_at, manifest_total_size as size",
	).
		From("manifests").
		Join("registries ON manifest_registry_id = registry_id").
		Where(
			"registry_parent_id = ? AND registry_name = ?"+
				" AND manifest_image_name = ? AND manifest_digest = ?", parentID, repoKey, imageName, digestBytes,
		)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	dst := new(ociVersionMetadataDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag metadata")
	}

	return t.mapToOciVersion(dst)
}

func (t tagDao) GetLatestTag(ctx context.Context, repoID int64, imageName string) (*types.Tag, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(tagDB{}), ",")).
		From("tags").
		Where("tag_registry_id = ? AND tag_image_name = ?", repoID, imageName).
		OrderBy("tag_updated_at DESC").Limit(1)

	db := dbtx.GetAccessor(ctx, t.db)

	dst := new(tagDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find tag")
	}

	return t.mapToTag(ctx, dst)
}

func (t tagDao) GetAllArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey string,
	sortByField string, sortByOrder string, limit int, offset int, search string,
	labels []string,
) (*[]types.ArtifactMetadata, error) {
	q := databaseg.Builder.Select(
		`r.registry_name as repo_name, t.tag_image_name as name, 
		r.registry_package_type as package_type, t.tag_name as latest_version, 
		t.tag_updated_at as modified_at, ar.image_labels as labels, 
		COALESCE(t2.download_count, 0) as download_count`,
	).
		From("tags t").
		Join(
			`(SELECT t.tag_id as id, ROW_NUMBER() OVER (PARTITION BY t.tag_registry_id, t.tag_image_name 
			ORDER BY t.tag_updated_at DESC) AS rank FROM tags t 
			JOIN registries r ON t.tag_registry_id = r.registry_id  
			WHERE r.registry_parent_id = ? AND r.registry_name = ? ) AS a 
			ON t.tag_id = a.id`, parentID, repoKey, // nolint:goconst
		).
		Join("registries r ON t.tag_registry_id = r.registry_id").
		Join(
			"images ar ON ar.image_registry_id = t.tag_registry_id"+
				" AND ar.image_name = t.tag_image_name",
		).
		LeftJoin(
			`( SELECT i.image_name, SUM(COALESCE(t1.download_count, 0)) as download_count FROM 
			( SELECT a.artifact_image_id, COUNT(d.download_stat_id) as download_count 
			FROM artifacts a 
			JOIN download_stats d ON d.download_stat_artifact_id = a.artifact_id GROUP BY 
			a.artifact_image_id ) as t1 
			JOIN images i ON i.image_id = t1.artifact_image_id 
			JOIN registries r ON r.registry_id = i.image_registry_id 
			WHERE r.registry_parent_id = ? AND r.registry_name = ? GROUP BY i.image_name) as t2 
			ON t.tag_image_name = t2.image_name`, parentID, repoKey,
		).
		Where("a.rank = 1 ")

	if search != "" {
		q = q.Where("tag_image_name LIKE ?", sqlPartialMatch(search))
	}

	if len(labels) > 0 {
		sort.Strings(labels)
		labelsVal := util.GetEmptySQLString(util.ArrToString(labels))
		labelsVal.String = labelSeparatorStart + labelsVal.String + labelSeparatorEnd
		q = q.Where("'^_' || ar.image_labels || '^_' LIKE ?", labelsVal)
	}

	sortField := "t.tag_" + sortByField
	if sortByField == downloadCount {
		sortField = downloadCount
	}
	q = q.OrderBy(sortField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset)) //nolint:gosec

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	dst := []*artifactMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return t.mapToArtifactMetadataList(dst)
}

// nolint:goconst
func (t tagDao) CountAllArtifactsByRepo(
	ctx context.Context, parentID int64, repoKey string,
	search string, labels []string,
) (int64, error) {
	q := databaseg.Builder.Select("COUNT(*)").
		From("tags t").
		Join(
			`(SELECT t.tag_id as id, ROW_NUMBER() OVER (PARTITION BY t.tag_registry_id, t.tag_image_name 
			ORDER BY t.tag_updated_at DESC) AS rank FROM tags t 
			JOIN registries r ON t.tag_registry_id = r.registry_id 
			WHERE r.registry_parent_id = ? AND r.registry_name = ? ) AS a ON t.tag_id = a.id`, parentID, repoKey,
		).
		Join("registries r ON t.tag_registry_id = r.registry_id").
		Join(
			"images ar ON ar.image_registry_id = t.tag_registry_id AND" +
				" ar.image_name = t.tag_image_name",
		).
		Where("a.rank = 1 ")

	if search != "" {
		q = q.Where("tag_image_name LIKE ?", sqlPartialMatch(search))
	}

	if len(labels) > 0 {
		sort.Strings(labels)
		labelsVal := util.GetEmptySQLString(util.ArrToString(labels))
		labelsVal.String = labelSeparatorStart + labelsVal.String + labelSeparatorEnd
		q = q.Where("'^_' || ar.image_labels || '^_' LIKE ?", labelsVal)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}
	db := dbtx.GetAccessor(ctx, t.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (t tagDao) GetAllTagsByRepoAndImage(
	ctx context.Context, parentID int64, repoKey string,
	image string, sortByField string, sortByOrder string, limit int, offset int,
	search string,
) (*[]types.OciVersionMetadata, error) {
	q := databaseg.Builder.Select(
		`t.tag_name as name, m.manifest_total_size as size, 
		r.registry_package_type as package_type, t.tag_updated_at as modified_at, 
		m.manifest_schema_version, m.manifest_non_conformant, m.manifest_payload, 
		mt.mt_media_type, m.manifest_digest`,
	).
		From("tags t").
		Join("registries r ON t.tag_registry_id = r.registry_id").
		Join("manifests m ON t.tag_manifest_id = m.manifest_id").
		Join("media_types mt ON mt.mt_id = m.manifest_media_type_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ? AND t.tag_image_name = ?",
			parentID, repoKey, image,
		)

	if search != "" {
		q = q.Where("tag_name LIKE ?", sqlPartialMatch(search))
	}
	q = q.OrderBy("t.tag_" + sortByField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset)) //nolint:gosec

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	dst := []*tagMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return t.mapToTagMetadataList(ctx, dst)
}

// GetQuarantineStatusForImages gets quarantine status for a list of image names
// Returns a boolean slice in the same order as the input image names.
func (t tagDao) GetQuarantineStatusForImages(
	ctx context.Context, imageNames []string, registryID int64,
) ([]bool, error) {
	if len(imageNames) == 0 {
		return []bool{}, nil
	}

	q := databaseg.Builder.Select(
		"DISTINCT i.image_name",
	).From("quarantined_paths qp").
		Join("images i ON qp.quarantined_path_image_id = i.image_id").
		Where("qp.quarantined_path_registry_id = ?", registryID).
		Where(sq.Eq{"i.image_name": imageNames})

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert quarantine status query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	type quarantineResult struct {
		ImageName string `db:"image_name"`
	}

	var results []quarantineResult
	if err = db.SelectContext(ctx, &results, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get quarantine status")
	}

	// Create a set of quarantined image names for quick lookup
	quarantinedImages := make(map[string]bool)
	for _, result := range results {
		quarantinedImages[result.ImageName] = true
	}

	// Build result slice in the same order as input
	resultSlice := make([]bool, len(imageNames))
	for i, imageName := range imageNames {
		resultSlice[i] = quarantinedImages[imageName]
	}

	return resultSlice, nil
}

func (t tagDao) GetAllOciVersionsByRepoAndImage(
	ctx context.Context, parentID int64, repoKey string,
	image string, sortByField string, sortByOrder string, limit int, offset int,
	search string,
) (*[]types.OciVersionMetadata, error) {
	// Choose aggregation function based on database driver
	var tagAggExpr string
	if t.db.DriverName() == SQLITE3 {
		tagAggExpr = sqliteGroupConcat + "(t.tag_name, ',')"
	} else {
		tagAggExpr = postgresStringAgg + "(t.tag_name, ',')"
	}

	q := databaseg.Builder.Select(
		"m.manifest_total_size as size",
		"r.registry_package_type as package_type",
		"m.manifest_created_at as modified_at",
		"m.manifest_schema_version",
		"m.manifest_non_conformant",
		"m.manifest_payload",
		"mt.mt_media_type",
		"m.manifest_digest",
		tagAggExpr+" AS tags",
	).
		From("manifests m").
		LeftJoin("tags t ON m.manifest_id = t.tag_manifest_id"). //
		Join("registries r ON m.manifest_registry_id = r.registry_id").
		Join("media_types mt ON mt.mt_id = m.manifest_media_type_id").
		Where(
			"r.registry_parent_id = ? AND r.registry_name = ? AND m.manifest_image_name = ?",
			parentID, repoKey, image,
		)

	if search != "" {
		digestBytes, err := types.GetDigestBytes(digest.Digest(search))
		if err != nil {
			return nil, fmt.Errorf("invalid digest: %s, error: %w", search, err)
		}
		q = q.Where("m.manifest_digest = ?", digestBytes)
	}

	q = q.GroupBy("m.manifest_total_size, r.registry_package_type, m.manifest_created_at, " +
		"m.manifest_schema_version, m.manifest_non_conformant," +
		" m.manifest_payload, mt.mt_media_type, m.manifest_digest")

	var sortField string
	switch sortByField {
	case "size":
		sortField = "m.manifest_total_size"
	case "updated_at":
		sortField = "m.manifest_created_at"
	}
	if sortField != "" {
		q = q.OrderBy(sortField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset)) //nolint:gosec
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	dst := []*ociVersionMetadataDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return t.mapToOciVersions(dst)
}

func (t tagDao) GetOciTagsInfo(
	ctx context.Context, registryID int64,
	image string, limit int, offset int,
	search string,
) (*[]types.TagInfo, error) {
	q := databaseg.Builder.Select(
		`t.tag_name as name, m.manifest_digest`,
	).
		From("tags t").
		Join("manifests m ON t.tag_manifest_id = m.manifest_id").
		Where(
			"m.manifest_registry_id = ? AND m.manifest_image_name = ?",
			registryID, image,
		)

	if search != "" {
		q = q.Where("t.tag_name LIKE ?", sqlPartialMatch(search))
	}
	q = q.Limit(uint64(limit)).Offset(uint64(offset)) //nolint:gosec

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	var dst []*tagInfoDB
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing get gats info query")
	}
	return t.mapToTagInfoList(dst)
}

func (t tagDao) CountAllTagsByRepoAndImage(
	ctx context.Context, parentID int64,
	repoKey string, image string, search string,
) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("tags").
		Join("registries ON tag_registry_id = registry_id").
		Join("manifests ON tag_manifest_id = manifest_id").
		Where(
			"registry_parent_id = ? AND registry_name = ?"+
				" AND tag_image_name = ?", parentID, repoKey, image,
		)

	if search != "" {
		stmt = stmt.Where("tag_name LIKE ?", sqlPartialMatch(search))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

// GetQuarantineInfoForArtifacts gets quarantine information for artifacts by name and version.
// Joins with images and artifacts tables to match by name, version, and registry_id.
func (t tagDao) GetQuarantineInfoForArtifacts(
	ctx context.Context, artifacts []types.ArtifactIdentifier, parentID int64,
) (map[types.ArtifactIdentifier]*types.QuarantineInfo, error) {
	if len(artifacts) == 0 {
		return make(map[types.ArtifactIdentifier]*types.QuarantineInfo), nil
	}

	// Build OR conditions for each artifact (name, version) pair
	orConditions := sq.Or{}
	for _, art := range artifacts {
		orConditions = append(orConditions, sq.And{
			sq.Eq{"i.image_name": art.Name},
			sq.Eq{"ar.artifact_version": art.Version},
		})
	}

	q := databaseg.Builder.Select(
		"i.image_name, ar.artifact_version, r.registry_name, qp.quarantined_path_reason, qp.quarantined_path_created_at",
	).From("quarantined_paths qp").
		Join("artifacts ar ON qp.quarantined_path_artifact_id = ar.artifact_id").
		Join("images i ON qp.quarantined_path_image_id = i.image_id").
		Join("registries r ON qp.quarantined_path_registry_id = r.registry_id AND r.registry_parent_id = ?", parentID).
		Where(orConditions).
		OrderBy("i.image_name, ar.artifact_version, qp.quarantined_path_created_at DESC")

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert quarantine query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	type quarantineResult struct {
		ImageName    string `db:"image_name"`
		Version      string `db:"artifact_version"`
		Reason       string `db:"quarantined_path_reason"`
		RegistryName string `db:"registry_name"`
		CreatedAt    int64  `db:"quarantined_path_created_at"`
	}

	var results []quarantineResult
	if err = db.SelectContext(ctx, &results, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get quarantine info")
	}

	// Build map with latest quarantine info for each artifact
	quarantineMap := make(map[types.ArtifactIdentifier]*types.QuarantineInfo)
	for _, result := range results {
		artifactKey := types.ArtifactIdentifier{
			Name:         result.ImageName,
			Version:      result.Version,
			RegistryName: result.RegistryName,
		}
		// Since we ordered by created_at DESC, the first entry for each artifact is the latest
		if _, exists := quarantineMap[artifactKey]; !exists {
			quarantineMap[artifactKey] = &types.QuarantineInfo{
				Reason:    result.Reason,
				CreatedAt: result.CreatedAt,
			}
		}
	}

	return quarantineMap, nil
}

func (t tagDao) CountOciVersionByRepoAndImage(
	ctx context.Context, parentID int64,
	repoKey string, image string, search string,
) (int64, error) {
	stmt := databaseg.Builder.Select("COUNT(*)").
		From("manifests").
		Join("registries ON manifest_registry_id = registry_id").
		Where(
			"registry_parent_id = ? AND registry_name = ?"+
				" AND manifest_image_name = ?", parentID, repoKey, image,
		)

	if search != "" {
		digestBytes, err := types.GetDigestBytes(digest.Digest(search))
		if err != nil {
			return 0, fmt.Errorf("invalid digest: %s, error: %w", search, err)
		}
		stmt = stmt.Where("manifest_digest = ?", digestBytes)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, t.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, databaseg.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

func (t tagDao) FindTag(
	ctx context.Context, repoID int64, imageName string,
	name string,
) (*types.Tag, error) {
	stmt := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(tagDB{}), ",")).
		From("tags").
		Where("tag_registry_id = ? AND tag_image_name = ? AND tag_name = ?", repoID, imageName, name)

	db := dbtx.GetAccessor(ctx, t.db)

	dst := new(tagDB)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find tag")
	}

	//TODO: validate for empty row
	return t.mapToTag(ctx, dst)
}

func (t tagDao) GetTagsByManifestID(
	ctx context.Context, manifestID int64,
) (*[]string, error) {
	stmt := databaseg.Builder.
		Select("tag_name").
		From("tags").
		Where("tag_manifest_id = ?", manifestID)

	db := dbtx.GetAccessor(ctx, t.db)

	dst := make([]string, 0)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find manifest tags")
	}

	return &dst, nil
}

func (t tagDao) DeleteTagsByImageName(
	ctx context.Context, registryID int64,
	imageName string,
) (err error) {
	stmt := databaseg.Builder.Delete("tags").
		Where(
			"tag_registry_id = ? AND tag_image_name = ?", registryID, imageName)

	toSQL, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert tag query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, t.db)

	_, err = db.ExecContext(ctx, toSQL, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (t tagDao) mapToInternalTag(ctx context.Context, in *types.Tag) *tagDB {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	return &tagDB{
		ID:         in.ID,
		Name:       in.Name,
		ImageName:  in.ImageName,
		RegistryID: in.RegistryID,
		ManifestID: in.ManifestID,
		CreatedAt:  in.CreatedAt.UnixMilli(),
		UpdatedAt:  in.UpdatedAt.UnixMilli(),
		CreatedBy:  sql.NullInt64{Int64: in.CreatedBy, Valid: true},
		UpdatedBy:  sql.NullInt64{Int64: in.UpdatedBy, Valid: true},
	}
}

func (t tagDao) mapToTag(_ context.Context, dst *tagDB) (*types.Tag, error) {
	createdBy := int64(-1)
	updatedBy := int64(-1)
	if dst.CreatedBy.Valid {
		createdBy = dst.CreatedBy.Int64
	}
	if dst.UpdatedBy.Valid {
		updatedBy = dst.UpdatedBy.Int64
	}
	return &types.Tag{
		ID:         dst.ID,
		Name:       dst.Name,
		ImageName:  dst.ImageName,
		RegistryID: dst.RegistryID,
		ManifestID: dst.ManifestID,
		CreatedAt:  time.UnixMilli(dst.CreatedAt),
		UpdatedAt:  time.UnixMilli(dst.UpdatedAt),
		CreatedBy:  createdBy,
		UpdatedBy:  updatedBy,
	}, nil
}

func (t tagDao) mapToTagList(ctx context.Context, dst []*tagDB) ([]*types.Tag, error) {
	tags := make([]*types.Tag, 0, len(dst))
	for _, d := range dst {
		tag, err := t.mapToTag(ctx, d)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (t tagDao) mapToArtifactMetadataList(
	dst []*artifactMetadataDB,
) (*[]types.ArtifactMetadata, error) {
	artifacts := make([]types.ArtifactMetadata, 0, len(dst))
	for _, d := range dst {
		a, err := t.mapToArtifactMetadata(d)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, *a)
	}
	return &artifacts, nil
}

func (t tagDao) mapToArtifactMetadata(
	dst *artifactMetadataDB,
) (*types.ArtifactMetadata, error) {
	version := dst.Version
	latestVersion := dst.LatestVersion
	if dst.Tag != nil {
		version = *dst.Tag
	} else if dst.PackageType == artifact.PackageTypeDOCKER || dst.PackageType == artifact.PackageTypeHELM {
		parsedDigest, err := types.Digest(strings.ToLower(dst.Version)).Parse()
		if err == nil {
			version = parsedDigest.String()
		}
		parsedLatestVersion, err := types.Digest(strings.ToLower(dst.LatestVersion)).Parse()
		if err == nil {
			latestVersion = parsedLatestVersion.String()
		}
	}

	artifactMetadata := &types.ArtifactMetadata{
		Name:             dst.Name,
		RepoName:         dst.RepoName,
		DownloadCount:    dst.DownloadCount,
		PackageType:      dst.PackageType,
		LatestVersion:    latestVersion,
		Labels:           util.StringToArr(dst.Labels.String),
		CreatedAt:        time.UnixMilli(dst.CreatedAt),
		ModifiedAt:       time.UnixMilli(dst.ModifiedAt),
		Version:          version,
		IsQuarantined:    dst.IsQuarantined,
		QuarantineReason: dst.QuarantineReason,
		ArtifactType:     dst.ArtifactType,
	}

	if dst.Tags != nil {
		artifactMetadata.Tags = strings.Split(*dst.Tags, ",")
	} else {
		artifactMetadata.Tags = []string{}
	}

	return artifactMetadata, nil
}

func (t tagDao) mapToTagMetadataList(
	ctx context.Context,
	dst []*tagMetadataDB,
) (*[]types.OciVersionMetadata, error) {
	ociVersion := make([]types.OciVersionMetadata, 0, len(dst))
	for _, d := range dst {
		tag, err := t.mapToTagMetadata(ctx, d)
		if err != nil {
			return nil, err
		}
		ociVersion = append(ociVersion, *tag)
	}
	return &ociVersion, nil
}

func (t tagDao) mapToTagInfoList(
	tagInfoDB []*tagInfoDB,
) (*[]types.TagInfo, error) {
	tagInfos := []types.TagInfo{}
	for _, d := range tagInfoDB {
		tagInfo := &types.TagInfo{
			Name: d.Name,
		}
		if d.Digest != nil {
			dgst, err := types.Digest(util.GetHexEncodedString(d.Digest)).Parse()
			if err != nil {
				return nil, fmt.Errorf("invalid digest: %s, error: %w", util.GetHexEncodedString(d.Digest), err)
			}
			tagInfo.Digest = string(dgst)
		}
		tagInfos = append(tagInfos, *tagInfo)
	}
	return &tagInfos, nil
}

func (t tagDao) mapToOciVersions(
	dst []*ociVersionMetadataDB,
) (*[]types.OciVersionMetadata, error) {
	ociVersions := make([]types.OciVersionMetadata, 0, len(dst))
	for _, d := range dst {
		tag, err := t.mapToOciVersion(d)
		if err != nil {
			return nil, err
		}
		ociVersions = append(ociVersions, *tag)
	}
	return &ociVersions, nil
}

func (t tagDao) mapToTagMetadata(
	_ context.Context,
	dst *tagMetadataDB,
) (*types.OciVersionMetadata, error) {
	tagMetadata := &types.OciVersionMetadata{
		Name:          dst.Name,
		Size:          dst.Size,
		PackageType:   dst.PackageType,
		DigestCount:   dst.DigestCount,
		ModifiedAt:    time.UnixMilli(dst.ModifiedAt),
		SchemaVersion: dst.SchemaVersion,
		NonConformant: dst.NonConformant,
		MediaType:     dst.MediaType,
		Payload:       dst.Payload,
		DownloadCount: dst.DownloadCount,
	}
	if dst.Digest != nil {
		dgst := types.Digest(util.GetHexEncodedString(dst.Digest))
		tagMetadata.Digest = string(dgst)
	}

	return tagMetadata, nil
}

func (t tagDao) mapToOciVersion(
	dst *ociVersionMetadataDB,
) (*types.OciVersionMetadata, error) {
	ociVersion := &types.OciVersionMetadata{
		Size:          dst.Size,
		PackageType:   dst.PackageType,
		DigestCount:   dst.DigestCount,
		ModifiedAt:    time.UnixMilli(dst.ModifiedAt),
		SchemaVersion: dst.SchemaVersion,
		NonConformant: dst.NonConformant,
		MediaType:     dst.MediaType,
		Payload:       dst.Payload,
	}
	if dst.Tags != nil {
		ociVersion.Tags = strings.Split(*dst.Tags, ",")
	} else {
		ociVersion.Tags = []string{}
	}

	dgst := types.Digest(util.GetHexEncodedString(dst.Digest))
	ociVersion.Digest = string(dgst)
	versionName, err := dgst.Parse()
	if err != nil {
		return nil, fmt.Errorf("invalid digest: %s, error: %w", util.GetHexEncodedString(dst.Digest), err)
	}
	ociVersion.Name = string(versionName)

	return ociVersion, nil
}

func (t tagDao) mapToTagDetail(
	_ context.Context,
	dst *tagDetailDB,
) (*types.TagDetail, error) {
	return &types.TagDetail{
		ID:            dst.ID,
		Name:          dst.Name,
		ImageName:     dst.ImageName,
		Size:          dst.Size,
		CreatedAt:     time.UnixMilli(dst.CreatedAt),
		UpdatedAt:     time.UnixMilli(dst.UpdatedAt),
		DownloadCount: dst.DownloadCount,
	}, nil
}

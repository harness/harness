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
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	gitness_store "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type UpstreamproxyDao struct {
	registryDao store.RegistryRepository
	db          *sqlx.DB
}

func NewUpstreamproxyDao(db *sqlx.DB, registryDao store.RegistryRepository) store.UpstreamProxyConfigRepository {
	return &UpstreamproxyDao{
		registryDao: registryDao,
		db:          db,
	}
}

// upstreamProxyConfigDB holds the record of an upstream_proxy_config in DB.
type upstreamProxyConfigDB struct {
	ID               int64          `db:"upstream_proxy_config_id"`
	RegistryID       int64          `db:"upstream_proxy_config_registry_id"`
	Source           string         `db:"upstream_proxy_config_source"`
	URL              string         `db:"upstream_proxy_config_url"`
	AuthType         string         `db:"upstream_proxy_config_auth_type"`
	UserName         string         `db:"upstream_proxy_config_user_name"`
	SecretIdentifier sql.NullString `db:"upstream_proxy_config_secret_identifier"`
	SecretSpaceID    sql.NullInt32  `db:"upstream_proxy_config_secret_space_id"`
	Token            string         `db:"upstream_proxy_config_token"`
	CreatedAt        int64          `db:"upstream_proxy_config_created_at"`
	UpdatedAt        int64          `db:"upstream_proxy_config_updated_at"`
	CreatedBy        int64          `db:"upstream_proxy_config_created_by"`
	UpdatedBy        int64          `db:"upstream_proxy_config_updated_by"`
}

type upstreamProxyDB struct {
	ID               int64                `db:"id"`
	RegistryID       int64                `db:"registry_id"`
	RepoKey          string               `db:"repo_key"`
	ParentID         string               `db:"parent_id"`
	PackageType      artifact.PackageType `db:"package_type"`
	AllowedPattern   sql.NullString       `db:"allowed_pattern"`
	BlockedPattern   sql.NullString       `db:"blocked_pattern"`
	Source           string               `db:"source"`
	RepoURL          string               `db:"repo_url"`
	RepoAuthType     string               `db:"repo_auth_type"`
	UserName         string               `db:"user_name"`
	SecretIdentifier sql.NullString       `db:"secret_identifier"`
	SecretSpaceID    sql.NullInt32        `db:"secret_space_id"`
	Token            string               `db:"token"`
	CreatedAt        int64                `db:"created_at"`
	UpdatedAt        int64                `db:"updated_at"`
	CreatedBy        sql.NullInt64        `db:"created_by"`
	UpdatedBy        sql.NullInt64        `db:"updated_by"`
}

func getUpstreamProxyQuery() squirrel.SelectBuilder {
	return databaseg.Builder.Select(
		" u.upstream_proxy_config_id as id," +
			" r.registry_id as registry_id," +
			" r.registry_name as repo_key," +
			" r.registry_parent_id as parent_id," +
			" r.registry_package_type as package_type," +
			" r.registry_allowed_pattern as allowed_pattern," +
			" r.registry_blocked_pattern as blocked_pattern," +
			" u.upstream_proxy_config_url as repo_url," +
			" u.upstream_proxy_config_source as source," +
			" u.upstream_proxy_config_auth_type as repo_auth_type," +
			" u.upstream_proxy_config_user_name as user_name," +
			" u.upstream_proxy_config_secret_identifier as secret_identifier," +
			" u.upstream_proxy_config_secret_space_id as secret_space_id," +
			" u.upstream_proxy_config_token as token," +
			" r.registry_created_at as created_at," +
			" r.registry_updated_at as updated_at ").
		From("registries r ").
		LeftJoin("upstream_proxy_configs u ON r.registry_id = u.upstream_proxy_config_registry_id ")
}

func (r UpstreamproxyDao) Get(ctx context.Context, id int64) (upstreamProxy *types.UpstreamProxy, err error) {
	q := getUpstreamProxyQuery()
	q = q.Where("r.registry_id = ? AND r.registry_type = 'UPSTREAM'", id)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := new(upstreamProxyDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}

	return r.mapToUpstreamProxy(ctx, dst)
}

func (r UpstreamproxyDao) GetByRegistryIdentifier(
	ctx context.Context,
	parentID int64,
	repoKey string,
) (upstreamProxy *types.UpstreamProxy, err error) {
	q := getUpstreamProxyQuery()
	q = q.Where("r.registry_parent_id = ? AND r.registry_name = ? AND r.registry_type = 'UPSTREAM'",
		parentID, repoKey)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := new(upstreamProxyDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}

	return r.mapToUpstreamProxy(ctx, dst)
}

func (r UpstreamproxyDao) GetByParentID(ctx context.Context, parentID string) (
	upstreamProxies *[]types.UpstreamProxy, err error) {
	q := getUpstreamProxyQuery()
	q = q.Where("r.registry_parent_id = ? AND r.registry_type = 'UPSTREAM'",
		parentID)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := []*upstreamProxyDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}

	return r.mapToUpstreamProxyList(ctx, dst)
}

func (r UpstreamproxyDao) Create(
	ctx context.Context,
	upstreamproxyRecord *types.UpstreamProxyConfig,
) (id int64, err error) {
	const sqlQuery = `
		INSERT INTO upstream_proxy_configs ( 
			upstream_proxy_config_registry_id
			,upstream_proxy_config_source
			,upstream_proxy_config_url
			,upstream_proxy_config_auth_type
			,upstream_proxy_config_user_name
			,upstream_proxy_config_secret_identifier
			,upstream_proxy_config_secret_space_id
			,upstream_proxy_config_token
			,upstream_proxy_config_created_at
			,upstream_proxy_config_updated_at
			,upstream_proxy_config_created_by
			,upstream_proxy_config_updated_by
		) VALUES (
			:upstream_proxy_config_registry_id
			,:upstream_proxy_config_source
			,:upstream_proxy_config_url
			,:upstream_proxy_config_auth_type
			,:upstream_proxy_config_user_name
			,:upstream_proxy_config_secret_identifier
			,:upstream_proxy_config_secret_space_id
			,:upstream_proxy_config_token
			,:upstream_proxy_config_created_at
			,:upstream_proxy_config_updated_at
			,:upstream_proxy_config_created_by
			,:upstream_proxy_config_updated_by
		) RETURNING upstream_proxy_config_registry_id`

	db := dbtx.GetAccessor(ctx, r.db)
	query, arg, err := db.BindNamed(sqlQuery, r.mapToInternalUpstreamProxy(ctx, upstreamproxyRecord))
	if err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx,
			err, "Failed to bind upstream proxy object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&upstreamproxyRecord.ID); err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return upstreamproxyRecord.ID, nil
}

func (r UpstreamproxyDao) Delete(ctx context.Context, parentID int64, repoKey string) (err error) {
	stmt := databaseg.Builder.Delete("upstream_proxy_configs").
		Where("upstream_proxy_config_registry_id in (SELECT registry_id from registries"+
			" WHERE registry_parent_id = ? AND registry_name = ?)", parentID, repoKey)

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

func (r UpstreamproxyDao) Update(
	ctx context.Context,
	upstreamProxyRecord *types.UpstreamProxyConfig,
) (err error) {
	var sqlQuery = " UPDATE upstream_proxy_configs SET " +
		util.GetSetDBKeys(upstreamProxyConfigDB{}, "upstream_proxy_config_id") +
		" WHERE upstream_proxy_config_id = :upstream_proxy_config_id "

	upstreamProxy := r.mapToInternalUpstreamProxy(ctx, upstreamProxyRecord)

	// update Version (used for optimistic locking) and Updated time
	upstreamProxy.UpdatedAt = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, r.db)

	query, arg, err := db.BindNamed(sqlQuery, upstreamProxy)
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

func (r UpstreamproxyDao) GetAll(
	ctx context.Context, parentID int64,
	packageTypes []string, sortByField string, sortByOrder string, limit int,
	offset int, search string,
) (upstreamProxies *[]types.UpstreamProxy, err error) {
	q := getUpstreamProxyQuery()
	q = q.Where("r.registry_parent_id = ? AND r.registry_type = 'UPSTREAM'",
		parentID)

	if search != "" {
		q = q.Where(" r.registry_name LIKE ?", sqlPartialMatch(search))
	}

	if len(packageTypes) > 0 {
		q = q.Where(" AND r.registry_package_type in ? ", packageTypes)
	}

	q = q.OrderBy(" r.registry_" + sortByField + " " + sortByOrder).Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	dst := []*upstreamProxyDB{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get tag detail")
	}
	return r.mapToUpstreamProxyList(ctx, dst)
}

func (r UpstreamproxyDao) CountAll(
	ctx context.Context, parentID string,
	packageTypes []string, search string,
) (count int64, err error) {
	q := databaseg.Builder.Select(" COUNT(*) ").
		From(" registries r").
		LeftJoin(" upstream_proxy_configs u ON r.registry_id = u.upstream_proxy_config_registry_id ").
		Where("r.registry_parent_id = ? AND r.registry_type = 'UPSTREAM'",
			parentID)

	if search != "" {
		q = q.Where(" r.registry_name LIKE '%" + search + "%' ")
	}

	if len(packageTypes) > 0 {
		q = q.Where(" AND r.registry_package_type in ? ", packageTypes)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, r.db)

	var total int64
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&total); err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get upstream proxy count")
	}
	return total, nil
}

func (r UpstreamproxyDao) mapToInternalUpstreamProxy(
	ctx context.Context,
	in *types.UpstreamProxyConfig,
) *upstreamProxyConfigDB {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	return &upstreamProxyConfigDB{
		ID:               in.ID,
		RegistryID:       in.RegistryID,
		Source:           in.Source,
		URL:              in.URL,
		AuthType:         in.AuthType,
		UserName:         in.UserName,
		SecretIdentifier: util.GetEmptySQLString(in.SecretIdentifier),
		SecretSpaceID:    util.GetEmptySQLInt32(in.SecretSpaceID),
		Token:            in.Token,
		CreatedAt:        in.CreatedAt.UnixMilli(),
		UpdatedAt:        in.UpdatedAt.UnixMilli(),
		CreatedBy:        in.CreatedBy,
		UpdatedBy:        in.UpdatedBy,
	}
}

func (r UpstreamproxyDao) mapToUpstreamProxy(
	_ context.Context,
	dst *upstreamProxyDB,
) (*types.UpstreamProxy, error) {
	createdBy := int64(-1)
	updatedBy := int64(-1)
	if dst.CreatedBy.Valid {
		createdBy = dst.CreatedBy.Int64
	}
	if dst.UpdatedBy.Valid {
		updatedBy = dst.UpdatedBy.Int64
	}
	return &types.UpstreamProxy{
		ID:               dst.ID,
		RegistryID:       dst.RegistryID,
		RepoKey:          dst.RepoKey,
		ParentID:         dst.ParentID,
		PackageType:      dst.PackageType,
		AllowedPattern:   util.StringToArr(dst.AllowedPattern.String),
		BlockedPattern:   util.StringToArr(dst.BlockedPattern.String),
		Source:           dst.Source,
		RepoURL:          dst.RepoURL,
		RepoAuthType:     dst.RepoAuthType,
		UserName:         dst.UserName,
		SecretIdentifier: dst.SecretIdentifier,
		SecretSpaceID:    dst.SecretSpaceID,
		Token:            dst.Token,
		CreatedAt:        time.UnixMilli(dst.CreatedAt),
		UpdatedAt:        time.UnixMilli(dst.UpdatedAt),
		CreatedBy:        createdBy,
		UpdatedBy:        updatedBy,
	}, nil
}

func (r UpstreamproxyDao) mapToUpstreamProxyList(
	ctx context.Context,
	dst []*upstreamProxyDB,
) (*[]types.UpstreamProxy, error) {
	upstreamProxies := make([]types.UpstreamProxy, 0, len(dst))
	for _, d := range dst {
		upstreamProxy, err := r.mapToUpstreamProxy(ctx, d)
		if err != nil {
			return nil, err
		}
		upstreamProxies = append(upstreamProxies, *upstreamProxy)
	}
	return &upstreamProxies, nil
}

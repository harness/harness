// Copyright 2023 Harness, Inc.
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
	"strings"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	gitspaceConfigInsertColumns = `
		gconf_uid,
		gconf_display_name,
		gconf_ide,
		gconf_infra_provider_resource_id,
		gconf_code_auth_type,
		gconf_code_auth_id,
		gconf_code_repo_type,
		gconf_code_repo_is_private,
		gconf_code_repo_url,
		gconf_devcontainer_path,
		gconf_branch,
		gconf_user_uid,
		gconf_space_id,
		gconf_created,
		gconf_updated,
        gconf_is_deleted,
        gconf_code_repo_ref
	`
	gitspaceConfigsTable        = `gitspace_configs`
	ReturningClause             = "RETURNING "
	gitspaceConfigSelectColumns = "gconf_id," + gitspaceConfigInsertColumns
)

type gitspaceConfig struct {
	ID                      int64                     `db:"gconf_id"`
	Identifier              string                    `db:"gconf_uid"`
	Name                    string                    `db:"gconf_display_name"`
	IDE                     enum.IDEType              `db:"gconf_ide"`
	InfraProviderResourceID int64                     `db:"gconf_infra_provider_resource_id"`
	CodeAuthType            string                    `db:"gconf_code_auth_type"`
	CodeRepoRef             null.String               `db:"gconf_code_repo_ref"`
	CodeAuthID              string                    `db:"gconf_code_auth_id"`
	CodeRepoType            enum.GitspaceCodeRepoType `db:"gconf_code_repo_type"`
	CodeRepoIsPrivate       bool                      `db:"gconf_code_repo_is_private"`
	CodeRepoURL             string                    `db:"gconf_code_repo_url"`
	DevcontainerPath        null.String               `db:"gconf_devcontainer_path"`
	Branch                  string                    `db:"gconf_branch"`
	UserUID                 string                    `db:"gconf_user_uid"`
	SpaceID                 int64                     `db:"gconf_space_id"`
	Created                 int64                     `db:"gconf_created"`
	Updated                 int64                     `db:"gconf_updated"`
	IsDeleted               bool                      `db:"gconf_is_deleted"`
}

var _ store.GitspaceConfigStore = (*gitspaceConfigStore)(nil)

// NewGitspaceConfigStore returns a new GitspaceConfigStore.
func NewGitspaceConfigStore(db *sqlx.DB) store.GitspaceConfigStore {
	return &gitspaceConfigStore{
		db: db,
	}
}

type gitspaceConfigStore struct {
	db *sqlx.DB
}

func (s gitspaceConfigStore) Count(ctx context.Context, filter *types.GitspaceFilter) (int64, error) {
	db := dbtx.GetAccessor(ctx, s.db)
	countStmt := database.Builder.
		Select("COUNT(*)").
		From(gitspaceConfigsTable).
		Where(squirrel.Eq{"gconf_is_deleted": false}).
		Where(squirrel.Eq{"gconf_user_uid": filter.UserID}).
		Where(squirrel.Eq{"gconf_space_id": filter.SpaceIDs})
	sql, args, err := countStmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing custom count query")
	}
	return count, nil
}

func (s gitspaceConfigStore) Find(ctx context.Context, id int64) (*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where("gconf_id = $1", id). //nolint:goconst
		Where("gconf_is_deleted = $2", false)
	dst := new(gitspaceConfig)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace config")
	}
	return s.mapToGitspaceConfig(ctx, dst)
}

func (s gitspaceConfigStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where("LOWER(gconf_uid) = $1", strings.ToLower(identifier)).
		Where("gconf_space_id = $2", spaceID).
		Where("gconf_is_deleted = $3", false)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	dst := new(gitspaceConfig)
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace config")
	}
	return s.mapToGitspaceConfig(ctx, dst)
}

func (s gitspaceConfigStore) Create(ctx context.Context, gitspaceConfig *types.GitspaceConfig) error {
	stmt := database.Builder.
		Insert(gitspaceConfigsTable).
		Columns(gitspaceConfigInsertColumns).
		Values(
			gitspaceConfig.Identifier,
			gitspaceConfig.Name,
			gitspaceConfig.IDE,
			gitspaceConfig.InfraProviderResourceID,
			gitspaceConfig.CodeAuthType,
			gitspaceConfig.CodeAuthID,
			gitspaceConfig.CodeRepoType,
			gitspaceConfig.CodeRepoIsPrivate,
			gitspaceConfig.CodeRepoURL,
			gitspaceConfig.DevcontainerPath,
			gitspaceConfig.Branch,
			gitspaceConfig.UserID,
			gitspaceConfig.SpaceID,
			gitspaceConfig.Created,
			gitspaceConfig.Updated,
			gitspaceConfig.IsDeleted,
			gitspaceConfig.CodeRepoRef,
		).
		Suffix(ReturningClause + "gconf_id")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&gitspaceConfig.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "gitspace config query failed")
	}
	return nil
}

func (s gitspaceConfigStore) Update(ctx context.Context,
	gitspaceConfig *types.GitspaceConfig) error {
	dbGitspaceConfig := mapToInternalGitspaceConfig(gitspaceConfig)
	stmt := database.Builder.
		Update(gitspaceConfigsTable).
		Set("gconf_display_name", dbGitspaceConfig.Name).
		Set("gconf_ide", dbGitspaceConfig.IDE).
		Set("gconf_updated", dbGitspaceConfig.Updated).
		Set("gconf_infra_provider_resource_id", dbGitspaceConfig.InfraProviderResourceID).
		Set("gconf_is_deleted", dbGitspaceConfig.IsDeleted).
		Where("gconf_id = $6", gitspaceConfig.ID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update gitspace config")
	}
	return nil
}

func mapToInternalGitspaceConfig(config *types.GitspaceConfig) *gitspaceConfig {
	return &gitspaceConfig{
		ID:                      config.ID,
		Identifier:              config.Identifier,
		Name:                    config.Name,
		IDE:                     config.IDE,
		InfraProviderResourceID: config.InfraProviderResourceID,
		CodeAuthType:            config.CodeAuthType,
		CodeAuthID:              config.CodeAuthID,
		CodeRepoIsPrivate:       config.CodeRepoIsPrivate,
		CodeRepoType:            config.CodeRepoType,
		CodeRepoRef:             null.StringFromPtr(config.CodeRepoRef),
		CodeRepoURL:             config.CodeRepoURL,
		DevcontainerPath:        null.StringFromPtr(config.DevcontainerPath),
		Branch:                  config.Branch,
		UserUID:                 config.UserID,
		SpaceID:                 config.SpaceID,
		IsDeleted:               config.IsDeleted,
		Created:                 config.Created,
		Updated:                 config.Updated,
	}
}

func (s gitspaceConfigStore) List(ctx context.Context, filter *types.GitspaceFilter) ([]*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where(squirrel.Eq{"gconf_is_deleted": false}).
		Where(squirrel.Eq{"gconf_user_uid": filter.UserID}).
		Where(squirrel.Eq{"gconf_space_id": filter.SpaceIDs})

	queryFilter := filter.QueryFilter
	stmt = stmt.Limit(database.Limit(queryFilter.Size))
	stmt = stmt.Offset(database.Offset(queryFilter.Page, queryFilter.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	var dst []*gitspaceConfig
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return s.mapToGitspaceConfigs(ctx, dst)
}

func (s gitspaceConfigStore) ListAll(
	ctx context.Context,
	userUID string,
) ([]*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where(squirrel.Eq{"gconf_is_deleted": false}).
		Where(squirrel.Eq{"gconf_user_uid": userUID})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	var dst []*gitspaceConfig
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return s.mapToGitspaceConfigs(ctx, dst)
}

func (s *gitspaceConfigStore) mapToGitspaceConfig(
	_ context.Context,
	in *gitspaceConfig,
) (*types.GitspaceConfig, error) {
	var res = &types.GitspaceConfig{
		ID:                      in.ID,
		Identifier:              in.Identifier,
		Name:                    in.Name,
		InfraProviderResourceID: in.InfraProviderResourceID,
		IDE:                     in.IDE,
		CodeRepoType:            in.CodeRepoType,
		CodeRepoRef:             in.CodeRepoRef.Ptr(),
		CodeRepoURL:             in.CodeRepoURL,
		Branch:                  in.Branch,
		DevcontainerPath:        in.DevcontainerPath.Ptr(),
		UserID:                  in.UserUID,
		SpaceID:                 in.SpaceID,
		CodeAuthType:            in.CodeAuthType,
		CodeAuthID:              in.CodeAuthID,
		CodeRepoIsPrivate:       in.CodeRepoIsPrivate,
		Created:                 in.Created,
		Updated:                 in.Updated,
	}
	return res, nil
}

func (s *gitspaceConfigStore) mapToGitspaceConfigs(
	ctx context.Context,
	configs []*gitspaceConfig,
) ([]*types.GitspaceConfig, error) {
	var err error
	res := make([]*types.GitspaceConfig, len(configs))
	for i := range configs {
		res[i], err = s.mapToGitspaceConfig(ctx, configs[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

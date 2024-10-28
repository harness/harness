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
	"fmt"
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
        gconf_code_repo_ref,
		gconf_ssh_token_identifier,
        gconf_created_by,
		gconf_is_marked_for_deletion
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
	// TODO: migrate to principal int64 id to use principal cache and consistent with Harness code.
	UserUID             string   `db:"gconf_user_uid"`
	SpaceID             int64    `db:"gconf_space_id"`
	Created             int64    `db:"gconf_created"`
	Updated             int64    `db:"gconf_updated"`
	IsDeleted           bool     `db:"gconf_is_deleted"`
	SSHTokenIdentifier  string   `db:"gconf_ssh_token_identifier"`
	CreatedBy           null.Int `db:"gconf_created_by"`
	IsMarkedForDeletion bool     `db:"gconf_is_marked_for_deletion"`
}

var _ store.GitspaceConfigStore = (*gitspaceConfigStore)(nil)

// NewGitspaceConfigStore returns a new GitspaceConfigStore.
func NewGitspaceConfigStore(
	db *sqlx.DB,
	pCache store.PrincipalInfoCache,
	rCache store.InfraProviderResourceCache,
) store.GitspaceConfigStore {
	return &gitspaceConfigStore{
		db:     db,
		pCache: pCache,
		rCache: rCache,
	}
}

type gitspaceConfigStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
	rCache store.InfraProviderResourceCache
}

func (s gitspaceConfigStore) Count(ctx context.Context, filter *types.GitspaceFilter) (int64, error) {
	db := dbtx.GetAccessor(ctx, s.db)
	countStmt := database.Builder.
		Select("COUNT(*)").
		From(gitspaceConfigsTable)

	if !filter.IncludeDeleted {
		countStmt = countStmt.Where(squirrel.Eq{"gconf_is_deleted": false})
	}
	if filter.UserID != "" {
		countStmt = countStmt.Where(squirrel.Eq{"gconf_user_uid": filter.UserID})
	}
	if filter.SpaceIDs != nil {
		countStmt = countStmt.Where(squirrel.Eq{"gconf_space_id": filter.SpaceIDs})
	}

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

func (s gitspaceConfigStore) Find(ctx context.Context, id int64, includeDeleted bool) (*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where("gconf_id = ?", id) //nolint:goconst

	if !includeDeleted {
		stmt = stmt.Where("gconf_is_deleted = ?", false)
	}

	dst := new(gitspaceConfig)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace config for %d", id)
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
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace config for %s", identifier)
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
			gitspaceConfig.InfraProviderResource.ID,
			gitspaceConfig.CodeRepo.AuthType,
			gitspaceConfig.CodeRepo.AuthID,
			gitspaceConfig.CodeRepo.Type,
			gitspaceConfig.CodeRepo.IsPrivate,
			gitspaceConfig.CodeRepo.URL,
			gitspaceConfig.DevcontainerPath,
			gitspaceConfig.Branch,
			gitspaceConfig.GitspaceUser.Identifier,
			gitspaceConfig.SpaceID,
			gitspaceConfig.Created,
			gitspaceConfig.Updated,
			gitspaceConfig.IsDeleted,
			gitspaceConfig.CodeRepo.Ref,
			gitspaceConfig.SSHTokenIdentifier,
			gitspaceConfig.GitspaceUser.ID,
			gitspaceConfig.IsMarkedForDeletion,
		).
		Suffix(ReturningClause + "gconf_id")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&gitspaceConfig.ID); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "gitspace config create query failed for %s", gitspaceConfig.Identifier)
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
		Set("gconf_is_marked_for_deletion", dbGitspaceConfig.IsMarkedForDeletion).
		Where("gconf_id = ?", gitspaceConfig.ID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "Failed to update gitspace config for %s", gitspaceConfig.Identifier)
	}
	return nil
}

func mapToInternalGitspaceConfig(config *types.GitspaceConfig) *gitspaceConfig {
	return &gitspaceConfig{
		ID:                      config.ID,
		Identifier:              config.Identifier,
		Name:                    config.Name,
		IDE:                     config.IDE,
		InfraProviderResourceID: config.InfraProviderResource.ID,
		CodeAuthType:            config.CodeRepo.AuthType,
		CodeAuthID:              config.CodeRepo.AuthID,
		CodeRepoIsPrivate:       config.CodeRepo.IsPrivate,
		CodeRepoType:            config.CodeRepo.Type,
		CodeRepoRef:             null.StringFromPtr(config.CodeRepo.Ref),
		CodeRepoURL:             config.CodeRepo.URL,
		DevcontainerPath:        null.StringFromPtr(config.DevcontainerPath),
		Branch:                  config.Branch,
		UserUID:                 config.GitspaceUser.Identifier,
		SpaceID:                 config.SpaceID,
		IsDeleted:               config.IsDeleted,
		IsMarkedForDeletion:     config.IsMarkedForDeletion,
		Created:                 config.Created,
		Updated:                 config.Updated,
		SSHTokenIdentifier:      config.SSHTokenIdentifier,
		CreatedBy:               null.IntFromPtr(config.GitspaceUser.ID),
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
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing gitspace config list space query")
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
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing gitspace config list all query")
	}
	return s.mapToGitspaceConfigs(ctx, dst)
}

func (s gitspaceConfigStore) FindAll(ctx context.Context, ids []int64) ([]*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where(squirrel.Eq{"gconf_id": ids}). //nolint:goconst
		Where("gconf_is_deleted = ?", false)
	var dst []*gitspaceConfig
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find all gitspace configs for %v", ids)
	}
	return s.mapToGitspaceConfigs(ctx, dst)
}

func (s *gitspaceConfigStore) mapToGitspaceConfig(
	ctx context.Context,
	in *gitspaceConfig,
) (*types.GitspaceConfig, error) {
	codeRepo := types.CodeRepo{
		URL:              in.CodeRepoURL,
		Ref:              in.CodeRepoRef.Ptr(),
		Type:             in.CodeRepoType,
		Branch:           in.Branch,
		DevcontainerPath: in.DevcontainerPath.Ptr(),
		IsPrivate:        in.CodeRepoIsPrivate,
		AuthType:         in.CodeAuthType,
		AuthID:           in.CodeAuthID,
	}
	var res = &types.GitspaceConfig{
		ID:                  in.ID,
		Identifier:          in.Identifier,
		Name:                in.Name,
		IDE:                 in.IDE,
		SpaceID:             in.SpaceID,
		Created:             in.Created,
		Updated:             in.Updated,
		SSHTokenIdentifier:  in.SSHTokenIdentifier,
		IsMarkedForDeletion: in.IsMarkedForDeletion,
		IsDeleted:           in.IsDeleted,
		CodeRepo:            codeRepo,
		GitspaceUser: types.GitspaceUser{
			ID:         in.CreatedBy.Ptr(),
			Identifier: in.UserUID},
	}
	if res.GitspaceUser.ID != nil {
		author, _ := s.pCache.Get(ctx, *res.GitspaceUser.ID)
		if author != nil {
			res.GitspaceUser.DisplayName = author.DisplayName
			res.GitspaceUser.Email = author.Email
		}
	}

	if resource, err := s.rCache.Get(ctx, in.InfraProviderResourceID); err == nil {
		res.InfraProviderResource = *resource
	} else {
		return nil, fmt.Errorf("couldn't set resource to the config in DB: %s", in.Identifier)
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

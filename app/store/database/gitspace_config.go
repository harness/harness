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
	"github.com/rs/zerolog/log"
)

const (
	gitspaceConfigsTable        = `gitspace_configs`
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
		gconf_is_marked_for_deletion,
		gconf_is_marked_for_reset,
        gconf_is_marked_for_infra_reset`
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
	UserUID               string   `db:"gconf_user_uid"`
	SpaceID               int64    `db:"gconf_space_id"`
	Created               int64    `db:"gconf_created"`
	Updated               int64    `db:"gconf_updated"`
	IsDeleted             bool     `db:"gconf_is_deleted"`
	SSHTokenIdentifier    string   `db:"gconf_ssh_token_identifier"`
	CreatedBy             null.Int `db:"gconf_created_by"`
	IsMarkedForDeletion   bool     `db:"gconf_is_marked_for_deletion"`
	IsMarkedForReset      bool     `db:"gconf_is_marked_for_reset"`
	IsMarkedForInfraReset bool     `db:"gconf_is_marked_for_infra_reset"`
}

type gitspaceConfigWithLatestInstance struct {
	gitspaceConfig
	// gitspace instance information
	gitspaceInstance
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
	gitsSelectStr := getLatestInstanceQuery()
	countStmt := squirrel.Select("COUNT(*)").
		From(gitspaceConfigsTable).
		LeftJoin("(" + gitsSelectStr +
			") AS gits ON gitspace_configs.gconf_id = gits.gits_gitspace_config_id AND gits.rn = 1").
		PlaceholderFormat(squirrel.Dollar)

	countStmt = addGitspaceFilter(countStmt, filter)
	countStmt = addGitspaceQueryFilter(countStmt, filter.QueryFilter)

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
	return s.mapDBToGitspaceConfig(ctx, dst)
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
	return s.mapDBToGitspaceConfig(ctx, dst)
}

func (s gitspaceConfigStore) FindAllByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifiers []string,
) ([]types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where(squirrel.Eq{"gconf_uid": identifiers}).
		Where(squirrel.Eq{"gconf_space_id": spaceID})

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	var dst []*gitspaceConfig
	db := dbtx.GetAccessor(ctx, s.db)
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace configs, identifiers: %s",
			identifiers)
	}
	gitspaceConfigs, err := s.mapToGitspaceConfigs(ctx, dst)
	if err != nil {
		return nil, err
	}

	return sortBy(gitspaceConfigs, identifiers), nil
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
			gitspaceConfig.IsMarkedForReset,
			gitspaceConfig.IsMarkedForInfraReset,
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
		Set("gconf_is_marked_for_reset", dbGitspaceConfig.IsMarkedForReset).
		Set("gconf_is_marked_for_infra_reset", dbGitspaceConfig.IsMarkedForInfraReset).
		Set("gconf_ssh_token_identifier", dbGitspaceConfig.SSHTokenIdentifier).
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
		IsMarkedForReset:        config.IsMarkedForReset,
		IsMarkedForInfraReset:   config.IsMarkedForInfraReset,
		Created:                 config.Created,
		Updated:                 config.Updated,
		SSHTokenIdentifier:      config.SSHTokenIdentifier,
		CreatedBy:               null.IntFromPtr(config.GitspaceUser.ID),
	}
}

// ListWithLatestInstance returns gitspace configs for the given filter with the latest gitspace instance information.
func (s gitspaceConfigStore) ListWithLatestInstance(
	ctx context.Context,
	filter *types.GitspaceFilter,
) ([]*types.GitspaceConfig, error) {
	gitsSelectStr := getLatestInstanceQuery()
	stmt := squirrel.Select(
		gitspaceConfigSelectColumns,
		gitspaceInstanceSelectColumns).
		From(gitspaceConfigsTable).
		LeftJoin("(" + gitsSelectStr +
			") AS gits ON gitspace_configs.gconf_id = gits.gits_gitspace_config_id AND gits.rn = 1").
		PlaceholderFormat(squirrel.Dollar)

	stmt = addGitspaceFilter(stmt, filter)
	stmt = addGitspaceQueryFilter(stmt, filter.QueryFilter)
	stmt = addOrderBy(stmt, filter)
	stmt = stmt.Limit(database.Limit(filter.QueryFilter.Size))
	stmt = stmt.Offset(database.Offset(filter.QueryFilter.Page, filter.QueryFilter.Size))

	sql, args, err := stmt.ToSql()

	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	var dst []*gitspaceConfigWithLatestInstance
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing list gitspace config query")
	}
	return s.ToGitspaceConfigs(ctx, dst)
}

func getLatestInstanceQuery() string {
	return fmt.Sprintf("SELECT %s, %s FROM %s",
		gitspaceInstanceSelectColumns,
		"ROW_NUMBER() OVER (PARTITION BY gits_gitspace_config_id ORDER BY gits_created DESC) AS rn",
		gitspaceInstanceTable,
	)
}

func addGitspaceFilter(stmt squirrel.SelectBuilder, filter *types.GitspaceFilter) squirrel.SelectBuilder {
	stmt = stmt.Where(squirrel.Gt{"gits_id": 0})

	if filter.Deleted != nil {
		stmt = stmt.Where(squirrel.Eq{"gconf_is_deleted": filter.Deleted})
	}

	if filter.Owner == enum.GitspaceOwnerSelf && filter.UserIdentifier != "" {
		stmt = stmt.Where(squirrel.Eq{"gconf_user_uid": filter.UserIdentifier})
	}

	if filter.MarkedForDeletion != nil {
		stmt = stmt.Where(squirrel.Eq{"gconf_is_marked_for_deletion": filter.MarkedForDeletion})
	}

	if len(filter.SpaceIDs) > 0 {
		stmt = stmt.Where(squirrel.Eq{"gconf_space_id": filter.SpaceIDs})
	}
	if len(filter.CodeRepoTypes) > 0 {
		stmt = stmt.Where(squirrel.Eq{"gconf_code_repo_type": filter.CodeRepoTypes})
	}

	if filter.LastHeartBeatBefore > 0 {
		stmt = stmt.Where(squirrel.Lt{"gits_last_heartbeat": filter.LastHeartBeatBefore})
	}

	if filter.LastUsedBefore > 0 {
		stmt = stmt.Where(squirrel.Lt{"gits_last_used": filter.LastUsedBefore})
	}

	if filter.LastUpdatedBefore > 0 {
		stmt = stmt.Where(squirrel.Lt{"gits_updated": filter.LastUpdatedBefore})
	}

	if len(filter.GitspaceFilterStates) > 0 && len(filter.States) > 0 {
		log.Warn().Msgf("both view list filter and states are set for gitspace, the states[] are ignored")
	}

	if len(filter.GitspaceFilterStates) > 0 {
		instanceStateTypes := make([]enum.GitspaceInstanceStateType, 0, len(filter.GitspaceFilterStates))
		for _, state := range filter.GitspaceFilterStates {
			switch state {
			case enum.GitspaceFilterStateError:
				instanceStateTypes =
					append(
						instanceStateTypes,
						enum.GitspaceInstanceStateError,
						enum.GitspaceInstanceStateUnknown,
					)
			case enum.GitspaceFilterStateRunning:
				instanceStateTypes =
					append(instanceStateTypes, enum.GitspaceInstanceStateRunning)
			case enum.GitspaceFilterStateStopped:
				instanceStateTypes = append(
					instanceStateTypes,
					enum.GitspaceInstanceStateStopped,
					enum.GitspaceInstanceStateCleaned,
					enum.GitspaceInstanceStateDeleted,
					enum.GitspaceInstanceStateUninitialized,
				)
			}
		}
		stmt = stmt.Where(squirrel.Eq{"gits_state": instanceStateTypes})
	} else if len(filter.States) > 0 {
		stmt = stmt.Where(squirrel.Eq{"gits_state": filter.States})
	}
	return stmt
}

func addOrderBy(stmt squirrel.SelectBuilder, filter *types.GitspaceFilter) squirrel.SelectBuilder {
	switch filter.Sort {
	case enum.GitspaceSortLastUsed:
		return stmt.OrderBy("gits_last_used " + filter.Order.String())
	case enum.GitspaceSortCreated:
		return stmt.OrderBy("gconf_created " + filter.Order.String())
	case enum.GitspaceSortLastActivated:
		return stmt.OrderBy("gits_created " + filter.Order.String())
	default:
		return stmt.OrderBy("gits_created " + filter.Order.String())
	}
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

func (s gitspaceConfigStore) ListActiveConfigsForInfraProviderResource(
	ctx context.Context,
	infraProviderResourceID int64,
) ([]*types.GitspaceConfig, error) {
	stmt := database.Builder.
		Select(gitspaceConfigSelectColumns).
		From(gitspaceConfigsTable).
		Where("gconf_infra_provider_resource_id = ?", infraProviderResourceID).
		Where("gconf_is_deleted = false").
		Where("gconf_is_marked_for_deletion = false")

	sql, args, err := stmt.ToSql()

	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, s.db)
	var dst []*gitspaceConfigWithLatestInstance
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing list gitspace config query")
	}
	return s.ToGitspaceConfigs(ctx, dst)
}

func (s gitspaceConfigStore) mapDBToGitspaceConfig(
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
	var result = &types.GitspaceConfig{
		ID:                    in.ID,
		Identifier:            in.Identifier,
		Name:                  in.Name,
		IDE:                   in.IDE,
		SpaceID:               in.SpaceID,
		Created:               in.Created,
		Updated:               in.Updated,
		SSHTokenIdentifier:    in.SSHTokenIdentifier,
		IsMarkedForDeletion:   in.IsMarkedForDeletion,
		IsMarkedForReset:      in.IsMarkedForReset,
		IsMarkedForInfraReset: in.IsMarkedForInfraReset,
		IsDeleted:             in.IsDeleted,
		CodeRepo:              codeRepo,
		GitspaceUser: types.GitspaceUser{
			ID:         in.CreatedBy.Ptr(),
			Identifier: in.UserUID},
	}
	if result.GitspaceUser.ID != nil {
		author, _ := s.pCache.Get(ctx, *result.GitspaceUser.ID)
		if author != nil {
			result.GitspaceUser.DisplayName = author.DisplayName
			result.GitspaceUser.Email = author.Email
		}
	}

	if resource, err := s.rCache.Get(ctx, in.InfraProviderResourceID); err == nil {
		result.InfraProviderResource = *resource
	} else {
		return nil, fmt.Errorf("couldn't set resource to the config in DB: %s", in.Identifier)
	}
	return result, nil
}

func (s gitspaceConfigStore) ToGitspaceConfig(
	ctx context.Context,
	in *gitspaceConfigWithLatestInstance,
) (*types.GitspaceConfig, error) {
	var result, err = s.mapDBToGitspaceConfig(ctx, &in.gitspaceConfig)
	if err != nil {
		return nil, err
	}
	instance, err := mapDBToGitspaceInstance(ctx, &in.gitspaceInstance)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("Failed to convert to gitspace instance, gitspace configID: %d",
			in.gitspaceInstance.ID,
		)
		instance = nil
	}

	if instance != nil {
		gitspaceStateType, err2 := instance.GetGitspaceState()
		if err2 != nil {
			return nil, err2
		}
		result.State = gitspaceStateType
	} else {
		result.State = enum.GitspaceStateUninitialized
	}
	result.GitspaceInstance = instance

	return result, nil
}

func (s gitspaceConfigStore) mapToGitspaceConfigs(
	ctx context.Context,
	configs []*gitspaceConfig,
) ([]*types.GitspaceConfig, error) {
	var err error
	res := make([]*types.GitspaceConfig, len(configs))
	for i := range configs {
		res[i], err = s.mapDBToGitspaceConfig(ctx, configs[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s gitspaceConfigStore) ToGitspaceConfigs(
	ctx context.Context,
	configs []*gitspaceConfigWithLatestInstance,
) ([]*types.GitspaceConfig, error) {
	var err error
	res := make([]*types.GitspaceConfig, len(configs))
	for i := range configs {
		res[i], err = s.ToGitspaceConfig(ctx, configs[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func addGitspaceQueryFilter(stmt squirrel.SelectBuilder, filter types.ListQueryFilter) squirrel.SelectBuilder {
	if filter.Query != "" {
		stmt = stmt.Where(squirrel.Or{
			squirrel.Expr(PartialMatch("gconf_uid", filter.Query)),
			squirrel.Expr(PartialMatch("gconf_display_name", filter.Query)),
		})
	}
	return stmt
}

func sortBy(configs []*types.GitspaceConfig, idsInOrder []string) []types.GitspaceConfig {
	idsIdxMap := make(map[string]int)
	for i, id := range idsInOrder {
		idsIdxMap[id] = i
	}

	orderedConfigs := make([]types.GitspaceConfig, len(configs))
	for _, config := range configs {
		orderedConfigs[idsIdxMap[config.Identifier]] = *config
	}

	return orderedConfigs
}

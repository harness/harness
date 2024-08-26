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

	gitnessAppStore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var _ gitnessAppStore.UserGroupStore = (*UserGroupStore)(nil)

func NewUserGroupStore(db *sqlx.DB) *UserGroupStore {
	return &UserGroupStore{
		db: db,
	}
}

type UserGroupStore struct {
	db *sqlx.DB
}

type UserGroup struct {
	SpaceID     int64  `db:"usergroup_space_id"`
	ID          int64  `db:"usergroup_id"`
	Identifier  string `db:"usergroup_identifier"`
	Name        string `db:"usergroup_name"`
	Description string `db:"usergroup_description"`
	Created     int64  `db:"usergroup_created"`
	Updated     int64  `db:"usergroup_updated"`
}

const (
	userGroupColumns = `
	usergroup_id
	,usergroup_identifier
	,usergroup_name
	,usergroup_description
	,usergroup_space_id
	,usergroup_created
	,usergroup_updated`

	userGroupSelectBase = `SELECT ` + userGroupColumns + ` FROM usergroups`
)

func mapUserGroup(ug *UserGroup) *types.UserGroup {
	return &types.UserGroup{
		ID:          ug.ID,
		Identifier:  ug.Identifier,
		Name:        ug.Name,
		Description: ug.Description,
		SpaceID:     ug.SpaceID,
		Created:     ug.Created,
		Updated:     ug.Updated,
	}
}

// FindByIdentifier returns a usergroup by its identifier.
func (s *UserGroupStore) FindByIdentifier(
	ctx context.Context,
	spaceID int64,
	identifier string,
) (*types.UserGroup, error) {
	const sqlQuery = userGroupSelectBase + ` WHERE usergroup_identifier = $1 AND usergroup_space_id = $2`
	db := dbtx.GetAccessor(ctx, s.db)
	dst := &UserGroup{}
	if err := db.GetContext(ctx, dst, sqlQuery, identifier, spaceID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find usergroup by identifier %s", identifier)
	}
	return mapUserGroup(dst), nil
}

// Find returns a usergroup by its id.
func (s *UserGroupStore) Find(ctx context.Context, id int64) (*types.UserGroup, error) {
	const sqlQuery = userGroupSelectBase + ` WHERE usergroup_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)
	dst := &UserGroup{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find usergroup by id %d", id)
	}
	return mapUserGroup(dst), nil
}

func (s *UserGroupStore) Map(ctx context.Context, ids []int64) (map[int64]*types.UserGroup, error) {
	result, err := s.FindManyByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, store.ErrResourceNotFound
	}
	mapResult := make(map[int64]*types.UserGroup, len(result))
	for _, r := range result {
		mapResult[r.ID] = r
	}
	return mapResult, nil
}

func (s *UserGroupStore) FindManyByIDs(ctx context.Context, ids []int64) ([]*types.UserGroup, error) {
	stmt := database.Builder.
		Select(userGroupColumns).
		From("usergroups").
		Where(squirrel.Eq{"usergroup_id": ids})
	db := dbtx.GetAccessor(ctx, s.db)

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to generate find many usergroups by ids query")
	}

	var dst []*UserGroup
	if err := db.SelectContext(ctx, &dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "find many by ids for usergroups query failed")
	}

	var result = make([]*types.UserGroup, len(dst))

	for i, u := range dst {
		result[i] = u.toUserGroupType()
	}

	return result, nil
}

func (s *UserGroupStore) FindManyByIdentifiersAndSpaceID(
	ctx context.Context,
	identifiers []string,
	spaceID int64,
) ([]*types.UserGroup, error) {
	stmt := database.Builder.
		Select(userGroupColumns).
		From("usergroups").
		Where(squirrel.Eq{"usergroup_identifier": identifiers}).Where(squirrel.Eq{"usergroup_space_id": spaceID})
	db := dbtx.GetAccessor(ctx, s.db)

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to generate find many usergroups query")
	}

	dst := []*UserGroup{}
	if err := db.SelectContext(ctx, &dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "find many by identifiers for usergroups query failed")
	}
	result := make([]*types.UserGroup, len(dst))
	for i, u := range dst {
		result[i] = mapUserGroup(u)
	}
	return result, nil
}

// Create Creates a usergroup in the database.
func (s *UserGroupStore) Create(
	ctx context.Context,
	spaceID int64,
	userGroup *types.UserGroup,
) error {
	const sqlQuery = `
	INSERT INTO usergroups (
		usergroup_identifier
		,usergroup_name
		,usergroup_description	
		,usergroup_space_id	
		,usergroup_created
		,usergroup_updated
	) values (	
		:usergroup_identifier
		,:usergroup_name
		,:usergroup_description
		,:usergroup_space_id	
		,:usergroup_created
		,:usergroup_updated
	) RETURNING usergroup_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalUserGroup(userGroup, spaceID))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind usergroup object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert usergroup")
	}

	return nil
}

func (s *UserGroupStore) CreateOrUpdate(
	ctx context.Context,
	spaceID int64,
	userGroup *types.UserGroup,
) error {
	const sqlQuery = `
	INSERT INTO usergroups (
		usergroup_identifier
		,usergroup_name
		,usergroup_description	
		,usergroup_space_id	
		,usergroup_created
		,usergroup_updated
	) values (	
		:usergroup_identifier
		,:usergroup_name
		,:usergroup_description
		,:usergroup_space_id	
		,:usergroup_created
		,:usergroup_updated
	) ON CONFLICT (usergroup_identifier, usergroup_space_id) DO UPDATE SET
		usergroup_name = EXCLUDED.usergroup_name,
		usergroup_description = EXCLUDED.usergroup_description,
		usergroup_updated = EXCLUDED.usergroup_updated
	`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, mapInternalUserGroup(userGroup, spaceID))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind usergroup object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert usergroup")
	}

	return nil
}

func mapInternalUserGroup(u *types.UserGroup, spaceID int64) *UserGroup {
	return &UserGroup{
		ID:          u.ID,
		SpaceID:     spaceID,
		Identifier:  u.Identifier,
		Name:        u.Name,
		Description: u.Description,
		Created:     u.Created,
		Updated:     u.Updated,
	}
}

func (u *UserGroup) toUserGroupType() *types.UserGroup {
	return &types.UserGroup{
		ID:          u.ID,
		Identifier:  u.Identifier,
		Name:        u.Name,
		Description: u.Description,
		Created:     u.Created,
		Updated:     u.Updated,
	}
}

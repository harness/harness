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

var _ store.GitspaceInstanceStore = (*gitspaceInstanceStore)(nil)

const (
	gitspaceInstanceInsertColumns = `
        gits_gitspace_config_id,  
        gits_url, 
        gits_state,               
        gits_user_uid,             
        gits_resource_usage,      
        gits_space_id,            
        gits_created, 
		gits_updated, 
        gits_last_used,           
        gits_total_time_used,     
        gits_tracked_changes,
		gits_access_key,
		gits_access_type,
		gits_machine_user,
		gits_uid`
	gitspaceInstanceSelectColumns = "gits_id," + gitspaceInstanceInsertColumns
	gitspaceInstanceTable         = `gitspaces`
)

type gitspaceInstance struct {
	ID               int64                          `db:"gits_id"`
	GitSpaceConfigID int64                          `db:"gits_gitspace_config_id"`
	URL              null.String                    `db:"gits_url"`
	State            enum.GitspaceInstanceStateType `db:"gits_state"`
	UserUID          string                         `db:"gits_user_uid"`
	ResourceUsage    null.String                    `db:"gits_resource_usage"`
	SpaceID          int64                          `db:"gits_space_id"`
	LastUsed         int64                          `db:"gits_last_used"`
	TotalTimeUsed    int64                          `db:"gits_total_time_used"`
	TrackedChanges   null.String                    `db:"gits_tracked_changes"`
	AccessKey        null.String                    `db:"gits_access_key"`
	AccessType       enum.GitspaceAccessType        `db:"gits_access_type"`
	MachineUser      null.String                    `db:"gits_machine_user"`
	Identifier       string                         `db:"gits_uid"`
	Created          int64                          `db:"gits_created"`
	Updated          int64                          `db:"gits_updated"`
}

// NewGitspaceInstanceStore returns a new GitspaceInstanceStore.
func NewGitspaceInstanceStore(db *sqlx.DB) store.GitspaceInstanceStore {
	return &gitspaceInstanceStore{
		db: db,
	}
}

type gitspaceInstanceStore struct {
	db *sqlx.DB
}

func (g gitspaceInstanceStore) Find(ctx context.Context, id int64) (*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		Where("gits_id = $1", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	gitspace := new(gitspaceInstance)
	db := dbtx.GetAccessor(ctx, g.db)
	if err := db.GetContext(ctx, gitspace, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace")
	}
	return g.mapToGitspaceInstance(ctx, gitspace)
}

func (g gitspaceInstanceStore) Create(ctx context.Context, gitspaceInstance *types.GitspaceInstance) error {
	stmt := database.Builder.
		Insert(gitspaceInstanceTable).
		Columns(gitspaceInstanceInsertColumns).
		Values(
			gitspaceInstance.GitSpaceConfigID,
			gitspaceInstance.URL,
			gitspaceInstance.State,
			gitspaceInstance.UserID,
			gitspaceInstance.ResourceUsage,
			gitspaceInstance.SpaceID,
			gitspaceInstance.Created,
			gitspaceInstance.Updated,
			gitspaceInstance.LastUsed,
			gitspaceInstance.TotalTimeUsed,
			gitspaceInstance.TrackedChanges,
			gitspaceInstance.AccessKey,
			gitspaceInstance.AccessType,
			gitspaceInstance.MachineUser,
			gitspaceInstance.Identifier,
		).
		Suffix(ReturningClause + "gits_id")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&gitspaceInstance.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "gitspace query failed")
	}
	return nil
}

func (g gitspaceInstanceStore) Update(
	ctx context.Context,
	gitspaceInstance *types.GitspaceInstance,
) error {
	stmt := database.Builder.
		Update(gitspaceInstanceTable).
		Set("gits_state", gitspaceInstance.State).
		Set("gits_last_used", gitspaceInstance.LastUsed).
		Set("gits_url", gitspaceInstance.URL).
		Set("gits_updated", gitspaceInstance.Updated).
		Where("gits_id = $5", gitspaceInstance.ID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update gitspace")
	}
	return nil
}

func (g gitspaceInstanceStore) FindLatestByGitspaceConfigID(
	ctx context.Context,
	gitspaceConfigID int64,
	spaceID int64,
) (*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		Where("gits_gitspace_config_id = $1", gitspaceConfigID).
		Where("gits_space_id = $2", spaceID).
		OrderBy("gits_created DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	gitspace := new(gitspaceInstance)
	db := dbtx.GetAccessor(ctx, g.db)
	if err := db.GetContext(ctx, gitspace, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace")
	}
	return g.mapToGitspaceInstance(ctx, gitspace)
}

func (g gitspaceInstanceStore) List(
	ctx context.Context,
	filter *types.GitspaceFilter,
) ([]*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		Where(squirrel.Eq{"gits_space_id": filter.SpaceIDs}).
		Where(squirrel.Eq{"gits_user_uid": filter.UserID}).
		OrderBy("gits_created ASC")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	var dst []*gitspaceInstance
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return g.mapToGitspaceInstances(ctx, dst)
}

func (g gitspaceInstanceStore) FindAllLatestByGitspaceConfigID(
	ctx context.Context,
	gitspaceConfigIDs []int64,
) ([]*types.GitspaceInstance, error) {
	var whereClause = "(1=0)"
	if len(gitspaceConfigIDs) > 0 {
		whereClause = fmt.Sprintf("gits_gitspace_config_id IN (%s)",
			strings.Trim(strings.Join(strings.Split(fmt.Sprint(gitspaceConfigIDs), " "), ","), "[]"))
	}
	baseSelect := squirrel.Select("*",
		"ROW_NUMBER() OVER (PARTITION BY gits_gitspace_config_id "+
			"ORDER BY gits_created DESC) AS rn").
		From(gitspaceInstanceTable).
		Where(whereClause)

	// Use the base select query in a common table expression (CTE)
	stmt := squirrel.Select(gitspaceInstanceSelectColumns).
		FromSelect(baseSelect, "RankedRows").
		Where("rn = 1")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	db := dbtx.GetAccessor(ctx, g.db)
	var dst []*gitspaceInstance
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}
	return g.mapToGitspaceInstances(ctx, dst)
}

func (g gitspaceInstanceStore) mapToGitspaceInstance(
	_ context.Context,
	in *gitspaceInstance,
) (*types.GitspaceInstance, error) {
	var res = &types.GitspaceInstance{
		ID:               in.ID,
		Identifier:       in.Identifier,
		GitSpaceConfigID: in.GitSpaceConfigID,
		URL:              in.URL.Ptr(),
		State:            in.State,
		UserID:           in.UserUID,
		ResourceUsage:    in.ResourceUsage.Ptr(),
		LastUsed:         in.LastUsed,
		TotalTimeUsed:    in.TotalTimeUsed,
		TrackedChanges:   in.TrackedChanges.Ptr(),
		AccessKey:        in.AccessKey.Ptr(),
		AccessType:       in.AccessType,
		MachineUser:      in.MachineUser.Ptr(),
		SpaceID:          in.SpaceID,
		Created:          in.Created,
		Updated:          in.Updated,
	}
	return res, nil
}

func (g gitspaceInstanceStore) mapToGitspaceInstances(
	ctx context.Context,
	instances []*gitspaceInstance,
) ([]*types.GitspaceInstance, error) {
	var err error
	res := make([]*types.GitspaceInstance, len(instances))
	for i := range instances {
		res[i], err = g.mapToGitspaceInstance(ctx, instances[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

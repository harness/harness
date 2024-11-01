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
	"database/sql"
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
		gits_access_type,
		gits_machine_user,
		gits_uid,
		gits_access_key_ref,
        gits_last_heartbeat,
		gits_active_time_started,
		gits_active_time_ended,
		gits_has_git_changes`
	gitspaceInstanceSelectColumns = "gits_id," + gitspaceInstanceInsertColumns
	gitspaceInstanceTable         = `gitspaces`
)

type gitspaceInstance struct {
	ID               int64                          `db:"gits_id"`
	GitSpaceConfigID int64                          `db:"gits_gitspace_config_id"`
	URL              null.String                    `db:"gits_url"`
	State            enum.GitspaceInstanceStateType `db:"gits_state"`
	// TODO: migrate to principal int64 id to use principal cache and consistent with Harness code.
	UserUID           string                  `db:"gits_user_uid"`
	ResourceUsage     null.String             `db:"gits_resource_usage"`
	SpaceID           int64                   `db:"gits_space_id"`
	LastUsed          null.Int                `db:"gits_last_used"`
	TotalTimeUsed     int64                   `db:"gits_total_time_used"`
	AccessType        enum.GitspaceAccessType `db:"gits_access_type"`
	AccessKeyRef      null.String             `db:"gits_access_key_ref"`
	MachineUser       null.String             `db:"gits_machine_user"`
	Identifier        string                  `db:"gits_uid"`
	Created           int64                   `db:"gits_created"`
	Updated           int64                   `db:"gits_updated"`
	LastHeartbeat     null.Int                `db:"gits_last_heartbeat"`
	ActiveTimeStarted null.Int                `db:"gits_active_time_started"`
	ActiveTimeEnded   null.Int                `db:"gits_active_time_ended"`
	HasGitChanges     null.Bool               `db:"gits_has_git_changes"`
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

func (g gitspaceInstanceStore) FindTotalUsage(
	ctx context.Context,
	fromTime int64,
	toTime int64,
	spaceIDs []int64,
) (int64, error) {
	var greatest = "MAX"
	var least = "MIN"
	if g.db.DriverName() == "postgres" {
		greatest = "GREATEST"
		least = "LEAST"
	}
	innerQuery := squirrel.Select(
		greatest+"(gits_active_time_started, ?) AS effective_start_time",
		least+"(COALESCE(gits_active_time_ended, ?), ?) AS effective_end_time",
	).
		From(gitspaceInstanceTable).
		Where(
			squirrel.And{
				squirrel.Lt{"gits_active_time_started": toTime},
				squirrel.Or{
					squirrel.Expr("gits_active_time_ended IS NULL"),
					squirrel.Gt{"gits_active_time_ended": fromTime},
				},
				squirrel.Eq{"gits_space_id": spaceIDs},
			},
		)

	innerQry, innerArgs, err := innerQuery.ToSql()
	if err != nil {
		return 0, err
	}

	query := squirrel.
		Select("SUM(effective_end_time - effective_start_time) AS total_active_time").
		From("(" + innerQry + ") AS subquery").PlaceholderFormat(squirrel.Dollar)

	qry, _, err := query.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}

	args := append([]any{fromTime, toTime, toTime}, innerArgs...)

	var totalActiveTime sql.NullInt64
	db := dbtx.GetAccessor(ctx, g.db)
	err = db.GetContext(ctx, &totalActiveTime, qry, args...)
	if err != nil {
		return 0, err
	}

	if totalActiveTime.Valid {
		return totalActiveTime.Int64, nil
	}

	return 0, nil
}

func (g gitspaceInstanceStore) Find(ctx context.Context, id int64) (*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		Where("gits_id = ?", id)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	gitspace := new(gitspaceInstance)
	db := dbtx.GetAccessor(ctx, g.db)
	if err := db.GetContext(ctx, gitspace, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace %d", id)
	}
	return mapDBToGitspaceInstance(ctx, gitspace)
}

func (g gitspaceInstanceStore) FindByIdentifier(
	ctx context.Context,
	identifier string,
) (*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		Where("gits_uid = ?", identifier)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	gitspace := new(gitspaceInstance)
	db := dbtx.GetAccessor(ctx, g.db)
	if err := db.GetContext(ctx, gitspace, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find gitspace %s", identifier)
	}
	return mapDBToGitspaceInstance(ctx, gitspace)
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
			gitspaceInstance.AccessType,
			gitspaceInstance.MachineUser,
			gitspaceInstance.Identifier,
			gitspaceInstance.AccessKeyRef,
			gitspaceInstance.LastHeartbeat,
			gitspaceInstance.ActiveTimeStarted,
			gitspaceInstance.ActiveTimeEnded,
			gitspaceInstance.HasGitChanges,
		).
		Suffix(ReturningClause + "gits_id")
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	if err = db.QueryRowContext(ctx, sql, args...).Scan(&gitspaceInstance.ID); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "gitspace instance query failed for %s", gitspaceInstance.Identifier)
	}
	return nil
}

func (g gitspaceInstanceStore) Update(
	ctx context.Context,
	gitspaceInstance *types.GitspaceInstance,
) error {
	validateActiveTimeDetails(gitspaceInstance)
	stmt := database.Builder.
		Update(gitspaceInstanceTable).
		Set("gits_state", gitspaceInstance.State).
		Set("gits_last_used", gitspaceInstance.LastUsed).
		Set("gits_last_heartbeat", gitspaceInstance.LastHeartbeat).
		Set("gits_url", gitspaceInstance.URL).
		Set("gits_active_time_started", gitspaceInstance.ActiveTimeStarted).
		Set("gits_active_time_ended", gitspaceInstance.ActiveTimeEnded).
		Set("gits_total_time_used", gitspaceInstance.TotalTimeUsed).
		Set("gits_has_git_changes", gitspaceInstance.HasGitChanges).
		Set("gits_updated", gitspaceInstance.Updated).
		Where("gits_id = ?", gitspaceInstance.ID)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(
			ctx, err, "Failed to update gitspace instance for %s", gitspaceInstance.Identifier)
	}
	return nil
}

func (g gitspaceInstanceStore) FindLatestByGitspaceConfigID(
	ctx context.Context,
	gitspaceConfigID int64,
) (*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		Where("gits_gitspace_config_id = ?", gitspaceConfigID).
		OrderBy("gits_created DESC")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	gitspace := new(gitspaceInstance)
	db := dbtx.GetAccessor(ctx, g.db)
	if err := db.GetContext(ctx, gitspace, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(
			ctx, err, "Failed to find latest gitspace instance for %d", gitspaceConfigID)
	}
	return mapDBToGitspaceInstance(ctx, gitspace)
}

func (g gitspaceInstanceStore) List(
	ctx context.Context,
	filter *types.GitspaceInstanceFilter,
) ([]*types.GitspaceInstance, error) {
	stmt := database.Builder.
		Select(gitspaceInstanceSelectColumns).
		From(gitspaceInstanceTable).
		OrderBy("gits_created ASC")
	stmt = addGitspaceInstanceFilter(stmt, filter)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert squirrel builder to sql")
	}
	db := dbtx.GetAccessor(ctx, g.db)
	var dst []*gitspaceInstance
	if err := db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing gitspace instance list query")
	}
	return g.mapToGitspaceInstances(ctx, dst)
}

func (g gitspaceInstanceStore) Count(ctx context.Context, filter *types.GitspaceInstanceFilter) (int64, error) {
	db := dbtx.GetAccessor(ctx, g.db)
	countStmt := database.Builder.
		Select("COUNT(*)").
		From(gitspaceInstanceTable)

	countStmt = addGitspaceInstanceFilter(countStmt, filter)

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
		return nil, database.ProcessSQLErrorf(
			ctx, err, "Failed executing all latest gitspace instance list query")
	}
	return g.mapToGitspaceInstances(ctx, dst)
}

func addGitspaceInstanceFilter(
	stmt squirrel.SelectBuilder,
	filter *types.GitspaceInstanceFilter,
) squirrel.SelectBuilder {
	if len(filter.SpaceIDs) > 0 {
		stmt = stmt.Where(squirrel.Eq{"gits_space_id": filter.SpaceIDs})
	}

	if filter.UserIdentifier != "" {
		stmt = stmt.Where(squirrel.Eq{"gits_user_id": filter.UserIdentifier})
	}

	if filter.LastHeartBeatBefore > 0 {
		stmt = stmt.Where(squirrel.Lt{"gits_last_heartbeat": filter.LastHeartBeatBefore})
	}

	if len(filter.States) > 0 {
		stmt = stmt.Where(squirrel.Eq{"gits_state": filter.States})
	}

	if filter.Limit > 0 {
		stmt = stmt.Limit(database.Limit(filter.Limit))
	}
	return stmt
}

func mapDBToGitspaceInstance(
	_ context.Context,
	in *gitspaceInstance,
) (*types.GitspaceInstance, error) {
	var res = &types.GitspaceInstance{
		ID:                in.ID,
		Identifier:        in.Identifier,
		GitSpaceConfigID:  in.GitSpaceConfigID,
		URL:               in.URL.Ptr(),
		State:             in.State,
		UserID:            in.UserUID,
		ResourceUsage:     in.ResourceUsage.Ptr(),
		LastUsed:          in.LastUsed.Ptr(),
		TotalTimeUsed:     in.TotalTimeUsed,
		AccessType:        in.AccessType,
		AccessKeyRef:      in.AccessKeyRef.Ptr(),
		MachineUser:       in.MachineUser.Ptr(),
		SpaceID:           in.SpaceID,
		Created:           in.Created,
		Updated:           in.Updated,
		LastHeartbeat:     in.LastHeartbeat.Ptr(),
		ActiveTimeEnded:   in.ActiveTimeEnded.Ptr(),
		ActiveTimeStarted: in.ActiveTimeStarted.Ptr(),
		HasGitChanges:     in.HasGitChanges.Ptr(),
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
		res[i], err = mapDBToGitspaceInstance(ctx, instances[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func validateActiveTimeDetails(gitspaceInstance *types.GitspaceInstance) {
	if (gitspaceInstance.State == enum.GitspaceInstanceStateStarting ||
		gitspaceInstance.State == enum.GitspaceInstanceStateUninitialized) &&
		(gitspaceInstance.ActiveTimeStarted != nil ||
			gitspaceInstance.ActiveTimeEnded != nil ||
			gitspaceInstance.TotalTimeUsed != 0) {
		log.Warn().Msgf("instance has incorrect active time, details: identifier %s state %s active time start "+
			"%d active time end %d total time used %d", gitspaceInstance.Identifier, gitspaceInstance.State,
			gitspaceInstance.ActiveTimeStarted, gitspaceInstance.ActiveTimeEnded, gitspaceInstance.TotalTimeUsed)
	}
	if (gitspaceInstance.State == enum.GitspaceInstanceStateRunning) &&
		(gitspaceInstance.ActiveTimeStarted == nil ||
			gitspaceInstance.ActiveTimeEnded != nil ||
			gitspaceInstance.TotalTimeUsed != 0) {
		log.Warn().Msgf(
			"instance is missing active time start or has incorrect end/total timestamps, details: "+
				" identifier %s state %s active time start %d active time end %d total time used %d",
			gitspaceInstance.Identifier, gitspaceInstance.State, gitspaceInstance.ActiveTimeStarted,
			gitspaceInstance.ActiveTimeEnded, gitspaceInstance.TotalTimeUsed)
	}
	if (gitspaceInstance.State == enum.GitspaceInstanceStateDeleted ||
		gitspaceInstance.State == enum.GitspaceInstanceStateStopping ||
		gitspaceInstance.State == enum.GitspaceInstanceStateError) &&
		(gitspaceInstance.ActiveTimeStarted == nil ||
			gitspaceInstance.ActiveTimeEnded == nil ||
			gitspaceInstance.TotalTimeUsed == 0) {
		log.Warn().Msgf("instance is missing active time start/end/total timestamp, details: "+
			" identifier %s state %s active time start %d active time end %d total time used %d",
			gitspaceInstance.Identifier, gitspaceInstance.State, gitspaceInstance.ActiveTimeStarted,
			gitspaceInstance.ActiveTimeEnded, gitspaceInstance.TotalTimeUsed)
	}
}

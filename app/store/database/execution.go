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
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	sqlxtypes "github.com/jmoiron/sqlx/types"
	"github.com/pkg/errors"
)

var _ store.ExecutionStore = (*executionStore)(nil)

// NewExecutionStore returns a new ExecutionStore.
func NewExecutionStore(db *sqlx.DB) store.ExecutionStore {
	return &executionStore{
		db: db,
	}
}

type executionStore struct {
	db *sqlx.DB
}

// execution represents an execution object stored in the database.
type execution struct {
	ID           int64              `db:"execution_id"`
	PipelineID   int64              `db:"execution_pipeline_id"`
	CreatedBy    int64              `db:"execution_created_by"`
	RepoID       int64              `db:"execution_repo_id"`
	Trigger      string             `db:"execution_trigger"`
	Number       int64              `db:"execution_number"`
	Parent       int64              `db:"execution_parent"`
	Status       enum.CIStatus      `db:"execution_status"`
	Error        string             `db:"execution_error"`
	Event        enum.TriggerEvent  `db:"execution_event"`
	Action       enum.TriggerAction `db:"execution_action"`
	Link         string             `db:"execution_link"`
	Timestamp    int64              `db:"execution_timestamp"`
	Title        string             `db:"execution_title"`
	Message      string             `db:"execution_message"`
	Before       string             `db:"execution_before"`
	After        string             `db:"execution_after"`
	Ref          string             `db:"execution_ref"`
	Fork         string             `db:"execution_source_repo"`
	Source       string             `db:"execution_source"`
	Target       string             `db:"execution_target"`
	Author       string             `db:"execution_author"`
	AuthorName   string             `db:"execution_author_name"`
	AuthorEmail  string             `db:"execution_author_email"`
	AuthorAvatar string             `db:"execution_author_avatar"`
	Sender       string             `db:"execution_sender"`
	Params       sqlxtypes.JSONText `db:"execution_params"`
	Cron         string             `db:"execution_cron"`
	Deploy       string             `db:"execution_deploy"`
	DeployID     int64              `db:"execution_deploy_id"`
	Debug        bool               `db:"execution_debug"`
	Started      int64              `db:"execution_started"`
	Finished     int64              `db:"execution_finished"`
	Created      int64              `db:"execution_created"`
	Updated      int64              `db:"execution_updated"`
	Version      int64              `db:"execution_version"`
}

type executionPipelineRepoJoin struct {
	execution
	PipelineUID sql.NullString `db:"pipeline_uid"`
	RepoUID     sql.NullString `db:"repo_uid"`
}

const (
	executionColumns = `
		execution_id
		,execution_pipeline_id
		,execution_created_by
		,execution_repo_id
		,execution_trigger
		,execution_number
		,execution_parent
		,execution_status
		,execution_error
		,execution_event
		,execution_action
		,execution_link
		,execution_timestamp
		,execution_title
		,execution_message
		,execution_before
		,execution_after
		,execution_ref
		,execution_source_repo
		,execution_source
		,execution_target
		,execution_author
		,execution_author_name
		,execution_author_email
		,execution_author_avatar
		,execution_sender
		,execution_params
		,execution_cron
		,execution_deploy
		,execution_deploy_id
		,execution_debug
		,execution_started
		,execution_finished
		,execution_created
		,execution_updated
		,execution_version
	`
)

// Find returns an execution given an execution ID.
func (s *executionStore) Find(ctx context.Context, id int64) (*types.Execution, error) {
	//nolint:goconst
	const findQueryStmt = `
	SELECT` + executionColumns + `
	FROM executions
	WHERE execution_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(execution)
	if err := db.GetContext(ctx, dst, findQueryStmt, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find execution")
	}
	return mapInternalToExecution(dst)
}

// FindByNumber returns an execution given a pipeline ID and an execution number.
func (s *executionStore) FindByNumber(
	ctx context.Context,
	pipelineID int64,
	executionNum int64,
) (*types.Execution, error) {
	const findQueryStmt = `
	SELECT` + executionColumns + `
	FROM executions
	WHERE execution_pipeline_id = $1 AND execution_number = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(execution)
	if err := db.GetContext(ctx, dst, findQueryStmt, pipelineID, executionNum); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find execution")
	}
	return mapInternalToExecution(dst)
}

// Create creates a new execution in the datastore.
func (s *executionStore) Create(ctx context.Context, execution *types.Execution) error {
	const executionInsertStmt = `
	INSERT INTO executions (
		execution_pipeline_id
		,execution_repo_id
		,execution_created_by
		,execution_trigger
		,execution_number
		,execution_parent
		,execution_status
		,execution_error
		,execution_event
		,execution_action
		,execution_link
		,execution_timestamp
		,execution_title
		,execution_message
		,execution_before
		,execution_after
		,execution_ref
		,execution_source_repo
		,execution_source
		,execution_target
		,execution_author
		,execution_author_name
		,execution_author_email
		,execution_author_avatar
		,execution_sender
		,execution_params
		,execution_cron
		,execution_deploy
		,execution_deploy_id
		,execution_debug
		,execution_started
		,execution_finished
		,execution_created
		,execution_updated
		,execution_version
	) VALUES (
		:execution_pipeline_id
		,:execution_repo_id
		,:execution_created_by
		,:execution_trigger
		,:execution_number
		,:execution_parent
		,:execution_status
		,:execution_error
		,:execution_event
		,:execution_action
		,:execution_link
		,:execution_timestamp
		,:execution_title
		,:execution_message
		,:execution_before
		,:execution_after
		,:execution_ref
		,:execution_source_repo
		,:execution_source
		,:execution_target
		,:execution_author
		,:execution_author_name
		,:execution_author_email
		,:execution_author_avatar
		,:execution_sender
		,:execution_params
		,:execution_cron
		,:execution_deploy
		,:execution_deploy_id
		,:execution_debug
		,:execution_started
		,:execution_finished
		,:execution_created
		,:execution_updated
		,:execution_version
	) RETURNING execution_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(executionInsertStmt, mapExecutionToInternal(execution))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind execution object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&execution.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Execution query failed")
	}

	return nil
}

// Update tries to update an execution in the datastore with optimistic locking.
func (s *executionStore) Update(ctx context.Context, e *types.Execution) error {
	const executionUpdateStmt = `
	UPDATE executions
	SET
		execution_status = :execution_status
		,execution_error = :execution_error
		,execution_event = :execution_event
		,execution_started = :execution_started
		,execution_finished = :execution_finished
		,execution_updated = :execution_updated
		,execution_version = :execution_version
	WHERE execution_id = :execution_id AND execution_version = :execution_version - 1`
	updatedAt := time.Now()
	stages := e.Stages

	execution := mapExecutionToInternal(e)

	execution.Version++
	execution.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(executionUpdateStmt, execution)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind execution object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update execution")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	m, err := mapInternalToExecution(execution)
	if err != nil {
		return fmt.Errorf("could not map execution object: %w", err)
	}
	*e = *m
	e.Version = execution.Version
	e.Updated = execution.Updated
	e.Stages = stages // stages are not mapped in database.
	return nil
}

// List lists the executions for a given pipeline ID.
// It orders them in descending order of execution number.
func (s *executionStore) List(
	ctx context.Context,
	pipelineID int64,
	pagination types.Pagination,
) ([]*types.Execution, error) {
	stmt := database.Builder.
		Select(executionColumns).
		From("executions").
		Where("execution_pipeline_id = ?", fmt.Sprint(pipelineID)).
		OrderBy("execution_number " + enum.OrderDesc.String())

	stmt = stmt.Limit(database.Limit(pagination.Size))
	stmt = stmt.Offset(database.Offset(pagination.Page, pagination.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*execution{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return mapInternalToExecutionList(dst)
}

// ListInSpace lists the executions in a given space.
// It orders them in descending order of execution id.
func (s *executionStore) ListInSpace(
	ctx context.Context,
	spaceID int64,
	filter types.ListExecutionsFilter,
) ([]*types.Execution, error) {
	const executionWithPipelineRepoColumn = executionColumns + `
	,pipeline_uid
	,repo_uid`

	stmt := database.Builder.
		Select(executionWithPipelineRepoColumn).
		From("executions").
		InnerJoin("pipelines ON execution_pipeline_id = pipeline_id").
		InnerJoin("repositories ON execution_repo_id = repo_id").
		Where("repo_parent_id = ?", spaceID).
		OrderBy("execution_" + string(filter.Sort) + " " + filter.Order.String())

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(pipeline_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
	}

	if filter.PipelineIdentifier != "" {
		stmt = stmt.Where("pipeline_uid = ?", filter.PipelineIdentifier)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*executionPipelineRepoJoin{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return convertExecutionPipelineRepoJoins(dst)
}

func (s executionStore) ListByPipelineIDs(
	ctx context.Context,
	pipelineIDs []int64,
	maxRows int64,
) (map[int64][]*types.ExecutionInfo, error) {
	stmt := database.Builder.
		Select("execution_number, execution_pipeline_id, execution_status").
		FromSelect(
			database.Builder.
				Select(`
					execution_number, execution_pipeline_id, execution_status, 
					ROW_NUMBER() OVER (
						PARTITION BY execution_pipeline_id
						ORDER BY execution_number DESC
					) AS row_num
				`).
				From("executions").
				Where(squirrel.Eq{"execution_pipeline_id": pipelineIDs}),
			"ranked",
		).
		Where("row_num <= ?", maxRows)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)
	var dst []*types.ExecutionInfo
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to list executions by pipeline IDs")
	}

	executionInfosMap := make(map[int64][]*types.ExecutionInfo)
	for _, info := range dst {
		executionInfosMap[info.PipelineID] = append(
			executionInfosMap[info.PipelineID],
			info,
		)
	}

	return executionInfosMap, nil
}

// Count of executions in a pipeline, if pipelineID is 0 then return total number of executions.
func (s *executionStore) Count(ctx context.Context, pipelineID int64) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("executions")

	if pipelineID > 0 {
		stmt = stmt.Where("execution_pipeline_id = ?", pipelineID)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

// CountInSpace counts the number of executions in a given space.
func (s *executionStore) CountInSpace(
	ctx context.Context,
	spaceID int64,
	filter types.ListExecutionsFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("executions").
		InnerJoin("pipelines ON execution_pipeline_id = pipeline_id").
		InnerJoin("repositories ON execution_repo_id = repo_id").
		Where("repo_parent_id = ?", spaceID)

	if filter.Query != "" {
		stmt = stmt.Where("LOWER(pipeline_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Query)))
	}

	if filter.PipelineIdentifier != "" {
		stmt = stmt.Where("pipeline_uid = ?", filter.PipelineIdentifier)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

// Delete deletes an execution given a pipeline ID and an execution number.
func (s *executionStore) Delete(ctx context.Context, pipelineID int64, executionNum int64) error {
	const executionDeleteStmt = `
		DELETE FROM executions
		WHERE execution_pipeline_id = $1 AND execution_number = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, executionDeleteStmt, pipelineID, executionNum); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete execution")
	}

	return nil
}

func convertExecutionPipelineRepoJoins(rows []*executionPipelineRepoJoin) ([]*types.Execution, error) {
	executions := make([]*types.Execution, len(rows))
	for i, k := range rows {
		e, err := convertExecutionPipelineRepoJoin(k)
		if err != nil {
			return nil, err
		}
		executions[i] = e
	}
	return executions, nil
}

func convertExecutionPipelineRepoJoin(join *executionPipelineRepoJoin) (*types.Execution, error) {
	e, err := mapInternalToExecution(&join.execution)
	if err != nil {
		return nil, err
	}
	e.RepoUID = join.RepoUID.String
	e.PipelineUID = join.PipelineUID.String
	return e, nil
}

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
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

var _ store.StageStore = (*stageStore)(nil)

const (
	stageColumns = `
	stage_id
	,stage_execution_id
	,stage_repo_id
	,stage_number
	,stage_name
	,stage_kind
	,stage_type
	,stage_status
	,stage_error
	,stage_errignore
	,stage_exit_code
	,stage_machine
	,stage_os
	,stage_arch
	,stage_variant
	,stage_kernel
	,stage_limit
	,stage_limit_repo
	,stage_started
	,stage_stopped
	,stage_created
	,stage_updated
	,stage_version
	,stage_on_success
	,stage_on_failure
	,stage_depends_on
	,stage_labels
	`
)

type stage struct {
	ID            int64              `db:"stage_id"`
	ExecutionID   int64              `db:"stage_execution_id"`
	RepoID        int64              `db:"stage_repo_id"`
	Number        int64              `db:"stage_number"`
	Name          string             `db:"stage_name"`
	Kind          string             `db:"stage_kind"`
	Type          string             `db:"stage_type"`
	Status        enum.CIStatus      `db:"stage_status"`
	Error         string             `db:"stage_error"`
	ParentGroupID int64              `db:"stage_parent_group_id"`
	ErrIgnore     bool               `db:"stage_errignore"`
	ExitCode      int                `db:"stage_exit_code"`
	Machine       string             `db:"stage_machine"`
	OS            string             `db:"stage_os"`
	Arch          string             `db:"stage_arch"`
	Variant       string             `db:"stage_variant"`
	Kernel        string             `db:"stage_kernel"`
	Limit         int                `db:"stage_limit"`
	LimitRepo     int                `db:"stage_limit_repo"`
	Started       int64              `db:"stage_started"`
	Stopped       int64              `db:"stage_stopped"`
	Created       int64              `db:"stage_created"`
	Updated       int64              `db:"stage_updated"`
	Version       int64              `db:"stage_version"`
	OnSuccess     bool               `db:"stage_on_success"`
	OnFailure     bool               `db:"stage_on_failure"`
	DependsOn     sqlxtypes.JSONText `db:"stage_depends_on"`
	Labels        sqlxtypes.JSONText `db:"stage_labels"`
}

// NewStageStore returns a new StageStore.
func NewStageStore(db *sqlx.DB) store.StageStore {
	return &stageStore{
		db: db,
	}
}

type stageStore struct {
	db *sqlx.DB
}

// FindByNumber returns a stage given an execution ID and a stage number.
func (s *stageStore) FindByNumber(ctx context.Context, executionID int64, stageNum int) (*types.Stage, error) {
	const findQueryStmt = `
		SELECT` + stageColumns + `
		FROM stages
		WHERE stage_execution_id = $1 AND stage_number = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(stage)
	if err := db.GetContext(ctx, dst, findQueryStmt, executionID, stageNum); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find stage")
	}
	return mapInternalToStage(dst)
}

// Create adds a stage in the database.
func (s *stageStore) Create(ctx context.Context, st *types.Stage) error {
	const stageInsertStmt = `
		INSERT INTO stages (
			stage_execution_id
			,stage_repo_id
			,stage_number
			,stage_name
			,stage_kind
			,stage_type
			,stage_status
			,stage_error
			,stage_errignore
			,stage_exit_code
			,stage_machine
			,stage_parent_group_id
			,stage_os
			,stage_arch
			,stage_variant
			,stage_kernel
			,stage_limit
			,stage_limit_repo
			,stage_started
			,stage_stopped
			,stage_created
			,stage_updated
			,stage_version
			,stage_on_success
			,stage_on_failure
			,stage_depends_on
			,stage_labels
		) VALUES (
			:stage_execution_id
			,:stage_repo_id
			,:stage_number
			,:stage_name
			,:stage_kind
			,:stage_type
			,:stage_status
			,:stage_error
			,:stage_errignore
			,:stage_exit_code
			,:stage_machine
			,:stage_parent_group_id
			,:stage_os
			,:stage_arch
			,:stage_variant
			,:stage_kernel
			,:stage_limit
			,:stage_limit_repo
			,:stage_started
			,:stage_stopped
			,:stage_created
			,:stage_updated
			,:stage_version
			,:stage_on_success
			,:stage_on_failure
			,:stage_depends_on
			,:stage_labels

		) RETURNING stage_id`
	db := dbtx.GetAccessor(ctx, s.db)

	stage := mapStageToInternal(st)
	query, arg, err := db.BindNamed(stageInsertStmt, stage)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind stage object")
	}
	if err = db.QueryRowContext(ctx, query, arg...).Scan(&stage.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Stage query failed")
	}
	return nil
}

// ListWithSteps returns a stage with information about all its containing steps.
func (s *stageStore) ListWithSteps(ctx context.Context, executionID int64) ([]*types.Stage, error) {
	const queryNumberWithSteps = `
	SELECT` + stageColumns + "," + stepColumns + `
	FROM stages
	LEFT JOIN steps
		ON stages.stage_id=steps.step_stage_id
	WHERE stages.stage_execution_id = $1
	ORDER BY
	stage_id ASC
	,step_id ASC
	`
	db := dbtx.GetAccessor(ctx, s.db)

	rows, err := db.QueryContext(ctx, queryNumberWithSteps, executionID)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to query stages and steps")
	}
	return scanRowsWithSteps(rows)
}

// Find returns a stage given the stage ID.
func (s *stageStore) Find(ctx context.Context, stageID int64) (*types.Stage, error) {
	const queryFind = `
	SELECT` + stageColumns + `
	FROM stages
	WHERE stage_id = $1
	`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(stage)
	if err := db.GetContext(ctx, dst, queryFind, stageID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find stage")
	}
	return mapInternalToStage(dst)
}

// ListIncomplete returns a list of stages with a pending status.
func (s *stageStore) ListIncomplete(ctx context.Context) ([]*types.Stage, error) {
	const queryListIncomplete = `
	SELECT` + stageColumns + `
	FROM stages
	WHERE stage_status IN ('pending','running')
	ORDER BY stage_id ASC
	`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*stage{}
	if err := db.SelectContext(ctx, &dst, queryListIncomplete); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find incomplete stages")
	}
	// map stages list
	return mapInternalToStageList(dst)
}

// List returns a list of stages corresponding to an execution ID.
func (s *stageStore) List(ctx context.Context, executionID int64) ([]*types.Stage, error) {
	const queryList = `
	SELECT` + stageColumns + `
	FROM stages
	WHERE stage_execution_id = $1
	ORDER BY stage_number ASC
	`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*stage{}
	if err := db.SelectContext(ctx, &dst, queryList, executionID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find stages")
	}
	// map stages list
	return mapInternalToStageList(dst)
}

// Update tries to update a stage in the datastore and returns a locking error
// if it was unable to do so.
func (s *stageStore) Update(ctx context.Context, st *types.Stage) error {
	const stageUpdateStmt = `
	UPDATE stages
	SET
		stage_status = :stage_status
		,stage_machine = :stage_machine
		,stage_started = :stage_started
		,stage_stopped = :stage_stopped
		,stage_exit_code = :stage_exit_code
		,stage_updated = :stage_updated
		,stage_version = :stage_version
		,stage_error = :stage_error
		,stage_on_success = :stage_on_success
		,stage_on_failure = :stage_on_failure
		,stage_errignore = :stage_errignore
		,stage_depends_on = :stage_depends_on
		,stage_labels = :stage_labels
	WHERE stage_id = :stage_id AND stage_version = :stage_version - 1`
	updatedAt := time.Now()
	steps := st.Steps

	stage := mapStageToInternal(st)

	stage.Version++
	stage.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(stageUpdateStmt, stage)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind stage object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update stage")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	m, err := mapInternalToStage(stage)
	if err != nil {
		return fmt.Errorf("could not map stage object: %w", err)
	}
	*st = *m
	st.Version = stage.Version
	st.Updated = stage.Updated
	st.Steps = steps // steps is not mapped in database.
	return nil
}

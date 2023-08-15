// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/jmoiron/sqlx"
	sqlxtypes "github.com/jmoiron/sqlx/types"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
)

var _ store.StageStore = (*stageStore)(nil)

const (
	stageColumns = `
	stage_id
	,stage_execution_id
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
	ID          int64              `db:"stage_id"`
	ExecutionID int64              `db:"stage_execution_id"`
	Number      int                `db:"stage_number"`
	Name        string             `db:"stage_name"`
	Kind        string             `db:"stage_kind"`
	Type        string             `db:"stage_type"`
	Status      string             `db:"stage_status"`
	Error       string             `db:"stage_error"`
	ErrIgnore   bool               `db:"stage_errignore"`
	ExitCode    int                `db:"stage_exit_code"`
	Machine     string             `db:"stage_machine"`
	OS          string             `db:"stage_os"`
	Arch        string             `db:"stage_arch"`
	Variant     string             `db:"stage_variant"`
	Kernel      string             `db:"stage_kernel"`
	Limit       int                `db:"stage_limit"`
	LimitRepo   int                `db:"stage_limit_repo"`
	Started     int64              `db:"stage_started"`
	Stopped     int64              `db:"stage_stopped"`
	Created     int64              `db:"stage_created"`
	Updated     int64              `db:"stage_updated"`
	Version     int64              `db:"stage_version"`
	OnSuccess   bool               `db:"stage_on_success"`
	OnFailure   bool               `db:"stage_on_failure"`
	DependsOn   sqlxtypes.JSONText `db:"stage_depends_on"`
	Labels      sqlxtypes.JSONText `db:"stage_labels"`
}

// NewStageStore returns a new StageStore.
func NewStageStore(db *sqlx.DB) *stageStore {
	return &stageStore{
		db: db,
	}
}

type stageStore struct {
	db *sqlx.DB
}

// FindNumbers returns a stage given an execution ID and a stage number.
func (s *stageStore) FindNumber(ctx context.Context, executionID int64, stageNum int) (*types.Stage, error) {
	const findQueryStmt = `
		SELECT` + stageColumns + `
		FROM stages
		WHERE stage_execution_id = $1 AND stage_number = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(stage)
	if err := db.GetContext(ctx, dst, findQueryStmt, executionID, stageNum); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find stage")
	}
	return mapInternalToStage(dst)
}

// ListSteps returns a stage with information about all its containing steps.
func (s *stageStore) ListSteps(ctx context.Context, executionID int64) ([]*types.Stage, error) {
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
		return nil, database.ProcessSQLErrorf(err, "Failed to query stages and steps")
	}
	return scanRowsWithSteps(rows)
}

// Find returns a stage given the stage ID
func (s *stageStore) Find(ctx context.Context, stageID int64) (*types.Stage, error) {
	const queryFind = `
	SELECT` + stageColumns + `
	FROM stages
	WHERE stage_id = $1
	`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(stage)
	if err := db.GetContext(ctx, dst, queryFind, stageID); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find stage")
	}
	return mapInternalToStage(dst)
}

// ListIncomplete returns a list of stages with a pending status
// TODO: Check whether mysql needs a separate syntax
// ref: https://github.com/harness/drone/blob/master/store/stage/stage.go#L110.
func (s *stageStore) ListIncomplete(ctx context.Context) ([]*types.Stage, error) {
	const queryListIncomplete = `
	SELECT` + stageColumns + `
	FROM stages
	WHERE stage_status IN ('pending','running')
	ORDER BY stage_id ASC
	`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*stage{}
	if err := db.GetContext(ctx, dst, queryListIncomplete); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find stage")
	}
	// map stages list
	return mapInternalToStageList(dst)
}

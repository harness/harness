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

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	sqlxtypes "github.com/jmoiron/sqlx/types"
)

var _ store.StepStore = (*stepStore)(nil)

const (
	stepColumns = `
	step_id
	,step_stage_id
	,step_number
	,step_name
	,step_status
	,step_error
	,step_errignore
	,step_exit_code
	,step_started
	,step_stopped
	,step_version
	,step_depends_on
	,step_image
	,step_detached
	,step_schema
	`
)

type step struct {
	ID            int64              `db:"step_id"`
	StageID       int64              `db:"step_stage_id"`
	Number        int64              `db:"step_number"`
	ParentGroupID int64              `db:"step_parent_group_id"`
	Name          string             `db:"step_name"`
	Status        enum.CIStatus      `db:"step_status"`
	Error         string             `db:"step_error"`
	ErrIgnore     bool               `db:"step_errignore"`
	ExitCode      int                `db:"step_exit_code"`
	Started       int64              `db:"step_started"`
	Stopped       int64              `db:"step_stopped"`
	Version       int64              `db:"step_version"`
	DependsOn     sqlxtypes.JSONText `db:"step_depends_on"`
	Image         string             `db:"step_image"`
	Detached      bool               `db:"step_detached"`
	Schema        string             `db:"step_schema"`
}

// NewStepStore returns a new StepStore.
func NewStepStore(db *sqlx.DB) store.StepStore {
	return &stepStore{
		db: db,
	}
}

type stepStore struct {
	db *sqlx.DB
}

// FindByNumber returns a step given a stage ID and a step number.
func (s *stepStore) FindByNumber(ctx context.Context, stageID int64, stepNum int) (*types.Step, error) {
	const findQueryStmt = `
		SELECT` + stepColumns + `
		FROM steps
		WHERE step_stage_id = $1 AND step_number = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(step)
	if err := db.GetContext(ctx, dst, findQueryStmt, stageID, stepNum); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find step")
	}
	return mapInternalToStep(dst)
}

// Create creates a step.
func (s *stepStore) Create(ctx context.Context, step *types.Step) error {
	const stepInsertStmt = `
	INSERT INTO steps (
		step_stage_id
		,step_number
		,step_name
		,step_status
		,step_error
		,step_parent_group_id
		,step_errignore
		,step_exit_code
		,step_started
		,step_stopped
		,step_version
		,step_depends_on
		,step_image
		,step_detached
		,step_schema
	) VALUES (
		:step_stage_id
		,:step_number
		,:step_name
		,:step_status
		,:step_error
		,:step_parent_group_id
		,:step_errignore
		,:step_exit_code
		,:step_started
		,:step_stopped
		,:step_version
		,:step_depends_on
		,:step_image
		,:step_detached
		,:step_schema
	) RETURNING step_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(stepInsertStmt, mapStepToInternal(step))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind step object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&step.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Step query failed")
	}

	return nil
}

// Update tries to update a step in the datastore and returns a locking error
// if it was unable to do so.
func (s *stepStore) Update(ctx context.Context, e *types.Step) error {
	const stepUpdateStmt = `
	UPDATE steps
	SET
		step_name = :step_name
		,step_status = :step_status
		,step_error = :step_error
		,step_errignore = :step_errignore
		,step_exit_code = :step_exit_code
		,step_started = :step_started
		,step_stopped = :step_stopped
		,step_depends_on = :step_depends_on
		,step_image = :step_image
		,step_detached = :step_detached
		,step_schema = :step_schema
		,step_version = :step_version
	WHERE step_id = :step_id AND step_version = :step_version - 1`
	step := mapStepToInternal(e)

	step.Version++

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(stepUpdateStmt, step)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind step object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update step")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	m, err := mapInternalToStep(step)
	if err != nil {
		return fmt.Errorf("could not map step object: %w", err)
	}
	*e = *m
	e.Version = step.Version
	return nil
}

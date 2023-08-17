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
	ID        int64              `db:"step_id"`
	StageID   int64              `db:"step_stage_id"`
	Number    int64              `db:"step_number"`
	Name      string             `db:"step_name"`
	Status    string             `db:"step_status"`
	Error     string             `db:"step_error"`
	ErrIgnore bool               `db:"step_errignore"`
	ExitCode  int                `db:"step_exit_code"`
	Started   int64              `db:"step_started"`
	Stopped   int64              `db:"step_stopped"`
	Version   int64              `db:"step_version"`
	DependsOn sqlxtypes.JSONText `db:"step_depends_on"`
	Image     string             `db:"step_image"`
	Detached  bool               `db:"step_detached"`
	Schema    string             `db:"step_schema"`
}

// NewStepStore returns a new StepStore.
func NewStepStore(db *sqlx.DB) *stepStore {
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
		return nil, database.ProcessSQLErrorf(err, "Failed to find step")
	}
	return mapInternalToStep(dst)
}

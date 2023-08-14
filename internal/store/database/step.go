// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.StepStore = (*stepStore)(nil)

const (
	stepQueryBase = `
		SELECT` +
		stepColumns + `
		FROM steps`

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
	,step_created
	,step_updated
	,step_version
	,step_depends_on
	,step_image
	,step_detached
	,step_schema
	`
)

// NewStepStore returns a new StepStore.
func NewStepStore(db *sqlx.DB) *stepStore {
	return &stepStore{
		db: db,
	}
}

type stepStore struct {
	db *sqlx.DB
}

// FindNumber returns a step given a stage ID and a step number.
func (s *stepStore) FindNumber(ctx context.Context, stageID int64, stepNum int) (*types.Step, error) {
	const findQueryStmt = `
		SELECT` + stepColumns + `
		FROM steps
		WHERE step_stage_id = $1 AND step_number = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Step)
	if err := db.GetContext(ctx, dst, findQueryStmt, stageID, stepNum); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find step")
	}
	return dst, nil
}

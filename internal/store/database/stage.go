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

var _ store.StageStore = (*stageStore)(nil)

const (
	stageQueryBase = `
		SELECT` +
		stageColumns + `
		FROM stages`

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

	dst := new(types.Stage)
	if err := db.GetContext(ctx, dst, findQueryStmt, executionID, stageNum); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find stage")
	}
	return dst, nil
}

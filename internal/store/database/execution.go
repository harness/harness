// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/types"

	"github.com/jmoiron/sqlx"
)

var _ store.ExecutionStore = (*ExecutionStore)(nil)

// NewExecutionStore returns a new ExecutionStore.
func NewExecutionStore(db *sqlx.DB) *ExecutionStore {
	return &ExecutionStore{db}
}

// ExecutionStore implements a ExecutionStore backed by a relational
// database.
type ExecutionStore struct {
	db *sqlx.DB
}

// Find finds the execution by id.
func (s *ExecutionStore) Find(ctx context.Context, id int64) (*types.Execution, error) {
	dst := new(types.Execution)
	err := s.db.Get(dst, executionSelectID, id)
	return dst, err
}

// FindSlug finds the execution by pipeline id and slug.
func (s *ExecutionStore) FindSlug(ctx context.Context, id int64, slug string) (*types.Execution, error) {
	dst := new(types.Execution)
	err := s.db.Get(dst, executionSelectSlug, id, slug)
	return dst, err
}

// List returns a list of executions.
func (s *ExecutionStore) List(ctx context.Context, id int64, opts types.Params) ([]*types.Execution, error) {
	dst := []*types.Execution{}
	err := s.db.Select(&dst, executionSelect, id, limit(opts.Size), offset(opts.Page, opts.Size))
	return dst, err
}

// Create saves the execution details.
func (s *ExecutionStore) Create(ctx context.Context, execution *types.Execution) error {
	query, arg, err := s.db.BindNamed(executionInsert, execution)
	if err != nil {
		return err
	}
	return s.db.QueryRow(query, arg...).Scan(&execution.ID)
}

// Update updates the execution details.
func (s *ExecutionStore) Update(ctx context.Context, execution *types.Execution) error {
	query, arg, err := s.db.BindNamed(executionUpdate, execution)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(query, arg...)
	return err
}

// Delete deletes the execution.
func (s *ExecutionStore) Delete(ctx context.Context, execution *types.Execution) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// delete the execution
	if _, err := tx.Exec(executionDelete, execution.ID); err != nil {
		return err
	}
	return tx.Commit()
}

const executionBase = `
SELECT
 execution_id
,execution_pipeline_id
,execution_slug
,execution_name
,execution_desc
,execution_created
,execution_updated
FROM executions
`

const executionSelect = executionBase + `
WHERE execution_pipeline_id = $1
ORDER BY execution_name ASC
LIMIT $2 OFFSET $3
`

const executionSelectID = executionBase + `
WHERE execution_id = $1
`

const executionSelectSlug = executionBase + `
WHERE execution_pipeline_id = $1
  AND execution_slug = $2
`

const executionInsert = `
INSERT INTO executions (
 execution_pipeline_id
,execution_slug
,execution_name
,execution_desc
,execution_created
,execution_updated
) values (
 :execution_pipeline_id
,:execution_slug
,:execution_name
,:execution_desc
,:execution_created
,:execution_updated
) RETURNING execution_id
`

const executionUpdate = `
UPDATE executions
SET
 execution_name    = :execution_name
,execution_desc    = :execution_desc
,execution_updated = :execution_updated
WHERE execution_id = :execution_id
`

const executionDelete = `
DELETE FROM executions
WHERE execution_id = $1
`

const executionDeletePipeline = `
DELETE FROM executions
WHERE execution_pipeline_id = $1
`

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

var _ store.PipelineStore = (*PipelineStore)(nil)

// NewPipelineStore returns a new PipelinetStore.
func NewPipelineStore(db *sqlx.DB) *PipelineStore {
	return &PipelineStore{db}
}

// PipelineStore implements a PipelineStore backed by a
// relational database.
type PipelineStore struct {
	db *sqlx.DB
}

// Find finds the pipeline by id.
func (s *PipelineStore) Find(ctx context.Context, id int64) (*types.Pipeline, error) {
	dst := new(types.Pipeline)
	err := s.db.Get(dst, pipelineSelectID, id)
	return dst, err
}

// FindToken finds the pipeline by token.
func (s *PipelineStore) FindToken(ctx context.Context, token string) (*types.Pipeline, error) {
	dst := new(types.Pipeline)
	err := s.db.Get(dst, pipelineSelectToken, token)
	return dst, err
}

// FindSlug finds the pipeline by slug.
func (s *PipelineStore) FindSlug(ctx context.Context, slug string) (*types.Pipeline, error) {
	dst := new(types.Pipeline)
	err := s.db.Get(dst, pipelineSelectSlug, slug)
	return dst, err
}

// List returns a list of pipelines by user.
func (s *PipelineStore) List(ctx context.Context, user int64, opts types.Params) ([]*types.Pipeline, error) {
	dst := []*types.Pipeline{}
	err := s.db.Select(&dst, pipelineSelect, limit(opts.Size), offset(opts.Page, opts.Size))
	return dst, err
}

// Create saves the pipeline details.
func (s *PipelineStore) Create(ctx context.Context, pipeline *types.Pipeline) error {
	query, arg, err := s.db.BindNamed(pipelineInsert, pipeline)
	if err != nil {
		return err
	}
	return s.db.QueryRow(query, arg...).Scan(&pipeline.ID)
}

// Update updates the pipeline details.
func (s *PipelineStore) Update(ctx context.Context, pipeline *types.Pipeline) error {
	query, arg, err := s.db.BindNamed(pipelineUpdate, pipeline)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(query, arg...)
	return err
}

// Delete deletes the pipeline.
func (s *PipelineStore) Delete(ctx context.Context, pipeline *types.Pipeline) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// pleae note that we are aware of foreign keys and
	// cascading deletes, however, we chose to implement
	// this logic in the application code in the event we
	// want to leverage citus postgres.
	//
	// to future developers: feel free to remove and
	// replace with foreign keys and cascading deletes
	// at your discretion.

	// delete the executions associated with the pipeline
	if _, err := tx.Exec(executionDeletePipeline, pipeline.ID); err != nil {
		return err
	}
	// delete the pipeline
	if _, err := tx.Exec(pipelineDelete, pipeline.ID); err != nil {
		return err
	}
	return tx.Commit()
}

const pipelineBase = `
SELECT
 pipeline_id
,pipeline_name
,pipeline_slug
,pipeline_desc
,pipeline_token
,pipeline_active
,pipeline_created
,pipeline_updated
FROM pipelines
`

const pipelineSelect = pipelineBase + `
ORDER BY pipeline_slug
LIMIT $1 OFFSET $2
`

const pipelineSelectID = pipelineBase + `
WHERE pipeline_id = $1
`

const pipelineSelectToken = pipelineBase + `
WHERE pipeline_token = $1
`

const pipelineSelectSlug = pipelineBase + `
WHERE pipeline_slug = $1
`

const pipelineDelete = `
DELETE FROM pipelines
WHERE pipeline_id = $1
`

const pipelineInsert = `
INSERT INTO pipelines (
 pipeline_name
,pipeline_slug
,pipeline_desc
,pipeline_token
,pipeline_active
,pipeline_created
,pipeline_updated
) values (
 :pipeline_name
,:pipeline_slug
,:pipeline_desc
,:pipeline_token
,:pipeline_active
,:pipeline_created
,:pipeline_updated
) RETURNING pipeline_id
`

const pipelineUpdate = `
UPDATE pipelines
SET
 pipeline_name      = :pipeline_name
,pipeline_desc      = :pipeline_desc
,pipeline_active    = :pipeline_active
,pipeline_updated   = :pipeline_updated
WHERE pipeline_id = :pipeline_id
`

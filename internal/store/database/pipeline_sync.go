// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/bradrydzewski/my-app/internal/store"
	"github.com/bradrydzewski/my-app/internal/store/database/mutex"
	"github.com/bradrydzewski/my-app/types"
)

var _ store.PipelineStore = (*PipelineStoreSync)(nil)

// NewPipelineStoreSync returns a new PipelineStoreSync.
func NewPipelineStoreSync(store *PipelineStore) *PipelineStoreSync {
	return &PipelineStoreSync{base: store}
}

// PipelineStoreSync synronizes read and write access to the
// pipeline store. This prevents race conditions when the database
// type is sqlite3.
type PipelineStoreSync struct{ base *PipelineStore }

// Find finds the pipeline by id.
func (s *PipelineStoreSync) Find(ctx context.Context, id int64) (*types.Pipeline, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindToken finds the pipeline by token.
func (s *PipelineStoreSync) FindToken(ctx context.Context, token string) (*types.Pipeline, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindToken(ctx, token)
}

// FindSlug finds the pipeline by slug.
func (s *PipelineStoreSync) FindSlug(ctx context.Context, slug string) (*types.Pipeline, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindSlug(ctx, slug)
}

// List returns a list of pipelines by user.
func (s *PipelineStoreSync) List(ctx context.Context, id int64, opts types.Params) ([]*types.Pipeline, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, id, opts)
}

// Create saves the pipeline details.
func (s *PipelineStoreSync) Create(ctx context.Context, pipeline *types.Pipeline) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Create(ctx, pipeline)
}

// Update updates the pipeline details.
func (s *PipelineStoreSync) Update(ctx context.Context, pipeline *types.Pipeline) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Update(ctx, pipeline)
}

// Delete deletes the pipeline.
func (s *PipelineStoreSync) Delete(ctx context.Context, pipeline *types.Pipeline) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Delete(ctx, pipeline)
}

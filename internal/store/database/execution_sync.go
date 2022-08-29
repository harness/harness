// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/mutex"
	"github.com/harness/gitness/types"
)

var _ store.ExecutionStore = (*ExecutionStoreSync)(nil)

// NewExecutionStoreSync returns a new ExecutionStoreSync.
func NewExecutionStoreSync(store *ExecutionStore) *ExecutionStoreSync {
	return &ExecutionStoreSync{base: store}
}

// ExecutionStoreSync synronizes read and write access to the
// execution store. This prevents race conditions when the database
// type is sqlite3.
type ExecutionStoreSync struct{ base *ExecutionStore }

// Find finds the execution by id.
func (s *ExecutionStoreSync) Find(ctx context.Context, id int64) (*types.Execution, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindSlug finds the execution by pipeline id and slug.
func (s *ExecutionStoreSync) FindSlug(ctx context.Context, id int64, slug string) (*types.Execution, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindSlug(ctx, id, slug)
}

// List returns a list of executions.
func (s *ExecutionStoreSync) List(ctx context.Context, id int64, opts types.Params) ([]*types.Execution, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, id, opts)
}

// Create saves the execution details.
func (s *ExecutionStoreSync) Create(ctx context.Context, execution *types.Execution) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Create(ctx, execution)
}

// Update updates the execution details.
func (s *ExecutionStoreSync) Update(ctx context.Context, execution *types.Execution) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Update(ctx, execution)
}

// Delete deletes the execution.
func (s *ExecutionStoreSync) Delete(ctx context.Context, execution *types.Execution) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Delete(ctx, execution)
}

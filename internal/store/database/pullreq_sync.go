// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/mutex"
	"github.com/harness/gitness/types"
)

var _ store.PullReqStore = (*PullReqStoreSync)(nil)

// NewPullReqStoreSync returns a new PullReqStoreSync.
func NewPullReqStoreSync(base *PullReqStore) *PullReqStoreSync {
	return &PullReqStoreSync{
		base: base,
	}
}

// PullReqStoreSync synchronizes read and write access to the
// pull request store. This prevents race conditions when the database
// type is sqlite3.
type PullReqStoreSync struct {
	base *PullReqStore
}

// Find finds the pull request by id.
func (s *PullReqStoreSync) Find(ctx context.Context, id int64) (*types.PullReqInfo, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindByNumber finds the pull request by repo ID and pull request number.
func (s *PullReqStoreSync) FindByNumber(ctx context.Context, repoID, number int64) (*types.PullReqInfo, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindByNumber(ctx, repoID, number)
}

// Create creates a new pull request.
func (s *PullReqStoreSync) Create(ctx context.Context, pullReq *types.PullReq) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Create(ctx, pullReq)
}

// Update updates the pull request.
func (s *PullReqStoreSync) Update(ctx context.Context, pullReq *types.PullReq) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Update(ctx, pullReq)
}

// Delete the pull request.
func (s *PullReqStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Delete(ctx, id)
}

// LastNumber return the number of the most recent pull request.
func (s *PullReqStoreSync) LastNumber(ctx context.Context, repoID int64) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.LastNumber(ctx, repoID)
}

// Count of pull requests for a repo.
func (s *PullReqStoreSync) Count(ctx context.Context, repoID int64, opts *types.PullReqFilter) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx, repoID, opts)
}

// List returns a list of pull requests for a repo.
func (s *PullReqStoreSync) List(
	ctx context.Context,
	repoID int64,
	opts *types.PullReqFilter,
) ([]*types.PullReqInfo, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, repoID, opts)
}

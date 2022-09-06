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

var _ store.RepoStore = (*RepoStoreSync)(nil)

// Returns a new RepoStoreSync.
func NewRepoStoreSync(base *RepoStore) *RepoStoreSync {
	return &RepoStoreSync{base}
}

// RepoStoreSync synronizes read and write access to the
// repo store. This prevents race conditions when the database
// type is sqlite3.
type RepoStoreSync struct {
	base *RepoStore
}

// Finds the repo by id.
func (s *RepoStoreSync) Find(ctx context.Context, id int64) (*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// Finds the repo by the full qualified repo name.
func (s *RepoStoreSync) FindFqn(ctx context.Context, fqn string) (*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindFqn(ctx, fqn)
}

// Creates a new repo
func (s *RepoStoreSync) Create(ctx context.Context, repo *types.Repository) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Create(ctx, repo)
}

// Updates the repo details.
func (s *RepoStoreSync) Update(ctx context.Context, repo *types.Repository) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Update(ctx, repo)
}

// Deletes the repo.
func (s *RepoStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Delete(ctx, id)
}

// List returns a list of repos in a space.
func (s *RepoStoreSync) List(ctx context.Context, spaceId int64, opts types.RepoFilter) ([]*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, spaceId, opts)
}

// Count of repos in a space.
func (s *RepoStoreSync) Count(ctx context.Context, spaceId int64) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx, spaceId)
}

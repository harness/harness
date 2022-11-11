// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"strconv"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/mutex"
	"github.com/harness/gitness/types"
)

var _ store.RepoStore = (*RepoStoreSync)(nil)

// NewRepoStoreSync returns a new RepoStoreSync.
func NewRepoStoreSync(base *RepoStore) *RepoStoreSync {
	return &RepoStoreSync{base}
}

// RepoStoreSync synchronizes read and write access to the
// repo store. This prevents race conditions when the database
// type is sqlite3.
type RepoStoreSync struct {
	base *RepoStore
}

// Find the repo by id.
func (s *RepoStoreSync) Find(ctx context.Context, id int64) (*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindByPath finds the repo by path.
func (s *RepoStoreSync) FindByPath(ctx context.Context, path string) (*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindByPath(ctx, path)
}

// FindRepoFromRef finds the repo by path or ref.
func (s *RepoStoreSync) FindRepoFromRef(ctx context.Context, repoRef string) (*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	id, err := strconv.ParseInt(repoRef, 10, 64)
	if err == nil {
		return s.Find(ctx, id)
	}

	return s.FindByPath(ctx, repoRef)
}

// Create a new repository.
func (s *RepoStoreSync) Create(ctx context.Context, repo *types.Repository) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Create(ctx, repo)
}

// Move an existing repo.
func (s *RepoStoreSync) Move(ctx context.Context, principalID int64, id int64, newParentID int64,
	newName string, keepAsAlias bool) (*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Move(ctx, principalID, id, newParentID, newName, keepAsAlias)
}

// Update the repo details.
func (s *RepoStoreSync) Update(ctx context.Context, repo *types.Repository) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Update(ctx, repo)
}

// Delete the repository.
func (s *RepoStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Delete(ctx, id)
}

// Count of repos in a space.
func (s *RepoStoreSync) Count(ctx context.Context, parentID int64, opts *types.RepoFilter) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx, parentID, opts)
}

// List returns a list of repos in a space.
func (s *RepoStoreSync) List(ctx context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, parentID, opts)
}

// CountPaths returns a count of all paths of a repo.
func (s *RepoStoreSync) CountPaths(ctx context.Context, id int64, opts *types.PathFilter) (int64, error) {
	return s.base.CountPaths(ctx, id, opts)
}

// ListPaths returns a list of all paths of a repo.
func (s *RepoStoreSync) ListPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error) {
	return s.base.ListPaths(ctx, id, opts)
}

// CreatePath an alias for a repository.
func (s *RepoStoreSync) CreatePath(ctx context.Context, repoID int64, params *types.PathParams) (*types.Path, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.CreatePath(ctx, repoID, params)
}

// DeletePath an alias of a repository.
func (s *RepoStoreSync) DeletePath(ctx context.Context, repoID int64, pathID int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.DeletePath(ctx, repoID, pathID)
}

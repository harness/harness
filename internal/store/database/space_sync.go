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

var _ store.SpaceStore = (*SpaceStoreSync)(nil)

// NewSpaceStoreSync returns a new SpaceStoreSync.
func NewSpaceStoreSync(base *SpaceStore) *SpaceStoreSync {
	return &SpaceStoreSync{base}
}

// SpaceStoreSync synchronizes read and write access to the
// space store. This prevents race conditions when the database
// type is sqlite3.
type SpaceStoreSync struct {
	base *SpaceStore
}

// Find the space by id.
func (s *SpaceStoreSync) Find(ctx context.Context, id int64) (*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// FindByPath find the space by path.
func (s *SpaceStoreSync) FindByPath(ctx context.Context, path string) (*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindByPath(ctx, path)
}

// Create a new space.
func (s *SpaceStoreSync) Create(ctx context.Context, space *types.Space) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Create(ctx, space)
}

// Move moves an existing space.
func (s *SpaceStoreSync) Move(ctx context.Context, principalID int64, spaceID int64, newParentID int64, newName string,
	keepAsAlias bool) (*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Move(ctx, principalID, spaceID, newParentID, newName, keepAsAlias)
}

// Update the space details.
func (s *SpaceStoreSync) Update(ctx context.Context, space *types.Space) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Update(ctx, space)
}

// Delete the space.
func (s *SpaceStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Delete(ctx, id)
}

// Count the child spaces of a space.
func (s *SpaceStoreSync) Count(ctx context.Context, id int64, opts *types.SpaceFilter) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx, id, opts)
}

// List returns a list of spaces under the parent space.
func (s *SpaceStoreSync) List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, id, opts)
}

// CountPaths returns a count of all paths of a space.
func (s *SpaceStoreSync) CountPaths(ctx context.Context, id int64, opts *types.PathFilter) (int64, error) {
	return s.base.CountPaths(ctx, id, opts)
}

// ListPaths returns a list of all paths of a space.
func (s *SpaceStoreSync) ListPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error) {
	return s.base.ListPaths(ctx, id, opts)
}

// CreatePath a path for a space.
func (s *SpaceStoreSync) CreatePath(ctx context.Context, spaceID int64, params *types.PathParams) (*types.Path, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.CreatePath(ctx, spaceID, params)
}

// DeletePath a path of a space.
func (s *SpaceStoreSync) DeletePath(ctx context.Context, spaceID int64, pathID int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.DeletePath(ctx, spaceID, pathID)
}

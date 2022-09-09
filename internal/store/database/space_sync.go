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

var _ store.SpaceStore = (*SpaceStoreSync)(nil)

// Returns a new SpaceStore.
func NewSpaceStoreSync(base *SpaceStore) *SpaceStoreSync {
	return &SpaceStoreSync{base}
}

// SpaceStoreSync synronizes read and write access to the
// space store. This prevents race conditions when the database
// type is sqlite3.
type SpaceStoreSync struct {
	base *SpaceStore
}

// Finds the space by id.
func (s *SpaceStoreSync) Find(ctx context.Context, id int64) (*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// Finds the space by path.
func (s *SpaceStoreSync) FindByPath(ctx context.Context, path string) (*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.FindByPath(ctx, path)
}

// Creates a new space
func (s *SpaceStoreSync) Create(ctx context.Context, space *types.Space) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Create(ctx, space)
}

// Moves an existing space.
func (s *SpaceStoreSync) Move(ctx context.Context, userId int64, spaceId int64, newParentId int64, newName string, keepAsAlias bool) (*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Move(ctx, userId, spaceId, newParentId, newName, keepAsAlias)
}

// Updates the space details.
func (s *SpaceStoreSync) Update(ctx context.Context, space *types.Space) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Update(ctx, space)
}

// Deletes the space.
func (s *SpaceStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Delete(ctx, id)
}

// Count the child spaces of a space.
func (s *SpaceStoreSync) Count(ctx context.Context, id int64) (int64, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Count(ctx, id)
}

// List returns a list of spaces under the parent space.
func (s *SpaceStoreSync) List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, id, opts)
}

// List returns a list of all paths of a space.
func (s *SpaceStoreSync) ListAllPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error) {
	return s.base.ListAllPaths(ctx, id, opts)
}

// Create a path for a space.
func (s *SpaceStoreSync) CreatePath(ctx context.Context, spaceId int64, params *types.PathParams) (*types.Path, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.CreatePath(ctx, spaceId, params)
}

// Delete a path of a space.
func (s *SpaceStoreSync) DeletePath(ctx context.Context, spaceId int64, pathId int64) error {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.DeletePath(ctx, spaceId, pathId)
}

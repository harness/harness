// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/pkg/errors"

	"github.com/jmoiron/sqlx"
)

var _ store.SpaceStore = (*SpaceStore)(nil)

// NewSpaceStore returns a new SpaceStore.
func NewSpaceStore(db *sqlx.DB) *SpaceStore {
	return &SpaceStore{db}
}

// SpaceStore implements a SpaceStore backed by a relational database.
type SpaceStore struct {
	db *sqlx.DB
}

// Find the space by id.
func (s *SpaceStore) Find(ctx context.Context, id int64) (*types.Space, error) {
	dst := new(types.Space)
	if err := s.db.GetContext(ctx, dst, spaceSelectByID, id); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

// FindByPath finds the space by path.
func (s *SpaceStore) FindByPath(ctx context.Context, path string) (*types.Space, error) {
	dst := new(types.Space)
	if err := s.db.GetContext(ctx, dst, spaceSelectByPath, path); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

// Create a new space.
func (s *SpaceStore) Create(ctx context.Context, space *types.Space) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// insert space first so we get id
	query, arg, err := s.db.BindNamed(spaceInsert, space)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind space object")
	}

	if err = tx.QueryRow(query, arg...).Scan(&space.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	// Get path (get parent if needed)
	path := space.PathName
	if space.ParentID > 0 {
		var parentPath *types.Path
		parentPath, err = FindPathTx(ctx, tx, enum.PathTargetTypeSpace, space.ParentID)
		if err != nil {
			return errors.Wrap(err, "Failed to find path of parent space")
		}

		// all existing paths are valid, space name is assumed to be valid.
		path = paths.Concatinate(parentPath.Value, space.PathName)
	}

	// create path only once we know the id of the space
	p := &types.Path{
		TargetType: enum.PathTargetTypeSpace,
		TargetID:   space.ID,
		IsAlias:    false,
		Value:      path,
		CreatedBy:  space.CreatedBy,
		Created:    space.Created,
		Updated:    space.Updated,
	}
	err = CreatePathTx(ctx, s.db, tx, p)
	if err != nil {
		return errors.Wrap(err, "Failed to create primary path of space")
	}

	// commit
	if err = tx.Commit(); err != nil {
		return processSQLErrorf(err, "Failed to commit transaction")
	}

	// update path in space object
	space.Path = p.Value

	return nil
}

// Move moves an existing space.
func (s *SpaceStore) Move(ctx context.Context, principalID int64, spaceID int64, newParentID int64, newName string,
	keepAsAlias bool) (*types.Space, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// always get currentpath (either it didn't change or we need to for validation)
	currentPath, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, spaceID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find the primary path of the space")
	}

	// get path of new parent if needed
	newPath := newName
	if newParentID > 0 {
		// get path of new parent space
		var spacePath *types.Path
		spacePath, err = FindPathTx(ctx, tx, enum.PathTargetTypeSpace, newParentID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to find the primary path of the new parent space")
		}

		newPath = paths.Concatinate(spacePath.Value, newName)
	}

	/*
	 * IMPORTANT
	 *   To avoid cycles in the primary graph, we have to ensure that the old path isn't a parent of the new path.
	 */
	if newPath == currentPath.Value {
		return nil, store.ErrNoChangeInRequestedMove
	} else if strings.HasPrefix(newPath, currentPath.Value+types.PathSeparator) {
		return nil, store.ErrIllegalMoveCyclicHierarchy
	}

	p := &types.Path{
		TargetType: enum.PathTargetTypeSpace,
		TargetID:   spaceID,
		IsAlias:    false,
		Value:      newPath,
		CreatedBy:  principalID,
		Created:    time.Now().UnixMilli(),
		Updated:    time.Now().UnixMilli(),
	}

	// replace the primary path (also updates all child primary paths)
	if err = ReplacePathTx(ctx, s.db, tx, p, keepAsAlias); err != nil {
		return nil, errors.Wrap(err, "Failed to update the primary path of the space")
	}

	// Update the space itself
	if _, err = tx.ExecContext(ctx, spaceUpdateNameAndParentID, newName, newParentID, spaceID); err != nil {
		return nil, processSQLErrorf(err, "Query for renaming and updating the parent id failed")
	}

	// TODO: return space as part of rename operation
	dst := new(types.Space)
	if err = tx.GetContext(ctx, dst, spaceSelectByID, spaceID); err != nil {
		return nil, processSQLErrorf(err, "Select query to get the space's latest state failed")
	}

	// commit
	if err = tx.Commit(); err != nil {
		return nil, processSQLErrorf(err, "Failed to commit transaction")
	}

	return dst, nil
}

// Updates the space details.
func (s *SpaceStore) Update(ctx context.Context, space *types.Space) error {
	query, arg, err := s.db.BindNamed(spaceUpdate, space)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind space object")
	}

	if _, err = s.db.ExecContext(ctx, query, arg...); err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	return nil
}

// Deletes the space.
func (s *SpaceStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// get primary path
	path, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, id)
	if err != nil {
		return errors.Wrap(err, "Failed to find the primary path of the space")
	}

	// Get child count and ensure there are none
	count, err := CountPrimaryChildPathsTx(ctx, tx, path.Value)
	if err != nil {
		return fmt.Errorf("child count error: %w", err)
	}
	if count > 0 {
		// TODO: still returns 500
		return store.ErrSpaceWithChildsCantBeDeleted
	}

	// delete all paths
	err = DeleteAllPaths(ctx, tx, enum.PathTargetTypeSpace, id)
	if err != nil {
		return errors.Wrap(err, "Failed to delete all paths of the space")
	}

	// delete the space
	if _, err = tx.Exec(spaceDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}

	if err = tx.Commit(); err != nil {
		return processSQLErrorf(err, "Failed to commit transaction")
	}

	return nil
}

// Count the child spaces of a space.
func (s *SpaceStore) Count(ctx context.Context, id int64, opts *types.SpaceFilter) (int64, error) {
	stmt := builder.
		Select("count(*)").
		From("spaces").
		Where("space_parentId = ?", id)

	if opts.Query != "" {
		stmt = stmt.Where("space_pathName LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	var count int64
	err = s.db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// List returns a list of spaces under the parent space.
func (s *SpaceStore) List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error) {
	dst := []*types.Space{}

	stmt := builder.
		Select("spaces.*,path_value AS space_path").
		From("spaces").
		InnerJoin("paths ON spaces.space_id=paths.path_targetId AND paths.path_targetType='space' AND paths.path_isAlias=0").
		Where("space_parentId = ?", fmt.Sprint(id))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	if opts.Query != "" {
		stmt = stmt.Where("space_pathName LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
	}

	switch opts.Sort {
	case enum.SpaceAttrName, enum.SpaceAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("space_name COLLATE NOCASE " + opts.Order.String())
	case enum.SpaceAttrCreated:
		stmt = stmt.OrderBy("space_created " + opts.Order.String())
	case enum.SpaceAttrUpdated:
		stmt = stmt.OrderBy("space_updated " + opts.Order.String())
	case enum.SpaceAttrPathName:
		stmt = stmt.OrderBy("space_pathName COLLATE NOCASE " + opts.Order.String())
	case enum.SpaceAttrPath:
		stmt = stmt.OrderBy("space_path COLLATE NOCASE " + opts.Order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = s.db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// CountPaths returns a count of all paths of a space.
func (s *SpaceStore) CountPaths(ctx context.Context, id int64, opts *types.PathFilter) (int64, error) {
	return CountPaths(ctx, s.db, enum.PathTargetTypeSpace, id, opts)
}

// ListPaths returns a list of all paths of a space.
func (s *SpaceStore) ListPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error) {
	return ListPaths(ctx, s.db, enum.PathTargetTypeSpace, id, opts)
}

// CreatePath creates an alias for a space.
func (s *SpaceStore) CreatePath(ctx context.Context, spaceID int64, params *types.PathParams) (*types.Path, error) {
	p := &types.Path{
		TargetType: enum.PathTargetTypeSpace,
		TargetID:   spaceID,
		IsAlias:    true,

		// get remaining infor from params
		Value:     params.Path,
		CreatedBy: params.CreatedBy,
		Created:   params.Created,
		Updated:   params.Updated,
	}

	return p, CreateAliasPath(ctx, s.db, p)
}

// DeletePath an alias of a space.
func (s *SpaceStore) DeletePath(ctx context.Context, spaceID int64, pathID int64) error {
	return DeletePath(ctx, s.db, pathID)
}

const spaceSelectBase = `
SELECT
 space_id
,space_pathName
,paths.path_value AS space_path
,space_parentId
,space_name
,space_description
,space_isPublic
,space_createdBy
,space_created
,space_updated
`

const spaceSelectBaseWithJoin = spaceSelectBase + `
FROM spaces
INNER JOIN paths
ON spaces.space_id=paths.path_targetId AND paths.path_targetType='space' AND paths.path_isAlias=0
`

const spaceSelectByID = spaceSelectBaseWithJoin + `
WHERE space_id = $1
`

const spaceSelectByPath = spaceSelectBase + `
FROM paths paths1
INNER JOIN spaces ON spaces.space_id=paths1.path_targetId AND paths1.path_targetType='space' AND paths1.path_value = $1
INNER JOIN paths ON spaces.space_id=paths.path_targetId AND paths.path_targetType='space' AND paths.path_isAlias=0
`

const spaceDelete = `
DELETE FROM spaces
WHERE space_id = $1
`

// TODO: do we have to worry about SQL injection for description?
const spaceInsert = `
INSERT INTO spaces (
    space_pathName
   ,space_parentId
   ,space_name
   ,space_description
   ,space_isPublic
   ,space_createdBy
   ,space_created
   ,space_updated
) values (
   :space_pathName
   ,:space_parentId
   ,:space_name
   ,:space_description
   ,:space_isPublic
   ,:space_createdBy
   ,:space_created
   ,:space_updated
) RETURNING space_id
`

const spaceUpdate = `
UPDATE spaces
SET
space_name   = :space_name
,space_description  = :space_description
,space_isPublic     = :space_isPublic
,space_updated      = :space_updated
WHERE space_id = :space_id
`

const spaceUpdateNameAndParentID = `
UPDATE spaces
SET
space_pathName = $1
,space_parentId = $2
WHERE space_id = $3
`

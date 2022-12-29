// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.SpaceStore = (*SpaceStore)(nil)

// NewSpaceStore returns a new SpaceStore.
func NewSpaceStore(db *sqlx.DB, pathTransformation store.PathTransformation) *SpaceStore {
	return &SpaceStore{
		db:                 db,
		pathTransformation: pathTransformation,
	}
}

// SpaceStore implements a SpaceStore backed by a relational database.
type SpaceStore struct {
	db                 *sqlx.DB
	pathTransformation store.PathTransformation
}

// Find the space by id.
func (s *SpaceStore) Find(ctx context.Context, id int64) (*types.Space, error) {
	dst := new(types.Space)
	if err := s.db.GetContext(ctx, dst, spaceSelectByID, id); err != nil {
		return nil, processSQLErrorf(err, "Failed to find space")
	}
	return dst, nil
}

// FindByPath finds the space by path.
func (s *SpaceStore) FindByPath(ctx context.Context, path string) (*types.Space, error) {
	// ensure we transform path before searching (otherwise casing might be wrong)
	pathUnique, err := s.pathTransformation(path)
	if err != nil {
		return nil, fmt.Errorf("failed to transform path '%s': %w", path, err)
	}

	dst := new(types.Space)
	if err = s.db.GetContext(ctx, dst, spaceSelectByPathUnique, pathUnique); err != nil {
		return nil, processSQLErrorf(err, "Failed to find space by path")
	}
	return dst, nil
}

func (s *SpaceStore) FindSpaceFromRef(ctx context.Context, spaceRef string) (*types.Space, error) {
	// check if ref is space ID - ASSUMPTION: digit only is no valid space name
	id, err := strconv.ParseInt(spaceRef, 10, 64)
	if err == nil {
		return s.Find(ctx, id)
	}

	return s.FindByPath(ctx, spaceRef)
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
	path := space.UID
	if space.ParentID > 0 {
		var parentPath *types.Path
		parentPath, err = FindPathTx(ctx, tx, enum.PathTargetTypeSpace, space.ParentID)
		if err != nil {
			return errors.Wrap(err, "Failed to find path of parent space")
		}

		// all existing paths are valid, space uid is assumed to be valid.
		path = paths.Concatinate(parentPath.Value, space.UID)
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
	err = CreatePathTx(ctx, s.db, tx, p, s.pathTransformation)
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
func (s *SpaceStore) Move(ctx context.Context, principalID int64, id int64, newParentID int64, newName string,
	keepAsAlias bool) (*types.Space, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// always get currentpath (either it didn't change or we need to for validation)
	currentPath, err := FindPathTx(ctx, tx, enum.PathTargetTypeSpace, id)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find the primary path of the space")
	}

	// get path of new parent if needed
	newPathValue := newName
	if newParentID > 0 {
		// get path of new parent space
		var spacePath *types.Path
		spacePath, err = FindPathTx(ctx, tx, enum.PathTargetTypeSpace, newParentID)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to find the primary path of the new parent space")
		}

		newPathValue = paths.Concatinate(spacePath.Value, newName)
	}

	// path is exactly the same => nothing to do
	if newPathValue == currentPath.Value {
		return nil, store.ErrNoChangeInRequestedMove
	}

	p := &types.Path{
		TargetType: enum.PathTargetTypeSpace,
		TargetID:   id,
		IsAlias:    false,
		Value:      newPathValue,
		CreatedBy:  principalID,
		Created:    time.Now().UnixMilli(),
		Updated:    time.Now().UnixMilli(),
	}

	// replace the primary path (also updates all child primary paths)
	if err = ReplacePathTx(ctx, s.db, tx, p, keepAsAlias, s.pathTransformation); err != nil {
		return nil, errors.Wrap(err, "Failed to update the primary path of the space")
	}

	// Update the space itself
	if _, err = tx.ExecContext(ctx, spaceUpdateUIDAndParentID, newName, newParentID, id); err != nil {
		return nil, processSQLErrorf(err, "Query for renaming and updating the parent id failed")
	}

	// TODO: return space as part of rename operation
	dst := new(types.Space)
	if err = tx.GetContext(ctx, dst, spaceSelectByID, id); err != nil {
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
	count, err := CountPrimaryChildPathsTx(ctx, tx, path.Value, s.pathTransformation)
	if err != nil {
		return fmt.Errorf("child count error: %w", err)
	}
	if count > 0 {
		// TODO: still returns 500
		return store.ErrSpaceWithChildsCantBeDeleted
	}

	// delete all paths
	if err = DeleteAllPaths(ctx, tx, enum.PathTargetTypeSpace, id); err != nil {
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
		Where("space_parent_id = ?", id)

	if opts.Query != "" {
		stmt = stmt.Where("space_uid LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
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
		InnerJoin(`paths ON spaces.space_id=paths.path_target_id AND paths.path_target_type='space'
		  AND paths.path_is_alias=false`).
		Where("space_parent_id = ?", fmt.Sprint(id))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	if opts.Query != "" {
		stmt = stmt.Where("space_uid LIKE ?", fmt.Sprintf("%%%s%%", opts.Query))
	}

	switch opts.Sort {
	case enum.SpaceAttrUID, enum.SpaceAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("space_uid " + opts.Order.String())
		//TODO: Postgres does not support COLLATE NOCASE for UTF8
		// stmt = stmt.OrderBy("space_uid COLLATE NOCASE " + opts.Order.String())
	case enum.SpaceAttrCreated:
		stmt = stmt.OrderBy("space_created " + opts.Order.String())
	case enum.SpaceAttrUpdated:
		stmt = stmt.OrderBy("space_updated " + opts.Order.String())
	case enum.SpaceAttrPath:
		stmt = stmt.OrderBy("space_path " + opts.Order.String())
		//TODO: Postgres does not support COLLATE NOCASE for UTF8
		// stmt = stmt.OrderBy("space_path COLLATE NOCASE " + opts.Order.String())
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
func (s *SpaceStore) CreatePath(ctx context.Context, id int64, params *types.PathParams) (*types.Path, error) {
	p := &types.Path{
		TargetType: enum.PathTargetTypeSpace,
		TargetID:   id,
		IsAlias:    true,

		// get remaining info from params
		Value:     params.Path,
		CreatedBy: params.CreatedBy,
		Created:   params.Created,
		Updated:   params.Updated,
	}

	return p, CreateAliasPath(ctx, s.db, p, s.pathTransformation)
}

// DeletePath an alias of a space.
func (s *SpaceStore) DeletePath(ctx context.Context, id int64, pathID int64) error {
	return DeletePath(ctx, s.db, pathID)
}

const spaceSelectBase = `
SELECT
 space_id
,space_parent_id
,space_uid
,paths.path_value AS space_path
,space_description
,space_is_public
,space_created_by
,space_created
,space_updated
`

const spaceSelectBaseWithJoin = spaceSelectBase + `
FROM spaces
INNER JOIN paths
ON spaces.space_id=paths.path_target_id AND paths.path_target_type='space' AND paths.path_is_alias=false
`

const spaceSelectByID = spaceSelectBaseWithJoin + `
WHERE space_id = $1
`

const spaceSelectByPathUnique = spaceSelectBase + `
FROM paths paths1
INNER JOIN spaces ON spaces.space_id=paths1.path_target_id AND paths1.path_target_type='space'
  AND paths1.path_value_unique = $1
INNER JOIN paths ON spaces.space_id=paths.path_target_id AND paths.path_target_type='space'
  AND paths.path_is_alias=false
`

const spaceDelete = `
DELETE FROM spaces
WHERE space_id = $1
`

// TODO: do we have to worry about SQL injection for description?
const spaceInsert = `
INSERT INTO spaces (
   space_parent_id
   ,space_uid
   ,space_description
   ,space_is_public
   ,space_created_by
   ,space_created
   ,space_updated
) values (
   :space_parent_id
   ,:space_uid
   ,:space_description
   ,:space_is_public
   ,:space_created_by
   ,:space_created
   ,:space_updated
) RETURNING space_id
`

const spaceUpdate = `
UPDATE spaces
SET
space_description = :space_description
,space_is_public  = :space_is_public
,space_updated    = :space_updated
WHERE space_id    = :space_id
`

const spaceUpdateUIDAndParentID = `
UPDATE spaces
SET
space_uid        = $1
,space_parent_id = $2
WHERE space_id   = $3
`

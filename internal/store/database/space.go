// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.SpaceStore = (*SpaceStore)(nil)

// NewSpaceStore returns a new SpaceStore.
func NewSpaceStore(db *sqlx.DB, pathCache store.PathCache) *SpaceStore {
	return &SpaceStore{
		db:        db,
		pathCache: pathCache,
	}
}

// SpaceStore implements a SpaceStore backed by a relational database.
type SpaceStore struct {
	db        *sqlx.DB
	pathCache store.PathCache
}

// space is an internal representation used to store space data in DB.
type space struct {
	ID      int64 `db:"space_id"`
	Version int64 `db:"space_version"`
	// IMPORTANT: We need to make parentID optional for spaces to allow it to be a foreign key.
	ParentID    null.Int `db:"space_parent_id"`
	UID         string   `db:"space_uid"`
	Path        string   `db:"space_path"`
	Description string   `db:"space_description"`
	IsPublic    bool     `db:"space_is_public"`
	CreatedBy   int64    `db:"space_created_by"`
	Created     int64    `db:"space_created"`
	Updated     int64    `db:"space_updated"`
}

const (
	spaceColumnsForJoin = `
		space_id
		,space_version
		,space_parent_id
		,space_uid
		,paths.path_value AS space_path
		,space_description
		,space_is_public
		,space_created_by
		,space_created
		,space_updated`

	spaceSelectBaseWithJoin = `
		SELECT` + spaceColumnsForJoin + `
		FROM spaces
		INNER JOIN paths
		ON spaces.space_id=paths.path_space_id AND paths.path_is_primary=true`
)

// Find the space by id.
func (s *SpaceStore) Find(ctx context.Context, id int64) (*types.Space, error) {
	const spaceSelectByID = spaceSelectBaseWithJoin + `
		WHERE space_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(space)
	if err := db.GetContext(ctx, dst, spaceSelectByID, id); err != nil {
		return nil, processSQLErrorf(err, "Failed to find space")
	}

	return mapToSpace(dst), nil
}

// FindByRef finds the space using the spaceRef as either the id or the space path.
func (s *SpaceStore) FindByRef(ctx context.Context, spaceRef string) (*types.Space, error) {
	// ASSUMPTION: digits only is not a valid space path
	id, err := strconv.ParseInt(spaceRef, 10, 64)
	if err != nil {
		var path *types.Path
		path, err = s.pathCache.Get(ctx, spaceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get path: %w", err)
		}

		if path.TargetType != enum.PathTargetTypeSpace {
			// IMPORTANT: expose as not found error as we didn't find the space!
			return nil, fmt.Errorf("path is not targeting a space - %w", store.ErrResourceNotFound)
		}

		id = path.TargetID
	}

	return s.Find(ctx, id)
}

// Create a new space.
func (s *SpaceStore) Create(ctx context.Context, space *types.Space) error {
	const sqlQuery = `
		INSERT INTO spaces (
			space_version
			,space_parent_id
			,space_uid
			,space_description
			,space_is_public
			,space_created_by
			,space_created
			,space_updated
		) values (
			:space_version
			,:space_parent_id
			,:space_uid
			,:space_description
			,:space_is_public
			,:space_created_by
			,:space_created
			,:space_updated
		) RETURNING space_id`

	dbSpace, err := mapToInternalSpace(space)
	if err != nil {
		return fmt.Errorf("failed to map space: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, dbSpace)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind space object")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&space.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Updates the space details.
func (s *SpaceStore) Update(ctx context.Context, space *types.Space) error {
	const sqlQuery = `
		UPDATE spaces
		SET
		    space_version		= :space_version
			,space_updated		= :space_updated
			,space_parent_id	= :space_parent_id
			,space_uid			= :space_uid
			,space_description	= :space_description
			,space_is_public	= :space_is_public
		WHERE space_id = :space_id AND space_version = :space_version - 1`

	dbSpace, err := mapToInternalSpace(space)
	if err != nil {
		return fmt.Errorf("failed to map space: %w", err)
	}

	// update Version (used for optimistic locking) and Updated time
	dbSpace.Version++
	dbSpace.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSpace)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind space object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Update query failed")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return processSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return store.ErrVersionConflict
	}

	space.Version = dbSpace.Version
	space.Updated = dbSpace.Updated

	return nil
}

// UpdateOptLock updates the space using the optimistic locking mechanism.
func (s *SpaceStore) UpdateOptLock(ctx context.Context,
	space *types.Space,
	mutateFn func(space *types.Space) error) (*types.Space, error) {
	for {
		dup := *space

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, store.ErrVersionConflict) {
			return nil, err
		}

		space, err = s.Find(ctx, space.ID)
		if err != nil {
			return nil, err
		}
	}
}

// Deletes the space.
func (s *SpaceStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM spaces
		WHERE space_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
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

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// List returns a list of spaces under the parent space.
func (s *SpaceStore) List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error) {
	stmt := builder.
		Select(spaceColumnsForJoin).
		From("spaces").
		InnerJoin(`paths ON spaces.space_id=paths.path_space_id AND paths.path_is_primary=true`).
		Where("space_parent_id = ?", fmt.Sprint(id))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(space_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
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

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*space{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Failed executing custom list query")
	}

	return mapToSpaces(dst), nil
}

func mapToSpace(s *space) *types.Space {
	res := &types.Space{
		ID:          s.ID,
		Version:     s.Version,
		UID:         s.UID,
		Path:        s.Path,
		Description: s.Description,
		IsPublic:    s.IsPublic,
		Created:     s.Created,
		CreatedBy:   s.CreatedBy,
		Updated:     s.Updated,
	}

	// Only overwrite ParentID if it's not a root space
	if s.ParentID.Valid {
		res.ParentID = s.ParentID.Int64
	}

	return res
}

func mapToSpaces(spaces []*space) []*types.Space {
	res := make([]*types.Space, len(spaces))
	for i := range spaces {
		res[i] = mapToSpace(spaces[i])
	}
	return res
}

func mapToInternalSpace(s *types.Space) (*space, error) {
	// space comes from outside.
	if s == nil {
		return nil, fmt.Errorf("space is nil")
	}

	res := &space{
		ID:          s.ID,
		Version:     s.Version,
		UID:         s.UID,
		Path:        s.Path,
		Description: s.Description,
		IsPublic:    s.IsPublic,
		Created:     s.Created,
		CreatedBy:   s.CreatedBy,
		Updated:     s.Updated,
	}

	// Only overwrite ParentID if it's not a root space
	// IMPORTANT: s.ParentID==0 has to be translated to nil as otherwise the foreign key fails
	if s.ParentID > 0 {
		res.ParentID = null.IntFrom(s.ParentID)
	}

	return res, nil
}

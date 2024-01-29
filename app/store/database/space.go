// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.SpaceStore = (*SpaceStore)(nil)

// NewSpaceStore returns a new SpaceStore.
func NewSpaceStore(
	db *sqlx.DB,
	spacePathCache store.SpacePathCache,
	spacePathStore store.SpacePathStore,
) *SpaceStore {
	return &SpaceStore{
		db:             db,
		spacePathCache: spacePathCache,
		spacePathStore: spacePathStore,
	}
}

// SpaceStore implements a SpaceStore backed by a relational database.
type SpaceStore struct {
	db             *sqlx.DB
	spacePathCache store.SpacePathCache
	spacePathStore store.SpacePathStore
}

// space is an internal representation used to store space data in DB.
type space struct {
	ID      int64 `db:"space_id"`
	Version int64 `db:"space_version"`
	// IMPORTANT: We need to make parentID optional for spaces to allow it to be a foreign key.
	ParentID    null.Int `db:"space_parent_id"`
	Identifier  string   `db:"space_uid"`
	Description string   `db:"space_description"`
	IsPublic    bool     `db:"space_is_public"`
	CreatedBy   int64    `db:"space_created_by"`
	Created     int64    `db:"space_created"`
	Updated     int64    `db:"space_updated"`
}

const (
	spaceColumns = `
		space_id
		,space_version
		,space_parent_id
		,space_uid
		,space_description
		,space_is_public
		,space_created_by
		,space_created
		,space_updated`

	spaceSelectBase = `
	SELECT` + spaceColumns + `
	FROM spaces`
)

// Find the space by id.
func (s *SpaceStore) Find(ctx context.Context, id int64) (*types.Space, error) {
	const sqlQuery = spaceSelectBase + `
		WHERE space_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(space)
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find space")
	}

	return mapToSpace(ctx, s.spacePathStore, dst)
}

// FindByRef finds the space using the spaceRef as either the id or the space path.
func (s *SpaceStore) FindByRef(ctx context.Context, spaceRef string) (*types.Space, error) {
	// ASSUMPTION: digits only is not a valid space path
	id, err := strconv.ParseInt(spaceRef, 10, 64)
	if err != nil {
		var path *types.SpacePath
		path, err = s.spacePathCache.Get(ctx, spaceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get path: %w", err)
		}

		id = path.SpaceID
	}

	return s.Find(ctx, id)
}

// GetRootSpace returns a space where space_parent_id is NULL.
func (s *SpaceStore) GetRootSpace(ctx context.Context, spaceID int64) (*types.Space, error) {
	query := `WITH RECURSIVE SpaceHierarchy AS (
	SELECT space_id, space_parent_id
	FROM spaces
	WHERE space_id = $1
	
	UNION
	
	SELECT s.space_id, s.space_parent_id
	FROM spaces s
	JOIN SpaceHierarchy h ON s.space_id = h.space_parent_id
)
SELECT space_id
FROM SpaceHierarchy
WHERE space_parent_id IS NULL;`

	db := dbtx.GetAccessor(ctx, s.db)

	var rootID int64
	if err := db.GetContext(ctx, &rootID, query, spaceID); err != nil {
		return nil, database.ProcessSQLErrorf(err, "failed to get root space_id")
	}

	return s.Find(ctx, rootID)
}

// Create a new space.
func (s *SpaceStore) Create(ctx context.Context, space *types.Space) error {
	if space == nil {
		return errors.New("space is nil")
	}

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

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, mapToInternalSpace(space))
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind space object")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&space.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Update updates the space details.
func (s *SpaceStore) Update(ctx context.Context, space *types.Space) error {
	if space == nil {
		return errors.New("space is nil")
	}

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

	dbSpace := mapToInternalSpace(space)

	// update Version (used for optimistic locking) and Updated time
	dbSpace.Version++
	dbSpace.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSpace)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind space object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Update query failed")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	space.Version = dbSpace.Version
	space.Updated = dbSpace.Updated

	// update path in case parent/identifier changed
	space.Path, err = getSpacePath(ctx, s.spacePathStore, space.ID)
	if err != nil {
		return err
	}

	return nil
}

// UpdateOptLock updates the space using the optimistic locking mechanism.
func (s *SpaceStore) UpdateOptLock(ctx context.Context,
	space *types.Space,
	mutateFn func(space *types.Space) error,
) (*types.Space, error) {
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
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		space, err = s.Find(ctx, space.ID)
		if err != nil {
			return nil, err
		}
	}
}

// Delete deletes a space.
func (s *SpaceStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM spaces
		WHERE space_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(err, "The delete query failed")
	}

	return nil
}

// Count the child spaces of a space.
func (s *SpaceStore) Count(ctx context.Context, id int64, opts *types.SpaceFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("spaces").
		Where("space_parent_id = ?", id)

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(space_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of spaces under the parent space.
func (s *SpaceStore) List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error) {
	stmt := database.Builder.
		Select(spaceColumns).
		From("spaces").
		Where("space_parent_id = ?", fmt.Sprint(id))

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(space_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	switch opts.Sort {
	case enum.SpaceAttrUID, enum.SpaceAttrIdentifier, enum.SpaceAttrNone:
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
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*space
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return s.mapToSpaces(ctx, dst)
}

func mapToSpace(
	ctx context.Context,
	spacePathStore store.SpacePathStore,
	in *space,
) (*types.Space, error) {
	var err error
	res := &types.Space{
		ID:          in.ID,
		Version:     in.Version,
		Identifier:  in.Identifier,
		Description: in.Description,
		IsPublic:    in.IsPublic,
		Created:     in.Created,
		CreatedBy:   in.CreatedBy,
		Updated:     in.Updated,
	}

	// Only overwrite ParentID if it's not a root space
	if in.ParentID.Valid {
		res.ParentID = in.ParentID.Int64
	}

	// backfill path
	res.Path, err = getSpacePath(ctx, spacePathStore, in.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary path for space %d: %w", in.ID, err)
	}

	return res, nil
}

func getSpacePath(
	ctx context.Context,
	spacePathStore store.SpacePathStore,
	spaceID int64,
) (string, error) {
	spacePath, err := spacePathStore.FindPrimaryBySpaceID(ctx, spaceID)
	if err != nil {
		return "", fmt.Errorf("failed to get primary path for space %d: %w", spaceID, err)
	}

	return spacePath.Value, nil
}

func (s *SpaceStore) mapToSpaces(
	ctx context.Context,
	spaces []*space,
) ([]*types.Space, error) {
	var err error
	res := make([]*types.Space, len(spaces))
	for i := range spaces {
		res[i], err = mapToSpace(ctx, s.spacePathStore, spaces[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func mapToInternalSpace(s *types.Space) *space {
	res := &space{
		ID:          s.ID,
		Version:     s.Version,
		Identifier:  s.Identifier,
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

	return res
}

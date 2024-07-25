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

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
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
	CreatedBy   int64    `db:"space_created_by"`
	Created     int64    `db:"space_created"`
	Updated     int64    `db:"space_updated"`
	Deleted     null.Int `db:"space_deleted"`
}

const (
	spaceColumns = `
		space_id
		,space_version
		,space_parent_id
		,space_uid
		,space_description
		,space_created_by
		,space_created
		,space_updated
		,space_deleted`

	spaceSelectBase = `
	SELECT` + spaceColumns + `
	FROM spaces`
)

// Find the space by id.
func (s *SpaceStore) Find(ctx context.Context, id int64) (*types.Space, error) {
	return s.find(ctx, id, nil)
}

func (s *SpaceStore) find(ctx context.Context, id int64, deletedAt *int64) (*types.Space, error) {
	stmt := database.Builder.
		Select(spaceColumns).
		From("spaces").
		Where("space_id = ?", id)

	if deletedAt != nil {
		stmt = stmt.Where("space_deleted = ?", *deletedAt)
	} else {
		stmt = stmt.Where("space_deleted IS NULL")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(space)
	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find space")
	}

	return mapToSpace(ctx, s.db, s.spacePathStore, dst)
}

// FindByRef finds the space using the spaceRef as either the id or the space path.
func (s *SpaceStore) FindByRef(ctx context.Context, spaceRef string) (*types.Space, error) {
	return s.findByRef(ctx, spaceRef, nil)
}

// FindByRefAndDeletedAt finds the space using the spaceRef as either the id or the space path and deleted timestamp.
func (s *SpaceStore) FindByRefAndDeletedAt(
	ctx context.Context,
	spaceRef string,
	deletedAt int64,
) (*types.Space, error) {
	// ASSUMPTION: digits only is not a valid space path
	id, err := strconv.ParseInt(spaceRef, 10, 64)
	if err != nil {
		return s.findByPathAndDeletedAt(ctx, spaceRef, deletedAt)
	}

	return s.find(ctx, id, &deletedAt)
}

func (s *SpaceStore) findByRef(ctx context.Context, spaceRef string, deletedAt *int64) (*types.Space, error) {
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
	return s.find(ctx, id, deletedAt)
}

func (s *SpaceStore) findByPathAndDeletedAt(
	ctx context.Context,
	spaceRef string,
	deletedAt int64,
) (*types.Space, error) {
	segments := paths.Segments(spaceRef)
	if len(segments) < 1 {
		return nil, fmt.Errorf("invalid space reference provided")
	}

	var stmt squirrel.SelectBuilder
	switch {
	case len(segments) == 1:
		stmt = database.Builder.
			Select("space_id").
			From("spaces").
			Where("space_uid = ? AND space_deleted = ? AND space_parent_id IS NULL", segments[0], deletedAt)

	case len(segments) > 1:
		stmt = buildRecursiveSelectQueryUsingPath(segments, deletedAt)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create sql query")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var spaceID int64
	if err = db.GetContext(ctx, &spaceID, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom select query")
	}

	return s.find(ctx, spaceID, &deletedAt)
}

const spaceRecursiveQuery = `
WITH RECURSIVE SpaceHierarchy(space_hierarchy_id, space_hierarchy_parent_id) AS (
	SELECT space_id, space_parent_id
	FROM spaces
	WHERE space_id = $1
	
	UNION
	
	SELECT s.space_id, s.space_parent_id
	FROM spaces s
	JOIN SpaceHierarchy h ON s.space_id = h.space_hierarchy_parent_id
)
`

// GetRootSpace returns a space where space_parent_id is NULL.
func (s *SpaceStore) GetRootSpace(ctx context.Context, spaceID int64) (*types.Space, error) {
	query := spaceRecursiveQuery + `
		SELECT space_hierarchy_id
		FROM SpaceHierarchy
		WHERE space_hierarchy_parent_id IS NULL;`

	db := dbtx.GetAccessor(ctx, s.db)

	var rootID int64
	if err := db.GetContext(ctx, &rootID, query, spaceID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to get root space_id")
	}

	return s.Find(ctx, rootID)
}

// GetAncestorIDs returns a list of all space IDs along the recursive path to the root space.
func (s *SpaceStore) GetAncestorIDs(ctx context.Context, spaceID int64) ([]int64, error) {
	query := spaceRecursiveQuery + `
		SELECT space_hierarchy_id FROM SpaceHierarchy`

	db := dbtx.GetAccessor(ctx, s.db)

	var spaceIDs []int64
	if err := db.SelectContext(ctx, &spaceIDs, query, spaceID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to get space hierarchy")
	}

	return spaceIDs, nil
}

func (s *SpaceStore) GetHierarchy(
	ctx context.Context,
	spaceID int64,
) ([]*types.Space, error) {
	query := spaceRecursiveQuery + `
		SELECT ` + spaceColumns + `
		FROM spaces INNER JOIN SpaceHierarchy ON space_id = space_hierarchy_id`

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*space
	if err := db.SelectContext(ctx, &dst, query, spaceID); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToSpaces(ctx, s.db, dst)
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
			,space_created_by
			,space_created
			,space_updated
			,space_deleted
		) values (
			:space_version
			,:space_parent_id
			,:space_uid
			,:space_description
			,:space_created_by
			,:space_created
			,:space_updated
			,:space_deleted
		) RETURNING space_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, mapToInternalSpace(space))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind space object")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&space.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
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
			,space_deleted 		= :space_deleted
		WHERE space_id = :space_id AND space_version = :space_version - 1`

	dbSpace := mapToInternalSpace(space)

	// update Version (used for optimistic locking) and Updated time
	dbSpace.Version++
	dbSpace.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbSpace)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind space object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Update query failed")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	space.Version = dbSpace.Version
	space.Updated = dbSpace.Updated

	// update path in case parent/identifier changed
	space.Path, err = getSpacePath(ctx, s.db, s.spacePathStore, space.ID)
	if err != nil {
		return err
	}

	return nil
}

// updateOptLock updates the space using the optimistic locking mechanism.
func (s *SpaceStore) updateOptLock(
	ctx context.Context,
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

		space, err = s.find(ctx, space.ID, space.Deleted)
		if err != nil {
			return nil, err
		}
	}
}

// UpdateOptLock updates the space using the optimistic locking mechanism.
func (s *SpaceStore) UpdateOptLock(
	ctx context.Context,
	space *types.Space,
	mutateFn func(space *types.Space) error,
) (*types.Space, error) {
	return s.updateOptLock(
		ctx,
		space,
		func(r *types.Space) error {
			if space.Deleted != nil {
				return gitness_store.ErrResourceNotFound
			}
			return mutateFn(r)
		},
	)
}

// UpdateDeletedOptLock updates a soft deleted space using the optimistic locking mechanism.
func (s *SpaceStore) updateDeletedOptLock(
	ctx context.Context,
	space *types.Space,
	mutateFn func(space *types.Space) error,
) (*types.Space, error) {
	return s.updateOptLock(
		ctx,
		space,
		func(r *types.Space) error {
			if space.Deleted == nil {
				return gitness_store.ErrResourceNotFound
			}
			return mutateFn(r)
		},
	)
}

// FindForUpdate finds the space and locks it for an update (should be called in a tx).
func (s *SpaceStore) FindForUpdate(ctx context.Context, id int64) (*types.Space, error) {
	// sqlite allows at most one write to proceed (no need to lock)
	if strings.HasPrefix(s.db.DriverName(), "sqlite") {
		return s.find(ctx, id, nil)
	}

	stmt := database.Builder.Select("space_id").
		From("spaces").
		Where("space_id = ? AND space_deleted IS NULL", id).
		Suffix("FOR UPDATE")

	sqlQuery, params, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to generate lock on spaces")
	}

	dst := new(space)
	db := dbtx.GetAccessor(ctx, s.db)
	if err = db.GetContext(ctx, dst, sqlQuery, params...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find space")
	}

	return mapToSpace(ctx, s.db, s.spacePathStore, dst)
}

// SoftDelete deletes a space softly.
func (s *SpaceStore) SoftDelete(
	ctx context.Context,
	space *types.Space,
	deletedAt int64,
) error {
	_, err := s.UpdateOptLock(ctx, space, func(s *types.Space) error {
		s.Deleted = &deletedAt
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// Purge deletes a space permanently.
func (s *SpaceStore) Purge(ctx context.Context, id int64, deletedAt *int64) error {
	stmt := database.Builder.
		Delete("spaces").
		Where("space_id = ?", id)

	if deletedAt != nil {
		stmt = stmt.Where("space_deleted = ?", *deletedAt)
	} else {
		stmt = stmt.Where("space_deleted IS NULL")
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge space query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

// Restore restores a soft deleted space.
func (s *SpaceStore) Restore(
	ctx context.Context,
	space *types.Space,
	newIdentifier *string,
	newParentID *int64,
) (*types.Space, error) {
	space, err := s.updateDeletedOptLock(ctx, space, func(s *types.Space) error {
		s.Deleted = nil
		if newParentID != nil {
			s.ParentID = *newParentID
		}

		if newIdentifier != nil {
			s.Identifier = *newIdentifier
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return space, nil
}

// Count the child spaces of a space.
func (s *SpaceStore) Count(ctx context.Context, id int64, opts *types.SpaceFilter) (int64, error) {
	if opts.Recursive {
		return s.countAll(ctx, id, opts)
	}
	return s.count(ctx, id, opts)
}

func (s *SpaceStore) count(
	ctx context.Context,
	id int64,
	opts *types.SpaceFilter,
) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("spaces").
		Where("space_parent_id = ?", id)

	if opts.Query != "" {
		stmt = stmt.Where("LOWER(space_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}

	stmt = s.applyQueryFilter(stmt, opts)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

func (s *SpaceStore) countAll(
	ctx context.Context,
	id int64,
	opts *types.SpaceFilter,
) (int64, error) {
	ctePrefix := `WITH RECURSIVE SpaceHierarchy AS (
		SELECT space_id, space_parent_id, space_deleted, space_uid
		FROM spaces
		WHERE space_id = ?

		UNION

		SELECT s.space_id, s.space_parent_id, s.space_deleted, s.space_uid
		FROM spaces s
		JOIN SpaceHierarchy h ON s.space_parent_id = h.space_id
	)`

	db := dbtx.GetAccessor(ctx, s.db)

	stmt := database.Builder.
		Select("COUNT(*)").
		Prefix(ctePrefix, id).
		From("SpaceHierarchy h1").
		Where("h1.space_id <> ?", id)

	stmt = s.applyQueryFilter(stmt, opts)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	var count int64
	if err = db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to count sub spaces")
	}

	return count, nil
}

// List returns a list of spaces under the parent space.
func (s *SpaceStore) List(
	ctx context.Context,
	id int64,
	opts *types.SpaceFilter,
) ([]*types.Space, error) {
	if opts.Recursive {
		return s.listAll(ctx, id, opts)
	}
	return s.list(ctx, id, opts)
}

func (s *SpaceStore) list(
	ctx context.Context,
	id int64,
	opts *types.SpaceFilter,
) ([]*types.Space, error) {
	stmt := database.Builder.
		Select(spaceColumns).
		From("spaces").
		Where("space_parent_id = ?", fmt.Sprint(id))

	stmt = s.applyQueryFilter(stmt, opts)
	stmt = s.applySortFilter(stmt, opts)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var dst []*space
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToSpaces(ctx, s.db, dst)
}

func (s *SpaceStore) listAll(ctx context.Context,
	id int64,
	opts *types.SpaceFilter,
) ([]*types.Space, error) {
	ctePrefix := `WITH RECURSIVE SpaceHierarchy AS (
		SELECT *
		FROM spaces
		WHERE space_id = ?
		
		UNION
		
		SELECT s.*
		FROM spaces s
		JOIN SpaceHierarchy h ON s.space_parent_id = h.space_id
	)`

	db := dbtx.GetAccessor(ctx, s.db)

	stmt := database.Builder.
		Select(spaceColumns).
		Prefix(ctePrefix, id).
		From("SpaceHierarchy h1").
		Where("h1.space_id <> ?", id)

	stmt = s.applyQueryFilter(stmt, opts)
	stmt = s.applySortFilter(stmt, opts)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	var dst []*space
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return s.mapToSpaces(ctx, s.db, dst)
}

func (s *SpaceStore) applyQueryFilter(
	stmt squirrel.SelectBuilder,
	opts *types.SpaceFilter,
) squirrel.SelectBuilder {
	if opts.Query != "" {
		stmt = stmt.Where("LOWER(space_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(opts.Query)))
	}
	//nolint:gocritic
	if opts.DeletedAt != nil {
		stmt = stmt.Where("space_deleted = ?", opts.DeletedAt)
	} else if opts.DeletedBeforeOrAt != nil {
		stmt = stmt.Where("space_deleted <= ?", opts.DeletedBeforeOrAt)
	} else {
		stmt = stmt.Where("space_deleted IS NULL")
	}

	return stmt
}

func getPathForDeletedSpace(
	ctx context.Context,
	sqlxdb *sqlx.DB,
	id int64,
) (string, error) {
	sqlQuery := spaceSelectBase + `
		where space_id = $1`

	path := ""
	nextSpaceID := null.IntFrom(id)

	db := dbtx.GetAccessor(ctx, sqlxdb)
	dst := new(space)

	for nextSpaceID.Valid {
		err := db.GetContext(ctx, dst, sqlQuery, nextSpaceID.Int64)
		if err != nil {
			return "", fmt.Errorf("failed to find the space %d: %w", id, err)
		}

		path = paths.Concatenate(dst.Identifier, path)
		nextSpaceID = dst.ParentID
	}

	return path, nil
}

func (s *SpaceStore) applySortFilter(
	stmt squirrel.SelectBuilder,
	opts *types.SpaceFilter,
) squirrel.SelectBuilder {
	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

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
	case enum.SpaceAttrDeleted:
		stmt = stmt.OrderBy("space_deleted " + opts.Order.String())
	}
	return stmt
}

func mapToSpace(
	ctx context.Context,
	sqlxdb *sqlx.DB,
	spacePathStore store.SpacePathStore,
	in *space,
) (*types.Space, error) {
	var err error
	res := &types.Space{
		ID:          in.ID,
		Version:     in.Version,
		Identifier:  in.Identifier,
		Description: in.Description,
		Created:     in.Created,
		CreatedBy:   in.CreatedBy,
		Updated:     in.Updated,
		Deleted:     in.Deleted.Ptr(),
	}

	// Only overwrite ParentID if it's not a root space
	if in.ParentID.Valid {
		res.ParentID = in.ParentID.Int64
	}

	// backfill path
	res.Path, err = getSpacePath(ctx, sqlxdb, spacePathStore, in.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary path for space %d: %w", in.ID, err)
	}

	return res, nil
}

func getSpacePath(
	ctx context.Context,
	sqlxdb *sqlx.DB,
	spacePathStore store.SpacePathStore,
	spaceID int64,
) (string, error) {
	spacePath, err := spacePathStore.FindPrimaryBySpaceID(ctx, spaceID)
	// delete space will delete paths; generate the path if space is soft deleted.
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		return getPathForDeletedSpace(ctx, sqlxdb, spaceID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get primary path for space %d: %w", spaceID, err)
	}

	return spacePath.Value, nil
}

func (s *SpaceStore) mapToSpaces(
	ctx context.Context,
	sqlxdb *sqlx.DB,
	spaces []*space,
) ([]*types.Space, error) {
	var err error
	res := make([]*types.Space, len(spaces))
	for i := range spaces {
		res[i], err = mapToSpace(ctx, sqlxdb, s.spacePathStore, spaces[i])
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
		Created:     s.Created,
		CreatedBy:   s.CreatedBy,
		Updated:     s.Updated,
		Deleted:     null.IntFromPtr(s.Deleted),
	}

	// Only overwrite ParentID if it's not a root space
	// IMPORTANT: s.ParentID==0 has to be translated to nil as otherwise the foreign key fails
	if s.ParentID > 0 {
		res.ParentID = null.IntFrom(s.ParentID)
	}

	return res
}

// buildRecursiveSelectQueryUsingPath builds the recursive select query using path among active or soft deleted spaces.
func buildRecursiveSelectQueryUsingPath(segments []string, deletedAt int64) squirrel.SelectBuilder {
	leaf := "s" + fmt.Sprint(len(segments)-1)

	// add the current space (leaf)
	stmt := database.Builder.
		Select(leaf+".space_id").
		From("spaces "+leaf).
		Where(leaf+".space_uid = ? AND "+leaf+".space_deleted = ?", segments[len(segments)-1], deletedAt)

	for i := len(segments) - 2; i >= 0; i-- {
		parentAlias := "s" + fmt.Sprint(i)
		alias := "s" + fmt.Sprint(i+1)

		stmt = stmt.InnerJoin(fmt.Sprintf("spaces %s ON %s.space_id = %s.space_parent_id", parentAlias, parentAlias, alias)).
			Where(parentAlias+".space_uid = ?", segments[i])
	}

	// add parent check for root
	stmt = stmt.Where("s0.space_parent_id IS NULL")

	return stmt
}

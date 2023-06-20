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

var _ store.PathStore = (*PathStore)(nil)

// NewSpaceStore returns a new PathStore.
func NewPathStore(db *sqlx.DB, pathTransformation store.PathTransformation) *PathStore {
	return &PathStore{
		db:                 db,
		pathTransformation: pathTransformation,
	}
}

// PathStore implements a store.PathStore backed by a relational database.
type PathStore struct {
	db                 *sqlx.DB
	pathTransformation store.PathTransformation
}

// path is an internal representation used to store path data in DB.
type path struct {
	ID      int64 `db:"path_id"`
	Version int64 `db:"path_version"`
	// Value is the original path that was provided
	Value string `db:"path_value"`
	// ValueUnique is a transformed version of Value which is used to ensure uniqueness guarantees
	ValueUnique string `db:"path_value_unique"`
	// IsPrimary indicates whether the path is the primary path of the repo/space
	// IMPORTANT: to allow DB enforcement of at most one primary path per repo/space
	// we have a unique index on repoID|spaceID + IsPrimary and set IsPrimary to true
	// for primary paths and to nil for non-primary paths.
	IsPrimary null.Bool `db:"path_is_primary"`
	SpaceID   null.Int  `db:"path_space_id"`
	RepoID    null.Int  `db:"path_repo_id"`
	CreatedBy int64     `db:"path_created_by"`
	Created   int64     `db:"path_created"`
	Updated   int64     `db:"path_updated"`
}

const (
	pathColumns = `
		path_id
		,path_version
		,path_value
		,path_value_unique
		,path_is_primary
		,path_space_id
		,path_repo_id
		,path_created_by
		,path_created
		,path_updated`

	pathSelectBase = `
		SELECT` + pathColumns + `
		FROM paths`
)

// Create creates a new path.
func (s *PathStore) Create(ctx context.Context, path *types.Path) error {
	const sqlQuery = `
		INSERT INTO paths (
			path_version
			,path_value
			,path_value_unique
			,path_is_primary
			,path_space_id
			,path_repo_id
			,path_created_by
			,path_created
			,path_updated
		) values (
			:path_version
			,:path_value
			,:path_value_unique
			,:path_is_primary
			,:path_space_id
			,:path_repo_id
			,:path_created_by
			,:path_created
			,:path_updated
		) RETURNING path_id`

	// map to internal path
	dbPath, err := s.mapToInternalPath(path)
	if err != nil {
		return fmt.Errorf("failed to map path: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbPath)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind path object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return database.ProcessSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// findInner finds the path for the given id and locks the record if requested.
func (s *PathStore) findInner(ctx context.Context, id int64, lock bool) (*types.Path, error) {
	sqlQuery := pathSelectBase + `
		WHERE path_id = $1`

	if lock && !strings.HasPrefix(s.db.DriverName(), "sqlite") {
		sqlQuery += "\n" + database.SQLForUpdate
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(path)
	err := db.GetContext(ctx, dst, sqlQuery, id)
	if err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find path")
	}

	res, err := mapToPath(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map path to external type: %w", err)
	}

	return res, nil
}

// Find finds the path for the given id.
func (s *PathStore) Find(ctx context.Context, id int64) (*types.Path, error) {
	return s.findInner(ctx, id, false)
}

// FindWithLock finds the path for the given id and locks the entry.
func (s *PathStore) FindWithLock(ctx context.Context, id int64) (*types.Path, error) {
	return s.findInner(ctx, id, true)
}

// FindValue finds the path for the given value.
func (s *PathStore) FindValue(ctx context.Context, value string) (*types.Path, error) {
	const sqlQuery = pathSelectBase + `
		WHERE path_value_unique = $1`

	// map the Value to unique Value before searching!
	valueUnique, err := s.pathTransformation(value)
	if err != nil {
		return nil, fmt.Errorf("failed to transform path '%s': %w", value, err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(path)
	err = db.GetContext(ctx, dst, sqlQuery, valueUnique)
	if err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find path")
	}

	res, err := mapToPath(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map path to external type: %w", err)
	}

	return res, nil
}

// findPrimaryInternal finds the  primary path for the given target and locks the record if requested.
func (s *PathStore) findPrimaryInternal(ctx context.Context,
	targetType enum.PathTargetType, targetID int64, lock bool) (*types.Path, error) {
	stmt := database.Builder.
		Select(pathColumns).
		From("paths").
		Where("path_is_primary = ?", true)

	stmt, err := wherePathTarget(stmt, targetType, targetID)
	if err != nil {
		return nil, err
	}

	if lock && !strings.HasPrefix(s.db.DriverName(), "sqlite") {
		stmt.Suffix(database.SQLForUpdate)
	}

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	// NOTE: there is at most one primary path (so no list needed)
	dst := new(path)
	err = db.GetContext(ctx, dst, sqlQuery, args...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find path")
	}

	res, err := mapToPath(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map path to external type: %w", err)
	}

	return res, nil
}

// FindPrimary finds the primary path for a target.
func (s *PathStore) FindPrimary(ctx context.Context,
	targetType enum.PathTargetType, targetID int64) (*types.Path, error) {
	return s.findPrimaryInternal(ctx, targetType, targetID, false)
}

// FindPrimaryWithLock finds the primary path for a target and locks the db entry.
func (s *PathStore) FindPrimaryWithLock(ctx context.Context,
	targetType enum.PathTargetType, targetID int64) (*types.Path, error) {
	return s.findPrimaryInternal(ctx, targetType, targetID, true)
}

// Update updates an existing path.
func (s *PathStore) Update(ctx context.Context, path *types.Path) error {
	const sqlQuery = `
		UPDATE paths
		SET
			path_version 		= :path_version
			,path_updated 		= :path_updated
			,path_value 		= :path_value
			,path_value_unique 	= :path_value_unique
			,path_is_primary 	= :path_is_primary
		WHERE path_id = :path_id and path_version = :path_version - 1`

	dbPath, err := s.mapToInternalPath(path)
	if err != nil {
		return fmt.Errorf("failed to map path: %w", err)
	}

	dbPath.Version++
	dbPath.Updated = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, args, err := db.BindNamed(sqlQuery, dbPath)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind path object")
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "failed to update path")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	path.Version = dbPath.Version
	path.Updated = dbPath.Updated

	return nil
}

// Delete deletes a specific path.
func (s *PathStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `
		DELETE FROM paths
		WHERE path_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(err, "Delete query failed")
	}

	return nil
}

// Count returns the count of paths for a target.
func (s *PathStore) Count(ctx context.Context, targetType enum.PathTargetType, targetID int64,
	opts *types.PathFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("paths")

	stmt, err := wherePathTarget(stmt, targetType, targetID)
	if err != nil {
		return 0, err
	}

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// List lists all paths for a target.
func (s *PathStore) List(ctx context.Context, targetType enum.PathTargetType, targetID int64,
	opts *types.PathFilter) ([]*types.Path, error) {
	// else we construct the sql statement.
	stmt := database.Builder.
		Select("*").
		From("paths")

	stmt, err := wherePathTarget(stmt, targetType, targetID)
	if err != nil {
		return nil, err
	}

	stmt = stmt.Limit(database.Limit(opts.Size))
	stmt = stmt.Offset(database.Offset(opts.Page, opts.Size))

	switch opts.Sort {
	case enum.PathAttrID, enum.PathAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("path_id " + opts.Order.String())
	case enum.PathAttrValue:
		stmt = stmt.OrderBy("path_value " + opts.Order.String())
	case enum.PathAttrCreated:
		stmt = stmt.OrderBy("path_created " + opts.Order.String())
	case enum.PathAttrUpdated:
		stmt = stmt.OrderBy("path_updated " + opts.Order.String())
	}

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*path{}
	if err = db.SelectContext(ctx, &dst, sqlQuery, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Path select query failed")
	}

	res, err := mapToPaths(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map paths to external type: %w", err)
	}

	return res, nil
}

// ListPrimaryDescendantsWithLock lists all primary paths that are descendants of the given path and locks them.
func (s *PathStore) ListPrimaryDescendantsWithLock(ctx context.Context, value string) ([]*types.Path, error) {
	var sqlQuery = pathSelectBase + `
		WHERE path_value_unique LIKE $1 AND path_is_primary = true`

	if !strings.HasPrefix(s.db.DriverName(), "sqlite") {
		sqlQuery += "\n" + database.SQLForUpdate
	}

	// map the Value to unique Value before searching!
	valueUnique, err := s.pathTransformation(value)
	if err != nil {
		return nil, fmt.Errorf("failed to transform path '%s': %w", value, err)
	}
	// prepare input for LIKE query (space1/space2 -> space1/space2/%)
	valueUniquePattern := paths.Concatinate(valueUnique, "%")

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*path{}
	if err = db.SelectContext(ctx, &dst, sqlQuery, valueUniquePattern); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to list paths")
	}

	res, err := mapToPaths(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to map paths to external type: %w", err)
	}

	return res, nil
}

// wherePathTarget adds a where statement for a path select statement filtering for the provided target.
func wherePathTarget(stmt squirrel.SelectBuilder,
	targetType enum.PathTargetType, targetID int64) (squirrel.SelectBuilder, error) {
	switch targetType {
	case enum.PathTargetTypeRepo:
		return stmt.Where("path_repo_id = ?", targetID), nil
	case enum.PathTargetTypeSpace:
		return stmt.Where("path_space_id = ?", targetID), nil
	default:
		return squirrel.SelectBuilder{}, fmt.Errorf("path target type '%s' is not supported", targetType)
	}
}

func mapToPath(p *path) (*types.Path, error) {
	res := &types.Path{
		ID:        p.ID,
		Version:   p.Version,
		Value:     p.Value,
		Created:   p.Created,
		CreatedBy: p.CreatedBy,
		Updated:   p.Updated,
	}

	if p.IsPrimary.Valid {
		res.IsPrimary = p.IsPrimary.Bool
	}

	switch {
	case p.RepoID.Valid && p.SpaceID.Valid:
		return nil, fmt.Errorf("both repoID and spaceID are set for path %d", p.ID)
	case p.RepoID.Valid:
		res.TargetType = enum.PathTargetTypeRepo
		res.TargetID = p.RepoID.Int64
	case p.SpaceID.Valid:
		res.TargetType = enum.PathTargetTypeSpace
		res.TargetID = p.SpaceID.Int64
	default:
		return nil, fmt.Errorf("neither repoID nor spaceID are set for path %d", p.ID)
	}

	return res, nil
}

func mapToPaths(paths []*path) ([]*types.Path, error) {
	var err error
	res := make([]*types.Path, len(paths))
	for i := range paths {
		res[i], err = mapToPath(paths[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *PathStore) mapToInternalPath(p *types.Path) (*path, error) {
	// path comes from outside.
	if p == nil {
		return nil, fmt.Errorf("path is nil")
	}

	res := &path{
		ID:        p.ID,
		Version:   p.Version,
		Value:     p.Value,
		Created:   p.Created,
		CreatedBy: p.CreatedBy,
		Updated:   p.Updated,
	}

	var err error
	if res.ValueUnique, err = s.pathTransformation(p.Value); err != nil {
		return nil, fmt.Errorf("failed to transform path: %w", err)
	}

	// only set IsPrimary to a value if it's true (Unique Index doesn't allow multiple false, hence keep it nil)
	if p.IsPrimary {
		res.IsPrimary = null.BoolFrom(true)
	}

	switch p.TargetType {
	case enum.PathTargetTypeRepo:
		res.RepoID = null.IntFrom(p.TargetID)
	case enum.PathTargetTypeSpace:
		res.SpaceID = null.IntFrom(p.TargetID)
	default:
		return nil, fmt.Errorf("path target type '%s' is not supported", p.TargetType)
	}

	return res, nil
}

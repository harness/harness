// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// path is a DB representation of a Path.
// It is required to allow storing transformed paths used for uniquness constraints and searching.
type path struct {
	types.Path
	ValueUnique string `db:"path_value_unique"`
}

// CreateAliasPath a new alias path (Don't call this for new path creation!)
func CreateAliasPath(ctx context.Context, db *sqlx.DB, path *types.Path,
	transformation store.PathTransformation) error {
	if !path.IsAlias {
		return store.ErrAliasPathRequired
	}

	// ensure path length is okay
	if check.IsPathTooDeep(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		log.Warn().Msgf("Path '%s' is too long.", path.Value)
		return store.ErrPathTooLong
	}

	// map to db path to ensure we store valueUnique.
	dbPath, err := mapToDBPath(path, transformation)
	if err != nil {
		return fmt.Errorf("failed to map db path: %w", err)
	}

	query, arg, err := db.BindNamed(pathInsert, dbPath)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind path object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// CreatePathTx creates a new path as part of a transaction.
func CreatePathTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, path *types.Path,
	transformation store.PathTransformation) error {
	// ensure path length is okay
	if check.IsPathTooDeep(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		log.Warn().Msgf("Path '%s' is too long.", path.Value)
		return store.ErrPathTooLong
	}

	// In case it's not an alias, ensure there are no duplicates
	if !path.IsAlias {
		if cnt, err := CountPathsTx(ctx, tx, path.TargetType, path.TargetID); err != nil {
			return err
		} else if cnt > 0 {
			return store.ErrPrimaryPathAlreadyExists
		}
	}

	// map to db path to ensure we store valueUnique.
	dbPath, err := mapToDBPath(path, transformation)
	if err != nil {
		return fmt.Errorf("failed to map db path: %w", err)
	}

	query, arg, err := db.BindNamed(pathInsert, dbPath)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind path object")
	}

	if err = tx.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

func CountPrimaryChildPathsTx(ctx context.Context, tx *sqlx.Tx, prefix string,
	transformation store.PathTransformation) (int64, error) {
	// map the Value to unique Value before searching!
	prefixUnique, err := transformation(prefix)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform path prefix '%s': %s", prefix, err.Error())
		return 0, store.ErrResourceNotFound
	}

	var count int64
	err = tx.QueryRowContext(ctx, pathCountPrimaryForPrefixUnique, paths.Concatinate(prefixUnique, "%")).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Count query failed")
	}
	return count, nil
}

func listPrimaryChildPathsTx(ctx context.Context, tx *sqlx.Tx, prefix string,
	transformation store.PathTransformation) ([]*path, error) {
	// map the Value to unique Value before searching!
	prefixUnique, err := transformation(prefix)
	if err != nil {
		// in case we fail to transform, return a not found (as it can't exist in the first place)
		log.Ctx(ctx).Debug().Msgf("failed to transform path prefix '%s': %s", prefix, err.Error())
		return nil, store.ErrResourceNotFound
	}

	childs := []*path{}

	if err = tx.SelectContext(ctx, &childs, pathSelectPrimaryForPrefixUnique,
		paths.Concatinate(prefixUnique, "%")); err != nil {
		return nil, processSQLErrorf(err, "Failed to list paths")
	}

	return childs, nil
}

// ReplacePathTx replaces the path for a target as part of a transaction - keeps the existing as alias if requested.
func ReplacePathTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, newPath *types.Path, keepAsAlias bool,
	transformation store.PathTransformation) error {
	if newPath.IsAlias {
		return store.ErrPrimaryPathRequired
	}

	// ensure new path length is okay
	if check.IsPathTooDeep(newPath.Value, newPath.TargetType == enum.PathTargetTypeSpace) {
		log.Warn().Msgf("Path '%s' is too long.", newPath.Value)
		return store.ErrPathTooLong
	}

	// dbExisting is always non-alias (as query filters for IsAlias=0)
	dbExisting := new(path)
	err := tx.GetContext(ctx, dbExisting, pathSelectPrimaryForTarget,
		string(newPath.TargetType), fmt.Sprint(newPath.TargetID))
	if err != nil {
		return processSQLErrorf(err, "Failed to get the existing primary path")
	}

	// map to db path to ensure we store valueUnique.
	dbNew, err := mapToDBPath(newPath, transformation)
	if err != nil {
		return fmt.Errorf("failed to map db path: %w", err)
	}

	// ValueUnique is the same => routing is the same, ensure we don't keep the old as alias (duplicate error)
	if dbNew.ValueUnique == dbExisting.ValueUnique {
		keepAsAlias = false
	}

	// Space specific checks.
	if newPath.TargetType == enum.PathTargetTypeSpace {
		/*
		 * IMPORTANT
		 *   To avoid cycles in the primary graph, we have to ensure that the old path isn't a parent of the new path.
		 *	 We have to look at the unique path here, as that is used for routing and duplicate detection.
		 */
		if strings.HasPrefix(dbNew.ValueUnique, dbExisting.ValueUnique+types.PathSeparator) {
			return store.ErrIllegalMoveCyclicHierarchy
		}
	}

	// Only look for children if the type can have children
	if newPath.TargetType == enum.PathTargetTypeSpace {
		err = replaceChildrenPathsTx(ctx, db, tx, &dbExisting.Path, newPath, keepAsAlias, transformation)
		if err != nil {
			return err
		}
	}

	// make existing an alias (or delete)
	// IMPORTANT: delete before insert as a casing only change in the path is a valid input.
	// It's part of a db transaction so it should be okay.
	query := pathDeleteID
	if keepAsAlias {
		query = pathMakeAliasID
	}
	if _, err = tx.ExecContext(ctx, query, dbExisting.ID); err != nil {
		return processSQLErrorf(err, "Failed to mark existing path '%s' as alias (or delete)", dbExisting.Value)
	}

	// insert the new Path
	query, arg, err := db.BindNamed(pathInsert, dbNew)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind path object")
	}

	_, err = tx.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to create new primary path '%s'", newPath.Value)
	}

	return nil
}

func replaceChildrenPathsTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx,
	existing *types.Path, updated *types.Path, keepAsAlias bool, transformation store.PathTransformation) error {
	var childPaths []*path
	// get all primary paths that start with the current path before updating (or we can run into recursion)
	childPaths, err := listPrimaryChildPathsTx(ctx, tx, existing.Value, transformation)
	if err != nil {
		return errors.Wrapf(err, "Failed to get primary child paths for '%s'", existing.Value)
	}

	for _, child := range childPaths {
		// create path with updated path (child already is primary)
		updatedChild := new(types.Path)
		*updatedChild = child.Path
		updatedChild.ID = 0 // will be regenerated
		updatedChild.Created = updated.Created
		updatedChild.Updated = updated.Updated
		updatedChild.CreatedBy = updated.CreatedBy
		updatedChild.Value = updated.Value + updatedChild.Value[len(existing.Value):]

		// ensure new child path length is okay
		if check.IsPathTooDeep(updatedChild.Value, updated.TargetType == enum.PathTargetTypeSpace) {
			log.Warn().Msgf("Path '%s' is too long.", updated.Value)
			return store.ErrPathTooLong
		}

		var (
			query string
			args  []interface{}
		)

		// make existing child path an alias (or delete)
		// IMPORTANT: delete before insert as a casing only change in the original path is a valid input.
		// It's part of a db transaction so it should be okay.
		query = pathDeleteID
		if keepAsAlias {
			query = pathMakeAliasID
		}
		if _, err = tx.ExecContext(ctx, query, child.ID); err != nil {
			return processSQLErrorf(err, "Failed to mark existing child path '%s' as alias (or delete)",
				updatedChild.Value)
		}

		// map to db path to ensure we store valueUnique.
		var dbUpdatedChild *path
		dbUpdatedChild, err = mapToDBPath(updatedChild, transformation)
		if err != nil {
			return fmt.Errorf("failed to map db path: %w", err)
		}

		query, args, err = db.BindNamed(pathInsert, dbUpdatedChild)
		if err != nil {
			return processSQLErrorf(err, "Failed to bind path object")
		}

		if _, err = tx.ExecContext(ctx, query, args...); err != nil {
			return processSQLErrorf(err, "Failed to create new primary child path '%s'", updatedChild.Value)
		}
	}

	return nil
}

// FindPathTx finds the primary path for a target.
func FindPathTx(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetID int64) (*types.Path, error) {
	dst := new(path)
	err := tx.GetContext(ctx, dst, pathSelectPrimaryForTarget, string(targetType), fmt.Sprint(targetID))
	if err != nil {
		return nil, processSQLErrorf(err, "Failed to find path")
	}

	return mapDBPath(dst), nil
}

// DeletePath deletes a specific path alias (primary can't be deleted, only with delete all).
func DeletePath(ctx context.Context, db *sqlx.DB, id int64) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// ensure path is an alias
	dst := new(path)
	if err = tx.GetContext(ctx, dst, pathSelectID, id); err != nil {
		return processSQLErrorf(err, "Failed to find path with id %d", id)
	}
	if !dst.IsAlias {
		return store.ErrPrimaryPathCantBeDeleted
	}

	// delete the path
	if _, err = tx.ExecContext(ctx, pathDeleteID, id); err != nil {
		return processSQLErrorf(err, "Delete query failed")
	}

	if err = tx.Commit(); err != nil {
		return processSQLErrorf(err, "Failed to commit transaction")
	}

	return nil
}

// DeleteAllPaths deletes all paths for a target as part of a transaction.
func DeleteAllPaths(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetID int64) error {
	// delete all entries for the target
	if _, err := tx.ExecContext(ctx, pathDeleteTarget, string(targetType), fmt.Sprint(targetID)); err != nil {
		return processSQLErrorf(err, "Query for deleting all pahts failed")
	}
	return nil
}

// CountPaths returns the count of paths for a specified target.
func CountPaths(ctx context.Context, db *sqlx.DB, targetType enum.PathTargetType, targetID int64,
	opts *types.PathFilter) (int64, error) {
	var count int64
	err := db.QueryRowContext(ctx, pathCount, string(targetType), fmt.Sprint(targetID)).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// ListPaths lists all paths for a target.
func ListPaths(ctx context.Context, db *sqlx.DB, targetType enum.PathTargetType, targetID int64,
	opts *types.PathFilter) ([]*types.Path, error) {
	dst := []*path{}
	// else we construct the sql statement.
	stmt := builder.
		Select("*").
		From("paths").
		Where("path_target_type = ? AND path_target_id = ?", string(targetType), fmt.Sprint(targetID))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.PathAttrPath, enum.PathAttrNone:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("path_value " + opts.Order.String())
	case enum.PathAttrCreated:
		stmt = stmt.OrderBy("path_created " + opts.Order.String())
	case enum.PathAttrUpdated:
		stmt = stmt.OrderBy("path_updated " + opts.Order.String())
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, processSQLErrorf(err, "Customer select query failed")
	}

	return mapDBPaths(dst), nil
}

// CountPathsTx counts paths for a target as part of a transaction.
func CountPathsTx(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, pathCount, string(targetType), fmt.Sprint(targetID)).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Query failed")
	}
	return count, nil
}

func mapDBPath(dbPath *path) *types.Path {
	return &dbPath.Path
}

func mapDBPaths(dbPaths []*path) []*types.Path {
	res := make([]*types.Path, len(dbPaths))
	for i := range dbPaths {
		res[i] = mapDBPath(dbPaths[i])
	}
	return res
}

func mapToDBPath(p *types.Path, transformation store.PathTransformation) (*path, error) {
	// path comes from outside.
	if p == nil {
		return nil, fmt.Errorf("path is nil")
	}

	valueUnique, err := transformation(p.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to transform path: %w", err)
	}
	dbPath := &path{
		Path:        *p,
		ValueUnique: valueUnique,
	}

	return dbPath, nil
}

const pathBase = `
SELECT
path_id
,path_value
,path_value_unique
,path_is_alias
,path_target_type
,path_target_id
,path_created_by
,path_created
,path_updated
FROM paths
`

// there's only one entry with a given target & targetId for isAlias -- false.
const pathSelectPrimaryForTarget = pathBase + `
WHERE path_target_type = $1 AND path_target_id = $2 AND path_is_alias = false
`

const pathSelectPrimaryForPrefixUnique = pathBase + `
WHERE path_value_unique LIKE $1 AND path_is_alias = false
`

const pathCount = `
SELECT count(*)
FROM paths
WHERE path_target_type = $1 AND path_target_id = $2
`

const pathCountPrimaryForPrefixUnique = `
SELECT count(*)
FROM paths
WHERE path_value_unique LIKE $1 AND path_is_alias = false
`

const pathInsert = `
INSERT INTO paths (
	path_value
	,path_value_unique
	,path_is_alias
	,path_target_type
	,path_target_id
	,path_created_by
	,path_created
	,path_updated
) values (
	:path_value
	,:path_value_unique
	,:path_is_alias
	,:path_target_type
	,:path_target_id
	,:path_created_by
	,:path_created
	,:path_updated
) RETURNING path_id
`

const pathSelectID = pathBase + `
WHERE path_id = $1
`

const pathDeleteID = `
DELETE FROM paths
WHERE path_id = $1
`

const pathDeleteTarget = `
DELETE FROM paths
WHERE path_target_type = $1 AND path_target_id = $2
`

const pathMakeAliasID = `
UPDATE paths
SET
path_is_alias		= 1
WHERE path_id = $1
`

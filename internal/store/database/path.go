// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/errs"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// Creates a new path
func CreatePath(ctx context.Context, db *sqlx.DB, path *types.Path) error {

	// ensure path length is okay
	if check.PathTooLong(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		return errs.WrapInPathTooLongf("Path '%s' is too long.", path.Value)
	}

	// In case it's not an alias, ensure there are no duplicates
	if !path.IsAlias {
		if cnt, err := CountPaths(ctx, db, path.TargetType, path.TargetId); err != nil {
			return err
		} else if cnt > 0 {
			return errs.PrimaryPathAlreadyExists
		}
	}

	query, arg, err := db.BindNamed(pathInsert, path)
	if err != nil {
		return wrapSqlErrorf(err, "Failed to bind path object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return wrapSqlErrorf(err, "Insert query failed")
	}

	return nil
}

// Creates a new path as part of a transaction
func CreatePathTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, path *types.Path) error {

	// ensure path length is okay
	if check.PathTooLong(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		return errs.WrapInPathTooLongf("Path '%s' is too long.", path.Value)
	}

	// In case it's not an alias, ensure there are no duplicates
	if !path.IsAlias {
		if cnt, err := CountPathsTx(ctx, tx, path.TargetType, path.TargetId); err != nil {
			return err
		} else if cnt > 0 {
			return errs.PrimaryPathAlreadyExists
		}
	}

	query, arg, err := db.BindNamed(pathInsert, path)
	if err != nil {
		return wrapSqlErrorf(err, "Failed to bind path object")
	}

	if err = tx.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return wrapSqlErrorf(err, "Insert query failed")
	}

	return nil
}

func CountPrimaryChildPathsTx(ctx context.Context, tx *sqlx.Tx, prefix string) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, pathCountPrimaryForPrefix, paths.Concatinate(prefix, "%")).Scan(&count)
	if err != nil {
		return 0, wrapSqlErrorf(err, "Count query failed")
	}
	return count, nil
}

func ListPrimaryChildPathsTx(ctx context.Context, tx *sqlx.Tx, prefix string) ([]*types.Path, error) {
	childs := []*types.Path{}

	if err := tx.SelectContext(ctx, &childs, pathSelectPrimaryForPrefix, paths.Concatinate(prefix, "%")); err != nil {
		return nil, wrapSqlErrorf(err, "Select query failed")
	}

	return childs, nil
}

// Replaces the path for a target as part of a transaction - keeps the existing as alias if requested.
func ReplacePathTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, path *types.Path, keepAsAlias bool) error {

	if path.IsAlias {
		return errs.PrimaryPathRequired
	}

	// ensure new path length is okay
	if check.PathTooLong(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		return errs.WrapInPathTooLongf("Path '%s' is too long.", path.Value)
	}

	// existing is always non-alias (as query filters for IsAlias=0)
	existing := new(types.Path)
	err := tx.GetContext(ctx, existing, pathSelectPrimaryForTarget, string(path.TargetType), fmt.Sprint(path.TargetId))
	if err != nil {
		return wrapSqlErrorf(err, "Failed to get the existing primary path")
	}

	// Only look for childs if the type can have childs
	if path.TargetType == enum.PathTargetTypeSpace {

		// get all primary paths that start with the current path before updating (or we can run into recursion)
		childs, err := ListPrimaryChildPathsTx(ctx, tx, existing.Value)
		if err != nil {
			return errors.Wrapf(err, "Failed to get primary child paths for '%s'", existing.Value)
		}

		for _, child := range childs {
			// create path with updated path (child already is primary)
			updatedChild := new(types.Path)
			*updatedChild = *child
			updatedChild.ID = 0 // will be regenerated
			updatedChild.Created = path.Created
			updatedChild.Updated = path.Updated
			updatedChild.CreatedBy = path.CreatedBy
			updatedChild.Value = path.Value + updatedChild.Value[len(existing.Value):]

			// ensure new child path length is okay
			if check.PathTooLong(updatedChild.Value, path.TargetType == enum.PathTargetTypeSpace) {
				return errs.WrapInPathTooLongf("Path '%s' is too long.", updatedChild.Value)
			}

			query, arg, err := db.BindNamed(pathInsert, updatedChild)
			if err != nil {
				return wrapSqlErrorf(err, "Failed to bind path object")
			}

			_, err = tx.ExecContext(ctx, query, arg...)
			if err != nil {
				return wrapSqlErrorf(err, "Failed to create new primary child path '%s'", updatedChild.Value)
			}

			// make current child an alias or delete it
			if keepAsAlias {
				_, err = tx.ExecContext(ctx, pathMakeAlias, child.ID)
			} else {
				_, err = tx.ExecContext(ctx, pathDeleteId, child.ID)
			}
			if err != nil {
				return wrapSqlErrorf(err, "Failed to mark existing child path '%s' as alias", updatedChild.Value)
			}
		}
	}

	// insert the new Path
	query, arg, err := db.BindNamed(pathInsert, path)
	if err != nil {
		return wrapSqlErrorf(err, "Failed to bind path object")
	}

	_, err = tx.ExecContext(ctx, query, arg...)
	if err != nil {
		return wrapSqlErrorf(err, "Failed to create new primary path '%s'", path.Value)
	}

	// make existing an alias
	if keepAsAlias {
		_, err = tx.ExecContext(ctx, pathMakeAlias, existing.ID)
	} else {
		_, err = tx.ExecContext(ctx, pathDeleteId, existing.ID)
	}
	if err != nil {
		return wrapSqlErrorf(err, "Failed to mark existing path '%s' as alias", existing.Value)
	}

	return nil
}

// Finds the primary path for a target.
func FindPathTx(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetId int64) (*types.Path, error) {
	dst := new(types.Path)
	err := tx.GetContext(ctx, dst, pathSelectPrimaryForTarget, string(targetType), fmt.Sprint(targetId))
	if err != nil {
		return nil, wrapSqlErrorf(err, "Select query failed")
	}

	return dst, nil
}

// Deletes a specific path alias (primary can't be deleted, only with delete all).
func DeletePath(ctx context.Context, db *sqlx.DB, id int64) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapSqlErrorf(err, "Failed to start a new transaction")
	}
	defer tx.Rollback()

	// ensure path is an alias
	dst := new(types.Path)
	err = tx.GetContext(ctx, dst, pathSelectId, id)
	if err != nil {
		return wrapSqlErrorf(err, "Failed to find path with id %d", id)
	} else if dst.IsAlias == false {
		return errs.PrimaryPathCantBeDeleted
	}

	// delete the path
	if _, err = tx.ExecContext(ctx, pathDeleteId, id); err != nil {
		return wrapSqlErrorf(err, "Delete query failed", id)
	}

	if err = tx.Commit(); err != nil {
		return wrapSqlErrorf(err, "Failed to commit transaction")
	}

	return nil
}

// Deletes all paths for a target as part of a transaction.
func DeleteAllPaths(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetId int64) error {
	// delete all entries for the target
	if _, err := tx.ExecContext(ctx, pathDeleteTarget, string(targetType), fmt.Sprint(targetId)); err != nil {
		return wrapSqlErrorf(err, "Query for deleting all pahts failed")
	}
	return nil
}

// Lists all paths for a target.
func ListPaths(ctx context.Context, db *sqlx.DB, targetType enum.PathTargetType, targetId int64, opts *types.PathFilter) ([]*types.Path, error) {
	dst := []*types.Path{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.PathAttrNone {
		err := db.SelectContext(ctx, &dst, pathSelect, string(targetType), fmt.Sprint(targetId), limit(opts.Size), offset(opts.Page, opts.Size))
		if err != nil {
			return nil, wrapSqlErrorf(err, "Default select query failed")
		}

		return dst, nil
	}

	// else we construct the sql statement.
	stmt := builder.Select("*").From("paths").Where("path_targetType = $1 AND path_targetId = $2", string(targetType), fmt.Sprint(targetId))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.PathAttrCreated:
		// NOTE: string concatination is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("path_created " + opts.Order.String())
	case enum.PathAttrUpdated:
		stmt = stmt.OrderBy("path_updated " + opts.Order.String())
	case enum.PathAttrId:
		stmt = stmt.OrderBy("path_id " + opts.Order.String())
	case enum.PathAttrPath:
		stmt = stmt.OrderBy("path_value" + opts.Order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql); err != nil {
		return nil, wrapSqlErrorf(err, "Customer select query failed")
	}

	return dst, nil
}

// Coutn paths for a target.
func CountPaths(ctx context.Context, db *sqlx.DB, targetType enum.PathTargetType, targetId int64) (int64, error) {
	var count int64
	err := db.QueryRowContext(ctx, pathCount, string(targetType), fmt.Sprint(targetId)).Scan(&count)
	if err != nil {
		return 0, wrapSqlErrorf(err, "Query failed")
	}
	return count, nil
}

// Count paths for a target as part of a transaction.
func CountPathsTx(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetId int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, pathCount, string(targetType), fmt.Sprint(targetId)).Scan(&count)
	if err != nil {
		return 0, wrapSqlErrorf(err, "Query failed")
	}
	return count, nil
}

const pathBase = `
SELECT
path_id
,path_value
,path_isAlias
,path_targetType
,path_targetId
,path_createdBy
,path_created
,path_updated
FROM paths
`
const pathSelect = pathBase + `
WHERE path_targetType = $1 AND path_targetId = $2
ORDER BY path_isAlias DESC, path_value ASC
LIMIT $3 OFFSET $4
`

// there's only one entry with a given target & targetId for isAlias -- false
const pathSelectPrimaryForTarget = pathBase + `
WHERE path_targetType = $1 AND path_targetId = $2 AND path_isAlias = 0
`

const pathSelectPrimaryForPrefix = pathBase + `
WHERE path_value LIKE $1 AND path_isAlias = 0
`

const pathCount = `
SELECT count(*)
FROM paths
WHERE path_targetType = $1 AND path_targetId = $2
`

const pathCountPrimaryForPrefix = `
SELECT count(*)
FROM paths
WHERE path_value LIKE $1 AND path_isAlias = 0
`

const pathInsert = `
INSERT INTO paths (
	path_value
	,path_isAlias
	,path_targetType
	,path_targetId
	,path_createdBy
	,path_created
	,path_updated
) values (
	:path_value
	,:path_isAlias
	,:path_targetType
	,:path_targetId
	,:path_createdBy
	,:path_created
	,:path_updated
) RETURNING path_id
`

const pathSelectId = pathBase + `
WHERE path_id = $1
`

const pathSelectPath = pathBase + `
WHERE path_value = $1
`

const pathDeleteId = `
DELETE FROM paths
WHERE path_id = $1
`

const pathDeleteTarget = `
DELETE FROM paths
WHERE path_targetType = $1 AND path_targetId = $2
`

const pathMakeAlias = `
UPDATE paths
SET
path_isAlias		= 1
WHERE path_id = :path_id
`

// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// CreateAliasPath a new alias path (Don't call this for new path creation!)
func CreateAliasPath(ctx context.Context, db *sqlx.DB, path *types.Path) error {
	if !path.IsAlias {
		return store.ErrAliasPathRequired
	}

	// ensure path length is okay
	if check.PathTooLong(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		log.Warn().Msgf("Path '%s' is too long.", path.Value)
		return store.ErrPathTooLong
	}

	query, arg, err := db.BindNamed(pathInsert, path)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind path object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// CreatePathTx creates a new path as part of a transaction.
func CreatePathTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, path *types.Path) error {
	// ensure path length is okay
	if check.PathTooLong(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
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

	query, arg, err := db.BindNamed(pathInsert, path)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind path object")
	}

	if err = tx.QueryRowContext(ctx, query, arg...).Scan(&path.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

func CountPrimaryChildPathsTx(ctx context.Context, tx *sqlx.Tx, prefix string) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, pathCountPrimaryForPrefix, paths.Concatinate(prefix, "%")).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Count query failed")
	}
	return count, nil
}

func ListPrimaryChildPathsTx(ctx context.Context, tx *sqlx.Tx, prefix string) ([]*types.Path, error) {
	childs := []*types.Path{}

	if err := tx.SelectContext(ctx, &childs, pathSelectPrimaryForPrefix, paths.Concatinate(prefix, "%")); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}

	return childs, nil
}

// ReplacePathTx replace the path for a target as part of a transaction - keeps the existing as alias if requested.
func ReplacePathTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, path *types.Path, keepAsAlias bool) error {
	if path.IsAlias {
		return store.ErrPrimaryPathRequired
	}

	// ensure new path length is okay
	if check.PathTooLong(path.Value, path.TargetType == enum.PathTargetTypeSpace) {
		log.Warn().Msgf("Path '%s' is too long.", path.Value)
		return store.ErrPathTooLong
	}

	// existing is always non-alias (as query filters for IsAlias=0)
	existing := new(types.Path)
	err := tx.GetContext(ctx, existing, pathSelectPrimaryForTarget, string(path.TargetType), fmt.Sprint(path.TargetID))
	if err != nil {
		return processSQLErrorf(err, "Failed to get the existing primary path")
	}

	// Only look for children if the type can have children
	if path.TargetType == enum.PathTargetTypeSpace {
		err = replaceChildrenPathsTx(ctx, db, tx, existing, path, keepAsAlias)
		if err != nil {
			return err
		}
	}

	// insert the new Path
	query, arg, err := db.BindNamed(pathInsert, path)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind path object")
	}

	_, err = tx.ExecContext(ctx, query, arg...)
	if err != nil {
		return processSQLErrorf(err, "Failed to create new primary path '%s'", path.Value)
	}

	// make existing an alias
	query = pathDeleteID
	if keepAsAlias {
		query = pathMakeAliasID
	}
	if _, err = tx.ExecContext(ctx, query, existing.ID); err != nil {
		return processSQLErrorf(err, "Failed to mark existing path '%s' as alias", existing.Value)
	}

	return nil
}

func replaceChildrenPathsTx(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx,
	existing *types.Path, path *types.Path, keepAsAlias bool) error {
	var childPaths []*types.Path
	// get all primary paths that start with the current path before updating (or we can run into recursion)
	childPaths, err := ListPrimaryChildPathsTx(ctx, tx, existing.Value)
	if err != nil {
		return errors.Wrapf(err, "Failed to get primary child paths for '%s'", existing.Value)
	}

	for _, child := range childPaths {
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
			log.Warn().Msgf("Path '%s' is too long.", path.Value)
			return store.ErrPathTooLong
		}

		var (
			query string
			args  []interface{}
		)

		query, args, err = db.BindNamed(pathInsert, updatedChild)
		if err != nil {
			return processSQLErrorf(err, "Failed to bind path object")
		}

		if _, err = tx.ExecContext(ctx, query, args...); err != nil {
			return processSQLErrorf(err, "Failed to create new primary child path '%s'", updatedChild.Value)
		}

		// make current child an alias or delete it
		query = pathDeleteID
		if keepAsAlias {
			query = pathMakeAliasID
		}
		if _, err = tx.ExecContext(ctx, query, child.ID); err != nil {
			return processSQLErrorf(err, "Failed to mark existing child path '%s' as alias",
				updatedChild.Value)
		}
	}

	return nil
}

// FindPathTx finds the primary path for a target.
func FindPathTx(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetID int64) (*types.Path, error) {
	dst := new(types.Path)
	err := tx.GetContext(ctx, dst, pathSelectPrimaryForTarget, string(targetType), fmt.Sprint(targetID))
	if err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}

	return dst, nil
}

// Deletes a specific path alias (primary can't be deleted, only with delete all).
func DeletePath(ctx context.Context, db *sqlx.DB, id int64) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return processSQLErrorf(err, "Failed to start a new transaction")
	}
	defer func(tx *sqlx.Tx) {
		_ = tx.Rollback()
	}(tx)

	// ensure path is an alias
	dst := new(types.Path)
	if err = tx.GetContext(ctx, dst, pathSelectID, id); err != nil {
		return processSQLErrorf(err, "Failed to find path with id %d", id)
	} else if !dst.IsAlias {
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

// Deletes all paths for a target as part of a transaction.
func DeleteAllPaths(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetID int64) error {
	// delete all entries for the target
	if _, err := tx.ExecContext(ctx, pathDeleteTarget, string(targetType), fmt.Sprint(targetID)); err != nil {
		return processSQLErrorf(err, "Query for deleting all pahts failed")
	}
	return nil
}

// Lists all paths for a target.
func ListPaths(ctx context.Context, db *sqlx.DB, targetType enum.PathTargetType, targetID int64,
	opts *types.PathFilter) ([]*types.Path, error) {
	dst := []*types.Path{}

	// if the principal does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.PathAttrNone {
		err := db.SelectContext(ctx, &dst, pathSelect, string(targetType), fmt.Sprint(targetID), limit(opts.Size),
			offset(opts.Page, opts.Size))
		if err != nil {
			return nil, processSQLErrorf(err, "Default select query failed")
		}

		return dst, nil
	}

	// else we construct the sql statement.
	stmt := builder.Select("*").From("paths").Where("path_targetType = $1 AND path_targetId = $2",
		string(targetType), fmt.Sprint(targetID))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.PathAttrCreated:
		// NOTE: string concatenation is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("path_created " + opts.Order.String())
	case enum.PathAttrUpdated:
		stmt = stmt.OrderBy("path_updated " + opts.Order.String())
	case enum.PathAttrID:
		stmt = stmt.OrderBy("path_id " + opts.Order.String())
	case enum.PathAttrPath:
		stmt = stmt.OrderBy("path_value" + opts.Order.String())
	case enum.PathAttrNone:
		// no sorting required
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err = db.SelectContext(ctx, &dst, sql); err != nil {
		return nil, processSQLErrorf(err, "Customer select query failed")
	}

	return dst, nil
}

// CountPathsTx Count paths for a target as part of a transaction.
func CountPathsTx(ctx context.Context, tx *sqlx.Tx, targetType enum.PathTargetType, targetID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, pathCount, string(targetType), fmt.Sprint(targetID)).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Query failed")
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

// there's only one entry with a given target & targetId for isAlias -- false.
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

const pathSelectID = pathBase + `
WHERE path_id = $1
`

const pathDeleteID = `
DELETE FROM paths
WHERE path_id = $1
`

const pathDeleteTarget = `
DELETE FROM paths
WHERE path_targetType = $1 AND path_targetId = $2
`

const pathMakeAliasID = `
UPDATE paths
SET
path_isAlias		= 1
WHERE path_id = $1
`

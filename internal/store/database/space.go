// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ store.SpaceStore = (*SpaceStore)(nil)

// Returns a new SpaceStore.
func NewSpaceStore(db *sqlx.DB) *SpaceStore {
	return &SpaceStore{db}
}

// Implements a SpaceStore backed by a relational database.
type SpaceStore struct {
	db *sqlx.DB
}

// Finds the space by id.
func (s *SpaceStore) Find(ctx context.Context, id int64) (*types.Space, error) {
	dst := new(types.Space)
	err := s.db.Get(dst, spaceSelectID, id)
	return dst, err
}

// Finds the space by the full qualified space name.
func (s *SpaceStore) FindFqn(ctx context.Context, fqn string) (*types.Space, error) {
	dst := new(types.Space)
	err := s.db.Get(dst, spaceSelectFqn, fqn)
	return dst, err
}

// Creates a new space
func (s *SpaceStore) Create(ctx context.Context, space *types.Space) error {
	// TODO: Ensure parent exists!!
	query, arg, err := s.db.BindNamed(spaceInsert, space)
	if err != nil {
		return err
	}
	return s.db.QueryRow(query, arg...).Scan(&space.ID)
}

// Updates the space details.
func (s *SpaceStore) Update(ctx context.Context, space *types.Space) error {
	query, arg, err := s.db.BindNamed(spaceUpdate, space)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(query, arg...)
	return err
}

// Deletes the space.
func (s *SpaceStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// ensure there are no child spaces
	var count int64
	if err := tx.QueryRow(spaceCount, id).Scan(&count); err != nil {
		return err
	} else if count > 0 {
		// TODO: still returns 500
		return errors.New(fmt.Sprintf("Space still contains %d child space(s).", count))
	}

	// delete the space
	if _, err := tx.Exec(spaceDelete, id); err != nil {
		return err
	}
	return tx.Commit()
}

// List returns a list of spaces under the parent space.
func (s *SpaceStore) List(ctx context.Context, id int64, opts types.SpaceFilter) ([]*types.Space, error) {
	dst := []*types.Space{}

	// if the user does not provide any customer filter
	// or sorting we use the default select statement.
	if opts.Sort == enum.SpaceAttrNone {
		err := s.db.Select(&dst, spaceSelect, id, limit(opts.Size), offset(opts.Page, opts.Size))
		return dst, err
	}

	// else we construct the sql statement.
	stmt := builder.Select("*").From("spaces").Where("space_parentId = " + fmt.Sprint(id))
	stmt = stmt.Limit(uint64(limit(opts.Size)))
	stmt = stmt.Offset(uint64(offset(opts.Page, opts.Size)))

	switch opts.Sort {
	case enum.SpaceAttrCreated:
		// NOTE: string concatination is safe because the
		// order attribute is an enum and is not user-defined,
		// and is therefore not subject to injection attacks.
		stmt = stmt.OrderBy("space_created " + opts.Order.String())
	case enum.SpaceAttrUpdated:
		stmt = stmt.OrderBy("space_updated " + opts.Order.String())
	case enum.SpaceAttrId:
		stmt = stmt.OrderBy("space_id " + opts.Order.String())
	case enum.SpaceAttrName:
		stmt = stmt.OrderBy("space_name " + opts.Order.String())
	case enum.SpaceAttrFqn:
		stmt = stmt.OrderBy("space_fqn " + opts.Order.String())
	}

	sql, _, err := stmt.ToSql()
	if err != nil {
		return dst, err
	}

	err = s.db.Select(&dst, sql)
	return dst, err
}

// Count the child spaces of a space.
func (s *SpaceStore) Count(ctx context.Context, id int64) (int64, error) {
	var count int64
	err := s.db.QueryRow(spaceCount, id).Scan(&count)
	return count, err
}

const spaceBase = `
SELECT
 space_id
,space_name
,space_fqn
,space_parentId
,space_displayName
,space_description
,space_isPublic
,space_createdBy
,space_created
,space_updated
FROM spaces
`

const spaceSelect = spaceBase + `
WHERE space_parentId = $1
ORDER BY space_fqn ASC
LIMIT $2 OFFSET $3
`

const spaceCount = `
SELECT count(*)
FROM spaces
WHERE space_parentId = $1
`

const spaceSelectID = spaceBase + `
WHERE space_id = $1
`

const spaceSelectFqn = spaceBase + `
WHERE space_fqn = $1
`

const spaceDelete = `
DELETE FROM spaces
WHERE space_id = $1
`

const spaceInsert = `
INSERT INTO spaces (
	space_name
   ,space_fqn
   ,space_parentId
   ,space_displayName
   ,space_description
   ,space_isPublic
   ,space_createdBy
   ,space_created
   ,space_updated
) values (
   :space_name
   ,:space_fqn
   ,:space_parentId
   ,:space_displayName
   ,:space_description
   ,:space_isPublic
   ,:space_createdBy
   ,:space_created
   ,:space_updated
   )RETURNING space_id
`

const spaceUpdate = `
UPDATE spaces
SET
space_displayName   = :space_displayName
,space_description  = :space_description
,space_isPublic     = :space_isPublic
,space_updated      = :space_updated
WHERE space_id = :space_id
`

// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.SpacePathStore = (*SpacePathStore)(nil)

// NewSpacePathStore returns a new SpacePathStore.
func NewSpacePathStore(db *sqlx.DB, pathTransformation store.SpacePathTransformation) *SpacePathStore {
	return &SpacePathStore{
		db:                      db,
		spacePathTransformation: pathTransformation,
	}
}

// SpacePathStore implements a store.SpacePathStore backed by a relational database.
type SpacePathStore struct {
	db                      *sqlx.DB
	spacePathTransformation store.SpacePathTransformation
}

// spacePathSegment is an internal representation of a segment of a space path.
type spacePathSegment struct {
	ID int64 `db:"space_path_id"`
	// UID is the original uid that was provided
	UID string `db:"space_path_uid"`
	// UIDUnique is a transformed version of UID which is used to ensure uniqueness guarantees
	UIDUnique string `db:"space_path_uid_unique"`
	// IsPrimary indicates whether the path is the primary path of the space
	// IMPORTANT: to allow DB enforcement of at most one primary path per repo/space
	// we have a unique index on spaceID + IsPrimary and set IsPrimary to true
	// for primary paths and to nil for non-primary paths.
	IsPrimary null.Bool `db:"space_path_is_primary"`
	ParentID  null.Int  `db:"space_path_parent_id"`
	SpaceID   int64     `db:"space_path_space_id"`
	CreatedBy int64     `db:"space_path_created_by"`
	Created   int64     `db:"space_path_created"`
	Updated   int64     `db:"space_path_updated"`
}

const (
	spacePathColumns = `
		space_path_uid
		,space_path_uid_unique
		,space_path_is_primary
		,space_path_parent_id
		,space_path_space_id
		,space_path_created_by
		,space_path_created
		,space_path_updated`

	spacePathSelectBase = `
		SELECT` + spacePathColumns + `
		FROM space_paths`
)

// InsertSegment inserts a space path segment to the table - returns the full path.
func (s *SpacePathStore) InsertSegment(ctx context.Context, segment *types.SpacePathSegment) error {
	const sqlQuery = `
		INSERT INTO space_paths (
			 space_path_uid
			,space_path_uid_unique
			,space_path_is_primary
			,space_path_parent_id
			,space_path_space_id
			,space_path_created_by
			,space_path_created
			,space_path_updated
		) values (
			:space_path_uid
			,:space_path_uid_unique
			,:space_path_is_primary
			,:space_path_parent_id
			,:space_path_space_id
			,:space_path_created_by
			,:space_path_created
			,:space_path_updated
		)`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, s.mapToInternalSpacePathSegment(segment))
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind path segment object")
	}

	if _, err = db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(err, "Insert query failed")
	}

	return nil
}

func (s *SpacePathStore) FindPrimaryBySpaceID(ctx context.Context, spaceID int64) (*types.SpacePath, error) {
	sqlQuery := spacePathSelectBase + `
		where space_path_space_id = $1 AND space_path_is_primary = TRUE`

	db := dbtx.GetAccessor(ctx, s.db)
	dst := new(spacePathSegment)

	path := ""
	nextSpaceID := null.IntFrom(spaceID)

	for nextSpaceID.Valid {
		err := db.GetContext(ctx, dst, sqlQuery, nextSpaceID.Int64)
		if err != nil {
			return nil, database.ProcessSQLErrorf(err, "Failed to find primary segment for %d", nextSpaceID.Int64)
		}

		path = paths.Concatinate(dst.UID, path)
		nextSpaceID = dst.ParentID
	}

	return &types.SpacePath{
		SpaceID:   spaceID,
		Value:     path,
		IsPrimary: true,
	}, nil
}
func (s *SpacePathStore) FindByPath(ctx context.Context, path string) (*types.SpacePath, error) {
	const sqlQueryParent = spacePathSelectBase + ` WHERE space_path_uid_unique = $1 AND space_path_parent_id = $2`
	const sqlQueryNoParent = spacePathSelectBase + ` WHERE space_path_uid_unique = $1 AND space_path_parent_id IS NULL`

	db := dbtx.GetAccessor(ctx, s.db)
	segment := new(spacePathSegment)

	segmentUIDs := paths.Segments(path)
	if len(segmentUIDs) == 0 {
		return nil, fmt.Errorf("path with no segments was passed '%s'", path)
	}

	var parentID int64
	sqlquery := sqlQueryNoParent
	originalPath := ""
	isPrimary := true
	for i, segmentUID := range segmentUIDs {
		uniqueSegmentUID := s.spacePathTransformation(segmentUID, i == 0)
		err := db.GetContext(ctx, segment, sqlquery, uniqueSegmentUID, parentID)
		if err != nil {
			return nil, database.ProcessSQLErrorf(err, "Failed to find segment for '%s' in '%s'", uniqueSegmentUID, path)
		}

		originalPath = paths.Concatinate(originalPath, segment.UID)
		parentID = segment.SpaceID
		isPrimary = isPrimary && segment.IsPrimary.ValueOrZero()
		sqlquery = sqlQueryParent
	}

	return &types.SpacePath{
		Value:     originalPath,
		IsPrimary: isPrimary,
		SpaceID:   segment.SpaceID,
	}, nil
}

// DeletePrimarySegment deletes the primary segment of the space.
func (s *SpacePathStore) DeletePrimarySegment(ctx context.Context, spaceID int64) error {
	const sqlQuery = `
		DELETE FROM space_paths
		WHERE space_path_space_id = $1 AND space_path_is_primary = TRUE`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, spaceID); err != nil {
		return database.ProcessSQLErrorf(err, "the delete query failed")
	}

	return nil
}

func (s *SpacePathStore) mapToInternalSpacePathSegment(p *types.SpacePathSegment) *spacePathSegment {
	res := &spacePathSegment{
		ID:        p.ID,
		UID:       p.UID,
		UIDUnique: s.spacePathTransformation(p.UID, p.ParentID == 0),
		SpaceID:   p.SpaceID,
		Created:   p.Created,
		CreatedBy: p.CreatedBy,
		Updated:   p.Updated,

		// ParentID:  is set below
		// IsPrimary: is set below
	}

	// only set IsPrimary to a value if it's true (Unique Index doesn't allow multiple false, hence keep it nil)
	if p.IsPrimary {
		res.IsPrimary = null.BoolFrom(true)
	}

	if p.ParentID > 0 {
		res.ParentID = null.IntFrom(p.ParentID)
	}

	return res
}

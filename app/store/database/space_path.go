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

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.SpacePathStore = (*SpacePathStore)(nil)

// NewSpacePathStore returns a new SpacePathStore.
func NewSpacePathStore(
	db *sqlx.DB,
	pathTransformation store.SpacePathTransformation,
) *SpacePathStore {
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
	// Identifier is the original identifier that was provided
	Identifier string `db:"space_path_uid"`
	// IdentifierUnique is a transformed version of Identifier which is used to ensure uniqueness guarantees
	IdentifierUnique string `db:"space_path_uid_unique"`
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
		) RETURNING space_path_id`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, s.mapToInternalSpacePathSegment(segment))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind path segment object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&segment.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
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
			return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find primary segment for %d", nextSpaceID.Int64)
		}

		path = paths.Concatenate(dst.Identifier, path)
		nextSpaceID = dst.ParentID
	}

	return &types.SpacePath{
		SpaceID:   spaceID,
		Value:     path,
		IsPrimary: true,
	}, nil
}
func (s *SpacePathStore) FindByPath(ctx context.Context, path string) (*types.SpacePath, error) {
	const sqlQueryNoParent = spacePathSelectBase + ` WHERE space_path_uid_unique = $1 AND space_path_parent_id IS NULL`
	const sqlQueryParent = spacePathSelectBase + ` WHERE space_path_uid_unique = $1 AND space_path_parent_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)
	segment := new(spacePathSegment)

	segmentIdentifiers := paths.Segments(path)
	if len(segmentIdentifiers) == 0 {
		return nil, fmt.Errorf("path with no segments was passed '%s'", path)
	}

	var err error
	var parentID int64
	originalPath := ""
	isPrimary := true
	for i, segmentIdentifier := range segmentIdentifiers {
		uniqueSegmentIdentifier := s.spacePathTransformation(segmentIdentifier, i == 0)

		if parentID == 0 {
			err = db.GetContext(ctx, segment, sqlQueryNoParent, uniqueSegmentIdentifier)
		} else {
			err = db.GetContext(ctx, segment, sqlQueryParent, uniqueSegmentIdentifier, parentID)
		}
		if err != nil {
			return nil, database.ProcessSQLErrorf(
				ctx,
				err,
				"Failed to find segment for '%s' in '%s'",
				uniqueSegmentIdentifier,
				path,
			)
		}

		originalPath = paths.Concatenate(originalPath, segment.Identifier)
		parentID = segment.SpaceID
		isPrimary = isPrimary && segment.IsPrimary.ValueOrZero()
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
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

// DeletePathsAndDescendandPaths deletes all space paths reachable from spaceID including itself.
func (s *SpacePathStore) DeletePathsAndDescendandPaths(ctx context.Context, spaceID int64) error {
	const sqlQuery = `WITH RECURSIVE DescendantPaths AS (
		SELECT space_path_id, space_path_space_id, space_path_parent_id
		FROM space_paths
		WHERE space_path_space_id = $1
	  
		UNION
	  
		SELECT sp.space_path_id, sp.space_path_space_id, sp.space_path_parent_id
		FROM space_paths sp
		JOIN DescendantPaths dp ON sp.space_path_parent_id = dp.space_path_space_id
	  )
	  DELETE FROM space_paths
	  WHERE space_path_id IN (SELECT space_path_id FROM DescendantPaths);`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, spaceID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (s *SpacePathStore) mapToInternalSpacePathSegment(p *types.SpacePathSegment) *spacePathSegment {
	res := &spacePathSegment{
		ID:               p.ID,
		Identifier:       p.Identifier,
		IdentifierUnique: s.spacePathTransformation(p.Identifier, p.ParentID == 0),
		SpaceID:          p.SpaceID,
		Created:          p.Created,
		CreatedBy:        p.CreatedBy,
		Updated:          p.Updated,

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

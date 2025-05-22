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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ store.FavoriteStore = (*FavoriteStore)(nil)

// NewFavoriteStore returns a new FavoriteStore.
func NewFavoriteStore(db *sqlx.DB) *FavoriteStore {
	return &FavoriteStore{db}
}

// FavoriteStore implements a store backed by a relational database.
type FavoriteStore struct {
	db *sqlx.DB
}

type favorite struct {
	ResourceID  int64 `db:"favorite_resource_id"`
	PrincipalID int64 `db:"favorite_principal_id"`
	Created     int64 `db:"favorite_created"`
}

// Create marks the resource as favorite.
func (s *FavoriteStore) Create(ctx context.Context, in *types.FavoriteResource) error {
	tableName, resourceColumnName, err := getTableAndColumnName(in.Type)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to fetch table and column name for favorite resource")
	}

	favoriteResourceInsert := fmt.Sprintf(favoriteInsert, tableName, resourceColumnName)

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(favoriteResourceInsert, favorite{
		ResourceID:  in.ID,
		PrincipalID: in.PrincipalID,
		Created:     time.Now().UnixMilli(),
	})
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to bind favorite object")
	}

	if _, err := db.ExecContext(ctx, query, arg...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to insert in %s", tableName)
	}

	return nil
}

// Map returns a map for the given resourceIDs and checks if the entity has been marked favorite or not.
func (s *FavoriteStore) Map(
	ctx context.Context,
	principalID int64,
	resourceType enum.ResourceType,
	resourceIDs []int64,
) (map[int64]bool, error) {
	tableName, resourceColumnName, err := getTableAndColumnName(resourceType)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to resolve table/column for resourceType %v", resourceType)
	}

	stmt := database.Builder.
		Select(resourceColumnName).
		From(tableName).
		Where("favorite_principal_id = ?", principalID)

	switch s.db.DriverName() {
	case SqliteDriverName:
		stmt = stmt.Where(squirrel.Eq{resourceColumnName: resourceIDs})
	case PostgresDriverName:
		query := fmt.Sprintf("%s = ANY(?)", resourceColumnName)
		stmt = stmt.Where(query, pq.Array(resourceIDs))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var foundIDs []int64
	if err := db.SelectContext(ctx, &foundIDs, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(
			ctx, err, "failed to fetch %s favorites for principal %d", resourceType, principalID)
	}

	result := make(map[int64]bool, len(resourceIDs))
	for _, id := range foundIDs {
		result[id] = true
	}

	return result, nil
}

// Delete unfavorites the resource.
func (s *FavoriteStore) Delete(ctx context.Context, in *types.FavoriteResource) error {
	tableName, resourceColumnName, err := getTableAndColumnName(in.Type)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to fetch table and column name for favorite resource")
	}

	favoriteResourceDelete := fmt.Sprintf(favoriteDelete, tableName, resourceColumnName)

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, favoriteResourceDelete, in.ID, in.PrincipalID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "delete query failed for %s", tableName)
	}

	return nil
}

func getTableAndColumnName(resourceType enum.ResourceType) (string, string, error) {
	switch resourceType { // nolint:exhaustive
	case enum.ResourceTypeRepo:
		return "favorite_repos", "favorite_repo_id", nil
	default:
		return "", "", fmt.Errorf("resource type %v not onboarded to favorites", resourceType)
	}
}

const favoriteInsert = `
INSERT INTO %s (
	%s,
	favorite_principal_id,
	favorite_created
) VALUES (
	:favorite_resource_id,
	:favorite_principal_id,
	:favorite_created
)
`

const favoriteDelete = `
DELETE FROM %s
WHERE %s = :favorite_resource_id AND favorite_principal_id = :favorite_principal_id
`

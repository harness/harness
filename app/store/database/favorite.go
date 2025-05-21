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

	"github.com/jmoiron/sqlx"
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

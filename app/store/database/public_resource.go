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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.PublicResource = (*PublicResourcesStore)(nil)

// NewPublicResourcesStore returns a new PublicResourcesStore.
func NewPublicResourcesStore(db *sqlx.DB) *PublicResourcesStore {
	return &PublicResourcesStore{
		db: db,
	}
}

// PublicResourcesStore implements store.SettingsStore backed by a relational database.
type PublicResourcesStore struct {
	db *sqlx.DB
}

type publicResource struct {
	ID      int64    `db:"public_resource_id"`
	Type    string   `db:"public_resource_type"`
	SpaceID null.Int `db:"public_resource_space_id"`
	RepoID  null.Int `db:"public_resource_repo_id"`
}

const (
	publicResourceColumns = `
	 public_resource_id
	,public_resource_type
	,public_resource_space_id
	,public_resource_repo_id
	`
)

func (p *PublicResourcesStore) Find(
	ctx context.Context,
	publicRsc *types.PublicResource,
) (bool, error) {
	stmt := database.Builder.
		Select(publicResourceColumns).
		From("public_resources").
		Where("public_resource_type = ?", publicRsc.Type)

	switch publicRsc.Type {
	case enum.PublicResourceTypeRepository:
		stmt = stmt.Where("public_resource_repo_id = ?", publicRsc.ResourceID)
	case enum.PublicResourceTypeSpace:
		stmt = stmt.Where("public_resource_space_id = ?", publicRsc.ResourceID)
	default:
		return false, fmt.Errorf("public resource type %q is not supported", publicRsc.Type)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, p.db)

	dst := &publicResource{}
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return true, nil
}

func (p *PublicResourcesStore) Create(
	ctx context.Context,
	publicRsc *types.PublicResource) error {
	stmt := database.Builder.
		Insert("").
		Into("public_resources").
		Columns(
			"public_resource_type",
			"public_resource_space_id",
			"public_resource_repo_id",
		)

	switch publicRsc.Type {
	case enum.PublicResourceTypeRepository:
		stmt = stmt.Values(enum.PublicResourceTypeRepository, null.Int{}, null.IntFrom(publicRsc.ResourceID))
	case enum.PublicResourceTypeSpace:
		stmt = stmt.Values(enum.PublicResourceTypeSpace, null.IntFrom(publicRsc.ResourceID), null.Int{})
	default:
		return fmt.Errorf("public resource type %q is not supported", publicRsc.Type)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, p.db)

	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

func (p *PublicResourcesStore) Delete(
	ctx context.Context,
	publicRsc *types.PublicResource) error {
	stmt := database.Builder.
		Delete("public_resources").
		Where("public_resource_type = ?", publicRsc.Type)

	switch publicRsc.Type {
	case enum.PublicResourceTypeRepository:
		stmt = stmt.Where("public_resource_repo_id = ?", publicRsc.ResourceID)
	case enum.PublicResourceTypeSpace:
		stmt = stmt.Where("public_resource_space_id = ?", publicRsc.ResourceID)
	default:
		return fmt.Errorf("public resource type %q is not supported", publicRsc.Type)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert delete public resource query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, p.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

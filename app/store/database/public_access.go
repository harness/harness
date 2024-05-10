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
	"github.com/harness/gitness/types/enum"

	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
)

var _ store.PublicAccessStore = (*PublicAccessStore)(nil)

// NewPublicAccessStore returns a new PublicAccessStore.
func NewPublicAccessStore(db *sqlx.DB) *PublicAccessStore {
	return &PublicAccessStore{
		db: db,
	}
}

// PublicAccessStore implements store.PublicAccessStore backed by a relational database.
type PublicAccessStore struct {
	db *sqlx.DB
}

type publicAccess struct {
	ID      int64    `db:"public_access_id"`
	SpaceID null.Int `db:"public_access_space_id"`
	RepoID  null.Int `db:"public_access_repo_id"`
}

const (
	publicAccessColumns = `
	 public_access_id
	,public_access_space_id
	,public_access_repo_id
	`
)

func (p *PublicAccessStore) Find(
	ctx context.Context,
	typ enum.PublicResourceType,
	id int64,
) error {
	stmt := database.Builder.
		Select(publicAccessColumns).
		From("public_access")

	switch typ {
	case enum.PublicResourceTypeRepo:
		stmt = stmt.Where("public_access_repo_id = ?", id)
	case enum.PublicResourceTypeSpace:
		stmt = stmt.Where("public_access_space_id = ?", id)
	default:
		return fmt.Errorf("public resource type %q is not supported", typ)
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, p.db)

	dst := &publicAccess{}
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return nil
}

func (p *PublicAccessStore) Create(
	ctx context.Context,
	typ enum.PublicResourceType,
	id int64,
) error {
	stmt := database.Builder.
		Insert("").
		Into("public_access")

	switch typ {
	case enum.PublicResourceTypeRepo:
		stmt = stmt.Columns("public_access_repo_id").Values(null.IntFrom(id))
	case enum.PublicResourceTypeSpace:
		stmt = stmt.Columns("public_access_space_id").Values(null.IntFrom(id))
	default:
		return fmt.Errorf("public resource type %q is not supported", typ)
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

func (p *PublicAccessStore) Delete(
	ctx context.Context,
	typ enum.PublicResourceType,
	id int64,
) error {
	stmt := database.Builder.
		Delete("public_access")

	switch typ {
	case enum.PublicResourceTypeRepo:
		stmt = stmt.Where("public_access_repo_id = ?", id)
	case enum.PublicResourceTypeSpace:
		stmt = stmt.Where("public_access_space_id = ?", id)
	default:
		return fmt.Errorf("public resource type %q is not supported", typ)
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

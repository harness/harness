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

func (p *PublicAccessStore) Find(
	ctx context.Context,
	typ enum.PublicResourceType,
	id int64,
) (bool, error) {
	var sqlQuery string

	switch typ {
	case enum.PublicResourceTypeRepo:
		sqlQuery = `SELECT EXISTS(SELECT * FROM public_access_repo WHERE public_access_repo_id = $1)`
	case enum.PublicResourceTypeSpace:
		sqlQuery = `SELECT EXISTS(SELECT * FROM public_access_space WHERE public_access_space_id = $1)`
	default:
		return false, fmt.Errorf("public resource type %q is not supported", typ)
	}

	var exists bool
	db := dbtx.GetAccessor(ctx, p.db)

	if err := db.QueryRowContext(ctx, sqlQuery, id).Scan(&exists); err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Select query failed")
	}

	return exists, nil
}

func (p *PublicAccessStore) Create(
	ctx context.Context,
	typ enum.PublicResourceType,
	id int64,
) error {
	var sqlQuery string
	switch typ {
	case enum.PublicResourceTypeRepo:
		sqlQuery = `INSERT INTO public_access_repo(public_access_repo_id) VALUES($1)`
	case enum.PublicResourceTypeSpace:
		sqlQuery = `INSERT INTO public_access_space(public_access_space_id) VALUES($1)`
	default:
		return fmt.Errorf("public resource type %q is not supported", typ)
	}

	db := dbtx.GetAccessor(ctx, p.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

func (p *PublicAccessStore) Delete(
	ctx context.Context,
	typ enum.PublicResourceType,
	id int64,
) error {
	var sqlQuery string
	switch typ {
	case enum.PublicResourceTypeRepo:
		sqlQuery = `DELETE FROM public_access_repo WHERE public_access_repo_id = $1`
	case enum.PublicResourceTypeSpace:
		sqlQuery = `DELETE FROM public_access_space WHERE public_access_space_id = $1`
	default:
		return fmt.Errorf("public resource type %q is not supported", typ)
	}

	db := dbtx.GetAccessor(ctx, p.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
)

var _ store.PublicKeySubKeyStore = PublicKeySubKeyStore{}

// NewPublicKeySubKeyStore returns a new PublicKeySubKeyStore.
func NewPublicKeySubKeyStore(db *sqlx.DB) PublicKeySubKeyStore {
	return PublicKeySubKeyStore{
		db: db,
	}
}

// PublicKeySubKeyStore implements a store.PublicKeySubKeyStore backed by a relational database.
type PublicKeySubKeyStore struct {
	db *sqlx.DB
}

// Create creates a new public key.
func (s PublicKeySubKeyStore) Create(ctx context.Context, publicKeyID int64, pgpKeyIDs []string) error {
	if len(pgpKeyIDs) == 0 {
		return nil
	}

	const sqlQuery = `
		INSERT INTO public_key_sub_keys(public_key_sub_key_public_key_id, public_key_sub_key_id)
		VALUES ($1, $2)`

	db := dbtx.GetAccessor(ctx, s.db)

	stmt, err := db.PrepareContext(ctx, sqlQuery)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to prepare insert public key subkey statement")
	}

	defer stmt.Close()

	for _, pgpKeyID := range pgpKeyIDs {
		_, err = stmt.ExecContext(ctx, publicKeyID, pgpKeyID)
		if err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Insert public key subkey query failed")
		}
	}

	return nil
}

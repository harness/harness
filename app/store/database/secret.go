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
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.SecretStore = (*secretStore)(nil)

const (
	secretQueryBase = `
		SELECT` + secretColumns + `
		FROM secrets`

	//nolint:gosec // wrong flagging
	secretColumns = `
	secret_id,
	secret_description,
	secret_space_id,
	secret_created_by,
	secret_uid,
	secret_data,
	secret_created,
	secret_updated,
	secret_version
	`
)

// NewSecretStore returns a new SecretStore.
func NewSecretStore(db *sqlx.DB) store.SecretStore {
	return &secretStore{
		db: db,
	}
}

type secretStore struct {
	db *sqlx.DB
}

// Find returns a secret given a secret ID.
func (s *secretStore) Find(ctx context.Context, id int64) (*types.Secret, error) {
	const findQueryStmt = secretQueryBase + `
		WHERE secret_id = $1`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Secret)
	if err := db.GetContext(ctx, dst, findQueryStmt, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find secret")
	}
	return dst, nil
}

// FindByIdentifier returns a secret in a given space with a given identifier.
func (s *secretStore) FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.Secret, error) {
	const findQueryStmt = secretQueryBase + `
		WHERE secret_space_id = $1 AND secret_uid = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Secret)
	if err := db.GetContext(ctx, dst, findQueryStmt, spaceID, identifier); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find secret")
	}
	return dst, nil
}

// Create creates a secret.
func (s *secretStore) Create(ctx context.Context, secret *types.Secret) error {
	//nolint:gosec // wrong flagging
	const secretInsertStmt = `
	INSERT INTO secrets (
		secret_description,
		secret_space_id,
		secret_created_by,
		secret_uid,
		secret_data,
		secret_created,
		secret_updated,
		secret_version
	) VALUES (
		:secret_description,
		:secret_space_id,
		:secret_created_by,
		:secret_uid,
		:secret_data,
		:secret_created,
		:secret_updated,
		:secret_version
	) RETURNING secret_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(secretInsertStmt, secret)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind secret object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&secret.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "secret query failed")
	}

	return nil
}

func (s *secretStore) Update(ctx context.Context, p *types.Secret) error {
	const secretUpdateStmt = `
	UPDATE secrets
	SET
		secret_description = :secret_description,
		secret_uid = :secret_uid,
		secret_data = :secret_data,
		secret_updated = :secret_updated,
		secret_version = :secret_version
	WHERE secret_id = :secret_id AND secret_version = :secret_version - 1`
	updatedAt := time.Now()
	secret := *p

	secret.Version++
	secret.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(secretUpdateStmt, secret)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind secret object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update secret")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	p.Version = secret.Version
	p.Updated = secret.Updated
	return nil
}

// UpdateOptLock updates the pipeline using the optimistic locking mechanism.
func (s *secretStore) UpdateOptLock(ctx context.Context,
	secret *types.Secret,
	mutateFn func(secret *types.Secret) error,
) (*types.Secret, error) {
	for {
		dup := *secret

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		secret, err = s.Find(ctx, secret.ID)
		if err != nil {
			return nil, err
		}
	}
}

// List lists all the secrets present in a space.
func (s *secretStore) List(ctx context.Context, parentID int64, filter types.ListQueryFilter) ([]*types.Secret, error) {
	stmt := database.Builder.
		Select(secretColumns).
		From("secrets").
		Where("secret_space_id = ?", fmt.Sprint(parentID))

	if filter.Query != "" {
		stmt = stmt.Where(PartialMatch("secret_uid", filter.Query))
	}

	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Secret{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return dst, nil
}

// ListAll lists all the secrets present in a space.
func (s *secretStore) ListAll(ctx context.Context, parentID int64) ([]*types.Secret, error) {
	stmt := database.Builder.
		Select(secretColumns).
		From("secrets").
		Where("secret_space_id = ?", fmt.Sprint(parentID))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Secret{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing custom list query")
	}

	return dst, nil
}

// Delete deletes a secret given a secret ID.
func (s *secretStore) Delete(ctx context.Context, id int64) error {
	//nolint:gosec // wrong flagging
	const secretDeleteStmt = `
		DELETE FROM secrets
		WHERE secret_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, secretDeleteStmt, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete secret")
	}

	return nil
}

// DeleteByIdentifier deletes a secret with a given identifier in a space.
func (s *secretStore) DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error {
	//nolint:gosec // wrong flagging
	const secretDeleteStmt = `
	DELETE FROM secrets
	WHERE secret_space_id = $1 AND secret_uid = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, secretDeleteStmt, spaceID, identifier); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Could not delete secret")
	}

	return nil
}

// Count of secrets in a space.
func (s *secretStore) Count(ctx context.Context, parentID int64, filter types.ListQueryFilter) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("secrets").
		Where("secret_space_id = ?", parentID)

	if filter.Query != "" {
		stmt = stmt.Where(PartialMatch("secret_uid", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}
	return count, nil
}

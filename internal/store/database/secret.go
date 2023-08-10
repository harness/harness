// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/internal/store"
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

	secretColumns = `
	secret_id,
	secret_description,
	secret_space_id,
	secret_uid,
	secret_data,
	secret_created,
	secret_updated,
	secret_version
	`
)

// NewSecretStore returns a new SecretStore.
func NewSecretStore(db *sqlx.DB) *secretStore {
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
		return nil, database.ProcessSQLErrorf(err, "Failed to find secret")
	}
	return dst, nil
}

// FindByUID returns a secret in a given space with a given UID.
func (s *secretStore) FindByUID(ctx context.Context, spaceID int64, uid string) (*types.Secret, error) {
	const findQueryStmt = secretQueryBase + `
		WHERE secret_space_id = $1 AND secret_uid = $2`
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Secret)
	if err := db.GetContext(ctx, dst, findQueryStmt, spaceID, uid); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed to find secret")
	}
	return dst, nil
}

// Create creates a secret.
func (s *secretStore) Create(ctx context.Context, secret *types.Secret) error {
	const secretInsertStmt = `
	INSERT INTO secrets (
		secret_description,
		secret_space_id,
		secret_uid,
		secret_data,
		secret_created,
		secret_updated,
		secret_version
	) VALUES (
		:secret_description,
		:secret_space_id,
		:secret_uid,
		:secret_data,
		:secret_created,
		:secret_updated,
		:secret_version
	) RETURNING secret_id`
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(secretInsertStmt, secret)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind secret object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&secret.ID); err != nil {
		return database.ProcessSQLErrorf(err, "secret query failed")
	}

	return nil
}

func (s *secretStore) Update(ctx context.Context, secret *types.Secret) error {
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

	secret.Version++
	secret.Updated = updatedAt.UnixMilli()

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(secretUpdateStmt, secret)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to bind secret object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to update secret")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

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
func (s *secretStore) List(ctx context.Context, parentID int64, pagination types.Pagination) ([]*types.Secret, error) {
	stmt := database.Builder.
		Select(secretColumns).
		From("secrets").
		Where("secret_space_id = ?", fmt.Sprint(parentID))

	if pagination.Query != "" {
		stmt = stmt.Where("LOWER(secret_uid) LIKE ?", fmt.Sprintf("%%%s%%", strings.ToLower(pagination.Query)))
	}

	stmt = stmt.Limit(database.Limit(pagination.Size))
	stmt = stmt.Offset(database.Offset(pagination.Page, pagination.Size))

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Secret{}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(err, "Failed executing custom list query")
	}

	return dst, nil
}

// Delete deletes a secret given a secret ID.
func (s *secretStore) Delete(ctx context.Context, id int64) error {
	const secretDeleteStmt = `
		DELETE FROM secrets
		WHERE secret_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, secretDeleteStmt, id); err != nil {
		return database.ProcessSQLErrorf(err, "Could not delete secret")
	}

	return nil
}

// DeleteByUID deletes a secret with a given UID in a space.
func (s *secretStore) DeleteByUID(ctx context.Context, spaceID int64, uid string) error {
	const secretDeleteStmt = `
	DELETE FROM secrets
	WHERE secret_space_id = $1 AND secret_uid = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, secretDeleteStmt, spaceID, uid); err != nil {
		return database.ProcessSQLErrorf(err, "Could not delete secret")
	}

	return nil
}

// Count of secrets in a space.
func (s *secretStore) Count(ctx context.Context, parentID int64, filter types.Pagination) (int64, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("secrets").
		Where("secret_space_id = ?", parentID)

	if filter.Query != "" {
		stmt = stmt.Where("secret_uid LIKE ?", fmt.Sprintf("%%%s%%", filter.Query))
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err = db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

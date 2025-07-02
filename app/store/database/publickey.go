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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ store.PublicKeyStore = PublicKeyStore{}

// NewPublicKeyStore returns a new PublicKeyStore.
func NewPublicKeyStore(db *sqlx.DB) PublicKeyStore {
	return PublicKeyStore{
		db: db,
	}
}

// PublicKeyStore implements a store.PublicKeyStore backed by a relational database.
type PublicKeyStore struct {
	db *sqlx.DB
}

type publicKey struct {
	ID int64 `db:"public_key_id"`

	PrincipalID int64 `db:"public_key_principal_id"`

	Created  int64    `db:"public_key_created"`
	Verified null.Int `db:"public_key_verified"`

	Identifier string `db:"public_key_identifier"`
	Usage      string `db:"public_key_usage"`

	Fingerprint string `db:"public_key_fingerprint"`
	Content     string `db:"public_key_content"`
	Comment     string `db:"public_key_comment"`
	Type        string `db:"public_key_type"`

	Scheme string `db:"public_key_scheme"`

	ValidFrom        null.Int    `db:"public_key_valid_from"`
	ValidTo          null.Int    `db:"public_key_valid_to"`
	RevocationReason null.String `db:"public_key_revocation_reason"`

	Metadata json.RawMessage `db:"public_key_metadata"`
}

const (
	publicKeyColumns = `
		 public_key_id
		,public_key_principal_id
		,public_key_created
		,public_key_verified
		,public_key_identifier
		,public_key_usage
		,public_key_fingerprint
		,public_key_content
		,public_key_comment
		,public_key_type
		,public_key_scheme
		,public_key_valid_from
		,public_key_valid_to
		,public_key_revocation_reason
		,public_key_metadata`

	publicKeySelectBase = `
		SELECT` + publicKeyColumns + `
		FROM public_keys`
)

// Find fetches a job by its unique identifier.
func (s PublicKeyStore) Find(ctx context.Context, id int64) (*types.PublicKey, error) {
	const sqlQuery = publicKeySelectBase + `
	WHERE public_key_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	result := &publicKey{}
	if err := db.GetContext(ctx, result, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find public key by id")
	}

	key := mapToPublicKey(result)

	return &key, nil
}

// FindByIdentifier returns a public key given a principal ID and an identifier.
func (s PublicKeyStore) FindByIdentifier(
	ctx context.Context,
	principalID int64,
	identifier string,
) (*types.PublicKey, error) {
	const sqlQuery = publicKeySelectBase + `
	WHERE public_key_principal_id = $1 and LOWER(public_key_identifier) = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	result := &publicKey{}
	if err := db.GetContext(ctx, result, sqlQuery, principalID, strings.ToLower(identifier)); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find public key by principal and identifier")
	}

	key := mapToPublicKey(result)

	return &key, nil
}

// Create creates a new public key.
func (s PublicKeyStore) Create(ctx context.Context, key *types.PublicKey) error {
	const sqlQuery = `
		INSERT INTO public_keys (
			 public_key_principal_id
			,public_key_created
			,public_key_verified
			,public_key_identifier
			,public_key_usage
			,public_key_fingerprint
			,public_key_content
			,public_key_comment
			,public_key_type
			,public_key_scheme
			,public_key_valid_from
			,public_key_valid_to
			,public_key_revocation_reason
			,public_key_metadata
		) values (
			 :public_key_principal_id
			,:public_key_created
			,:public_key_verified
			,:public_key_identifier
			,:public_key_usage
			,:public_key_fingerprint
			,:public_key_content
			,:public_key_comment
			,:public_key_type
			,:public_key_scheme
			,:public_key_valid_from
			,:public_key_valid_to
			,:public_key_revocation_reason
			,:public_key_metadata
		) RETURNING public_key_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dbKey := mapToInternalPublicKey(key)

	query, arg, err := db.BindNamed(sqlQuery, &dbKey)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind public key object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&dbKey.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert public key query failed")
	}

	key.ID = dbKey.ID

	return nil
}

func (s PublicKeyStore) Update(ctx context.Context, publicKey *types.PublicKey) error {
	const sqlQuery = `
		UPDATE public_keys
		SET
			 public_key_valid_from = :public_key_valid_from
			,public_key_valid_to = :public_key_valid_to
			,public_key_revocation_reason = :public_key_revocation_reason
		WHERE public_key_id = :public_key_id`

	dbPublicKey := mapToInternalPublicKey(publicKey)

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, dbPublicKey)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind public key object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update public key")
	}

	return nil
}

// DeleteByIdentifier deletes a public key.
func (s PublicKeyStore) DeleteByIdentifier(ctx context.Context, principalID int64, identifier string) error {
	const sqlQuery = `DELETE FROM public_keys WHERE public_key_principal_id = $1 and LOWER(public_key_identifier) = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, principalID, strings.ToLower(identifier))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Delete public key query failed")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "RowsAffected after delete of public key failed")
	}

	if count == 0 {
		return errors.NotFound("Key not found")
	}

	return nil
}

// MarkAsVerified updates the public key to mark it as verified.
func (s PublicKeyStore) MarkAsVerified(ctx context.Context, id int64, verified int64) error {
	const sqlQuery = `
		UPDATE public_keys
		SET public_key_verified = $1
		WHERE public_key_id = $2`

	if _, err := dbtx.GetAccessor(ctx, s.db).ExecContext(ctx, sqlQuery, verified, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to mark public key as varified")
	}

	return nil
}

func (s PublicKeyStore) Count(
	ctx context.Context,
	principalID *int64,
	filter *types.PublicKeyFilter,
) (int, error) {
	stmt := database.Builder.
		Select("count(*)").
		From("public_keys")

	if principalID != nil {
		stmt = stmt.Where("public_key_principal_id = ?", *principalID)
	}

	stmt = s.applyQueryFilter(stmt, filter)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int

	if err := db.QueryRowContext(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to execute count public keys query")
	}

	return count, nil
}

// List returns the public keys for the principal.
func (s PublicKeyStore) List(
	ctx context.Context,
	principalID *int64,
	filter *types.PublicKeyFilter,
) ([]types.PublicKey, error) {
	return s.list(ctx, func(stmt squirrel.SelectBuilder) squirrel.SelectBuilder {
		if principalID != nil {
			stmt = stmt.Where("public_key_principal_id = ?", *principalID)
		}

		stmt = s.applyQueryFilter(stmt, filter)
		stmt = s.applyPagination(stmt, filter.Pagination)
		stmt = s.applySortFilter(stmt, filter.Sort, filter.Order)

		return stmt
	})
}

// ListByFingerprint returns public keys given a fingerprint and key usage.
func (s PublicKeyStore) ListByFingerprint(
	ctx context.Context,
	fingerprint string,
	principalID *int64,
	usages []enum.PublicKeyUsage,
	schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	return s.list(ctx, func(stmt squirrel.SelectBuilder) squirrel.SelectBuilder {
		stmt = stmt.Where("public_key_fingerprint = ?", fingerprint)
		if principalID != nil {
			stmt = stmt.Where("public_key_principal_id = ?", *principalID)
		}
		stmt = s.applyUsages(stmt, usages)
		stmt = s.applySchemes(stmt, schemes)
		stmt = s.applySortFilter(stmt, enum.PublicKeySortCreated, enum.OrderDesc)
		return stmt
	})
}

func (s PublicKeyStore) ListBySubKeyID(
	ctx context.Context,
	subKeyID string,
	principalID *int64,
	usages []enum.PublicKeyUsage,
	schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	return s.list(ctx, func(stmt squirrel.SelectBuilder) squirrel.SelectBuilder {
		stmt = stmt.Join("public_key_sub_keys ON public_key_sub_key_public_key_id = public_key_id").
			Where("public_key_sub_key_id = ?", subKeyID)
		if principalID != nil {
			stmt = stmt.Where("public_key_principal_id = ?", *principalID)
		}
		stmt = s.applyUsages(stmt, usages)
		stmt = s.applySchemes(stmt, schemes)
		stmt = s.applySortFilter(stmt, enum.PublicKeySortCreated, enum.OrderDesc)
		return stmt
	})
}

// List returns the public keys for the principal.
func (s PublicKeyStore) list(
	ctx context.Context,
	builder func(squirrel.SelectBuilder) squirrel.SelectBuilder,
) ([]types.PublicKey, error) {
	stmt := database.Builder.
		Select(publicKeyColumns).
		From("public_keys")

	stmt = builder(stmt)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	keys := make([]publicKey, 0)
	if err = db.SelectContext(ctx, &keys, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to execute list public keys query")
	}

	return mapToPublicKeys(keys), nil
}

func (s PublicKeyStore) applyQueryFilter(
	stmt squirrel.SelectBuilder,
	filter *types.PublicKeyFilter,
) squirrel.SelectBuilder {
	if filter.Query != "" {
		stmt = stmt.Where(PartialMatch("public_key_identifier", filter.Query))
	}

	stmt = s.applyUsages(stmt, filter.Usages)
	stmt = s.applySchemes(stmt, filter.Schemes)

	return stmt
}

func (s PublicKeyStore) applyUsages(
	stmt squirrel.SelectBuilder,
	usages []enum.PublicKeyUsage,
) squirrel.SelectBuilder {
	if len(usages) == 1 {
		stmt = stmt.Where("public_key_usage = ?", usages[0])
	} else if len(usages) > 1 {
		switch s.db.DriverName() {
		case SqliteDriverName:
			stmt = stmt.Where(squirrel.Eq{"public_key_usage": usages})
		case PostgresDriverName:
			stmt = stmt.Where("public_key_usage = ANY(?)", pq.Array(usages))
		}
	}
	return stmt
}

func (s PublicKeyStore) applySchemes(
	stmt squirrel.SelectBuilder,
	schemes []enum.PublicKeyScheme,
) squirrel.SelectBuilder {
	if len(schemes) == 1 {
		stmt = stmt.Where("public_key_scheme = ?", schemes[0])
	} else if len(schemes) > 1 {
		switch s.db.DriverName() {
		case SqliteDriverName:
			stmt = stmt.Where(squirrel.Eq{"public_key_scheme": schemes})
		case PostgresDriverName:
			stmt = stmt.Where("public_key_scheme = ANY(?)", pq.Array(schemes))
		}
	}
	return stmt
}

func (PublicKeyStore) applyPagination(
	stmt squirrel.SelectBuilder,
	filter types.Pagination,
) squirrel.SelectBuilder {
	stmt = stmt.Limit(database.Limit(filter.Size))
	stmt = stmt.Offset(database.Offset(filter.Page, filter.Size))
	return stmt
}

func (PublicKeyStore) applySortFilter(
	stmt squirrel.SelectBuilder,
	sort enum.PublicKeySort,
	order enum.Order,
) squirrel.SelectBuilder {
	if order == enum.OrderDefault {
		order = enum.OrderAsc
	}

	switch sort {
	case enum.PublicKeySortIdentifier:
		stmt = stmt.OrderBy("public_key_identifier " + order.String())
	case enum.PublicKeySortCreated:
		stmt = stmt.OrderBy("public_key_created " + order.String())
	}

	return stmt
}

func mapToInternalPublicKey(in *types.PublicKey) publicKey {
	var revocationReason null.String
	if in.RevocationReason != nil {
		revocationReason.Valid = true
		revocationReason.String = string(*in.RevocationReason)
	}
	return publicKey{
		ID:               in.ID,
		PrincipalID:      in.PrincipalID,
		Created:          in.Created,
		Verified:         null.IntFromPtr(in.Verified),
		Identifier:       in.Identifier,
		Usage:            string(in.Usage),
		Fingerprint:      in.Fingerprint,
		Content:          in.Content,
		Comment:          in.Comment,
		Type:             in.Type,
		Scheme:           string(in.Scheme),
		ValidFrom:        null.IntFromPtr(in.ValidFrom),
		ValidTo:          null.IntFromPtr(in.ValidTo),
		RevocationReason: revocationReason,
		Metadata:         in.Metadata,
	}
}

func mapToPublicKey(in *publicKey) types.PublicKey {
	return types.PublicKey{
		ID:               in.ID,
		PrincipalID:      in.PrincipalID,
		Created:          in.Created,
		Verified:         in.Verified.Ptr(),
		Identifier:       in.Identifier,
		Usage:            enum.PublicKeyUsage(in.Usage),
		Fingerprint:      in.Fingerprint,
		Content:          in.Content,
		Comment:          in.Comment,
		Type:             in.Type,
		Scheme:           enum.PublicKeyScheme(in.Scheme),
		ValidFrom:        in.ValidFrom.Ptr(),
		ValidTo:          in.ValidTo.Ptr(),
		RevocationReason: (*enum.RevocationReason)(in.RevocationReason.Ptr()),
		Metadata:         in.Metadata,
	}
}

func mapToPublicKeys(
	keys []publicKey,
) []types.PublicKey {
	res := make([]types.PublicKey, len(keys))
	for i := 0; i < len(keys); i++ {
		res[i] = mapToPublicKey(&keys[i])
	}
	return res
}

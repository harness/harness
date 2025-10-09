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
	"strings"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var _ store.TokenStore = (*TokenStore)(nil)

// NewTokenStore returns a new TokenStore.
func NewTokenStore(db *sqlx.DB) *TokenStore {
	return &TokenStore{db}
}

// TokenStore implements a TokenStore backed by a relational database.
type TokenStore struct {
	db *sqlx.DB
}

// Find finds the token by id.
func (s *TokenStore) Find(ctx context.Context, id int64) (*types.Token, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Token)
	if err := db.GetContext(ctx, dst, TokenSelectByID, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find token")
	}

	return dst, nil
}

// FindByIdentifier finds the token by principalId and token identifier.
func (s *TokenStore) FindByIdentifier(ctx context.Context, principalID int64, identifier string) (*types.Token, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := new(types.Token)
	if err := db.GetContext(
		ctx,
		dst,
		TokenSelectByPrincipalIDAndIdentifier,
		principalID,
		strings.ToLower(identifier),
	); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find token by identifier")
	}

	return dst, nil
}

// Create saves the token details.
func (s *TokenStore) Create(ctx context.Context, token *types.Token) error {
	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(tokenInsert, token)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind token object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&token.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

// Delete deletes the token with the given id.
func (s *TokenStore) Delete(ctx context.Context, id int64) error {
	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, tokenDelete, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "The delete query failed")
	}

	return nil
}

// DeleteExpiredBefore deletes all tokens that expired before the provided time.
// If tokenTypes are provided, then only tokens of that type are deleted.
func (s *TokenStore) DeleteExpiredBefore(
	ctx context.Context,
	before time.Time,
	tknTypes []enum.TokenType,
) (int64, error) {
	stmt := database.Builder.
		Delete("tokens").
		Where("token_expires_at < ?", before.UnixMilli())

	if len(tknTypes) > 0 {
		stmt = stmt.Where(squirrel.Eq{"token_type": tknTypes})
	}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to convert delete token query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to execute delete token query")
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "failed to get number of deleted tokens")
	}

	return n, nil
}

// Count returns a count of tokens of a specifc type for a specific principal.
func (s *TokenStore) Count(ctx context.Context,
	principalID int64, tokenType enum.TokenType) (int64, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	err := db.QueryRowContext(ctx, tokenCountForPrincipalIDOfType, principalID, tokenType).Scan(&count)
	if err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing count query")
	}

	return count, nil
}

// List returns a list of tokens of a specific type for a specific principal.
func (s *TokenStore) List(ctx context.Context,
	principalID int64, tokenType enum.TokenType) ([]*types.Token, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	dst := []*types.Token{}

	// TODO: custom filters / sorting for tokens.

	err := db.SelectContext(ctx, &dst, tokenSelectForPrincipalIDOfType, principalID, tokenType)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing token list query")
	}
	return dst, nil
}

const tokenSelectBase = `
SELECT
token_id
,token_type
,token_uid
,token_principal_id
,token_expires_at
,token_issued_at
,token_created_by
FROM tokens
` //#nosec G101

const tokenSelectForPrincipalIDOfType = tokenSelectBase + `
WHERE token_principal_id = $1 AND token_type = $2
ORDER BY token_issued_at DESC
` //#nosec G101

const tokenCountForPrincipalIDOfType = `
SELECT count(*)
FROM tokens
WHERE token_principal_id = $1 AND token_type = $2
` //#nosec G101

const TokenSelectByID = tokenSelectBase + `
WHERE token_id = $1
`

const TokenSelectByPrincipalIDAndIdentifier = tokenSelectBase + `
WHERE token_principal_id = $1 AND LOWER(token_uid) = $2
`

const tokenDelete = `
DELETE FROM tokens
WHERE token_id = $1
`

const tokenInsert = `
INSERT INTO tokens (
	token_type
	,token_uid
	,token_principal_id
	,token_expires_at
	,token_issued_at
	,token_created_by
) values (
	:token_type
	,:token_uid
	,:token_principal_id
	,:token_expires_at
	,:token_issued_at
	,:token_created_by
) RETURNING token_id
`

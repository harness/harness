// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

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
	dst := new(types.Token)
	if err := s.db.GetContext(ctx, dst, TokenSelectByID, id); err != nil {
		return nil, processSQLErrorf(err, "Select query failed")
	}
	return dst, nil
}

// Create saves the token details.
func (s *TokenStore) Create(ctx context.Context, token *types.Token) error {
	query, arg, err := s.db.BindNamed(tokenInsert, token)
	if err != nil {
		return processSQLErrorf(err, "Failed to bind token object")
	}

	if err = s.db.QueryRowContext(ctx, query, arg...).Scan(&token.ID); err != nil {
		return processSQLErrorf(err, "Insert query failed")
	}

	return nil
}

// Delete deletes the token with the given id.
func (s *TokenStore) Delete(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, tokenDelete, id); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}

	return nil
}

// DeleteForPrincipal deletes all tokens for a specific principal.
func (s *TokenStore) DeleteForPrincipal(ctx context.Context, principalID int64) error {
	if _, err := s.db.ExecContext(ctx, tokenDeleteForPrincipal, principalID); err != nil {
		return processSQLErrorf(err, "The delete query failed")
	}

	return nil
}

// Count returns a count of tokens of a specifc type for a specific principal.
func (s *TokenStore) Count(ctx context.Context,
	principalID int64, tokenType enum.TokenType) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, tokenCountForPrincipalIDOfType, principalID, tokenType).Scan(&count)
	if err != nil {
		return 0, processSQLErrorf(err, "Failed executing count query")
	}
	return count, nil
}

// List returns a list of tokens of a specific type for a specific principal.
func (s *TokenStore) List(ctx context.Context,
	principalID int64, tokenType enum.TokenType) ([]*types.Token, error) {
	dst := []*types.Token{}

	// TODO: custom filters / sorting for tokens.

	err := s.db.SelectContext(ctx, &dst, tokenSelectForPrincipalIDOfType, principalID, tokenType)
	if err != nil {
		return nil, processSQLErrorf(err, "Failed executing token list query")
	}
	return dst, nil
}

const tokenSelectBase = `
SELECT
token_id
,token_type
,token_name
,token_principalId
,token_expiresAt
,token_grants
,token_issuedAt
,token_createdBy
FROM tokens
` //#nosec G101

const tokenSelectForPrincipalIDOfType = tokenSelectBase + `
WHERE token_principalId = $1 and token_type = $2
ORDER BY token_issuedAt DESC
` //#nosec G101

const tokenCountForPrincipalIDOfType = `
SELECT count(*)
FROM tokens
WHERE token_principalId = $1 and token_type = $2
` //#nosec G101

const TokenSelectByID = tokenSelectBase + `
WHERE token_id = $2
`

const tokenDelete = `
DELETE FROM tokens
WHERE token_id = $1
`

const tokenDeleteForPrincipal = `
DELETE FROM tokens
WHERE token_principalId = $1
`

const tokenInsert = `
INSERT INTO tokens (
	token_type
	,token_name
	,token_principalId
	,token_expiresAt
	,token_grants
	,token_issuedAt
	,token_createdBy
) values (
	:token_type
	,:token_name
	,:token_principalId
	,:token_expiresAt
	,:token_grants
	,:token_issuedAt
	,:token_createdBy
) RETURNING token_id
`

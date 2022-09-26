// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/database/mutex"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ store.TokenStore = (*TokenStoreSync)(nil)

// NewTokenStoreSync returns a new TokenStoreSync.
func NewTokenStoreSync(store *TokenStore) *TokenStoreSync {
	return &TokenStoreSync{base: store}
}

// TokenStoreSync synronizes read and write access to the
// token store. This prevents race conditions when the database
// type is sqlite3.
type TokenStoreSync struct{ base *TokenStore }

// Find finds the token by id.
func (s *TokenStoreSync) Find(ctx context.Context, id int64) (*types.Token, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.Find(ctx, id)
}

// Create saves the token details.
func (s *TokenStoreSync) Create(ctx context.Context, token *types.Token) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Create(ctx, token)
}

// Delete deletes the token with the given id.
func (s *TokenStoreSync) Delete(ctx context.Context, id int64) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Delete(ctx, id)
}

// DeleteForPrincipal deletes all tokens for a specific principal.
func (s *TokenStoreSync) DeleteForPrincipal(ctx context.Context, principalID int64) error {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.DeleteForPrincipal(ctx, principalID)
}

// Count returns a count of tokens of a specifc type for a specific principal.
func (s *TokenStoreSync) Count(ctx context.Context, principalID int64,
	tokenType enum.TokenType) (int64, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return s.base.Count(ctx, principalID, tokenType)
}

// List returns a list of tokens of a specific type for a specific principal.
func (s *TokenStoreSync) List(ctx context.Context, principalID int64,
	tokenType enum.TokenType) ([]*types.Token, error) {
	mutex.RLock()
	defer mutex.RUnlock()
	return s.base.List(ctx, principalID, tokenType)
}

// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"

	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/crypto/bcrypt"
)

type Controller struct {
	principalUIDCheck check.PrincipalUID
	authorizer        authz.Authorizer
	principalStore    store.PrincipalStore
	tokenStore        store.TokenStore
}

func NewController(principalUIDCheck check.PrincipalUID, authorizer authz.Authorizer,
	principalStore store.PrincipalStore, tokenStore store.TokenStore) *Controller {
	return &Controller{
		principalUIDCheck: principalUIDCheck,
		authorizer:        authorizer,
		principalStore:    principalStore,
		tokenStore:        tokenStore,
	}
}

var hashPassword = bcrypt.GenerateFromPassword

func findUserFromUID(ctx context.Context,
	principalStore store.PrincipalStore, userUID string) (*types.User, error) {
	return principalStore.FindUserByUID(ctx, userUID)
}

func findUserFromID(ctx context.Context, principalStore store.PrincipalStore, userID int64) (*types.User, error) {
	return principalStore.FindUser(ctx, userID)
}
func findUserFromEmail(ctx context.Context,
	principalStore store.PrincipalStore, email string) (*types.User, error) {
	return principalStore.FindUserByEmail(ctx, email)
}

func isUserTokenType(tokenType enum.TokenType) bool {
	return tokenType == enum.TokenTypePAT || tokenType == enum.TokenTypeSession
}

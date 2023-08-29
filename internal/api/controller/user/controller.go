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

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type Controller struct {
	db                *sqlx.DB
	principalUIDCheck check.PrincipalUID
	authorizer        authz.Authorizer
	principalStore    store.PrincipalStore
	tokenStore        store.TokenStore
	membershipStore   store.MembershipStore
}

func NewController(
	db *sqlx.DB,
	principalUIDCheck check.PrincipalUID,
	authorizer authz.Authorizer,
	principalStore store.PrincipalStore,
	tokenStore store.TokenStore,
	membershipStore store.MembershipStore,
) *Controller {
	return &Controller{
		db:                db,
		principalUIDCheck: principalUIDCheck,
		authorizer:        authorizer,
		principalStore:    principalStore,
		tokenStore:        tokenStore,
		membershipStore:   membershipStore,
	}
}

var hashPassword = bcrypt.GenerateFromPassword

func findUserFromUID(ctx context.Context,
	principalStore store.PrincipalStore, userUID string,
) (*types.User, error) {
	return principalStore.FindUserByUID(ctx, userUID)
}

func findUserFromEmail(ctx context.Context,
	principalStore store.PrincipalStore, email string,
) (*types.User, error) {
	return principalStore.FindUserByEmail(ctx, email)
}

func isUserTokenType(tokenType enum.TokenType) bool {
	return tokenType == enum.TokenTypePAT || tokenType == enum.TokenTypeSession
}

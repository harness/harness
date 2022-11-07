// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package token

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	userTokenLifeTime time.Duration = 24 * time.Hour   // 1 day.
	oathTokenLifeTime time.Duration = 30 * time.Minute // 30 min.
)

func CreateUserSession(ctx context.Context, tokenStore store.TokenStore,
	user *types.User, uid string) (*types.Token, string, error) {
	principal := types.PrincipalFromUser(user)
	return Create(
		ctx,
		tokenStore,
		enum.TokenTypeSession,
		principal,
		principal,
		uid,
		userTokenLifeTime,
		enum.AccessGrantAll,
	)
}

func CreatePAT(ctx context.Context, tokenStore store.TokenStore,
	createdBy *types.Principal, createdFor *types.User,
	uid string, lifetime time.Duration, grants enum.AccessGrant) (*types.Token, string, error) {
	return Create(
		ctx,
		tokenStore,
		enum.TokenTypePAT,
		createdBy,
		types.PrincipalFromUser(createdFor),
		uid,
		lifetime,
		grants,
	)
}

func CreateSAT(ctx context.Context, tokenStore store.TokenStore,
	createdBy *types.Principal, createdFor *types.ServiceAccount,
	uid string, lifetime time.Duration, grants enum.AccessGrant) (*types.Token, string, error) {
	return Create(
		ctx,
		tokenStore,
		enum.TokenTypeSAT,
		createdBy,
		types.PrincipalFromServiceAccount(createdFor),
		uid,
		lifetime,
		grants,
	)
}

func CreateOAuth(ctx context.Context, tokenStore store.TokenStore,
	createdBy *types.Principal, createdFor *types.User,
	name string, grants enum.AccessGrant) (*types.Token, string, error) {
	return Create(
		ctx,
		tokenStore,
		enum.TokenTypeOAuth2,
		createdBy,
		types.PrincipalFromUser(createdFor),
		name,
		oathTokenLifeTime,
		grants,
	)
}

func Create(ctx context.Context, tokenStore store.TokenStore,
	tokenType enum.TokenType, createdBy *types.Principal, createdFor *types.Principal,
	uid string, lifetime time.Duration, grants enum.AccessGrant) (*types.Token, string, error) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(lifetime)

	// create db entry first so we get the id.
	token := types.Token{
		Type:        tokenType,
		UID:         uid,
		PrincipalID: createdFor.ID,
		IssuedAt:    issuedAt.UnixMilli(),
		ExpiresAt:   expiresAt.UnixMilli(),
		Grants:      grants,
		CreatedBy:   createdBy.ID,
	}

	err := tokenStore.Create(ctx, &token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to store token in db: %w", err)
	}

	// create jwt token.
	jwtToken, err := GenerateJWTForToken(&token, createdFor.Salt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create jwt token: %w", err)
	}

	return &token, jwtToken, nil
}

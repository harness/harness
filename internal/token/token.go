// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package token

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/jwt"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

const (
	userTokenLifeTime time.Duration = 24 * time.Hour // 1 day.
)

func CreateUserSession(
	ctx context.Context,
	tokenStore store.TokenStore,
	user *types.User,
	uid string,
) (*types.Token, string, error) {
	principal := user.ToPrincipal()
	return create(
		ctx,
		tokenStore,
		enum.TokenTypeSession,
		principal,
		principal,
		uid,
		ptr.Duration(userTokenLifeTime),
	)
}

func CreatePAT(
	ctx context.Context,
	tokenStore store.TokenStore,
	createdBy *types.Principal,
	createdFor *types.User,
	uid string,
	lifetime *time.Duration,
) (*types.Token, string, error) {
	return create(
		ctx,
		tokenStore,
		enum.TokenTypePAT,
		createdBy,
		createdFor.ToPrincipal(),
		uid,
		lifetime,
	)
}

func CreateSAT(
	ctx context.Context,
	tokenStore store.TokenStore,
	createdBy *types.Principal,
	createdFor *types.ServiceAccount,
	uid string,
	lifetime *time.Duration,
) (*types.Token, string, error) {
	return create(
		ctx,
		tokenStore,
		enum.TokenTypeSAT,
		createdBy,
		createdFor.ToPrincipal(),
		uid,
		lifetime,
	)
}

func create(
	ctx context.Context,
	tokenStore store.TokenStore,
	tokenType enum.TokenType,
	createdBy *types.Principal,
	createdFor *types.Principal,
	uid string,
	lifetime *time.Duration,
) (*types.Token, string, error) {
	issuedAt := time.Now()

	var expiresAt *int64
	if lifetime != nil {
		expiresAt = ptr.Int64(issuedAt.Add(*lifetime).UnixMilli())
	}

	// create db entry first so we get the id.
	token := types.Token{
		Type:        tokenType,
		UID:         uid,
		PrincipalID: createdFor.ID,
		IssuedAt:    issuedAt.UnixMilli(),
		ExpiresAt:   expiresAt,
		CreatedBy:   createdBy.ID,
	}

	err := tokenStore.Create(ctx, &token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to store token in db: %w", err)
	}

	// create jwt token.
	jwtToken, err := jwt.GenerateForToken(&token, createdFor.Salt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create jwt token: %w", err)
	}

	return &token, jwtToken, nil
}

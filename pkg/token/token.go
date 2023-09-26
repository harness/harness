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

package token

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/pkg/jwt"
	"github.com/harness/gitness/pkg/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

const (
	// userSessionTokenLifeTime is the duration a login / register token is valid.
	// NOTE: Users can list / delete session tokens via rest API if they want to cleanup earlier.
	userSessionTokenLifeTime time.Duration = 30 * 24 * time.Hour // 30 days.
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
		ptr.Duration(userSessionTokenLifeTime),
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

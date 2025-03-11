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
	"math/rand/v2"
	"time"

	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

const (
	// userSessionTokenLifeTime is the duration a login / register token is valid.
	// NOTE: Users can list / delete session tokens via rest API if they want to cleanup earlier.
	userSessionTokenLifeTime                  time.Duration = 30 * 24 * time.Hour // 30 days.
	sessionTokenWithAccessPermissionsLifeTime time.Duration = 24 * time.Hour      // 24 hours.
	RemoteAuthTokenLifeTime                   time.Duration = 15 * time.Minute    // 15 minutes.
)

func CreateUserWithAccessPermissions(
	user *types.User,
	accessPermissions *jwt.SubClaimsAccessPermissions,
) (string, error) {
	principal := user.ToPrincipal()
	return createWithAccessPermissions(
		principal,
		ptr.Duration(sessionTokenWithAccessPermissionsLifeTime),
		accessPermissions,
	)
}

func CreateUserSession(
	ctx context.Context,
	tokenStore store.TokenStore,
	user *types.User,
	identifier string,
) (*types.Token, string, error) {
	principal := user.ToPrincipal()
	return create(
		ctx,
		tokenStore,
		enum.TokenTypeSession,
		principal,
		principal,
		identifier,
		ptr.Duration(userSessionTokenLifeTime),
	)
}

func CreatePAT(
	ctx context.Context,
	tokenStore store.TokenStore,
	createdBy *types.Principal,
	createdFor *types.User,
	identifier string,
	lifetime *time.Duration,
) (*types.Token, string, error) {
	return create(
		ctx,
		tokenStore,
		enum.TokenTypePAT,
		createdBy,
		createdFor.ToPrincipal(),
		identifier,
		lifetime,
	)
}

func CreateSAT(
	ctx context.Context,
	tokenStore store.TokenStore,
	createdBy *types.Principal,
	createdFor *types.ServiceAccount,
	identifier string,
	lifetime *time.Duration,
) (*types.Token, string, error) {
	return create(
		ctx,
		tokenStore,
		enum.TokenTypeSAT,
		createdBy,
		createdFor.ToPrincipal(),
		identifier,
		lifetime,
	)
}

func CreateRemoteAuthToken(
	ctx context.Context,
	tokenStore store.TokenStore,
	principal *types.Principal,
	identifier string,
) (*types.Token, string, error) {
	return create(
		ctx,
		tokenStore,
		enum.TokenTypeRemoteAuth,
		principal,
		principal,
		identifier,
		ptr.Duration(RemoteAuthTokenLifeTime),
	)
}

func GenerateIdentifier(prefix string) string {
	//nolint:gosec // math/rand is sufficient for this use case
	r := rand.IntN(0x10000)
	return fmt.Sprintf("%s-%08x-%04x", prefix, time.Now().Unix(), r)
}

func create(
	ctx context.Context,
	tokenStore store.TokenStore,
	tokenType enum.TokenType,
	createdBy *types.Principal,
	createdFor *types.Principal,
	identifier string,
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
		Identifier:  identifier,
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

func createWithAccessPermissions(
	createdFor *types.Principal,
	lifetime *time.Duration,
	accessPermissions *jwt.SubClaimsAccessPermissions,
) (string, error) {
	jwtToken, err := jwt.GenerateForTokenWithAccessPermissions(
		createdFor.ID, lifetime, createdFor.Salt, accessPermissions,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create jwt token: %w", err)
	}

	return jwtToken, nil
}

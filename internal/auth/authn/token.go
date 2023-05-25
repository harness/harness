// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"

	"github.com/dgrijalva/jwt-go"
)

var _ Authenticator = (*TokenAuthenticator)(nil)

/*
 * Authenticates a user by checking for an access token in the
 * "Authorization" header or the "access_token" form value.
 */
type TokenAuthenticator struct {
	principalStore store.PrincipalStore
	tokenStore     store.TokenStore
}

func NewTokenAuthenticator(
	principalStore store.PrincipalStore,
	tokenStore store.TokenStore) *TokenAuthenticator {
	return &TokenAuthenticator{
		principalStore: principalStore,
		tokenStore:     tokenStore,
	}
}

func (a *TokenAuthenticator) Authenticate(r *http.Request, sourceRouter SourceRouter) (*auth.Session, error) {
	ctx := r.Context()
	str := extractToken(r)

	if len(str) == 0 {
		return nil, ErrNoAuthData
	}

	var principal *types.Principal
	var err error
	claims := &token.JWTClaims{}
	parsed, err := jwt.ParseWithClaims(str, claims, func(token_ *jwt.Token) (interface{}, error) {
		principal, err = a.principalStore.Find(ctx, claims.PrincipalID)
		if err != nil {
			return nil, fmt.Errorf("failed to get principal for token: %w", err)
		}
		return []byte(principal.Salt), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing of JWT claims failed: %w", err)
	}

	if !parsed.Valid {
		return nil, errors.New("parsed JWT token is invalid")
	}

	if _, ok := parsed.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, errors.New("invalid HMAC signature for JWT")
	}

	// ensure tkn exists
	tkn, err := a.tokenStore.Find(ctx, claims.TokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to find token in db: %w", err)
	}

	// protect against faked JWTs for other principals in case of single salt leak
	if principal.ID != tkn.PrincipalID {
		return nil, fmt.Errorf("JWT was for principal %d while db token was for principal %d",
			principal.ID, tkn.PrincipalID)
	}

	return &auth.Session{
		Principal: *principal,
		Metadata: &auth.TokenMetadata{
			TokenType: tkn.Type,
			TokenID:   tkn.ID,
			Grants:    tkn.Grants,
		},
	}, nil
}

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		return r.FormValue("access_token")
	}
	// pull/push git operations will require auth using
	// Basic realm
	if strings.HasPrefix(bearer, "Basic") {
		_, tkn, ok := r.BasicAuth()
		if !ok {
			return ""
		}
		return tkn
	}

	bearer = strings.TrimPrefix(bearer, "Bearer ")
	return bearer
}

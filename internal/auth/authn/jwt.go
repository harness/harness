// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/jwt"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	gojwt "github.com/dgrijalva/jwt-go"
)

var _ Authenticator = (*JWTAuthenticator)(nil)

// JWTAuthenticator uses the provided JWT to authenticate the caller.
type JWTAuthenticator struct {
	principalStore store.PrincipalStore
	tokenStore     store.TokenStore
}

func NewTokenAuthenticator(
	principalStore store.PrincipalStore,
	tokenStore store.TokenStore) *JWTAuthenticator {
	return &JWTAuthenticator{
		principalStore: principalStore,
		tokenStore:     tokenStore,
	}
}

func (a *JWTAuthenticator) Authenticate(r *http.Request, sourceRouter SourceRouter) (*auth.Session, error) {
	ctx := r.Context()
	str := extractToken(r)

	if len(str) == 0 {
		return nil, ErrNoAuthData
	}

	var principal *types.Principal
	var err error
	claims := &jwt.Claims{}
	parsed, err := gojwt.ParseWithClaims(str, claims, func(token_ *gojwt.Token) (interface{}, error) {
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

	if _, ok := parsed.Method.(*gojwt.SigningMethodHMAC); !ok {
		return nil, errors.New("invalid HMAC signature for JWT")
	}

	var metadata auth.Metadata
	switch {
	case claims.Token != nil:
		metadata, err = a.metadataFromTokenClaims(ctx, principal, claims.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata from token claims: %w", err)
		}
	case claims.Membership != nil:
		metadata, err = a.metadataFromMembershipClaims(claims.Membership)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata from membership claims: %w", err)
		}
	default:
		return nil, fmt.Errorf("jwt is missing sub-claims")
	}

	return &auth.Session{
		Principal: *principal,
		Metadata:  metadata,
	}, nil
}

func (a *JWTAuthenticator) metadataFromTokenClaims(
	ctx context.Context,
	principal *types.Principal,
	tknClaims *jwt.SubClaimsToken,
) (auth.Metadata, error) {
	// ensure tkn exists
	tkn, err := a.tokenStore.Find(ctx, tknClaims.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find token in db: %w", err)
	}

	// protect against faked JWTs for other principals in case of single salt leak
	if principal.ID != tkn.PrincipalID {
		return nil, fmt.Errorf("JWT was for principal %d while db token was for principal %d",
			principal.ID, tkn.PrincipalID)
	}

	return &auth.TokenMetadata{
		TokenType: tkn.Type,
		TokenID:   tkn.ID,
	}, nil
}

func (a *JWTAuthenticator) metadataFromMembershipClaims(
	mbsClaims *jwt.SubClaimsMembership,
) (auth.Metadata, error) {
	// We could check if space exists - but also okay to fail later (saves db call)
	return &auth.MembershipMetadata{
		SpaceID: mbsClaims.SpaceID,
		Role:    mbsClaims.Role,
	}, nil
}

func extractToken(r *http.Request) string {
	// Check query param first (as that's most immediately visible to caller)
	if queryToken, ok := request.GetAccessTokenFromQuery(r); ok {
		return queryToken
	}

	// check authorization header next
	headerToken := r.Header.Get(request.HeaderAuthorization)
	switch {
	// in case of git push / pull it would be basic auth and token is in password
	case strings.HasPrefix(headerToken, "Basic "):
		// return pwd either way - if it's invalid pwd is empty string which we'd return anyway
		_, pwd, _ := r.BasicAuth()
		return pwd
	// strip bearer prefix if present
	case strings.HasPrefix(headerToken, "Bearer "):
		return headerToken[7:]
	// otherwise use value as is
	case headerToken != "":
		return headerToken
	}

	// check cookies last (as that's least visible to caller)
	if cookieToken, ok := request.GetTokenFromCookie(r); ok {
		return cookieToken
	}

	// no token found
	return ""
}

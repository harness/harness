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

package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	gojwt "github.com/golang-jwt/jwt/v5"
)

const (
	headerTokenPrefixBearer = "Bearer "
	//nolint:gosec // wrong flagging
	HeaderTokenPrefixRemoteAuth = "RemoteAuth "
)

var _ Authenticator = (*JWTAuthenticator)(nil)

// JWTAuthenticator uses the provided JWT to authenticate the caller.
type JWTAuthenticator struct {
	cookieName     string
	principalStore store.PrincipalStore
	tokenStore     store.TokenStore
}

func NewTokenAuthenticator(
	principalStore store.PrincipalStore,
	tokenStore store.TokenStore,
	cookieName string,
) *JWTAuthenticator {
	return &JWTAuthenticator{
		cookieName:     cookieName,
		principalStore: principalStore,
		tokenStore:     tokenStore,
	}
}

func (a *JWTAuthenticator) Authenticate(r *http.Request) (*auth.Session, error) {
	ctx := r.Context()
	str := extractToken(r, a.cookieName)

	if len(str) == 0 {
		return nil, ErrNoAuthData
	}

	// First, parse claims just to get the principal ID (minimal parsing)
	claims := &jwt.Claims{}
	token, _, err := new(gojwt.Parser).ParseUnverified(str, claims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token format: %w", err)
	}

	// Check if it's the expected token format before proceeding
	if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
		return nil, errors.New("invalid signature method for JWT")
	}

	// Fetch the principal
	principal, err := a.principalStore.Find(ctx, claims.PrincipalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get principal for token: %w", err)
	}

	// Support for multiple secrets (comma-separated)
	saltValues := strings.Split(principal.Salt, ",")
	var lastErr error

	// Try each salt value until one works
	for _, salt := range saltValues {
		salt = strings.TrimSpace(salt)
		// Parse with this salt
		verifiedClaims := &jwt.Claims{}
		parsedToken, err := gojwt.ParseWithClaims(
			str,
			verifiedClaims,
			func(_ *gojwt.Token) (interface{}, error) {
				return []byte(salt), nil
			},
		)

		if err == nil && parsedToken.Valid {
			// Use the helper function to create the session
			return createSessionFromClaims(ctx, a, principal, verifiedClaims)
		}

		lastErr = err
	}

	// All verification attempts failed
	if lastErr != nil {
		return nil, fmt.Errorf("JWT verification failed: %w", lastErr)
	}

	return nil, errors.New("JWT verification failed with all provided salts")
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
		return nil, fmt.Errorf(
			"JWT was for principal %d while db token was for principal %d",
			principal.ID, tkn.PrincipalID,
		)
	}

	return &auth.TokenMetadata{
		TokenType: tkn.Type,
		TokenID:   tkn.ID,
	}, nil
}

func (a *JWTAuthenticator) metadataFromMembershipClaims(
	mbsClaims *jwt.SubClaimsMembership,
) auth.Metadata {
	// We could check if space exists - but also okay to fail later (saves db call)
	return &auth.MembershipMetadata{
		SpaceID: mbsClaims.SpaceID,
		Role:    mbsClaims.Role,
	}
}

func (a *JWTAuthenticator) metadataFromAccessPermissions(
	s *jwt.SubClaimsAccessPermissions,
) auth.Metadata {
	return &auth.AccessPermissionMetadata{
		AccessPermissions: s,
	}
}

func extractToken(r *http.Request, cookieName string) string {
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
	case strings.HasPrefix(headerToken, headerTokenPrefixBearer):
		return headerToken[len(headerTokenPrefixBearer):]
	// for ssh git-lfs-authenticate the returned token prefix would be RemoteAuth of type JWT
	case strings.HasPrefix(headerToken, HeaderTokenPrefixRemoteAuth):
		return headerToken[len(HeaderTokenPrefixRemoteAuth):]
	// otherwise use value as is
	case headerToken != "":
		return headerToken
	}

	// check cookies last (as that's least visible to caller)
	if cookieToken, ok := request.GetTokenFromCookie(r, cookieName); ok {
		return cookieToken
	}

	// no token found
	return ""
}

// createSessionFromClaims creates an auth session from verified JWT claims.
func createSessionFromClaims(
	ctx context.Context,
	a *JWTAuthenticator,
	principal *types.Principal,
	claims *jwt.Claims,
) (*auth.Session, error) {
	var metadata auth.Metadata
	switch {
	case claims.Token != nil:
		tokenMetadata, err := a.metadataFromTokenClaims(ctx, principal, claims.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to get metadata from token claims: %w", err)
		}

		return &auth.Session{
			Principal: *principal,
			Metadata:  tokenMetadata,
		}, nil
	case claims.Membership != nil:
		metadata = a.metadataFromMembershipClaims(claims.Membership)
	case claims.AccessPermissions != nil:
		metadata = a.metadataFromAccessPermissions(claims.AccessPermissions)
	default:
		return nil, fmt.Errorf("jwt is missing sub-claims")
	}

	return &auth.Session{
		Principal: *principal,
		Metadata:  metadata,
	}, nil
}

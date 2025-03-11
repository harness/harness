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

	gojwt "github.com/golang-jwt/jwt"
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

	var principal *types.Principal
	var err error
	claims := &jwt.Claims{}
	parsed, err := gojwt.ParseWithClaims(
		str, claims, func(_ *gojwt.Token) (interface{}, error) {
			principal, err = a.principalStore.Find(ctx, claims.PrincipalID)
			if err != nil {
				return nil, fmt.Errorf("failed to get principal for token: %w", err)
			}
			return []byte(principal.Salt), nil
		},
	)
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

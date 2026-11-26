// Copyright 2026 Harness, Inc.
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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/store"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type infPrincipalStore struct {
	store.PrincipalStore

	principals map[int64]*types.Principal
}

func (s *infPrincipalStore) Find(_ context.Context, id int64) (*types.Principal, error) {
	p, ok := s.principals[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return p, nil
}

type infTokenStore struct {
	store.TokenStore

	tokens map[int64]*types.Token
}

func (s *infTokenStore) Find(_ context.Context, id int64) (*types.Token, error) {
	t, ok := s.tokens[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return t, nil
}

const (
	infSalt     = "alice-salt-0000"
	infAnonSalt = "anonymous-secret-0000"
	infCookie   = "token-cookie"
)

func infAuthenticator() *JWTAuthenticator {
	return NewTokenAuthenticator(
		&infPrincipalStore{principals: map[int64]*types.Principal{
			7: {ID: 7, UID: "alice", Salt: infSalt},
			8: {ID: 8, UID: "rotated", Salt: "new-salt-0000, old-salt-0000"},
		}},
		&infTokenStore{tokens: map[int64]*types.Token{
			100: {ID: 100, PrincipalID: 7, Type: enum.TokenTypePAT},
			101: {ID: 101, PrincipalID: 99, Type: enum.TokenTypePAT},
		}},
		infCookie,
		infAnonSalt,
	)
}

func infRequest(authHeader string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api", nil)
	if authHeader != "" {
		r.Header.Set("Authorization", authHeader)
	}
	return r
}

func infTokenJWT(t *testing.T, id, principalID int64, salt string) string {
	t.Helper()
	token, err := jwt.GenerateForToken(&types.Token{
		ID: id, PrincipalID: principalID, Type: enum.TokenTypePAT, IssuedAt: time.Now().UnixMilli(),
	}, salt)
	require.NoError(t, err)
	return token
}

func TestInfAuthenticate_TokenClaims(t *testing.T) {
	a := infAuthenticator()
	session, err := a.Authenticate(infRequest("Bearer " + infTokenJWT(t, 100, 7, infSalt)))
	require.NoError(t, err)
	require.EqualValues(t, 7, session.Principal.ID)
	require.Equal(t, &auth.TokenMetadata{TokenType: enum.TokenTypePAT, TokenID: 100}, session.Metadata)
}

func TestInfAuthenticate_NoAuthData(t *testing.T) {
	_, err := infAuthenticator().Authenticate(infRequest(""))
	require.ErrorIs(t, err, ErrNoAuthData)
}

func TestInfAuthenticate_MembershipAndAccessPermissions(t *testing.T) {
	a := infAuthenticator()

	token, err := jwt.GenerateWithMembership(7, 42, enum.MembershipRoleContributor, time.Hour, infSalt)
	require.NoError(t, err)
	session, err := a.Authenticate(infRequest("Bearer " + token))
	require.NoError(t, err)
	require.Equal(t, &auth.MembershipMetadata{SpaceID: 42, Role: enum.MembershipRoleContributor}, session.Metadata)

	lifetime := time.Hour
	perms := &jwt.SubClaimsAccessPermissions{
		Source:      jwt.OciSource,
		Permissions: []jwt.AccessPermissions{{SpaceID: 1, Permissions: []enum.Permission{enum.PermissionRepoView}}},
	}
	token, err = jwt.GenerateForTokenWithAccessPermissions(7, &lifetime, infSalt, perms)
	require.NoError(t, err)
	session, err = a.Authenticate(infRequest("Bearer " + token))
	require.NoError(t, err)
	meta, ok := session.Metadata.(*auth.AccessPermissionMetadata)
	require.True(t, ok)
	require.Equal(t, perms, meta.AccessPermissions)
}

func TestInfAuthenticate_Failures(t *testing.T) {
	a := infAuthenticator()

	t.Run("wrong salt", func(t *testing.T) {
		_, err := a.Authenticate(infRequest("Bearer " + infTokenJWT(t, 100, 7, "some-other-salt")))
		require.ErrorContains(t, err, "JWT verification failed")
	})
	t.Run("unknown principal", func(t *testing.T) {
		_, err := a.Authenticate(infRequest("Bearer " + infTokenJWT(t, 100, 12345, infSalt)))
		require.ErrorContains(t, err, "failed to get principal for token")
	})
	t.Run("not a jwt", func(t *testing.T) {
		_, err := a.Authenticate(infRequest("Bearer not-a-jwt"))
		require.ErrorContains(t, err, "failed to parse token format")
	})
	t.Run("non HMAC signing method", func(t *testing.T) {
		token := gojwt.NewWithClaims(gojwt.SigningMethodNone, jwt.Claims{PrincipalID: 7})
		str, err := token.SignedString(gojwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		_, err = a.Authenticate(infRequest("Bearer " + str))
		require.ErrorContains(t, err, "invalid signature method")
	})
	t.Run("token not in db", func(t *testing.T) {
		_, err := a.Authenticate(infRequest("Bearer " + infTokenJWT(t, 555, 7, infSalt)))
		require.ErrorContains(t, err, "failed to find token in db")
	})
	t.Run("token of a different principal", func(t *testing.T) {
		_, err := a.Authenticate(infRequest("Bearer " + infTokenJWT(t, 101, 7, infSalt)))
		require.ErrorContains(t, err, "JWT was for principal 7 while db token was for principal 99")
	})
	t.Run("missing sub claims", func(t *testing.T) {
		lifetime := time.Hour
		token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, jwt.Claims{
			RegisteredClaims: gojwt.RegisteredClaims{ExpiresAt: gojwt.NewNumericDate(time.Now().Add(lifetime))},
			PrincipalID:      7,
		})
		str, err := token.SignedString([]byte(infSalt))
		require.NoError(t, err)
		_, err = a.Authenticate(infRequest("Bearer " + str))
		require.ErrorContains(t, err, "jwt is missing sub-claims")
	})
	t.Run("expired token", func(t *testing.T) {
		token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, jwt.Claims{
			RegisteredClaims: gojwt.RegisteredClaims{ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-time.Hour))},
			PrincipalID:      7,
			Membership:       &jwt.SubClaimsMembership{SpaceID: 1, Role: enum.MembershipRoleReader},
		})
		str, err := token.SignedString([]byte(infSalt))
		require.NoError(t, err)
		_, err = a.Authenticate(infRequest("Bearer " + str))
		require.ErrorContains(t, err, "JWT verification failed")
	})
}

func TestInfAuthenticate_SaltRotation(t *testing.T) {
	a := infAuthenticator()
	for _, salt := range []string{"new-salt-0000", "old-salt-0000"} {
		token, err := jwt.GenerateWithMembership(8, 1, enum.MembershipRoleReader, time.Hour, salt)
		require.NoError(t, err)
		session, err := a.Authenticate(infRequest("Bearer " + token))
		require.NoError(t, err, salt)
		require.EqualValues(t, 8, session.Principal.ID)
	}
}

func TestInfAuthenticate_AnonymousPrincipal(t *testing.T) {
	a := infAuthenticator()
	token, err := jwt.GenerateWithMembership(-1, 1, enum.MembershipRoleReader, time.Hour, infAnonSalt)
	require.NoError(t, err)
	session, err := a.Authenticate(infRequest("Bearer " + token))
	require.NoError(t, err)
	require.True(t, auth.IsAnonymousSession(session))
	require.EqualValues(t, auth.AnonymousPrincipal.ID, session.Principal.ID)

	other, err := jwt.GenerateWithMembership(-1, 1, enum.MembershipRoleReader, time.Hour, infSalt)
	require.NoError(t, err)
	_, err = a.Authenticate(infRequest("Bearer " + other))
	require.Error(t, err, "anonymous tokens must be signed with the anonymous secret")
}

func TestInfExtractToken(t *testing.T) {
	basic := func(user, pwd string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api", nil)
		r.SetBasicAuth(user, pwd)
		return r
	}
	withCookie := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api", nil)
		r.AddCookie(&http.Cookie{Name: infCookie, Value: "cookie-token"})
		return r
	}
	tests := []struct {
		name string
		req  *http.Request
		want string
	}{
		{"query parameter wins", func() *http.Request {
			r := httptest.NewRequest(http.MethodGet, "/api?access_token=query-token", nil)
			r.Header.Set("Authorization", "Bearer header-token")
			return r
		}(), "query-token"},
		{"basic auth uses the password", basic("user", "pwd-token"), "pwd-token"},
		{"bearer", infRequest("Bearer bearer-token"), "bearer-token"},
		{"remote auth", infRequest("RemoteAuth remote-token"), "remote-token"},
		{"raw header value", infRequest("raw-token"), "raw-token"},
		{"cookie", withCookie(), "cookie-token"},
		{"nothing", infRequest(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, extractToken(tt.req, infCookie))
		})
	}
	require.Equal(t, "", extractToken(withCookie(), "other-cookie"))
}

func TestInfAuthenticate_ErrorsAreWrapped(t *testing.T) {
	a := NewTokenAuthenticator(&infPrincipalStore{}, &infTokenStore{}, infCookie, infAnonSalt)
	_, err := a.Authenticate(infRequest("Bearer " + infTokenJWT(t, 1, 7, infSalt)))
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNoAuthData))
}

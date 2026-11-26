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

package token

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/harness/gitness/app/jwt"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type infTokenStore struct {
	store.TokenStore

	created []*types.Token
	err     error
}

func (s *infTokenStore) Create(_ context.Context, t *types.Token) error {
	if s.err != nil {
		return s.err
	}
	t.ID = int64(len(s.created) + 100)
	s.created = append(s.created, t)
	return nil
}

func infParse(t *testing.T, token, salt string) *jwt.Claims {
	t.Helper()
	claims := &jwt.Claims{}
	_, err := gojwt.ParseWithClaims(token, claims, func(*gojwt.Token) (any, error) { return []byte(salt), nil })
	require.NoError(t, err)
	return claims
}

func infUser() *types.User {
	return &types.User{ID: 7, UID: "alice", Salt: "user-salt-0000"}
}

func TestInfCreateUserSession(t *testing.T) {
	ts := &infTokenStore{}
	tok, jwtStr, err := CreateUserSession(context.Background(), ts, infUser(), "login-session")
	require.NoError(t, err)

	require.Equal(t, enum.TokenTypeSession, tok.Type)
	require.Equal(t, "login-session", tok.Identifier)
	require.EqualValues(t, 7, tok.PrincipalID)
	require.EqualValues(t, 7, tok.CreatedBy)
	require.EqualValues(t, 100, tok.ID)
	require.NotNil(t, tok.ExpiresAt)
	require.WithinDuration(t, time.Now().Add(userSessionTokenLifeTime), time.UnixMilli(*tok.ExpiresAt), time.Minute)

	claims := infParse(t, jwtStr, "user-salt-0000")
	require.EqualValues(t, 100, claims.Token.ID)
	require.Equal(t, enum.TokenTypeSession, claims.Token.Type)
}

func TestInfCreatePAT(t *testing.T) {
	ts := &infTokenStore{}
	admin := &types.Principal{ID: 1, Salt: "admin-salt-0000"}

	t.Run("with lifetime", func(t *testing.T) {
		lifetime := time.Hour
		tok, jwtStr, err := CreatePAT(context.Background(), ts, admin, infUser(), "my-pat", &lifetime)
		require.NoError(t, err)
		require.Equal(t, enum.TokenTypePAT, tok.Type)
		require.EqualValues(t, 7, tok.PrincipalID, "token belongs to the user it was created for")
		require.EqualValues(t, 1, tok.CreatedBy)
		require.NotNil(t, tok.ExpiresAt)
		require.Equal(t, enum.TokenTypePAT, infParse(t, jwtStr, "user-salt-0000").Token.Type)
	})
	t.Run("without lifetime never expires", func(t *testing.T) {
		tok, _, err := CreatePAT(context.Background(), ts, admin, infUser(), "forever", nil)
		require.NoError(t, err)
		require.Nil(t, tok.ExpiresAt)
	})
}

func TestInfCreateSAT(t *testing.T) {
	ts := &infTokenStore{}
	sa := &types.ServiceAccount{ID: 9, Salt: "sa-salt-0000"}
	creator := &types.Principal{ID: 1}
	lifetime := time.Hour

	tok, jwtStr, err := CreateSAT(context.Background(), ts, creator, sa, "my-sat", &lifetime)
	require.NoError(t, err)
	require.Equal(t, enum.TokenTypeSAT, tok.Type)
	require.EqualValues(t, 9, tok.PrincipalID)
	require.EqualValues(t, 1, tok.CreatedBy)
	infParse(t, jwtStr, "sa-salt-0000")
}

func TestInfCreateRemoteAuthToken(t *testing.T) {
	ts := &infTokenStore{}
	principal := &types.Principal{ID: 3, Salt: "p-salt-0000"}

	tok, jwtStr, err := CreateRemoteAuthToken(context.Background(), ts, principal, "remote")
	require.NoError(t, err)
	require.Equal(t, enum.TokenTypeRemoteAuth, tok.Type)
	require.WithinDuration(t, time.Now().Add(RemoteAuthTokenLifeTime), time.UnixMilli(*tok.ExpiresAt), time.Minute)
	infParse(t, jwtStr, "p-salt-0000")
}

func TestInfCreate_Errors(t *testing.T) {
	t.Run("store failure", func(t *testing.T) {
		ts := &infTokenStore{err: errors.New("db down")}
		_, _, err := CreateUserSession(context.Background(), ts, infUser(), "x")
		require.ErrorContains(t, err, "failed to store token in db")
	})
	t.Run("missing salt", func(t *testing.T) {
		ts := &infTokenStore{}
		_, _, err := CreateUserSession(context.Background(), ts, &types.User{ID: 1}, "x")
		require.ErrorContains(t, err, "failed to create jwt token")
	})
}

func TestInfCreateUserWithAccessPermissions(t *testing.T) {
	perms := &jwt.SubClaimsAccessPermissions{
		Source:      jwt.OciSource,
		Permissions: []jwt.AccessPermissions{{SpaceID: 1, Permissions: []enum.Permission{enum.PermissionRepoView}}},
	}
	token, err := CreateUserWithAccessPermissions(infUser(), perms)
	require.NoError(t, err)

	claims := infParse(t, token, "user-salt-0000")
	require.EqualValues(t, 7, claims.PrincipalID)
	require.Equal(t, perms, claims.AccessPermissions)

	_, err = CreateUserWithAccessPermissions(&types.User{ID: 1}, perms)
	require.ErrorContains(t, err, "failed to create jwt token")
}

func TestInfGenerateIdentifier(t *testing.T) {
	id := GenerateIdentifier("pat")
	require.Regexp(t, regexp.MustCompile(`^pat-[0-9a-f]{8}-[0-9a-f]{4}$`), id)
	require.Regexp(t, regexp.MustCompile(`^-[0-9a-f]{8}-[0-9a-f]{4}$`), GenerateIdentifier(""))
}

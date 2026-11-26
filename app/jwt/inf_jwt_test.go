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

package jwt

import (
	"testing"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const infSecret = "test-secret-0000"

func infParse(t *testing.T, token, secret string) *Claims {
	t.Helper()
	claims := &Claims{}
	parsed, err := gojwt.ParseWithClaims(token, claims, func(*gojwt.Token) (any, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	require.Equal(t, gojwt.SigningMethodHS256, parsed.Method)
	return claims
}

func TestInfExtractFirstSecretFromList(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "single", in: "secret-a", want: "secret-a"},
		{name: "single with spaces", in: "  secret-a  ", want: "secret-a"},
		{name: "list takes the first", in: "secret-a,secret-b,secret-c", want: "secret-a"},
		{name: "list trims the first", in: " secret-a , secret-b", want: "secret-a"},
		{name: "empty", in: "", wantErr: true},
		{name: "blank", in: "   ", wantErr: true},
		{name: "empty first entry", in: ",secret-b", wantErr: true},
		{name: "blank first entry", in: "  ,secret-b", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractFirstSecretFromList(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestInfGenerateForToken(t *testing.T) {
	issued := time.Now().Add(-time.Minute).UnixMilli()
	expires := time.Now().Add(time.Hour).UnixMilli()

	t.Run("with expiry", func(t *testing.T) {
		token, err := GenerateForToken(&types.Token{
			ID: 5, PrincipalID: 7, Type: enum.TokenTypePAT, IssuedAt: issued, ExpiresAt: &expires,
		}, infSecret)
		require.NoError(t, err)

		claims := infParse(t, token, infSecret)
		require.EqualValues(t, 7, claims.PrincipalID)
		require.Equal(t, issuer, claims.Issuer)
		require.NotNil(t, claims.Token)
		require.EqualValues(t, 5, claims.Token.ID)
		require.Equal(t, enum.TokenTypePAT, claims.Token.Type)
		require.NotNil(t, claims.ExpiresAt)
		require.Equal(t, expires/1000, claims.ExpiresAt.Unix())
		require.Equal(t, issued/1000, claims.IssuedAt.Unix())
		require.Nil(t, claims.Membership)
		require.Nil(t, claims.AccessPermissions)
	})
	t.Run("without expiry", func(t *testing.T) {
		token, err := GenerateForToken(&types.Token{ID: 1, PrincipalID: 2, IssuedAt: issued}, infSecret)
		require.NoError(t, err)
		require.Nil(t, infParse(t, token, infSecret).ExpiresAt)
	})
	t.Run("secret rotation signs with the first secret", func(t *testing.T) {
		token, err := GenerateForToken(&types.Token{ID: 1, PrincipalID: 2, IssuedAt: issued},
			"new-secret-0000,old-secret-0000")
		require.NoError(t, err)
		infParse(t, token, "new-secret-0000")

		_, err = gojwt.ParseWithClaims(token, &Claims{}, func(*gojwt.Token) (any, error) {
			return []byte("old-secret-0000"), nil
		})
		require.Error(t, err)
	})
	t.Run("invalid secret", func(t *testing.T) {
		_, err := GenerateForToken(&types.Token{}, "")
		require.ErrorContains(t, err, "failed to get first secret")
	})
}

func TestInfGenerateWithMembership(t *testing.T) {
	token, err := GenerateWithMembership(7, 42, enum.MembershipRoleContributor, 2*time.Hour, infSecret)
	require.NoError(t, err)

	claims := infParse(t, token, infSecret)
	require.EqualValues(t, 7, claims.PrincipalID)
	require.NotNil(t, claims.Membership)
	require.EqualValues(t, 42, claims.Membership.SpaceID)
	require.Equal(t, enum.MembershipRoleContributor, claims.Membership.Role)
	require.Nil(t, claims.Token)
	require.WithinDuration(t, time.Now().Add(2*time.Hour), claims.ExpiresAt.Time, time.Minute)

	_, err = GenerateWithMembership(1, 1, enum.MembershipRoleReader, time.Hour, " ")
	require.ErrorContains(t, err, "failed to get first secret")
}

func TestInfGenerateForTokenWithAccessPermissions(t *testing.T) {
	lifetime := 30 * time.Minute
	perms := &SubClaimsAccessPermissions{
		Source: OciSource,
		Permissions: []AccessPermissions{
			{SpaceID: 3, Permissions: []enum.Permission{enum.PermissionRepoView}},
		},
	}

	token, err := GenerateForTokenWithAccessPermissions(7, &lifetime, infSecret, perms)
	require.NoError(t, err)

	claims := infParse(t, token, infSecret)
	require.EqualValues(t, 7, claims.PrincipalID)
	require.Equal(t, perms, claims.AccessPermissions)
	require.WithinDuration(t, time.Now().Add(lifetime), claims.ExpiresAt.Time, time.Minute)

	_, err = GenerateForTokenWithAccessPermissions(7, nil, infSecret, perms)
	require.ErrorContains(t, err, "token lifetime is required")

	_, err = GenerateForTokenWithAccessPermissions(7, &lifetime, "", perms)
	require.ErrorContains(t, err, "failed to get first secret")
}

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

package auth

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestInfAnonymousPrincipal(t *testing.T) {
	require.EqualValues(t, -1, AnonymousPrincipal.ID)
	require.Equal(t, types.AnonymousPrincipalUID, AnonymousPrincipal.UID)
	require.Equal(t, enum.PrincipalTypeUser, AnonymousPrincipal.Type)
	require.True(t, AnonymousPrincipal.IsAnonymous())
}

func TestInfIsAnonymousSession(t *testing.T) {
	require.False(t, IsAnonymousSession(nil))
	require.True(t, IsAnonymousSession(&Session{Principal: AnonymousPrincipal}))
	require.False(t, IsAnonymousSession(&Session{Principal: types.Principal{ID: 5, UID: "alice"}}))
}

func TestInfMetadataImpactOnAuthorization(t *testing.T) {
	require.False(t, (&EmptyMetadata{}).ImpactsAuthorization())
	require.False(t, (&TokenMetadata{TokenType: enum.TokenTypePAT, TokenID: 1}).ImpactsAuthorization())
	require.True(t, (&MembershipMetadata{SpaceID: 1, Role: enum.MembershipRoleReader}).ImpactsAuthorization())
	require.True(t, (&AccessPermissionMetadata{}).ImpactsAuthorization())
}

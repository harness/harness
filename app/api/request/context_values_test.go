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

package request

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestContextValues(t *testing.T) {
	ctx := t.Context()

	t.Run("empty context has no values", func(t *testing.T) {
		_, ok := AuthSessionFrom(ctx)
		require.False(t, ok)
		_, ok = PrincipalFrom(ctx)
		require.False(t, ok)
		_, ok = UserFrom(ctx)
		require.False(t, ok)
		_, ok = ServiceAccountFrom(ctx)
		require.False(t, ok)
		_, ok = SpaceFrom(ctx)
		require.False(t, ok)
		_, ok = RepoFrom(ctx)
		require.False(t, ok)
		_, ok = RequestIDFrom(ctx)
		require.False(t, ok)
	})

	t.Run("auth session and principal", func(t *testing.T) {
		session := &auth.Session{Principal: types.Principal{ID: 7, UID: "jane"}}
		withSession := WithAuthSession(ctx, session)

		got, ok := AuthSessionFrom(withSession)
		require.True(t, ok)
		require.Same(t, session, got)

		principal, ok := PrincipalFrom(withSession)
		require.True(t, ok)
		require.Equal(t, int64(7), principal.ID)
		require.Equal(t, "jane", principal.UID)

		var nilSession *auth.Session
		_, ok = AuthSessionFrom(WithAuthSession(ctx, nilSession))
		require.False(t, ok, "a nil session counts as missing")
	})

	t.Run("user", func(t *testing.T) {
		user := &types.User{ID: 1}
		got, ok := UserFrom(WithUser(ctx, user))
		require.True(t, ok)
		require.Same(t, user, got)

		var nilUser *types.User
		_, ok = UserFrom(WithUser(ctx, nilUser))
		require.False(t, ok)
	})

	t.Run("service account", func(t *testing.T) {
		sa := &types.ServiceAccount{ID: 2}
		got, ok := ServiceAccountFrom(WithServiceAccount(ctx, sa))
		require.True(t, ok)
		require.Same(t, sa, got)

		var nilSA *types.ServiceAccount
		_, ok = ServiceAccountFrom(WithServiceAccount(ctx, nilSA))
		require.False(t, ok)
	})

	t.Run("space and repo", func(t *testing.T) {
		space := &types.Space{ID: 3}
		repo := &types.Repository{ID: 4}
		both := WithRepo(WithSpace(ctx, space), repo)

		gotSpace, ok := SpaceFrom(both)
		require.True(t, ok)
		require.Same(t, space, gotSpace)
		gotRepo, ok := RepoFrom(both)
		require.True(t, ok)
		require.Same(t, repo, gotRepo)

		var nilSpace *types.Space
		_, ok = SpaceFrom(WithSpace(ctx, nilSpace))
		require.False(t, ok)
		var nilRepo *types.Repository
		_, ok = RepoFrom(WithRepo(ctx, nilRepo))
		require.False(t, ok)
	})

	t.Run("request id", func(t *testing.T) {
		got, ok := RequestIDFrom(WithRequestID(ctx, "req-1"))
		require.True(t, ok)
		require.Equal(t, "req-1", got)

		_, ok = RequestIDFrom(WithRequestID(ctx, ""))
		require.False(t, ok, "an empty id counts as missing")
	})

	t.Run("values are isolated per key", func(t *testing.T) {
		withID := WithRequestID(ctx, "req-1")
		_, ok := UserFrom(withID)
		require.False(t, ok)
		_, ok = RequestIDFrom(context.WithValue(ctx, "other", "value")) //nolint:staticcheck,revive
		require.False(t, ok)
	})
}

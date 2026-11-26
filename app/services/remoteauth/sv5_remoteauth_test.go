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

package remoteauth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5PrincipalStore struct {
	store.PrincipalStore

	principal *types.Principal
	err       error
}

func (s sv5PrincipalStore) Find(context.Context, int64) (*types.Principal, error) {
	return s.principal, s.err
}

type sv5TokenStore struct {
	store.TokenStore

	err     error
	created *types.Token
}

func (s *sv5TokenStore) Create(_ context.Context, t *types.Token) error {
	s.created = t
	return s.err
}

func TestSv5GenerateToken(t *testing.T) {
	ctx := context.Background()
	principal := &types.Principal{ID: 12, Salt: "salt-value"}

	t.Run("creates a remote auth token", func(t *testing.T) {
		tokens := &sv5TokenStore{}
		var svc Service = NewService(tokens, sv5PrincipalStore{principal: principal})

		jwt, err := svc.GenerateToken(ctx, 12, enum.PrincipalTypeUser, "resource")
		require.NoError(t, err)
		require.Len(t, strings.Split(jwt, "."), 3, "a signed JWT has three segments")

		require.NotNil(t, tokens.created)
		require.Equal(t, enum.TokenTypeRemoteAuth, tokens.created.Type)
		require.Equal(t, int64(12), tokens.created.PrincipalID)
		require.True(t, strings.HasPrefix(tokens.created.Identifier, "remoteAuth-"))
		require.NotNil(t, tokens.created.ExpiresAt)
	})

	t.Run("principal lookup failure", func(t *testing.T) {
		svc := NewService(&sv5TokenStore{}, sv5PrincipalStore{err: errSv5Boom})
		_, err := svc.GenerateToken(ctx, 12, enum.PrincipalTypeUser, "")
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "find principal 12")
	})

	t.Run("token store failure", func(t *testing.T) {
		svc := NewService(&sv5TokenStore{err: errSv5Boom}, sv5PrincipalStore{principal: principal})
		_, err := svc.GenerateToken(ctx, 12, enum.PrincipalTypeUser, "")
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "remote auth token")
	})
}

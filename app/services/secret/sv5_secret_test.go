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

package secret

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

type sv5Cache[K comparable, V any] struct {
	values map[K]V
}

func (sv5Cache[K, V]) Stats() (int64, int64)    { return 0, 0 }
func (sv5Cache[K, V]) Evict(context.Context, K) {}

func (c sv5Cache[K, V]) Get(_ context.Context, key K) (V, error) {
	v, ok := c.values[key]
	if !ok {
		var zero V
		return zero, errSv5Boom
	}
	return v, nil
}

type sv5SecretStore struct {
	store.SecretStore

	secret *types.Secret
	err    error
}

func (s sv5SecretStore) FindByIdentifier(_ context.Context, spaceID int64, identifier string) (*types.Secret, error) {
	if s.err != nil {
		return nil, s.err
	}
	if spaceID != 5 || identifier != "token" {
		return nil, errSv5Boom
	}
	return s.secret, nil
}

type sv5Encrypter struct{ err error }

func (e sv5Encrypter) Encrypt(p string) ([]byte, error) { return []byte(p), nil }

func (e sv5Encrypter) Decrypt(c []byte) (string, error) {
	if e.err != nil {
		return "", e.err
	}
	return "plain:" + string(c), nil
}

func sv5SpaceFinder() refcache.SpaceFinder {
	return refcache.NewSpaceFinder(
		sv5Cache[int64, *types.SpaceCore]{values: map[int64]*types.SpaceCore{5: {ID: 5}}},
		sv5Cache[string, *types.SpacePath]{values: map[string]*types.SpacePath{"acct/org": {SpaceID: 5, Value: "acct/org"}}},
		nil,
		cache.Evictor[*types.SpaceCore]{},
	)
}

func TestSv5DecryptSecret(t *testing.T) {
	ctx := context.Background()
	stored := &types.Secret{Identifier: "token", Data: "cipher"}

	t.Run("decrypts the stored secret", func(t *testing.T) {
		svc := NewService(sv5SecretStore{secret: stored}, sv5Encrypter{}, sv5SpaceFinder())
		got, err := svc.DecryptSecret(ctx, "acct/org", "token")
		require.NoError(t, err)
		require.Equal(t, "plain:cipher", got)
		require.Equal(t, "cipher", stored.Data, "the stored secret is not modified")
	})

	t.Run("space not found", func(t *testing.T) {
		svc := NewService(sv5SecretStore{secret: stored}, sv5Encrypter{}, sv5SpaceFinder())
		_, err := svc.DecryptSecret(ctx, "unknown/space", "token")
		require.ErrorContains(t, err, "failed to find space path")
	})

	t.Run("secret not found", func(t *testing.T) {
		svc := NewService(sv5SecretStore{err: errSv5Boom}, sv5Encrypter{}, sv5SpaceFinder())
		_, err := svc.DecryptSecret(ctx, "acct/org", "token")
		require.ErrorContains(t, err, "failed to find secret")
		require.ErrorIs(t, err, errSv5Boom)
	})

	t.Run("decryption failure", func(t *testing.T) {
		svc := NewService(sv5SecretStore{secret: stored}, sv5Encrypter{err: errSv5Boom}, sv5SpaceFinder())
		_, err := svc.DecryptSecret(ctx, "acct/org", "token")
		require.ErrorContains(t, err, "failed to decrypt secret")
	})
}

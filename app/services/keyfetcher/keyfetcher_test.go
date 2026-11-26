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

package keyfetcher

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type fakePublicKeyStore struct {
	store.PublicKeyStore

	keys []types.PublicKey
	err  error

	gotFingerprint string
	gotSubKeyID    string
	gotPrincipalID *int64
	gotUsages      []enum.PublicKeyUsage
	gotSchemes     []enum.PublicKeyScheme
}

func (f *fakePublicKeyStore) ListByFingerprint(
	_ context.Context, fingerprint string, principalID *int64,
	usages []enum.PublicKeyUsage, schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.gotFingerprint, f.gotPrincipalID, f.gotUsages, f.gotSchemes = fingerprint, principalID, usages, schemes
	return f.keys, f.err
}

func (f *fakePublicKeyStore) ListBySubKeyID(
	_ context.Context, subKeyID string, principalID *int64,
	usages []enum.PublicKeyUsage, schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	f.gotSubKeyID, f.gotPrincipalID, f.gotUsages, f.gotSchemes = subKeyID, principalID, usages, schemes
	return f.keys, f.err
}

func TestService_FetchByFingerprint(t *testing.T) {
	usages := []enum.PublicKeyUsage{enum.PublicKeyUsageSign}
	schemes := []enum.PublicKeyScheme{enum.PublicKeySchemeSSH}

	t.Run("passes the filters to the store", func(t *testing.T) {
		fake := &fakePublicKeyStore{keys: []types.PublicKey{{Identifier: "k"}}}
		keys, err := NewService(fake).FetchByFingerprint(t.Context(), "SHA256:abc", 7, usages, schemes)
		require.NoError(t, err)
		require.Equal(t, []types.PublicKey{{Identifier: "k"}}, keys)
		require.Equal(t, "SHA256:abc", fake.gotFingerprint)
		require.Equal(t, int64(7), *fake.gotPrincipalID)
		require.Equal(t, usages, fake.gotUsages)
		require.Equal(t, schemes, fake.gotSchemes)
	})

	t.Run("wraps store errors", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := NewService(&fakePublicKeyStore{err: boom}).FetchByFingerprint(t.Context(), "x", 1, usages, schemes)
		require.ErrorIs(t, err, boom)
	})
}

func TestService_FetchBySubKeyID(t *testing.T) {
	usages := []enum.PublicKeyUsage{enum.PublicKeyUsageSign}
	schemes := []enum.PublicKeyScheme{enum.PublicKeySchemePGP}

	t.Run("passes the filters to the store", func(t *testing.T) {
		fake := &fakePublicKeyStore{keys: []types.PublicKey{{Identifier: "k"}}}
		keys, err := NewService(fake).FetchBySubKeyID(t.Context(), "ABCD", 9, usages, schemes)
		require.NoError(t, err)
		require.Len(t, keys, 1)
		require.Equal(t, "ABCD", fake.gotSubKeyID)
		require.Equal(t, int64(9), *fake.gotPrincipalID)
		require.Equal(t, schemes, fake.gotSchemes)
	})

	t.Run("wraps store errors", func(t *testing.T) {
		boom := errors.New("boom")
		_, err := NewService(&fakePublicKeyStore{err: boom}).FetchBySubKeyID(t.Context(), "x", 1, usages, schemes)
		require.ErrorIs(t, err, boom)
	})
}

func TestProvideService(t *testing.T) {
	require.NotNil(t, ProvideService(&fakePublicKeyStore{}))
}

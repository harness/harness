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

package keyfetcher

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type svcPublicKeyStore struct {
	store.PublicKeyStore
	keys         []types.PublicKey
	err          error
	gotKey       string
	gotPrincipal *int64
	gotUsages    []enum.PublicKeyUsage
	gotSchemes   []enum.PublicKeyScheme
	bySubKey     bool
}

func (s *svcPublicKeyStore) ListByFingerprint(
	_ context.Context, fp string, principalID *int64, usages []enum.PublicKeyUsage, schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	s.gotKey, s.gotPrincipal, s.gotUsages, s.gotSchemes = fp, principalID, usages, schemes
	return s.keys, s.err
}

func (s *svcPublicKeyStore) ListBySubKeyID(
	_ context.Context, id string, principalID *int64, usages []enum.PublicKeyUsage, schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	s.bySubKey = true
	s.gotKey, s.gotPrincipal, s.gotUsages, s.gotSchemes = id, principalID, usages, schemes
	return s.keys, s.err
}

func TestSvcFetchers(t *testing.T) {
	ctx := context.Background()
	usages := []enum.PublicKeyUsage{enum.PublicKeyUsageSign}
	schemes := []enum.PublicKeyScheme{enum.PublicKeySchemePGP}

	t.Run("by fingerprint", func(t *testing.T) {
		st := &svcPublicKeyStore{keys: []types.PublicKey{{ID: 1}}}
		svc := ProvideService(st)

		keys, err := svc.FetchByFingerprint(ctx, "fp", 7, usages, schemes)
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.False(t, st.bySubKey)
		assert.Equal(t, "fp", st.gotKey)
		require.NotNil(t, st.gotPrincipal)
		assert.Equal(t, int64(7), *st.gotPrincipal)
		assert.Equal(t, usages, st.gotUsages)
		assert.Equal(t, schemes, st.gotSchemes)
	})
	t.Run("by sub key id", func(t *testing.T) {
		st := &svcPublicKeyStore{keys: []types.PublicKey{{ID: 2}}}
		keys, err := NewService(st).FetchBySubKeyID(ctx, "ABCD", 8, usages, schemes)
		require.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.True(t, st.bySubKey)
		assert.Equal(t, "ABCD", st.gotKey)
		assert.Equal(t, int64(8), *st.gotPrincipal)
	})
	t.Run("store errors are wrapped", func(t *testing.T) {
		boom := errors.New("db down")
		svc := NewService(&svcPublicKeyStore{err: boom})
		_, err := svc.FetchByFingerprint(ctx, "fp", 1, usages, schemes)
		require.ErrorIs(t, err, boom)
		_, err = svc.FetchBySubKeyID(ctx, "id", 1, usages, schemes)
		require.ErrorIs(t, err, boom)
	})
}

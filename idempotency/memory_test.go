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

// Package idempotency provides stores that remember which operations were already performed.
package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_ClaimExpires(t *testing.T) {
	t.Parallel()

	store := NewInMemory()

	state, err := store.Claim(context.Background(), "k", 10*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)

	state, err = store.Claim(context.Background(), "k", 10*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, StateInFlight, state)

	time.Sleep(20 * time.Millisecond)

	state, err = store.Claim(context.Background(), "k", 10*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
}

func TestInMemoryStore_ExpiredClaimCanBeTakenBeforeSweep(t *testing.T) {
	t.Parallel()

	store := NewInMemory()
	ctx := context.Background()

	// The first call sweeps; the next ones within the sweep interval must still honor expiry per key.
	state, err := store.Claim(ctx, "other", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
	state, err = store.Claim(ctx, "k", 10*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)

	time.Sleep(20 * time.Millisecond)

	state, err = store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
}

// A committed key is told apart from a held claim.
func TestInMemoryStore_CommittedKeyIsDoneAndClaimedKeyIsInFlight(t *testing.T) {
	t.Parallel()

	store := NewInMemory()
	ctx := context.Background()

	state, err := store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)

	state, err = store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateInFlight, state)

	require.NoError(t, store.Commit(ctx, "k", time.Hour))

	state, err = store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateDone, state)
}

// Committing works even if the claim already expired.
func TestInMemoryStore_CommitWorksWithoutAClaim(t *testing.T) {
	t.Parallel()

	store := NewInMemory()
	ctx := context.Background()

	require.NoError(t, store.Commit(ctx, "k", time.Hour))

	state, err := store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateDone, state)
}

func TestInMemoryStore_NonPositiveTTLIsRejected(t *testing.T) {
	t.Parallel()

	store := NewInMemory()
	ctx := context.Background()

	_, err := store.Claim(ctx, "k", 0)
	require.ErrorIs(t, err, ErrInvalidTTL)

	_, err = store.Claim(ctx, "k", -time.Second)
	require.ErrorIs(t, err, ErrInvalidTTL)

	require.ErrorIs(t, store.Commit(ctx, "k", 0), ErrInvalidTTL)

	// A rejected claim must not have stored anything.
	state, err := store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
}

func TestInMemoryStore_ReleaseAllowsClaimingAgain(t *testing.T) {
	t.Parallel()

	store := NewInMemory()
	ctx := context.Background()

	_, err := store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.Release(ctx, "k"))

	state, err := store.Claim(ctx, "k", time.Hour)
	require.NoError(t, err)
	require.Equal(t, StateClaimed, state)
}

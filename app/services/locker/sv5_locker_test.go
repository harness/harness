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

package locker

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/lock"

	"github.com/stretchr/testify/require"
)

func sv5NewLocker() *Locker {
	return NewLocker(lock.NewInMemory(lock.Config{
		App: "test", Expiry: 10 * time.Second, Tries: 2, RetryDelay: 10 * time.Millisecond,
		DriftFactor: 0.01, TimeoutFactor: 0.25,
	}))
}

func TestSv5LockPR(t *testing.T) {
	ctx := context.Background()
	l := sv5NewLocker()

	unlock, err := l.LockPR(ctx, 1, 7, time.Minute)
	require.NoError(t, err)

	_, err = l.LockPR(ctx, 1, 7, time.Minute)
	require.ErrorContains(t, err, "failed to lock mutex for pr 7 in repo 1")

	// Other pull requests and other repositories are independent.
	other, err := l.LockPR(ctx, 1, 8, time.Minute)
	require.NoError(t, err)
	other()
	otherRepo, err := l.LockPR(ctx, 2, 7, time.Minute)
	require.NoError(t, err)
	otherRepo()

	unlock()
	again, err := l.LockPR(ctx, 1, 7, time.Minute)
	require.NoError(t, err)
	again()
}

func TestSv5LockPRWithoutNumberLocksRepoPulls(t *testing.T) {
	ctx := context.Background()
	l := sv5NewLocker()

	unlock, err := l.LockPR(ctx, 3, 0, time.Minute)
	require.NoError(t, err)
	defer unlock()

	_, err = l.LockPR(ctx, 3, 0, time.Minute)
	require.ErrorContains(t, err, "pr 0 in repo 3")
}

func TestSv5LockDefaultBranch(t *testing.T) {
	ctx := context.Background()
	l := sv5NewLocker()

	unlock, err := l.LockDefaultBranch(ctx, 4, "trunk", time.Minute)
	require.NoError(t, err)

	_, err = l.LockDefaultBranch(ctx, 4, "other", time.Minute)
	require.ErrorContains(t, err, "update default branch to other")

	unlock()
	again, err := l.LockDefaultBranch(ctx, 4, "other", time.Minute)
	require.NoError(t, err)
	again()
}

func TestSv5LockResource(t *testing.T) {
	ctx := context.Background()
	l := sv5NewLocker()

	unlock, err := l.LockResource(ctx, "registry/1", time.Minute)
	require.NoError(t, err)

	_, err = l.LockResource(ctx, "registry/1", time.Minute)
	require.ErrorContains(t, err, "failed to lock mutex for key [registry/1]")

	unlock()
	again, err := l.LockResource(ctx, "registry/1", time.Minute)
	require.NoError(t, err)
	again()
}

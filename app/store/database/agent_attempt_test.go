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

package database_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestAgentAttemptStoreConcurrentMaxActive(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAgentAttemptStore(db, dbtx.New(db))

	const (
		workers   = 20
		maxActive = 3
	)
	now := time.Now()
	var started, denied atomic.Int32
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, deny, err := s.TryStart(ctx, types.AgentAttemptStart{
				RepoID: 1, TaskKey: "task-" + string(rune('a'+i)), MaxActive: maxActive,
				Cooldown: time.Minute, MaxFailed: 3, At: now,
			})
			if err != nil {
				t.Errorf("TryStart: %v", err)
				return
			}
			if deny == "" && id != 0 {
				started.Add(1)
			} else if deny == types.AgentAttemptDenyMaxActive {
				denied.Add(1)
			}
		}()
	}
	wg.Wait()

	require.EqualValues(t, maxActive, started.Load())
	require.EqualValues(t, workers-maxActive, denied.Load())
	n, err := s.CountActive(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, maxActive, n)
}

func TestAgentAttemptStoreRules(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	s := database.NewAgentAttemptStore(db, dbtx.New(db))

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	start := func(repo int64, key string, maxActive int, at time.Time) (int64, string) {
		id, deny, err := s.TryStart(ctx, types.AgentAttemptStart{
			RepoID: repo, TaskKey: key, MaxActive: maxActive,
			Cooldown: 30 * time.Minute, MaxFailed: 2, At: at,
		})
		require.NoError(t, err)
		return id, deny
	}

	// cooldown: second attempt of the same task is refused until the cooldown elapsed
	id1, deny := start(1, "t", 5, now)
	require.Empty(t, deny)
	require.NotZero(t, id1)
	require.NoError(t, s.Finish(ctx, id1, types.AgentAttemptFailed, "boom"))

	id, deny := start(1, "t", 5, now.Add(10*time.Minute))
	require.Zero(t, id)
	require.Equal(t, types.AgentAttemptDenyCooldown, deny)

	id2, deny := start(1, "t", 5, now.Add(31*time.Minute))
	require.Empty(t, deny)
	require.NoError(t, s.Finish(ctx, id2, types.AgentAttemptFailed, "boom"))

	// failed count: two failures reach MaxFailed
	id, deny = start(1, "t", 5, now.Add(2*time.Hour))
	require.Zero(t, id)
	require.Equal(t, types.AgentAttemptDenyTooManyFailures, deny)

	// other tasks and other repos are independent; max active is per repo
	idA, deny := start(1, "other", 1, now)
	require.Empty(t, deny)
	_, deny = start(1, "other2", 1, now)
	require.Equal(t, types.AgentAttemptDenyMaxActive, deny)
	_, deny = start(2, "other3", 1, now)
	require.Empty(t, deny)

	// finishing frees the slot
	require.NoError(t, s.Finish(ctx, idA, types.AgentAttemptSucceeded, ""))
	_, deny = start(1, "other2", 1, now)
	require.Empty(t, deny)

	list, err := s.ListByTask(ctx, "t")
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, types.AgentAttemptFailed, list[0].State)
	require.Equal(t, "boom", list[0].Error)
	require.NotNil(t, list[0].Finished)
	require.Equal(t, now.UnixMilli(), list[0].Started)

	require.ErrorIs(t, s.Finish(ctx, 9999, types.AgentAttemptFailed, ""), gitnessstore.ErrResourceNotFound)
	require.Error(t, s.Finish(ctx, id1, types.AgentAttemptRunning, ""))
}

func TestAgentAttemptStorePersistsAcrossStoreInstances(t *testing.T) {
	db, teardown := setupDB(t)
	defer teardown()
	ctx := context.Background()
	now := time.Now()

	first := database.NewAgentAttemptStore(db, dbtx.New(db))
	id, deny, err := first.TryStart(ctx, types.AgentAttemptStart{
		RepoID: 7, TaskKey: "t", MaxActive: 1, Cooldown: time.Hour, MaxFailed: 3, At: now,
	})
	require.NoError(t, err)
	require.Empty(t, deny)
	require.NotZero(t, id)

	// a new store on the same database (as after a restart) still sees the running attempt
	second := database.NewAgentAttemptStore(db, dbtx.New(db))
	n, err := second.CountActive(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	_, deny, err = second.TryStart(ctx, types.AgentAttemptStart{
		RepoID: 7, TaskKey: "u", MaxActive: 1, Cooldown: time.Hour, MaxFailed: 3, At: now,
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentAttemptDenyMaxActive, deny)
}

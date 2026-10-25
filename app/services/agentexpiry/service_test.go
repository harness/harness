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

package agentexpiry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type fakeAttempts struct {
	olderThan, now int64
	out            []*types.AgentAttempt
	err            error
}

func (f *fakeAttempts) ExpireRunning(_ context.Context, olderThan, now int64) ([]*types.AgentAttempt, error) {
	f.olderThan, f.now = olderThan, now
	return f.out, f.err
}

type fakeTasks struct {
	runs      map[int64]task.AgentRunInput
	blocked   []int64
	recordErr error
}

func (f *fakeTasks) RecordAgentRun(_ context.Context, _, id int64, in task.AgentRunInput) error {
	if f.runs == nil {
		f.runs = map[int64]task.AgentRunInput{}
	}
	f.runs[id] = in
	return f.recordErr
}

func (f *fakeTasks) Transition(_ context.Context, _, id int64, to enum.TaskState, _ string) (*types.Task, error) {
	if to != enum.TaskStateBlocked {
		return nil, errors.New("unexpected state")
	}
	f.blocked = append(f.blocked, id)
	return nil, nil
}

func newMutexes() lock.MutexManager {
	return lock.NewInMemory(lock.Config{App: "t", Expiry: time.Minute, Tries: 1, RetryDelay: time.Millisecond})
}

func TestRunExpiresAndBlocksTasks(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	at := &fakeAttempts{out: []*types.AgentAttempt{
		{ID: 1, TaskKey: "task:7"},
		{ID: 2, TaskKey: "shield:abc"},
		{ID: 3, TaskKey: "task:x"},
	}}
	ts := &fakeTasks{recordErr: errors.New("boom")} // errors are best effort
	s := New(at, ts, newMutexes(), 2*time.Hour)
	s.now = func() time.Time { return now }
	s.principal = func() int64 { return 1 }

	res, err := s.Run(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, res.Expired)
	require.Equal(t, now.Add(-2*time.Hour).UnixMilli(), at.olderThan)
	require.Equal(t, now.UnixMilli(), at.now)
	require.Equal(t, []int64{7}, ts.blocked)
	require.Equal(t, OutcomeExpired, ts.runs[7].Outcome)
	require.Len(t, ts.runs, 1)
}

func TestRunSkippedWhenLocked(t *testing.T) {
	m := newMutexes()
	mx, err := m.NewMutex(lockKey)
	require.NoError(t, err)
	require.NoError(t, mx.Lock(context.Background()))

	at := &fakeAttempts{out: []*types.AgentAttempt{{ID: 1, TaskKey: "task:1"}}}
	res, err := New(at, &fakeTasks{}, m, time.Hour).Run(context.Background())
	require.NoError(t, err)
	require.True(t, res.Skipped)
	require.Zero(t, at.now)
}

func TestRunDisabledWithoutTimeout(t *testing.T) {
	at := &fakeAttempts{}
	res, err := New(at, &fakeTasks{}, newMutexes(), 0).Run(context.Background())
	require.NoError(t, err)
	require.True(t, res.Skipped)
	require.Zero(t, at.now)
}

func TestRunStoreError(t *testing.T) {
	at := &fakeAttempts{err: errors.New("db down")}
	_, err := New(at, &fakeTasks{}, newMutexes(), time.Hour).Run(context.Background())
	require.Error(t, err)
}
